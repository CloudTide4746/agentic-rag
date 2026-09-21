"""问答路由 — SSE 问答流核心接口（设计书 5.2 / 5.3）"""
from fastapi import APIRouter, HTTPException
from starlette.responses import FileResponse, StreamingResponse

from .. import db
from ..agent.orchestrator import run_ask
from ..models import AskRequest
from ..services import chat_images, sessions

router = APIRouter(prefix="/api/chat", tags=["chat"])


@router.post("/sessions", status_code=201)
async def create_session() -> dict:
    return await sessions.create_session()


@router.get("/sessions")
async def list_sessions() -> list[dict]:
    return await sessions.list_sessions()


@router.get("/sessions/{session_id}/messages")
async def get_messages(session_id: int) -> list[dict]:
    sess = await db.aquery_one("SELECT id FROM chat_session WHERE id=%s", (session_id,))
    if sess is None:
        raise HTTPException(404, "会话不存在")
    return await sessions.get_messages(session_id)


@router.delete("/sessions/{session_id}")
async def delete_session(session_id: int) -> dict:
    sess = await db.aquery_one("SELECT id FROM chat_session WHERE id=%s", (session_id,))
    if sess is None:
        raise HTTPException(404, "会话不存在")
    await sessions.delete_session(session_id)
    return {"deleted": session_id}


@router.delete("/sessions/{session_id}/last-assistant")
async def purge_last_assistant(session_id: int) -> dict:
    """重新生成前置清理：移除最后一条 assistant 消息及其决策日志"""
    removed = await sessions.purge_last_assistant(session_id)
    return {"removed": removed}


@router.get("/images/{name}")
async def get_chat_image(name: str) -> FileResponse:
    """问答图片静态服务（chat_message.images 引用的站内 URL）"""
    path = chat_images.image_path(name)
    if path is None:
        raise HTTPException(404, "图片不存在")
    ext = name.rsplit(".", 1)[-1]
    return FileResponse(path, media_type=chat_images.EXT_MIME[ext])


@router.post("/sessions/{session_id}/ask")
async def ask(session_id: int, body: AskRequest):
    sess = await db.aquery_one("SELECT id FROM chat_session WHERE id=%s", (session_id,))
    if sess is None:
        raise HTTPException(404, "会话不存在")
    try:
        image_urls = chat_images.save_images(body.images or [])
    except ValueError as e:
        raise HTTPException(400, str(e))
    gen = run_ask(session_id, body.question.strip(), body.kb_ids, body.skip_user_persist, image_urls)
    return StreamingResponse(
        gen,
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
