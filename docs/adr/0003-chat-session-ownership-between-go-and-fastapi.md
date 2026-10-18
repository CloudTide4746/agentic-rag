# ADR-0003：Go 与 FastAPI 的聊天会话边界和数据所有权

- 状态：已采纳
- 日期：2026-10-16
- 决策范围：`apps/api` 的聊天公开接口与 `backend` Agent 运行时之间的会话、消息和问答调用

## 背景

仓库同时存在 Go TeamFlow API 和 FastAPI Agent。旧 FastAPI 暴露了会话创建、会话列表、消息历史和问答流接口，但它没有平台用户身份、工作区归属和知识库授权边界。浏览器如果直接使用这些接口，会绕过 Go 的认证和授权规则；如果 Go 直接读写 FastAPI 的数据库，又会破坏模块数据所有权。

因此需要明确 `CreateChat` 的职责：它既不能只在 Go 内生成一个没有 Agent 对应关系的 ID，也不能让 HTTP handler 直接调用 FastAPI 或把 Agent 的内部 ID 返回给浏览器。

## 决策

采用“双层会话”设计：

1. Go 是唯一公开入口。Go 负责认证身份、工作区授权、平台会话 ID、会话状态、分页、错误映射和生命周期编排。
2. FastAPI 继续作为内部 Agent 运行时。FastAPI 负责 Agent 会话、消息正文、引用、图片关联、决策日志、检索过程和模型调用。
3. Go 不跨库查询或写入 FastAPI 的 Agent 表。Go 通过专用 `AgentClient` 调用 FastAPI 的内部接口。
4. 浏览器只接触 Go 生成的平台会话 ID。FastAPI 的数字会话 ID 只存放在 Go 的内部映射中。
5. `CreateChat` 由 Go 的应用服务编排：先校验调用者，再创建 Go 平台会话映射，再调用 FastAPI 创建 Agent 会话，最后返回平台会话。

推荐的依赖方向如下：

```text
HTTP Handler -> Go ChatService -> Go ChatRepository
                           -> AgentClient -> FastAPI Agent API
```

`AgentClient` 由 Go 应用侧定义最小端口，HTTP 适配器在外层实现：

```go
type AgentClient interface {
	CreateSession(ctx context.Context, title string, idempotencyKey string) (string, error)
	GetMessages(ctx context.Context, agentSessionID string) ([]AgentMessage, error)
	StartRun(ctx context.Context, agentSessionID string, req AgentRunRequest) (AgentStream, error)
	DeleteSession(ctx context.Context, agentSessionID string) error
}
```

## Go 需要保留的内容

Go 侧应保留一个平台会话映射实体，至少包含：

| 字段 | 用途 |
| --- | --- |
| `id` | 对外平台字符串 ID |
| `owner_id` | 会话所有者 |
| `workspace_id` | 授权范围 |
| `agent_session_id` | FastAPI 内部会话 ID，仅内部使用 |
| `title` | 列表展示标题，可由首条问题同步或异步更新 |
| `status` | `creating`、`active`、`pending`、`failed`、`deleting` 等生命周期状态 |
| `created_at`、`updated_at` | 平台排序和审计 |
| `last_error_code` | 可恢复失败的机器可读原因，可选 |

Go 负责以下行为：

- 从认证上下文取得 `owner_id`，不信任客户端提交的操作者 ID。
- 校验工作区和知识库权限。
- 对外使用 `/api/v1/chat/sessions` 和 `/api/v1/chat/sessions/{id}/messages` 等平台路径。
- 将平台会话 ID映射为 Agent 会话 ID，再调用 FastAPI。
- 对 FastAPI 的错误映射为稳定的 `401/403/404/409/502/503/504` 等公开状态。
- 对列表做授权过滤、分页和排序。
- 记录创建结果未知时的 `pending` 状态，并提供对账或恢复入口。
- 在删除时先校验平台会话归属，再协调 Agent 数据清理和平台映射删除。

## FastAPI 需要保留的内容

FastAPI 保留所有 Agent 运行时数据和行为：

