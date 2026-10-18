# 补全 ChatService 其余三个用例（按 ADR-0003 双层会话）

## 目标

按 ADR-0003 把 `apps/api/internal/chat` 从原型补齐为可用的 chat 垂直切片：
`CreateChat` 修正 + `GetChat` + `ListSessions` + `Ask`，并接入 `/api/v1` 路由。

## 已核实的前提（非推测）

- `apps/api/migrations/0001_teamflow_core.up.sql` 只有 users/teams/team_members/projects/project_members/tasks，**没有 chat 映射表**。
- `platform/router.go` 无任何 chat 引用，模块未接入 HTTP。
- `ChatRepository` 无数据库，直接 HTTP 调 Python，`Create` 把 Agent 返回 ID 当平台 ID 返回（违反 ADR-0003 §4/§5）。
- 命名错位：`SendMessage` 注释为 ask、`GetMessages` 实为会话列表（返回 `Items/Total/Page/PageSize`）。
- Python 侧：`chat_session(id,title,created_at,updated_at)`、`chat_message(id,session_id,role,content,citations json,images json,decision_log_id,created_at)`；`citations`/`images` 读时自动 JSON 解码、写时须 `json.dumps`；`aexecute` 返回 lastrowid。
- Python 内部路由已有 `X-Agent-Token` 鉴权（`internal_auth.py`），前缀 `/internal/rag/chat`。
- `kb_ids` 仅按请求传入，`None` = 全库未限定；FastAPI 不做授权判断。

## 决策

- **不做** workspace（无 workspace 表与成员模型，ADR 表格里的 `workspace_id` 暂以 `owner_id` 单列授权代替，并在 ADR 后续任务中补 workspace）。
  理由：引入无来源的 workspace 会造出空权限模型；owner 单列与现有 team/project 的 owner 语义一致。
- 平台 ID 用 `platform_sessions.public_id`（随机字符串），Agent 数字 ID 存 `agent_session_id`，**不出网**。
- `status` 采用 ADR-0003 枚举：`creating|active|pending|failed|deleting`。
- 命名纠正：`GetMessages` → `ListSessions`；`SendMessage` → `Ask`（返回 SSE 流）。

## 实施步骤

1. **迁移** `migrations/0002_chat_platform_sessions.up.sql` / `.down.sql`
   - 表 `chat_platform_sessions`：`id`、`public_id`(uniq)、`owner_id`(FK users)、`agent_session_id`(nullable)、`title`、`status`、`last_error_code`、`idempotency_key`(uniq)、`created_at`/`updated_at`、`deleted_at`。
   - CHECK status 枚举；`agent_session_id` 可空以承载 `creating`/`pending`。
2. **AgentClient 端口** `internal/chat/service` 内定义最小接口，适配器放 `internal/chat/repository`（HTTP，含超时、响应体限制、`X-Agent-Token`、幂等键头）。
   - `CreateSession(ctx,title,idempotencyKey) (string,error)`、`GetMessages(ctx,agentSessionID) ([]Message,error)`、`StartRun(ctx,agentSessionID,RunRequest) (io.ReadCloser,error)`、`DeleteSession(ctx,agentSessionID) error`
   - 错误分类：`ErrAgentUnavailable` / `ErrAgentTimeout` / `ErrAgentRejected`，映射 502/504/502。
3. **平台 Repository** `internal/chat/repository`（GORM，同 project/task 风格）：`Create`、`GetByPublicID`、`ListByOwner`、`MarkActive`、`MarkFailed`、`MarkPending`。
4. **Service** 重写 `ChatService.go`：
   - `CreateChat`：鉴权 → 校验标题 → 落 `creating` 记录 + 幂等键 → 调 AgentClient → 成功置 `active`；明确拒绝置 `failed`；超时/未知置 `pending`。
   - `GetChat`：按 `public_id` + `owner_id` 校验归属 → AgentClient 读消息 → 映射公开 DTO（数字 ID 不泄露）。
   - `ListSessions`：按 owner 过滤 + 分页（沿用 `normalizePage`）。
   - `Ask`：校验归属、问题长度（≤4000）、图片数量（≤4）、KB 授权 → 透传 SSE。
5. **Handler DTO 修正**：`ChatHandler.go` 改名、补 `ListSessions` 分页入参、`Ask` 返回流、清理未使用字段。
6. **路由接线**：`platform/router.go` 新增 `protected` 下的 `/chat/sessions`、`/chat/sessions/:id`、`/chat/sessions/:id/messages`、`/chat/sessions/:id/runs`（SSE）。
7. **OpenAPI**：`api/openapi.yaml` 增加 `/api/v1/chat/*` 契约。
8. **测试**：service 层 fake 测试（未登录、他人 session、分页、Agent 失败/超时、Ask 校验），沿用 `fakeXxx` + `errors.Is` 风格。
9. **门禁**：`gofmt`、`go vet ./...`、`go test ./...`、`go build ./...`。

## 明确不在本次范围

- workspace 表与成员模型（需单独 ADR）。
- Python 侧新增 Go 适配端点（现有内部路由已带鉴权，先复用；`skip_user_persist` 不再由浏览器传入）。
- 前端 DTO 与页面切换（仍走旧 `/api/chat`，单独任务）。
- `pending` 对账后台任务（本次只落状态与错误码，对账入口后续补）。

## 风险

- Python `chat_session` 无 owner 列，归属只能由 Go 映射表保证；若有人绕过 Go 直连 Python，归属失效。缓解：部署层限制 Python 内网可达（ADR-0003 已要求）。
- SSE 经 Go 转发需处理取消传播与 `X-Accel-Buffering`；本步骤只做透传，不做事件改写。
