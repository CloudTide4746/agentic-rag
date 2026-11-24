/** @format */

import type { ApiResponse } from "../../types/api";
import type { SseEvent } from "../../types/legacy";
import { SseParser } from "./sseParser";
import type {
  AskReq,
  AuthResp,
  ChatMessage,
  ChatSession,
  CreateChatReq,
  CreateProjectReq,
  CreateTaskReq,
  CreateTeamReq,
  EmptyResp,
  ListResp,
  ListTasksQuery,
  LoginReq,
  PageQuery,
  Project,
  RegisterReq,
  Task,
  Team,
  UpdateProjectReq,
  UpdateTaskReq,
  UpdateTeamReq,
  User,
} from "./dto";

export type * from "./dto";
export type { ApiResponse };

const API_BASE = "/api/v1";
let accessToken: string | undefined;
let sessionController = new AbortController();
const expiryListeners = new Set<() => void>();

type ApiEnvelope<T> = ApiResponse<T>;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: string,
    public readonly requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/** Shared HTTP code publishes expiry without depending on the auth feature. */
export function subscribeAuthenticationExpired(listener: () => void): () => void {
  expiryListeners.add(listener);
  return () => { expiryListeners.delete(listener); };
}

function expireSession(token: string | undefined): void {
  if (!token || accessToken !== token) return;
  clearAccessToken();
  for (const listener of expiryListeners) listener();
}

function requestSignal(signal?: AbortSignal | null): AbortSignal {
  return signal
    ? AbortSignal.any([signal, sessionController.signal])
    : sessionController.signal;
}

async function readError(response: Response): Promise<ApiError> {
  const fallback = `HTTP ${response.status}`;
  try {
    const body: unknown = await response.json();
    if (isRecord(body)) {
      const error = body.error;
      if (isRecord(error) && typeof error.message === "string") {
        return new ApiError(error.message, response.status,
          typeof error.code === "string" ? error.code : "HTTP_ERROR",
          typeof body.request_id === "string" ? body.request_id : undefined);
      }
      if (typeof body.message === "string") return new ApiError(body.message, response.status, "HTTP_ERROR");
      if (typeof body.detail === "string") return new ApiError(body.detail, response.status, "HTTP_ERROR");
    }
  } catch {
    // Keep the status-based message when the response is not JSON.
  }
  return new ApiError(fallback, response.status, "HTTP_ERROR");
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  decode: (value: unknown) => T,
): Promise<ApiEnvelope<T>> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const token = !path.startsWith("/auth/") ? accessToken : undefined;
  const signal = requestSignal(init.signal);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    signal,
    credentials: "same-origin",
  });
  if (!response.ok) {
    const error = await readError(response);
    signal.throwIfAborted();
    if (response.status === 401) expireSession(token);
    throw error;
  }

  let body: unknown;
  try {
    body = await response.json();
  } catch (cause) {
    signal.throwIfAborted();
    throw new ApiError(
      cause instanceof SyntaxError ? "API 未返回有效 JSON" : "API 响应读取失败",
      response.status, "INVALID_RESPONSE",
    );
  }
  signal.throwIfAborted();
  if (
    !isRecord(body) ||
    !("data" in body) ||
    typeof body.request_id !== "string"
  ) {
    throw new ApiError("API 响应格式无效", response.status, "INVALID_RESPONSE");
  }
  try {
    return { data: decode(body.data), request_id: body.request_id };
  } catch {
    throw new ApiError("API 响应数据无效", response.status, "INVALID_RESPONSE", body.request_id);
  }
}

function record(value: unknown): Record<string, unknown> {
  if (!isRecord(value)) throw new TypeError("Expected object");
  return value;
}

function string(value: unknown): string {
  if (typeof value !== "string") throw new TypeError("Expected string");
  return value;
}

function integer(value: unknown, minimum = 0): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < minimum) {
    throw new TypeError("Expected integer");
  }
  return value;
}

function id(value: unknown): string {
  const result = string(value);
  if (!result.trim()) throw new TypeError("Expected identifier");
  return result;
}

function optionalString(value: unknown): string | undefined {
  return value === undefined ? undefined : string(value);
}

function user(value: unknown): User {
  const v = record(value);
  return { id: id(v.id), username: string(v.username), email: string(v.email),
    nickname: string(v.nickname), avatar: string(v.avatar), created_at: string(v.created_at) };
}

