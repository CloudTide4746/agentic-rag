# Python Agent 对话链路接入 Go

本文按当前 `backend/`、`apps/api/`、`apps/web/` 源码说明聊天链路；已实现与未对齐事项分开列出。

## 1. 架构

Agent 编排和模型调用继续由 Python 执行。Go 是浏览器访问聊天能力的公开入口，负责认证、平台会话映射、请求校验和 SSE 代理。前端通过 POST + fetch 读取 Go 流，再用现有 `SseParser` 解析。对话生成使用 HTTP/SSE，不使用 WebSocket。

```text
React ChatPage -> web/lib/api/chat.ts -> Go Gin /api/v1/chat/*
  -> ChatService -> AgentClient（内部 HTTP + X-Agent-Token）
  -> FastAPI /internal/rag/chat/* -> Python run_ask / Agent 编排
```

## 2. 当前代码状态

| 区域 | 当前行为 |
| --- | --- |
| Python 编排 | `backend/app/agent/orchestrator.py:run_ask` 执行路由、改写、检索、评估、生成、反思和引用，并产生 SSE 事件。 |
| Python API | `backend/app/routers/chat.py` 提供受内部 token 保护的 `/internal/rag/chat/sessions/*`；问答端点保存图片后调用 `run_ask`。 |
| Go 公开接口 | `apps/api/internal/chat/handler/http.go` 注册 `/api/v1/chat/sessions`、历史读取和 `POST /api/v1/chat/sessions/:id/runs`，由认证保护组挂载。 |
| Go 会话映射 | `apps/api/internal/chat` 保存公开字符串 ID、owner、内部 Agent session ID 和状态；Agent ID 不序列化到公开 Session DTO。状态包含 `creating`、`active`、`pending`、`failed`、`deleting`。 |
| Go Agent 适配器 | `apps/api/internal/chat/repository/agent_client.go` 集中处理内部 URL、`X-Agent-Token`、会话创建/读取和 SSE 请求。 |
| Go 流转发 | Handler 将上游响应体逐块写给浏览器并 Flush，不解析或重写 SSE frame。请求 context 传给上游 HTTP 请求。 |
| 前端 | `apps/web/src/lib/api/sseParser.ts` 解析 `event:`/`data:`；`chat.ts` 通过 fetch、ReadableStream、TextDecoder 消费 POST 流。当前 `chat.ts` 已请求 Go `/api/v1/chat/*`，使用不透明字符串 session ID；会话与历史由 React Query 管理，Zustand 仅保存当前流。 |

## 3. 一轮对话的调用过程

### 3.1 创建会话

调用 `POST /api/v1/chat/sessions`。Go 从认证上下文取得 actor，创建平台会话映射，再通过 AgentClient 调 Python 创建 Agent session。成功后将映射设为 `active`，公开返回平台字符串 ID。明确失败记为 `failed`；超时或响应丢失等结果未知记为 `pending`。创建结果未知时禁止盲目重试，以免重复创建 Agent 会话。

### 3.2 读取历史

调用 `GET /api/v1/chat/sessions/{platform_session_id}` 或 `/messages`。Go 先按平台 ID 和当前 actor 查授权映射，再用映射中的 Agent 数字 ID 请求 Python。浏览器只持有平台 ID，不提交 Agent ID。

### 3.3 提交问题

浏览器公开请求：

```http
POST /api/v1/chat/sessions/{platform_session_id}/runs
Authorization: Bearer <go-access-token>
Accept: text/event-stream
Content-Type: application/json

{"question":"如何申请设备？","knowledge_base_ids":["12"],"images":[]}
```

Go 校验身份、问题、图片、知识库 ID 格式、会话归属及 Agent session 状态，然后通过内部请求调用 Python：

```http
POST /internal/rag/chat/sessions/{agent_session_id}/ask
X-Agent-Token: <internal-token>
Accept: text/event-stream
Content-Type: application/json

{"question":"如何申请设备？","kb_ids":["12"],"images":[],"skip_user_persist":false}
```

`skip_user_persist` 不是浏览器字段，由平台控制。FastAPI 调用 `run_ask`，Python 按 Agent 阶段生成 SSE，Go 透传并 Flush，前端解码和解析后更新流式 UI。

## 4. SSE 事件

Python 产生的 frame 形如：

```text
event: generate
data: {"step":6,"ts":1790000000000,"detail":"","delta":"回答片段"}

```

事件名来自 `event:` 行，JSON payload 不一定重复包含 `type`；前端 parser 将 event 名映射为 `SseEvent.type`。

| 事件 | 用途 | 常见字段 |
| --- | --- | --- |
| `router` | 意图路由 | `intent`、`confidence` |
| `rewrite` | 查询改写 | `rewritten_query`、`sub_queries`、`hyde_used` |
| `retrieve` | 检索进度 | `source`、`hit_count` |
| `grade` | 检索评估 | `score`、`verdict`、`reason` |
| `correct` | 纠错/重试决策 | `action`、`round` |
| `generate` | 回答增量 | `delta` |
| `reflect` | 生成后反思 | `result`、`round`、`missing_info` |
| `citation` | 引用 | `citations` |
| `done` | 正常终态 | `message_id`、`elapsed_ms`、`token_usage` |
| `error` | 错误终态 | `code`、`message` |

