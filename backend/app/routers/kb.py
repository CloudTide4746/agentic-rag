"""知识库管理路由 — M1（设计书 5.1）"""
import asyncio

from fastapi import APIRouter, BackgroundTasks, File, HTTPException, UploadFile

from .. import db
from ..services import ingestion

router = APIRouter(prefix="/api/admin", tags=["kb"])


@router.get("/kb")
async def list_kbs() -> list[dict]:
    rows = await db.aquery_all(
        """SELECT k.*,
                  (SELECT COUNT(*) FROM kb_document d WHERE d.kb_id = k.id AND d.status = 'ready') AS doc_count,
                  (SELECT COALESCE(SUM(d.chunk_count), 0) FROM kb_document d WHERE d.kb_id = k.id AND d.status = 'ready') AS chunk_count
           FROM kb_database k ORDER BY k.id ASC"""
    )
    return rows


@router.post("/kb", status_code=201)
async def create_kb(body: dict) -> dict:
    name = (body.get("name") or "").strip()
    description = (body.get("description") or "").strip()
    if not name:
        raise HTTPException(400, "知识库名称不能为空")
    dup = await db.aquery_one("SELECT id FROM kb_database WHERE name=%s", (name,))
    if dup:
        raise HTTPException(409, f"知识库名称「{name}」已存在（name 唯一约束）")
    kid = await db.aexecute("INSERT INTO kb_database (name, description) VALUES (%s, %s)", (name, description))
    row = await db.aquery_one(
        """SELECT k.*, 0 AS doc_count, 0 AS chunk_count FROM kb_database k WHERE k.id=%s""", (kid,)
    )
    return row  # type: ignore[return-value]


@router.get("/kb/{kb_id}/documents")
async def list_documents(kb_id: int) -> list[dict]:
    return await db.aquery_all(
        "SELECT id, kb_id, file_name, file_type, status, chunk_count, error_msg, created_at, updated_at"
        " FROM kb_document WHERE kb_id=%s ORDER BY created_at DESC, id DESC",
        (kb_id,),
    )


@router.post("/kb/{kb_id}/documents", status_code=202)
async def upload_document(kb_id: int, background: BackgroundTasks, file: UploadFile = File(...)) -> dict:
    kb = await db.aquery_one("SELECT id FROM kb_database WHERE id=%s", (kb_id,))
    if kb is None:
        raise HTTPException(404, "知识库不存在")
    content = await file.read()
    try:
        doc_id, _, _ = await asyncio.to_thread(ingestion.save_upload, kb_id, file.filename or "untitled", content)
    except ValueError as e:
        raise HTTPException(400, str(e))
    background.add_task(ingestion.process_document, doc_id)
    doc = await db.aquery_one(
        "SELECT id, kb_id, file_name, file_type, status, chunk_count, error_msg, created_at, updated_at"
        " FROM kb_document WHERE id=%s",
        (doc_id,),
    )
    return doc  # type: ignore[return-value]


@router.delete("/kb/{kb_id}/documents/{doc_id}")
async def delete_document(kb_id: int, doc_id: int) -> dict:
    doc = await db.aquery_one("SELECT id FROM kb_document WHERE id=%s AND kb_id=%s", (doc_id, kb_id))
    if doc is None:
        raise HTTPException(404, "文档不存在")
    await ingestion.delete_document(kb_id, doc_id)
    return {"deleted": doc_id}


@router.post("/kb/{kb_id}/documents/{doc_id}/retry", status_code=202)
async def retry_document(kb_id: int, doc_id: int, background: BackgroundTasks) -> dict:
    try:
        doc = await ingestion.reset_for_retry(kb_id, doc_id)
    except ValueError as e:
        raise HTTPException(400, str(e))
    background.add_task(ingestion.process_document, doc_id)
    return doc


@router.get("/kb/{kb_id}/documents/{doc_id}/chunks")
async def list_document_chunks(kb_id: int, doc_id: int, page: int = 1, page_size: int = 8) -> dict:
    doc = await db.aquery_one("SELECT * FROM kb_document WHERE id=%s AND kb_id=%s", (doc_id, kb_id))
    if doc is None:
        raise HTTPException(404, "文档不存在")
    if doc["status"] != "ready":
        raise HTTPException(400, "文档尚未完成入库，暂无切片")
    page = max(1, page)
    page_size = min(50, max(1, page_size))
    total_row = await db.aquery_one("SELECT COUNT(*) AS n FROM kb_chunk WHERE doc_id=%s", (doc_id,))
    total = int(total_row["n"]) if total_row else 0
    items = await db.aquery_all(
        "SELECT id, doc_id, chunk_no, section_path, content, token_count FROM kb_chunk"
        " WHERE doc_id=%s ORDER BY chunk_no ASC LIMIT %s OFFSET %s",
        (doc_id, page_size, (page - 1) * page_size),
    )
    return {"items": items, "total": total, "page": page, "page_size": page_size}


@router.get("/chunks/by-citation")
async def get_chunk_by_citation(doc_name: str, chunk_no: int) -> dict:
    """引用溯源定位：doc_name 为文档名去扩展名（与 ES 元数据一致）"""
    doc = await db.aquery_one(
        "SELECT id, kb_id, file_name, file_type, status, chunk_count, error_msg, created_at, updated_at"
        " FROM kb_document WHERE status='ready' AND SUBSTRING_INDEX(file_name, '.', 1)=%s ORDER BY id DESC LIMIT 1",
        (doc_name,),
    )
    if doc is None:
        raise HTTPException(404, f"未找到文档《{doc_name}》：可能已被删除或未完成入库")
    chunk = await db.aquery_one(
        "SELECT id, doc_id, chunk_no, section_path, content, token_count FROM kb_chunk"
        " WHERE doc_id=%s AND chunk_no=%s",
        (doc["id"], chunk_no),
    )
    if chunk is None:
        raise HTTPException(404, f"《{doc_name}》不存在切片 #{chunk_no}，可能文档已重新入库")
    return {"doc": doc, "chunk": chunk}