- Agent 会话记录及其内部 ID。
- `chat_message` 消息正文、角色、引用、图片 URL、决策日志关联和创建时间。
- Agent 决策日志、路由、检索、重写、生成、反思和引用事件。
- Agent 的模型调用、知识库检索、引用组装和 SSE 事件流。
- Agent 运行时需要的会话历史窗口和消息持久化逻辑。
- Agent 图片文件的处理和内部消息图片关联，前提是通过 Go 的受保护附件入口对外提供。
- 面向 Go `AgentClient` 的内部接口，例如创建 Agent 会话、读取消息、启动问答流和删除 Agent 会话。

FastAPI 的接口必须是内部接口，至少具备内部鉴权、超时、请求体限制和幂等键支持。它不负责判断 TeamFlow 用户是否属于某个工作区，也不接受浏览器直接传入的 Go 平台身份或未授权知识库 ID。

## FastAPI 不需要保留的内容

以下职责不应继续作为 FastAPI 的平台职责：

- 浏览器可直接访问的公开聊天入口。
- TeamFlow 用户、团队、工作区、项目成员和知识库授权判断。
- Go 平台会话 ID的生成、公开、分页和归属判断。
- 通过 URL 或请求体接收并信任 `owner_id`、`workspace_id` 等平台身份字段。
- 直接决定调用者能看到哪些会话或消息。
- 把 FastAPI 的整数 `session_id` 作为公开 API ID 返回。
- 让 Go 直接连接 Agent 数据库读取或写入 `chat_session`、`chat_message`、决策日志等表。
- 旧的 `skip_user_persist` 公共参数。是否持久化由 Go 的用例和内部 Agent 协议决定，不能由浏览器绕过平台审计。
- 未经 Go 授权的全局知识库 ID 查询和引用查询。
- 旧的无授权 `/api/chat/*` 浏览器入口；迁移完成后应关闭、限制为内部网络，或由部署层阻断。

这不要求立即删除 FastAPI 内部实现。迁移期间可以保留兼容路由，但兼容路由只能供 `AgentClient` 使用，并应标记为内部协议，不能继续作为公开契约。

## 创建会话流程

```text
1. 浏览器 -> Go: POST /api/v1/chat/sessions
2. Go: 验证身份、工作区和标题长度
3. Go: 创建平台记录，状态 creating，生成幂等键
4. Go -> FastAPI: 带内部鉴权和幂等键创建 Agent 会话
5. FastAPI: 创建 Agent session，返回 agent_session_id
6. Go: 保存映射，状态 active
7. Go -> 浏览器: 返回平台 session_id 和 session DTO
```

如果 FastAPI 明确拒绝，Go 将平台记录置为 `failed` 并返回可识别错误。如果网络超时或连接中断导致结果未知，Go 将记录置为 `pending`；不得无条件重试创建请求，因为这可能产生两个 Agent 会话。恢复机制应优先使用幂等键查询或后台对账。

## 消息和问答流程

读取消息时，Go 先根据平台会话 ID校验归属，再由 `AgentClient` 使用内部 Agent ID读取 FastAPI 消息；FastAPI 返回的正文必须经过 Go 的公开 DTO 映射。

发送问题时，浏览器只提交问题和允许的附件引用。Go 校验会话、输入长度、附件和知识库权限，并根据授权结果生成内部 Agent 请求。FastAPI 产生的 SSE 由 Go 转发；浏览器不直接连接 FastAPI。流开始后发生错误时，使用约定的 SSE `error` 事件；未收到 `done` 的 EOF 不能标记为成功。

## 删除和失败恢复

删除流程由 Go 发起并授权。Go 将平台记录置为 `deleting`，调用 FastAPI 删除 Agent 会话和关联附件，再删除或标记平台映射。外部删除结果未知时保留可重试状态和对账信息，不返回假成功。

FastAPI 不可用时，Go 应返回上游不可用或超时错误，并保留平台记录的状态。禁止在创建、发送问题或删除等结果未知的写操作上进行没有幂等保证的盲目重试。

## 备选方案

