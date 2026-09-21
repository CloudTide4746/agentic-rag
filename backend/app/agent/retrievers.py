"""检索适配层 — 三类 Retriever 实现统一 search(query, top_k) -> list[RetrieverResult]

新增数据源零侵入（设计书 2.2）：VectorRetriever / KeywordRetriever 走 ES，
DBRetriever 走 NL2SQL 只读查询。
"""
import asyncio
import logging
import re
from typing import Any

from pydantic import BaseModel

from .. import db
from ..es import elastic
from . import prompts
from .embeddings import embed_query
from .llm import judge
from .rrf import RetrieverResult

logger = logging.getLogger("agentic.retrievers")


class SqlOut(BaseModel):
    sql: str = ""


def _es_hit_to_result(source: str, hit: dict) -> RetrieverResult:
    doc_name = hit.get("doc_name", "")
    section = hit.get("section_path", "")
    chunk_no = int(hit.get("chunk_no", 0))
    return RetrieverResult(
        source=source,
        score=round(float(hit.get("score", 0.0)), 4),
        title=f"{doc_name} · {section}" if section else doc_name,
        content=hit.get("content", ""),
        metadata={
            "doc_name": doc_name,
            "section_path": section,
            "chunk_no": chunk_no,
            "doc_id": hit.get("doc_id"),
        },
    )


class VectorRetriever:
    """语义检索：百炼向量化（query 侧）→ ES 顶层 kNN"""

    async def search(self, query: str, kb_ids: list[int] | None, top_k: int = 5) -> list[RetrieverResult]:
        vector = await embed_query(query)
        hits = await elastic.knn_search(vector, kb_ids, k=top_k)
        return [_es_hit_to_result("vector", h) for h in hits]


class KeywordRetriever:
    """关键词检索：ES BM25 multi_match（content + section_path 加权）"""

    async def search(self, query: str, kb_ids: list[int] | None, top_k: int = 5) -> list[RetrieverResult]:
        hits = await elastic.match_search(query, kb_ids, k=top_k)
        return [_es_hit_to_result("keyword", h) for h in hits]


_FORBIDDEN = re.compile(
    r"(insert|update|delete|drop|alter|create|truncate|grant|revoke|replace|merge|call|lock|;|--|/\*)",
    re.IGNORECASE,
)


def validate_select(sql: str) -> str:
    """SELECT 白名单校验 + LIMIT 20 截断（设计书 9 安全）"""
    sql = sql.strip().rstrip(";").strip()
    if not re.match(r"^select\s", sql, re.IGNORECASE):
        raise ValueError("仅允许 SELECT 查询")
    if _FORBIDDEN.search(sql):
        raise ValueError("SQL 含危险关键字，已拦截")
    m = re.search(r"limit\s+(\d+)", sql, re.IGNORECASE)
    if m:
        if int(m.group(1)) > 20:
            sql = sql[: m.start()] + f"LIMIT 20"
    else:
        sql += " LIMIT 20"
    return sql


class DBRetriever:
    """业务数据查询：LLM 按 sales Schema 生成 SELECT → 只读账号执行（行数 ≤20）"""

    async def search(self, natural_question: str, top_k: int = 5) -> tuple[str, list[RetrieverResult]]:
        out = await judge(SqlOut, prompts.NL2SQL_SYS, prompts.nl2sql_user(natural_question))
        sql = validate_select(out.sql)
        rows = await db.aquery_all(sql)
        results: list[RetrieverResult] = []
        for i, row in enumerate(rows[:20]):
            product = str(row.get("product", row.get("region", "记录")))
            quarter = str(row.get("quarter", ""))
            region = row.get("region", "")
            snippet = "，".join(f"{k}={v}" for k, v in row.items() if k != "id")
            results.append(
                RetrieverResult(
                    source="database",
                    score=round(1.0 - i * 0.01, 2),
                    title=f"{product} · {quarter}".strip(" ·"),
                    content=f"业务数据库 sales 表查询结果：{snippet}" + (f"（{region}区）" if region else ""),
                    # 引用溯源需要 SQL + 原始行作为证据（rows 已过 _serialize，可 JSON 落库）
                    metadata={"db": True, "sql": sql, "rows": rows[:20]},
                )
            )
        return sql, results


vector_retriever = VectorRetriever()
keyword_retriever = KeywordRetriever()
db_retriever = DBRetriever()


async def enabled_sources() -> dict[str, bool]:
    """读取 datasource_config 生效的检索源集合（M2 → Q3 裁剪）"""
    rows = await db.aquery_all("SELECT source_type, enabled FROM datasource_config")
    mapping = {r["source_type"]: bool(r["enabled"]) for r in rows}
    return {
        "vector": mapping.get("vector", True),
        "keyword": mapping.get("keyword", True),
        "database": mapping.get("database", True),
    }


async def gather_lanes(
    query: str, kb_ids: list[int] | None, top_k: int, sources: list[str]
) -> dict[str, list[RetrieverResult]]:
    """按启用的检索源并行召回（gather + return_exceptions，单路失败不影响其余路）"""
    runners: dict[str, Any] = {}
    if "vector" in sources:
        runners["vector"] = vector_retriever.search(query, kb_ids, top_k)
    if "keyword" in sources:
        runners["keyword"] = keyword_retriever.search(query, kb_ids, top_k)
    keys = list(runners.keys())
    results = await asyncio.gather(*runners.values(), return_exceptions=True)
    lanes: dict[str, list[RetrieverResult]] = {}
    for key, res in zip(keys, results):
        if isinstance(res, BaseException):
            logger.warning("检索路 %s 失败：%s", key, res)
            lanes[key] = []
        else:
            lanes[key] = res
    return lanes
