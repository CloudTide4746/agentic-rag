"""Agentic RAG 后端入口 — FastAPI + SSE

启动流程：依赖自检（ES / DeepSeek / DashScope 密钥，异常仅终端提示）→ ES 建索引。
MySQL 库表与基础数据（datasource_config / sales）由项目根目录 agentic_rag.sql 前置导入。
"""
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from . import db
from .config import settings
from .es import elastic
from .routers import chat, datasources, decision_logs, kb, retrieval_test
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


app = FastAPI(title="Agentic RAG API", version="1.0.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173", "http://127.0.0.1:5173"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(kb.router)
app.include_router(datasources.router)
app.include_router(decision_logs.router)
app.include_router(chat.router)
app.include_router(retrieval_test.router)


@app.get("/api/health")
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
        "model": settings.llm_model,
        "mysql": mysql_ok,
        "elasticsearch": es_ok,
    }