function auth(value: unknown): AuthResp {
  const v = record(value);
  return { token: id(v.token), user: user(v.user) };
}

function team(value: unknown): Team {
  const v = record(value);
  return { id: id(v.id), name: string(v.name), description: string(v.description),
    owner_id: id(v.owner_id), avatar: string(v.avatar), created_at: string(v.created_at), updated_at: string(v.updated_at) };
}

function project(value: unknown): Project {
  const v = record(value);
  return { id: id(v.id), name: string(v.name), description: string(v.description),
    owner_id: id(v.owner_id), team_id: id(v.team_id), status: string(v.status),
    start_date: optionalString(v.start_date), end_date: optionalString(v.end_date),
    created_at: string(v.created_at), updated_at: string(v.updated_at) };
}

function task(value: unknown): Task {
  const v = record(value);
  return { id: id(v.id), title: string(v.title), description: string(v.description),
    project_id: id(v.project_id), creator_id: id(v.creator_id), assignee_id: optionalString(v.assignee_id),
    status: string(v.status), priority: string(v.priority), due_date: optionalString(v.due_date),
    version: integer(v.version), created_at: string(v.created_at), updated_at: string(v.updated_at) };
}

function session(value: unknown): ChatSession {
  const v = record(value);
  return { id: id(v.id), title: string(v.title), status: string(v.status), owner_id: id(v.owner_id),
    created_at: string(v.created_at), updated_at: string(v.updated_at) };
}

function array<T>(value: unknown, decode: (item: unknown) => T): T[] {
  if (!Array.isArray(value)) throw new TypeError("Expected array");
  return value.map(decode);
}

function message(value: unknown): ChatMessage {
  const v = record(value);
  return { role: string(v.role), content: string(v.content), created_at: string(v.created_at),
    citations: v.citations, images: v.images === undefined ? undefined : array(v.images, string),
    decision_log_id: optionalString(v.decision_log_id) };
}

function list<T>(decode: (value: unknown) => T): (value: unknown) => ListResp<T> {
  return (value) => {
    const v = record(value);
    return { items: array(v.items, decode), total: integer(v.total), page: integer(v.page, 1), page_size: integer(v.page_size, 1) };
  };
}

const teamResponse = (value: unknown) => ({ team: team(record(value).team) });
const projectResponse = (value: unknown) => ({ project: project(record(value).project) });
const taskResponse = (value: unknown) => ({ task: task(record(value).task) });
const sessionResponse = (value: unknown) => ({ session: session(record(value).session) });
function empty(value: unknown): EmptyResp {
  if (Object.keys(record(value)).length) throw new TypeError("Expected empty object");
  return {};
}

function jsonBody(
  method: "POST" | "PUT",
  body?: unknown,
  signal?: AbortSignal,
): RequestInit {
  return {
    method,
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    signal,
  };
}