只有 `done` 表示成功；`error` 表示失败；无终态 EOF 表示中断。当前 Go 不检查终态，也不合成终态事件；读写错误会结束转发。前端不能把 reader 结束直接视为成功。

## 5. 前端对接步骤

1. 将 `apps/web/src/lib/api/chat.ts` 的创建、列表、历史和提问路径切到 Go `/api/v1/chat/*`。
2. 会话 ID 类型改为 Go 返回的不透明字符串；Agent session ID 不进入前端 DTO、URL 或状态。
3. 按 Go 认证契约携带 bearer token。问答是 JSON POST，继续使用 fetch，不使用只支持 GET 的原生 EventSource。
4. 用 `TextDecoder` 增量解码 `ReadableStream`，文本交给 `SseParser`；流结束时 flush decoder。
5. `generate.delta` 用于增量回答；处理 `citation`、`done` 和 `error`。无终态 EOF 显示为中断。
6. 用 `AbortController` 支持用户取消；取消后停止写入 UI 状态。Go 将请求 context 传给上游，但 Python 模型调用是否停止须用集成测试确认。
7. 收到终态后从 Go 刷新历史；流式草稿只是临时 UI 状态，不是持久化事实。

## 6. 尚未打通的事项

- **前端接入已完成、真实模型未验收：** 当前 `chat.ts` 已访问 Go `/api/v1/chat/*` 并使用字符串 ID。登录、会话分页/历史、流式回答、阶段数据和终态已接入；本机 Agent 运行环境与模型配置尚未就绪，未完成真实模型端到端验收。
- **知识库授权缺口：** 当前 Go 对 `knowledge_base_ids` 进行数字格式和重复值校验后传给 Agent；代码中尚未看到按当前用户权限映射平台 KB 到 Agent `kb_ids`。接入真实用户前必须补上服务端授权与映射，不信任浏览器提交的 ID。
- **SSE 元数据未补充：** Go 当前原样透传，没有注入 `run_id`、公开 `request_id` 或递增 `seq`。若客户端需要这些字段，先定义契约，再实现并测试。
- **中途断流：** headers 发出后无法再返回 JSON 错误；当前 Go 读写错误时结束流。前端应将无 `done`/`error` 的 EOF 视为中断，不要自动重放可能已落库的问题。
- **取消语义：** context 取消传到上游 HTTP 请求，不足以单凭源码证明 Python/模型任务停止；需端到端验证。
- **KB 空值：** 不得把无授权知识库集合改成 Python 侧可能表示“全库”的 `null`。授权映射完成前，不开放真实知识库检索。
- **动态 payload：** SSE JSON 来自 Python 运行时，TS 类型不是运行时校验；解析失败或未知事件不能伪装成成功。

## 7. 配置与联调

Go 使用 `AGENT_BASE_URL` 和 `INTERNAL_API_TOKEN`；Python 使用 `INTERNAL_API_TOKEN` 校验 `X-Agent-Token`。两边 token 必须一致。真实 token 不提交、不下发浏览器、不写日志。

联调顺序：启动 Python 及依赖；启动 Go 并登录；创建会话并确认平台 ID/`active` 状态；用 `curl -N` 调 Go runs 检查 SSE frame 是否逐步到达；切换前端路径后验证创建、历史、增量文本、引用和终态；再验证无效身份、他人会话、Agent 不可达/超时、越权 KB、取消和异常 EOF。

```bash
curl -N \
  -H 'Authorization: Bearer <GO_ACCESS_TOKEN>' \
  -H 'Accept: text/event-stream' \
  -H 'Content-Type: application/json' \
  -d '{"question":"请概述这份资料","knowledge_base_ids":[],"images":[]}' \
  'http://127.0.0.1:8080/api/v1/chat/sessions/<PLATFORM_SESSION_ID>/runs'
```

## 8. 验收

- [ ] 浏览器只访问 Go `/api/v1/chat/*`；Agent 内部 ID 不暴露。
- [ ] Go 对会话和知识库做服务端授权；内部 token 不进入浏览器、URL 或日志。
- [ ] OpenAPI、Go DTO 和前端 DTO 字段一致；SSE 响应为 `text/event-stream` 并及时 Flush。
- [ ] 前端增量显示 `generate.delta`，并区分 `done`、`error`、异常 EOF。
- [ ] 超时、取消、Agent 不可用、重复创建和跨用户会话访问有测试证据。
- [ ] 前端运行 `npm run typecheck`、`npm test`、`npm run build`；Go 运行 `gofmt`、`go vet ./...`、`go test ./...`、`go build ./...`；端到端联调结果另行记录。
