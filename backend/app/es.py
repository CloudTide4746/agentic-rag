"""Elasticsearch 客户端 — kb_chunks 索引（dense_vector kNN + BM25 双路检索）

单索引承载全部知识库切片，kb_id 字段隔离多库（设计书 6.1）。
IK 分词插件未安装时自动退回 standard 分词器。
"""
import json
import logging

import httpx

from .config import settings

logger = logging.getLogger("agentic.es")

INDEX = "kb_chunks"
_SOURCE_FIELDS = ["kb_id", "doc_id", "chunk_uid", "doc_name", "section_path", "chunk_no", "content"]


class Elastic:
    def __init__(self) -> None:
        auth = (settings.es_user, settings.es_password) if settings.es_user else None
        self.client = httpx.AsyncClient(base_url=settings.es_url, auth=auth, timeout=30)

    async def close(self) -> None:
        await self.client.aclose()

    async def probe_analyzer(self) -> str:
        try:
            r = await self.client.post("/_analyze", json={"analyzer": "ik_max_word", "text": "中文分词测试"})
            if r.status_code == 200:
                return "ik_max_word"
        except httpx.HTTPError:
            pass
        logger.info("IK 插件不可用，content/section_path 退回 standard 分词器")
        return "standard"

    async def ensure_index(self) -> None:
        r = await self.client.head(f"/{INDEX}")
        if r.status_code == 200:
            return
        analyzer = await self.probe_analyzer()
        mapping = {
            "settings": {"number_of_shards": 1, "number_of_replicas": 0},
            "mappings": {
                "properties": {
                    "kb_id": {"type": "integer"},
                    "doc_id": {"type": "long"},
                    "chunk_uid": {"type": "keyword"},
                    "doc_name": {"type": "keyword", "ignore_above": 256},
                    "section_path": {"type": "text", "analyzer": analyzer},
                    "chunk_no": {"type": "integer"},
                    "content": {"type": "text", "analyzer": analyzer},
                    "embedding": {
                        "type": "dense_vector",
                        "dims": settings.embed_dims,
                        "similarity": "cosine",
                        "index": True,
                        "index_options": {"type": "hnsw", "m": 16, "ef_construction": 100},
                    },
                    "created_at": {"type": "date"},
                }
            },
        }
        r = await self.client.put(f"/{INDEX}", json=mapping)
        r.raise_for_status()
        logger.info("已创建 ES 索引 %s（analyzer=%s）", INDEX, analyzer)

    @staticmethod
    def _kb_filter(kb_ids: list[int] | None) -> dict:
        if kb_ids:
            return {"term": {"kb_id": kb_ids[0]}} if len(kb_ids) == 1 else {"terms": {"kb_id": kb_ids}}
        return {"match_all": {}}

    async def knn_search(self, query_vector: list[float], kb_ids: list[int] | None, k: int = 5) -> list[dict]:
        """语义检索：顶层 knn + kb_id 过滤，_score = (1+cosine)/2"""
        body = {
            "size": k,
            "_source": _SOURCE_FIELDS,
            "query": {"bool": {"filter": self._kb_filter(kb_ids)}},
            "knn": {
                "field": "embedding",
                "query_vector": query_vector,
                "k": k,
                "num_candidates": 100,
            },
        }
        r = await self.client.post(f"/{INDEX}/_search", json=body)
        r.raise_for_status()
        hits = r.json().get("hits", {}).get("hits", [])
        return [{"score": h["_score"] or 0.0, **h["_source"]} for h in hits]

    async def match_search(self, query: str, kb_ids: list[int] | None, k: int = 5) -> list[dict]:
        """关键词检索：BM25 match（content 主字段 + section_path 加权）"""
        body = {
            "size": k,
            "_source": _SOURCE_FIELDS,
            "query": {
                "bool": {
                    "must": {
                        "multi_match": {"query": query, "fields": ["content^1", "section_path^2"], "type": "best_fields"}
                    },
                    "filter": self._kb_filter(kb_ids),
                }
            },
        }
        r = await self.client.post(f"/{INDEX}/_search", json=body)
        r.raise_for_status()
        hits = r.json().get("hits", {}).get("hits", [])
        return [{"score": h["_score"] or 0.0, **h["_source"]} for h in hits]

    async def bulk_index(self, docs: list[dict]) -> None:
        lines: list[str] = []
        for d in docs:
            lines.append('{"index": {"_index": "%s", "_id": "%s"}}' % (INDEX, d["chunk_uid"]))
            lines.append(json.dumps(d, ensure_ascii=False))
        body = "\n".join(lines) + "\n"
        r = await self.client.post("/_bulk", content=body.encode("utf-8"), headers={"Content-Type": "application/x-ndjson"})
        r.raise_for_status()
        resp = r.json()
        if resp.get("errors"):
            raise RuntimeError(f"ES bulk 写入存在失败项：{str(resp)[:200]}")

    async def delete_by_doc(self, doc_id: int) -> None:
        """删除文档：先 ES 后 MySQL（设计书 4.2 一致性策略）"""
        r = await self.client.post(
            f"/{INDEX}/_delete_by_query",
            json={"query": {"term": {"doc_id": doc_id}}, "conflicts": "proceed"},
            params={"refresh": "true"},
        )
        r.raise_for_status()

    async def count(self) -> int:
        r = await self.client.get(f"/{INDEX}/_count")
        r.raise_for_status()
        return int(r.json().get("count", 0))

    async def ping(self) -> bool:
        try:
            r = await self.client.get("/")
            return r.status_code == 200
        except httpx.HTTPError:
            return False


elastic = Elastic()