- **只在 Go 创建会话**：拒绝。会产生没有 Agent 运行时对应关系的空平台记录，首次提问时还要重新处理映射和失败恢复。
- **只在 FastAPI 创建会话**：拒绝。Go 无法在创建时建立平台归属、授权和生命周期状态，且会暴露内部资源边界。
- **Go 直接访问 FastAPI 数据库**：拒绝。跨模块查表破坏数据所有权，绕过 Agent 适配协议并增加迁移耦合。
- **浏览器继续直连 FastAPI**：拒绝。无法保证 TeamFlow 身份、工作区授权、错误契约和平台审计。

## 安全影响

平台 ID 与 Agent ID 分离可以减少内部资源枚举风险。Go 必须在每次读取、发送和删除前执行授权；FastAPI 必须验证来自 Go 的内部身份和请求签名或服务凭据。日志中不得记录令牌、完整问题、完整回答或 Agent 内部密钥。

## 迁移与验证

1. 先实现 Go 平台会话映射表和 `AgentClient` 内部协议。
2. 将旧 FastAPI 路由改为内部调用，补充幂等键、超时和错误码。
3. 实现 Go 的创建、列表、历史和 runs 用例，再同步 OpenAPI 与前端 DTO。
4. 验证创建成功、FastAPI 明确失败、超时结果未知、重复幂等键、他人会话、空知识库授权、消息读取和删除恢复路径。
5. 迁移完成后关闭旧的浏览器直连路由，并保留内部协议的版本和兼容窗口。

实现阶段至少运行 `gofmt`、`go vet ./...`、`go test ./...`、`go build ./...`，并校验 OpenAPI、数据库 migration 和 AgentClient contract 测试。本文的采纳不代表这些实现已经完成。

## 实施记录（2026-10-16）

本轮已在 `apps/api` 落地创建、列表、历史和问答用例，并接入 `/api/v1/chat/*`。以下三点与本 ADR 原文的差异已明确，不是遗漏：

1. **`workspace_id` 暂未落地。** 本仓库当前没有 workspace 表与成员模型，映射表以 `owner_id` 单列做授权范围。引入无来源的 workspace 会造出空权限模型，因此留待 workspace 能力单独设计后再补列，届时本表增加 `workspace_id` 并把授权条件从 owner 扩展为工作区成员。
2. **状态枚举采用本文的 `creating|active|pending|failed|deleting`**，`docs/fastapi-to-go-tutorial/01-chat-sessions-and-sse.md` 中的 `pending|ready|failed` 不再使用。
3. **复用 FastAPI 现有内部路由**（`/internal/rag/chat/*`，已具备 `X-Agent-Token` 内部鉴权），未新增独立的 Go 适配端点。教程指出的“FastAPI 尚无适合 Go 的会话创建/列表内部协议”在本轮以内部路由 + 令牌鉴权 + 超时 + 幂等键满足；`skip_user_persist` 改为由平台决定，不再接受浏览器传值。

已验证的行为（真实 MySQL 8.4 + 真实 HTTP 调用，非仅单元测试）：

- 迁移 `0002_chat_platform_sessions` 在空库上正反向执行成功，CHECK、两个唯一索引和 owner 外键均按预期拒绝非法数据。
- 创建：Agent 明确拒绝 → 502 `AGENT_REJECTED` 且映射 `failed`；连接不可达 → 503 `AGENT_UNAVAILABLE` 且映射 `failed`；Agent 不响应 → 504 `AGENT_TIMEOUT` 且映射 `pending`。
- 授权：他人会话的读取、消息和提问均返回 404（不泄露存在性），无令牌返回 401，列表按 owner 过滤且分页正确。
- 问答：SSE 以 `text/event-stream` 逐帧转发，内部请求携带平台决定的 `kb_ids`。

尚未实现，不得视为已完成：

- `pending` 记录的后台对账/恢复入口。
- 删除会话用例（`deleting` 状态已预留，但无公开删除接口）。
- 前端从旧 `/api/chat` 切换到 `/api/v1/chat/*`，以及 workspace 授权范围。
