/** @format */

export interface PageQuery {
  page?: number;
  page_size?: number;
}

export interface ListResp<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

/** 后端 DeleteXxxResp 是空结构体，序列化为 {} */
export type EmptyResp = Record<string, never>;

/* ============================================================
 * 实体（对齐各模块 types.go 的 json tag）
 * ============================================================ */

export interface User {
  id: string;
  username: string;
  email: string;
  nickname: string;
  avatar: string;
  created_at: string;
}

export interface Team {
  id: string;
  name: string;
  description: string;
  owner_id: string;
  avatar: string;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  team_id: string;
  owner_id: string;
  status: string;
  start_date?: string;
  end_date?: string;
  created_at: string;
  updated_at: string;
}

export interface Task {
  id: string;
  title: string;
  description: string;
  project_id: string;
  assignee_id?: string;
  creator_id: string;
  status: string;
  priority: string;
  due_date?: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface ChatSession {
  id: string;
  title: string;
  status: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export interface ChatMessage {
  role: string;
  content: string;
  citations?: unknown;
  images?: string[];
  decision_log_id?: string;
  created_at: string;
}

/* ============================================================
 * 请求体（对齐各模块 handler 的 XxxReq 的 json tag）
 * ActorID / TeamID / ProjectID / TaskID 等 json:"-" 字段由服务端注入，前端不传
 * ============================================================ */

export interface RegisterReq {
  username: string;
  email: string;
  password: string;
}

export interface LoginReq {
  username: string;
  password: string;
}

export interface AuthResp {
  token: string;
  user: User;
}

export interface CreateTeamReq {
  name: string;
  description: string;
}

export interface UpdateTeamReq {
  name?: string;
  description?: string;
  avatar?: string;
}

export interface CreateProjectReq {
  name: string;
  description: string;
  start_date?: string;
  end_date?: string;
}

export interface UpdateProjectReq {
  name?: string;
  description?: string;
  status?: string;
  start_date?: string;
  end_date?: string;
}

export interface CreateTaskReq {
  project_id: string;
  title: string;
  description: string;
  priority: string;
  due_date?: string;
}

export interface UpdateTaskReq {
  version?: number;
  title?: string;
  description?: string;
  priority?: string;
  due_date?: string;
}

export interface ListTasksQuery extends PageQuery {
  project_id?: string;
}

export interface CreateChatReq {
  title?: string;
}

export interface AskReq {
  question: string;
  knowledge_base_ids?: string[];
  images?: string[];
}
