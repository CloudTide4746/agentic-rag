"""检索测试 Playground 路由 — 单路召回 + 应用层 RRF 融合 + 真实 Grader 评分

与 Q3/Q4 链路同构：复用检索适配层与 Grader，用于管理端直观验证。
"""
import logging
import time

from fastapi import APIRouter, HTTPException

from ..agent import orchestrator
from ..agent.retrievers import db_retriever, vector_retriever, keyword_retriever
from ..agent.rrf import RetrieverResult, item_key, rrf_fuse
from ..models import FuseRequest, LaneRequest

router = APIRouter(prefix="/api/admin/retrieval-test", tags=["playground"])

logger = logging.getLogger("agentic.playground")


def _hit(item: RetrieverResult) -> dict:
    return {
        "title": item.title,
        "snippet": item.content,
        "score": item.score,
        "meta": {k: v for k, v in item.metadata.items() if k != "doc_id"},
    }


@router.post("/lane")
async def test_lane(body: LaneRequest) -> dict:
    t0 = time.time()
    if body.source == "vector":
        results = await vector_retriever.search(body.query, body.kb_ids, body.top_k)
    elif body.source == "keyword":
        results = await keyword_retriever.search(body.query, body.kb_ids, body.top_k)
    elif body.source == "database":
        # 非数据类查询 NL2SQL 可能生成无法执行/不合法的 SQL：降级为空命中，不阻断其余路
        try:
            _, results = await db_retriever.search(body.query, body.top_k)
        except Exception as e:  # noqa: BLE001
            logger.warning("Playground database 路查询失败：%s", e)
            results = []
    else:
        raise HTTPException(400, f"未知检索源：{body.source}")
    return {
        "source": body.source,
        "latency_ms": int((time.time() - t0) * 1000),
        "hits": [_hit(r) for r in results],
    }


@router.post("/fuse")
async def test_fuse(body: FuseRequest) -> dict:
    lanes: dict[str, list[RetrieverResult]] = {}
    for lane in body.lanes:
        lanes[lane.source] = [
            RetrieverResult(
                source=lane.source,
                score=h.score,
                title=h.title,
                content=h.snippet,
                metadata=dict(h.meta),
            )
            for h in lane.hits
        ]
    fused = rrf_fuse(lanes, 8)
    items = [f["item"] for f in fused]
    ctx_blocks = orchestrator.build_context(items)
    query = body.query.strip() or "（综合各路检索查询）"
    score, verdict, reason = await orchestrator.do_grade(query, [], ctx_blocks, {})
    return {
        "fused": [
            {"key": item_key(f["item"]), "rrf": f["rrf"], "from_lanes": f["from_lanes"], "hit": _hit(f["item"])}
            for f in fused
        ],
        "grade": {"score": score, "verdict": verdict, "reason": reason},
    }
