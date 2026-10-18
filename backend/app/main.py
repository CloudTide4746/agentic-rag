"""Agentic RAG 后端入口 — FastAPI + SSE

启动流程：依赖自检（ES / DeepSeek / DashScope 密钥，异常仅终端提示）→ ES 建索引。
MySQL 库表与基础数据（datasource_config / sales）由项目根目录 agentic_rag.sql 前置导入。
"""
import logging
from contextlib import asynccontextmanager

from fastapi import Depends, FastAPI

from . import db
from .config import settings
from .es import elastic
from .internal_auth import require_internal_token
from .routers import chat, kb
from .startup_check import run_startup_checks

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(levelname)s %(message)s")
logger = logging.getLogger("agentic.main")


@asynccontextmanager
async def lifespan(app: FastAPI):
    await run_startup_checks()
    try:
        await elastic.ensure_index()
    except Exception as e:  # noqa: BLE001
        logger.error("ES 索引初始化失败，检索功能不可用（服务降级启动）：%s", e)
    yield
    await elastic.close()


if not settings.internal_api_token:
    raise RuntimeError("INTERNAL_API_TOKEN 必须配置后才能启动 Agent 服务")

app = FastAPI(
    title="Agentic RAG Internal Service",
    version="1.0.0",
    lifespan=lifespan,
    docs_url=None,
    redoc_url=None,
    openapi_url=None,
)

app.include_router(kb.router)
app.include_router(chat.router)


@app.get("/internal/healthz", dependencies=[Depends(require_internal_token)])
async def health() -> dict:
    mysql_ok = True
    es_ok = await elastic.ping()
    try:
        await db.aquery_one("SELECT 1 AS ok")
    except Exception:  # noqa: BLE001
        mysql_ok = False
    return {
        "status": "ok" if (mysql_ok and es_ok) else "degraded",
        "version": app.version,
        "mysql": mysql_ok,
        "elasticsearch": es_ok,
    }
