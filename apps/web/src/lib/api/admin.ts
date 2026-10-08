import type {
  DecisionLog,
  DecisionLogQuery,
  DatasourceItem,
  FusedItem,
  KbChunk,
  KbDocument,
  KnowledgeBase,
  Page,
  RetrievalGrade,
  RetrievalLane,
  RetrievalSource,
} from '../../types/legacy'
import { apiJson, jsonInit } from './http'

/* ============================================================
 * 管理端 API — 路径与协议对齐设计书 5.1 节
 * ============================================================ */

/** GET /api/admin/kb */
export async function listKbs(): Promise<KnowledgeBase[]> {
  return apiJson<KnowledgeBase[]>('/api/admin/kb')
}

/** POST /api/admin/kb */
export async function createKb(name: string, description: string): Promise<KnowledgeBase> {
  return apiJson<KnowledgeBase>('/api/admin/kb', jsonInit('POST', { name, description }))
}

/** GET /api/admin/kb/{kb_id}/documents */
export async function listDocuments(kbId: number): Promise<KbDocument[]> {
  return apiJson<KbDocument[]>(`/api/admin/kb/${kbId}/documents`)
}

/** POST /api/admin/kb/{kb_id}/documents（multipart 上传，202 异步入库） */
export async function uploadDocument(kbId: number, file: File): Promise<KbDocument> {
  const fd = new FormData()
  fd.append('file', file)
  return apiJson<KbDocument>(`/api/admin/kb/${kbId}/documents`, { method: 'POST', body: fd })
}

/** DELETE /api/admin/kb/{kb_id}/documents/{doc_id}（先 ES 后 MySQL） */
export async function deleteDocument(kbId: number, docId: number): Promise<void> {
  await apiJson(`/api/admin/kb/${kbId}/documents/${docId}`, { method: 'DELETE' })
}

/** GET /api/admin/kb/{kb_id}/documents/{doc_id}/chunks（切片明细，分页） */
export async function listDocumentChunks(kbId: number, docId: number, page: number, pageSize = 8): Promise<Page<KbChunk>> {
  return apiJson<Page<KbChunk>>(`/api/admin/kb/${kbId}/documents/${docId}/chunks?page=${page}&page_size=${pageSize}`)
}

/** GET /api/admin/chunks/by-citation?doc_name=&chunk_no=（引用溯源定位单切片） */
export async function getChunkByCitation(docName: string, chunkNo: number): Promise<{ doc: KbDocument; chunk: KbChunk }> {
  return apiJson<{ doc: KbDocument; chunk: KbChunk }>(
    `/api/admin/chunks/by-citation?doc_name=${encodeURIComponent(docName)}&chunk_no=${chunkNo}`,
  )
}

/** POST /api/admin/kb/{kb_id}/documents/{doc_id}/retry */
export async function retryDocument(kbId: number, docId: number): Promise<KbDocument> {
  return apiJson<KbDocument>(`/api/admin/kb/${kbId}/documents/${docId}/retry`, { method: 'POST' })
}

/** GET /api/admin/datasources */
export async function getDatasources(): Promise<DatasourceItem[]> {
  return apiJson<DatasourceItem[]>('/api/admin/datasources')
}

/** PUT /api/admin/datasources */
export async function updateDatasources(items: DatasourceItem[]): Promise<{ updated: number }> {
  return apiJson<{ updated: number }>('/api/admin/datasources', jsonInit('PUT', { items }))
}

/** GET /api/admin/decision-logs（过滤：会话 ID、时间范围 + 分页） */
export async function queryDecisionLogs(query: DecisionLogQuery): Promise<Page<DecisionLog>> {
  const params = new URLSearchParams()
  if (query.session_id != null) params.set('session_id', String(query.session_id))
  if (query.start) params.set('start', query.start)
  if (query.end) params.set('end', query.end)
  params.set('page', String(query.page))
  params.set('page_size', String(query.page_size))
  return apiJson<Page<DecisionLog>>(`/api/admin/decision-logs?${params.toString()}`)
}

/** GET /api/admin/decision-logs/{log_id} */
export async function getDecisionLog(logId: number): Promise<DecisionLog> {
  return apiJson<DecisionLog>(`/api/admin/decision-logs/${logId}`)
}

/* ============================================================
 * 检索测试 Playground — 四路召回 / 应用层 RRF / 真实 Grader
 * 调用 POST /api/admin/retrieval-test/{lane,fuse}
 * ============================================================ */

/** 单路召回（各路独立调用，走后端检索适配层） */
export async function testRetrieveLane(query: string, source: RetrievalSource, topK = 5, kbIds?: number[]): Promise<RetrievalLane> {
  return apiJson<RetrievalLane>(
    '/api/admin/retrieval-test/lane',
    jsonInit('POST', { query, source, top_k: topK, kb_ids: kbIds ?? null }),
  )
}

/** 应用层 RRF 融合（k=60）+ Grader 评分（LLM-as-Judge） */
export async function fuseAndGrade(lanes: RetrievalLane[], query = ''): Promise<{ fused: FusedItem[]; grade: RetrievalGrade }> {
  return apiJson<{ fused: FusedItem[]; grade: RetrievalGrade }>(
    '/api/admin/retrieval-test/fuse',
    jsonInit('POST', { query, lanes }),
  )
}

/** 数据源类型 → 展示元数据 */
export const SOURCE_META: Record<RetrievalSource, { label: string; desc: string; icon: string; color: string }> = {
  vector: { label: '向量检索', desc: 'ES dense_vector kNN 语义检索（text-embedding-v4 · 1024 维）', icon: 'search', color: 'text-src-vector' },
  keyword: { label: '关键词检索', desc: 'ES BM25 match 精确匹配（ik_max_word 中文分词）', icon: 'zap', color: 'text-src-keyword' },
  database: { label: '业务数据库', desc: 'MySQL 只读 NL2SQL 查询（SELECT 白名单 + LIMIT 截断）', icon: 'db', color: 'text-src-db' },
}
