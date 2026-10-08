/**
 * Types migrated from the archived Vue frontend. These describe the legacy Agent contract.
 *
 * @format
 */

export type IntentType = "chitchat" | "kb_qa" | "data_query" | "summarize";
export type RetrievalSource = "vector" | "keyword" | "database";
export type DocStatus =
  | "parsing"
  | "chunking"
  | "embedding"
  | "ready"
  | "failed";

interface SseBase {
  step: number;
  ts: number;
  detail: string;
}
export interface RouterEvent extends SseBase {
  type: "router";
  intent: IntentType;
  confidence: number;
}
export interface RewriteEvent extends SseBase {
  type: "rewrite";
  rewritten_query: string;
  sub_queries: string[];
  hyde_used: boolean;
}
export interface RetrieveEvent extends SseBase {
  type: "retrieve";
  source: RetrievalSource;
  hit_count: number;
}
export interface GradeEvent extends SseBase {
  type: "grade";
  score: number;
  verdict: "pass" | "partial" | "fail";
  reason: string;
}
export interface CorrectEvent extends SseBase {
  type: "correct";
  action: "rewrite_retry" | "give_up";
  round: number;
}
export interface GenerateEvent extends SseBase {
  type: "generate";
  delta: string;
}
export interface ReflectEvent extends SseBase {
  type: "reflect";
  result: "pass" | "fail";
  round: number;
  missing_info?: string;
}
export interface Citation {
  doc_name: string;
  section_path: string;
  chunk_no: number;
  source_type?: "db";
  sql?: string;
  rows?: Record<string, string | number | null>[];
}
export interface CitationEvent extends SseBase {
  type: "citation";
  citations: Citation[];
}
export interface TokenUsage {
  prompt: number;
  completion: number;
}
export interface DoneEvent extends SseBase {
  type: "done";
  message_id: number;
  elapsed_ms: number;
  token_usage: TokenUsage;
}
export interface ErrorEvent extends SseBase {
  type: "error";
  code: number;
  message: string;
}
export type SseEvent =
  | RouterEvent
  | RewriteEvent
  | RetrieveEvent
  | GradeEvent
  | CorrectEvent
  | GenerateEvent
  | ReflectEvent
  | CitationEvent
  | DoneEvent
  | ErrorEvent;
export type SseEventType = SseEvent["type"];
export type StepVisual = "pending" | "done" | "warn";

export interface KnowledgeBase {
  id: number;
  name: string;
  description: string;
  doc_count: number;
  chunk_count: number;
  created_at: string;
  updated_at: string;
}
export interface KbDocument {
  id: number;
  kb_id: number;
  file_name: string;
  file_type: "md" | "txt" | "pdf" | "docx";
  status: DocStatus;
  chunk_count: number;
  error_msg?: string;
  created_at: string;
  updated_at: string;
}
export interface KbChunk {
  id: number;
  doc_id: number;
  chunk_no: number;
  section_path: string;
  content: string;
  token_count: number;
}
export interface ChatSession {
  id: number;
  title: string;
  created_at: string;
  updated_at: string;
}
export interface ChatMessage {
  id: number;
  session_id: number;
  role: "user" | "assistant";
  content: string;
  citations?: Citation[];
  images?: string[];
  decision_log_id?: number;
  created_at: string;
}
export interface DecisionStep {
  step: number;
  type: SseEventType;
  detail: string;
  payload: Record<string, unknown>;
  elapsed_ms: number;
}
export interface DecisionLog {
  id: number;
  session_id: number;
  message_id: number;
  question: string;
  final_answer: string;
  steps: DecisionStep[];
  total_rounds: number;
  elapsed_ms: number;
  created_at: string;
}
export interface DatasourceItem {
  source_type: RetrievalSource;
  enabled: boolean;
  config: Record<string, string>;
  updated_at: string;
}
export interface RetrievalHit {
  title: string;
  snippet: string;
  score: number;
  meta: { doc_name?: string; section_path?: string; chunk_no?: number };
}
export interface RetrievalLane {
  source: RetrievalSource;
  latency_ms: number;
  hits: RetrievalHit[];
}
export interface FusedItem {
  key: string;
  rrf: number;
  from_lanes: RetrievalSource[];
  hit: RetrievalHit;
}
export interface RetrievalGrade {
  score: number;
  verdict: "pass" | "partial" | "fail";
  reason: string;
}
export interface RetrievalTestResult {
  lanes: RetrievalLane[];
  fused: FusedItem[];
  grade: RetrievalGrade;
}
export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}
export interface DecisionLogQuery {
  session_id?: number;
  start?: string;
  end?: string;
  page: number;
  page_size: number;
}
export interface SseHandlers {
  onEvent: (event: SseEvent) => void;
  signal?: AbortSignal;
}
export interface ApiResponse<T> {
  data: T;
  request_id: string;
}
export interface ApiErrorResponse {
  error: { code: string; message: string };
  request_id: string;
}
export interface Health {
  status: string;
  service: string;
}

/** UI model held by the old chat store; retained for component migration only. */
export interface DecisionCard {
  step: number;
  type: SseEventType;
  status: StepVisual;
  detail: string;
  payload: Record<string, unknown>;
  ts: number;
  elapsedMs: number;
}
export interface ViewingLog {
  id: number;
  session_id: number;
  message_id: number;
  question: string;
  created_at: string;
}
