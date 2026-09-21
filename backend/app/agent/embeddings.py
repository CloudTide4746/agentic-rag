"""Embedding 客户端 — 阿里云百炼 text-embedding-v4（1024 维，OpenAI 兼容接口）

批次 ≤10 条、单文本 ≤8192 token（设计书 2.3），失败重试 1 次。
"""
import logging

import httpx

from ..config import settings

logger = logging.getLogger("agentic.embeddings")

_BATCH = 10


async def _embed_batch(client: httpx.AsyncClient, texts: list[str]) -> list[list[float]]:
    resp = await client.post(
        "/embeddings",
        json={
            "model": settings.embed_model,
            "input": texts,
            "dimensions": settings.embed_dims,
            "encoding_format": "float",
        },
    )
    resp.raise_for_status()
    data = sorted(resp.json()["data"], key=lambda x: x["index"])
    return [item["embedding"] for item in data]


async def embed_texts(texts: list[str]) -> list[list[float]]:
    """批量向量化（每批 ≤10 条，超长文本截断到 6000 字符兜底）"""
    texts = [t[:6000] if t else " " for t in texts]
    out: list[list[float]] = []
    async with httpx.AsyncClient(
        base_url=settings.dashscope_base_url,
        headers={"Authorization": f"Bearer {settings.dashscope_api_key}"},
        timeout=60,
    ) as client:
        for i in range(0, len(texts), _BATCH):
            batch = texts[i : i + _BATCH]
            for attempt in range(2):
                try:
                    out.extend(await _embed_batch(client, batch))
                    break
                except Exception as e:  # noqa: BLE001
                    if attempt == 1:
                        raise
                    logger.warning("Embedding 批次失败重试：%s", e)
    return out


async def embed_query(text: str) -> list[float]:
    """单条查询向量化"""
    vecs = await embed_texts([text])
    return vecs[0]