function pageQuery(query?: PageQuery): string {
  const params = new URLSearchParams();
  if (query?.page !== undefined) params.set("page", String(query.page));
  if (query?.page_size !== undefined) {
    params.set("page_size", String(query.page_size));
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

function numericPathId(id: number, label: string): string {
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw new TypeError(`${label} 必须是正安全整数`);
  }
  return String(id);
}

function opaquePathId(id: string): string {
  if (!id.trim()) throw new TypeError("sessionId 不能为空");
  return encodeURIComponent(id);
}

/** 设置仅驻留在当前页面内存中的 Bearer token。 */
export function setAccessToken(token: string): void {
  if (typeof token !== "string") throw new TypeError("token 必须是字符串");
  const normalized = token.trim();
  if (!normalized) throw new TypeError("token 不能为空");
  clearAccessToken();
  accessToken = normalized;
}

/** 清理当前页面内存中的 Bearer token。 */
export function clearAccessToken(): void {
  accessToken = undefined;
  sessionController.abort();
  sessionController = new AbortController();
}

/** POST /api/v1/auth/register */
export function register(
  body: RegisterReq,
  signal?: AbortSignal,
): Promise<ApiResponse<User>> {
  return request("/auth/register", jsonBody("POST", body, signal), user);
}

/** POST /api/v1/auth/login; the returned token is kept in memory only. */
export async function login(
  body: LoginReq,
  signal?: AbortSignal,
): Promise<ApiResponse<AuthResp>> {
  const lifecycleSignal = sessionController.signal;
  const response = await request<AuthResp>(
    "/auth/login",
    jsonBody("POST", body, signal),
    auth,
  );
  lifecycleSignal.throwIfAborted();
  signal?.throwIfAborted();
  setAccessToken(response.data.token);
  return response;
}

/**
 * There is no public authenticate endpoint; authentication is enforced by the
 * protected API middleware. Supply the in-memory token to subsequent calls.
 */
export { setAccessToken as authenticate };

/** POST /api/v1/teams */
export function addTeam(
  body: CreateTeamReq,
  signal?: AbortSignal,
): Promise<ApiResponse<{ team: Team }>> {
  return request("/teams", jsonBody("POST", body, signal), teamResponse);
}

/** GET /api/v1/teams?page=&page_size= */
export function getTeamList(
  query?: PageQuery,
  signal?: AbortSignal,
): Promise<ApiResponse<ListResp<Team>>> {
  return request(`/teams${pageQuery(query)}`, { signal }, list(team));
}

/** GET /api/v1/teams/:id */
export function getTeamById(
  teamId: number,
  signal?: AbortSignal,
): Promise<ApiResponse<{ team: Team }>> {
  return request(`/teams/${numericPathId(teamId, "teamId")}`, { signal }, teamResponse);
}

/** PUT /api/v1/teams/:id */
export function updateTeam(
  teamId: number,
  body: UpdateTeamReq,
  signal?: AbortSignal,
): Promise<ApiResponse<{ team: Team }>> {
  return request(
    `/teams/${numericPathId(teamId, "teamId")}`,
    jsonBody("PUT", body, signal),
    teamResponse,
  );
}

/** DELETE /api/v1/teams/:id */
export function deleteTeam(
  teamId: number,
  signal?: AbortSignal,
): Promise<ApiResponse<EmptyResp>> {
  return request(`/teams/${numericPathId(teamId, "teamId")}`, {
    method: "DELETE",
    signal,
  }, empty);
}

/** POST /api/v1/teams/:id/projects */
export function addProject(
  teamId: number,
  body: CreateProjectReq,
  signal?: AbortSignal,
): Promise<ApiResponse<{ project: Project }>> {
  return request(
    `/teams/${numericPathId(teamId, "teamId")}/projects`,
    jsonBody("POST", body, signal),
    projectResponse,
  );
}

/** GET /api/v1/teams/:id/projects?page=&page_size= */
export function getProjectList(
  teamId: number,
  query?: PageQuery,
  signal?: AbortSignal,
): Promise<ApiResponse<ListResp<Project>>> {
  return request(
    `/teams/${numericPathId(teamId, "teamId")}/projects${pageQuery(query)}`,
    { signal },
    list(project),
  );
}

/** GET /api/v1/projects/:id */
export function getProjectById(
  projectId: number,
  signal?: AbortSignal,
): Promise<ApiResponse<{ project: Project }>> {
  return request(`/projects/${numericPathId(projectId, "projectId")}`, {
    signal,
  }, projectResponse);
}

/** PUT /api/v1/projects/:id */
export function updateProject(
  projectId: number,
  body: UpdateProjectReq,
  signal?: AbortSignal,
): Promise<ApiResponse<{ project: Project }>> {
  return request(
    `/projects/${numericPathId(projectId, "projectId")}`,
    jsonBody("PUT", body, signal),
    projectResponse,
  );
}

/** DELETE /api/v1/projects/:id */
export function deleteProject(
  projectId: number,
  signal?: AbortSignal,
): Promise<ApiResponse<EmptyResp>> {
  return request(`/projects/${numericPathId(projectId, "projectId")}`, {
    method: "DELETE",
    signal,
  }, empty);
}

/** POST /api/v1/tasks */
export function addTask(
  body: CreateTaskReq,
  signal?: AbortSignal,
): Promise<ApiResponse<{ task: Task }>> {
  return request("/tasks", jsonBody("POST", body, signal), taskResponse);
}

/** GET /api/v1/tasks?project_id=&page=&page_size= */
export function getTaskList(
  query?: ListTasksQuery,
  signal?: AbortSignal,
): Promise<ApiResponse<ListResp<Task>>> {
  const params = new URLSearchParams(pageQuery(query).replace(/^\?/, ""));
  if (query?.project_id !== undefined) {
    params.set("project_id", query.project_id);
  }
  const queryString = params.toString();
  return request(`/tasks${queryString ? `?${queryString}` : ""}`, { signal }, list(task));
}

/** GET /api/v1/tasks/:id */
export function getTaskById(
  taskId: number,
  signal?: AbortSignal,
): Promise<ApiResponse<{ task: Task }>> {
  return request(`/tasks/${numericPathId(taskId, "taskId")}`, { signal }, taskResponse);
}

/** PUT /api/v1/tasks/:id; pass the version unchanged for optimistic locking. */
export function updateTask(
  taskId: number,
  body: UpdateTaskReq,
  signal?: AbortSignal,
): Promise<ApiResponse<{ task: Task }>> {
  return request(
    `/tasks/${numericPathId(taskId, "taskId")}`,
    jsonBody("PUT", body, signal),
    taskResponse,
  );
}

/** DELETE /api/v1/tasks/:id */
export function deleteTask(
  taskId: number,
  signal?: AbortSignal,
): Promise<ApiResponse<EmptyResp>> {
  return request(`/tasks/${numericPathId(taskId, "taskId")}`, {
    method: "DELETE",
    signal,
  }, empty);
}

/** POST /api/v1/chat/sessions */
export function createChat(
  body?: CreateChatReq,
  signal?: AbortSignal,
): Promise<ApiResponse<{ session: ChatSession }>> {
  return request("/chat/sessions", jsonBody("POST", body ?? {}, signal), sessionResponse);
}

/** GET /api/v1/chat/sessions?page=&page_size= */
export function listSessions(
  query?: PageQuery,
  signal?: AbortSignal,
): Promise<ApiResponse<ListResp<ChatSession>>> {
  return request(`/chat/sessions${pageQuery(query)}`, { signal }, list(session));
}

/** GET /api/v1/chat/sessions/:id */
export function getChat(
  sessionId: string,
  signal?: AbortSignal,
): Promise<ApiResponse<{ session: ChatSession; messages: ChatMessage[] }>> {
  return request(`/chat/sessions/${opaquePathId(sessionId)}`, { signal }, (value) => {
    const v = record(value);
    return { session: session(v.session), messages: array(v.messages, message) };
  });
}

/**
 * POST /api/v1/chat/sessions/:id/runs. The callback receives parsed events;
 * AbortSignal cancellation is propagated to fetch and the stream reader.
 */
export async function ask(
  sessionId: string,
  body: AskReq,
  onEvent: (event: SseEvent) => void,
  signal?: AbortSignal,
): Promise<void> {
  const headers = new Headers({
    Accept: "text/event-stream",
    "Content-Type": "application/json",
  });
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  const token = accessToken;
  const combinedSignal = requestSignal(signal);

  const response = await fetch(
    `${API_BASE}/chat/sessions/${opaquePathId(sessionId)}/runs`,
    {
      method: "POST",
      headers,
      credentials: "same-origin",
      body: JSON.stringify(body),
      signal: combinedSignal,
    },
  );
  if (!response.ok) {
    const error = await readError(response);
    combinedSignal.throwIfAborted();
    if (response.status === 401) expireSession(token);
    throw error;
  }
  combinedSignal.throwIfAborted();
  if (!response.headers.get('content-type')?.toLowerCase().startsWith('text/event-stream')) {
    throw new ApiError('问答接口未返回事件流', response.status, 'INVALID_RESPONSE');
  }
  if (!response.body) throw new Error("SSE 响应缺少可读取的数据流");

  const parser = new SseParser();
  const decoder = new TextDecoder('utf-8', { fatal: true });
  const reader = response.body.getReader();
  let terminal = false;
  try {
    for (;;) {
      const { done, value } = await reader.read();
      combinedSignal.throwIfAborted();
      const text = done
        ? decoder.decode()
        : decoder.decode(value, { stream: true });
      const events = parser.feed(text);
      if (done) events.push(...parser.finish());
      for (const event of events) {
        combinedSignal.throwIfAborted();
        onEvent(event);
        if (event.type === "error") {
          throw new Error(event.message || "问答流返回错误");
        }
        if (event.type === "done") {
          terminal = true;
          break;
        }
      }
      if (done || terminal) break;
    }
    if (terminal) await reader.cancel();
  } catch (error) {
    await reader.cancel(error).catch(() => undefined);
    throw error;
  } finally {
    reader.releaseLock();
  }
  if (!terminal) throw new Error("SSE 流在收到终态事件前结束");
}
