# 教程 1：迁移聊天会话与 SSE 流式问答

本章覆盖旧会话列表/创建和问答流。重点是让 Go 完成身份、会话授权与公开协议，Python 继续负责 Agent 编排，并由 Go 通过内部 HTTP 逐帧转发 SSE。

## 1. 先看清旧行为

当前 Python 实现位于 `backend/app/routers/chat.py`：

| 旧接口 | 当前行为 |
| --- | --- |
| `GET /api/chat/sessions` | 返回 `sessions.list_sessions()` 的数组；旧查询按更新时间和 ID 倒序，接口没有调用者过滤。 |
| `POST /api/chat/sessions` | 无请求体，创建会话，返回 201。 |
| `GET /api/chat/sessions/{session_id}/messages` | 检查 Agent 会话存在后返回消息数组。 |
| `POST /api/chat/sessions/{session_id}/ask` | 校验会话并保存图片，然后调用 `run_ask`，以 `text/event-stream` 返回。 |

旧接口没有用户鉴权。迁移不能把“没有过滤”原样复制成 Go 行为。也不能因为 Agent 数据库里有数字 `session_id`，就把它作为浏览器端的授权依据。

## 2. 会话列表/创建怎么迁

### 2.1 先定平台映射

平台侧需要一条由 Go 独占写入的会话映射记录，至少包含：

```text
platform_session_id (公开字符串 ID)
workspace_id
owner_user_id
agent_instance_id
agent_session_id (内部数字 ID，可空)
status: pending | ready | failed
created_at / updated_at
```

Agent 的 session/message 表仍由 Python 写。Go 只维护平台授权和资源映射，不复制一份可编辑的消息历史。映射表属于平台数据，因此 schema、唯一约束和失败对账流程在实施前要通过 ADR/设计确认；这属于数据所有权与跨服务创建语义变化。

### 2.2 定义 Go 公共接口

建议：

```http
GET  /api/v1/chat/sessions?page=1&page_size=20
POST /api/v1/chat/sessions
```

受保护接口从已验证身份取 `actor_id`，不可接受客户端传入 owner 或 Agent session ID。

- `GET`：按当前用户有权访问的 workspace 过滤，分页，排序稳定；响应为 `{"data":{"items":[],"total":0,"page":1,"page_size":20},"request_id":"..."}`。
- `POST`：由身份/当前工作空间决定 owner；创建平台映射并创建 Agent 会话。Agent 不可用时不能返回假成功。
- 若两个系统间调用失败或响应丢失，创建结果可能未知。不要盲目重试创建；记录 `pending`，提供按关联键查询/对账能力，最终转 `ready` 或 `failed`。不要声称有跨库事务。

服务调用顺序建议：

```text
Gin Handler -> Chat Application Service -> 平台 Repository
                                  └-----> AgentClient (Python 内部 API)
```

Handler 只解析 HTTP 和映射响应；Service 校验用户和工作空间、处理 pending/失败状态；Repository 只访问平台库；AgentClient 封装 Python URL、内部鉴权、超时与协议。不要让 Handler 自行拼 Agent URL。

当前 FastAPI 并没有适合 Go 调用的会话创建/列表内部协议（它们是旧公开路由）。正式启用 Go 代理前，需新增有内部鉴权的 Agent 适配端点，或另行设计安全的内部协议。不能直接假定旧端点已具备服务身份校验。

## 3. SSE 请求与数据流

### 3.1 浏览器只请求 Go

迁移后由浏览器请求：

```http
POST /api/v1/chat/sessions/{platform_session_id}/runs
Authorization: Bearer <Go access token>
Content-Type: application/json
Accept: text/event-stream

{"question":"如何申请设备？","knowledge_base_ids":["kb-platform-id"],"images":[]}
```

Go 先认证，再校验会话属于调用者可访问范围，校验每个知识库和附件权限，然后从平台映射得到 Agent 数字 session/KB ID。绝不信任浏览器提供的 Agent ID。授权知识库集合为空时直接拒绝；不可把空集合转成 `null` 后交给旧 Agent。

Go 内部请求 FastAPI（示意）：

```http
POST http://agent-internal/api/chat/sessions/{agent_session_id}/ask
Authorization: Bearer <service credential>
Content-Type: application/json
Accept: text/event-stream

{"question":"...","kb_ids":[12,18],"images":[...]}
```

内部协议只能在受限网络中使用，并应由 Agent 旁的内部代理或明确实现的 Agent 端点校验服务凭据。浏览器不持有内部凭据。旧 `skip_user_persist` 是两次请求重答方案的一部分，不应直接成为新公共字段。

### 3.2 Go 侧转发策略

普通 JSON 客户端/`io.ReadAll` 会缓存完整响应，不适合 SSE。AgentClient 应使用 `http.Client.Do` 获取响应后，持续从 `resp.Body` 读取；Handler 将每个完整 SSE frame 写到 `http.ResponseWriter` 并调用 `http.Flusher.Flush()`。

时序：

```text
浏览器 -> Go Handler -> 授权/映射 -> AgentClient -> FastAPI Agent
浏览器 <- Go Flush  <- 解析/补公共字段 <- SSE frame <- run_ask
```

建议将协议解析放在 `internal/agentclient`，让它发出完整帧或结构化事件；Chat Handler 负责写 HTTP。若采用 `io.Pipe` 做透明转发，也要有专门的 SSE 验证，且 Go 仍需识别 `done/error/EOF` 来更新运行状态。

## 4. Go 中读取并转发 SSE 的核心示意

下面展示核心控制流，省略项目 DTO/统一错误函数和具体鉴权代码。它不是可直接粘贴的完整实现：需要配合本项目的错误 envelope、Chat 用例、Agent 内部鉴权及 SSE parser 完成。

```go
func (h *ChatHandler) CreateRun(c *gin.Context) {
	ctx := c.Request.Context() // 浏览器断开时向下游传播取消
	actorID := authenticatedActorID(c)
	platformSessionID := c.Param("session_id")

	var req CreateRunRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Question) == "" {
		writeAPIError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "问题不能为空")
		return
	}

	// Service 必须校验会话归属、KB 权限和附件；并返回内部 Agent ID。
	plan, err := h.chat.PrepareRun(ctx, actorID, platformSessionID, req)
	if err != nil {
		writeChatError(c, err) // 在写 SSE headers 之前才能回标准 JSON 错误
		return
	}

	upstream, err := h.agent.Ask(ctx, plan.AgentSessionID, plan.AgentKBIDs, req.Question, plan.AgentImages)
	if err != nil {
		writeUpstreamError(c, err)
		return
	}
	defer upstream.Close()

	if upstream.StatusCode < 200 || upstream.StatusCode >= 300 {
		writeUpstreamStatusError(c, upstream.StatusCode)
		return
	}
	if !strings.HasPrefix(strings.ToLower(upstream.ContentType), "text/event-stream") {
		writeUpstreamProtocolError(c)
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		writeAPIError(c, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "服务不支持流式响应")
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher.Flush() // 从此刻起不能再改成普通 JSON 状态/错误响应

	terminal := false
	for {
		frame, err := upstream.NextFrame(ctx) // 正确解析 SSE 边界，不是每次 Read 都是一个事件
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_ = h.chat.MarkRunFailed(ctx, plan.RunID, classifyStreamError(err))
			writeSSEError(c.Writer, flusher, plan.RunID, "UPSTREAM_STREAM_ERROR", "问答流中断")
			return
		}

		publicFrame, isTerminal, err := h.chat.MapAgentFrame(ctx, plan, frame)
		if err != nil {
			_ = h.chat.MarkRunFailed(ctx, plan.RunID, "invalid_agent_event")
			writeSSEError(c.Writer, flusher, plan.RunID, "INVALID_AGENT_EVENT", "问答事件解析失败")
			return
		}
		if _, err := c.Writer.Write(publicFrame); err != nil {
			// 通常表示浏览器断开。request context 会取消上游；运行状态设为 interrupted。
			_ = h.chat.MarkRunInterrupted(context.Background(), plan.RunID, "client_disconnected")
			return
		}
		flusher.Flush()
		if isTerminal { // 只有 done 算 completed；error 按 failed 处理
			terminal = true
			break
		}
	}

	if !terminal {
		// EOF 没有 done/error 是 interrupted，不能当成功。
		_ = h.chat.MarkRunInterrupted(context.Background(), plan.RunID, "eof_without_terminal_event")
		return
	}
	_ = h.chat.MarkRunCompleted(ctx, plan.RunID)
}
```

**实施注意：**示例里后台记录状态的错误不能长期忽略。正式实现要定义状态写入失败时的日志/告警和恢复策略，并避免将客户端断开后的状态写入绑在已取消的请求 context 上（同时给后台状态更新设置短 deadline）。

## 5. SSE parser 为什么不能按 Read 切帧

TCP/HTTP body 是字节流。一次 `Read` 可能只有半个 UTF-8 字符、半行 `data:`、一个事件的一部分，也可能同时包含多个事件。简单 `bufio.Scanner` 默认 token 上限也可能不适合长事件。

Parser 至少要正确处理：

- `\n` 和 `\r\n` 行结束；空行才结束一个 SSE event。
- 多行 `data:` 合并为一个 data payload。
- 一次 Read 含多个事件，或事件跨多次 Read。
- UTF-8 字符跨读取边界；设置有明确上限的事件大小并把超限作为协议错误。
- comment 行（以 `:` 开头）可忽略；未知事件名记指标/日志后按契约策略处理。
- `event: done` 才是完成；`event: error` 失败；没有终态的 EOF 是 interrupted。

Agent 旧事件名包括 `router/rewrite/retrieve/grade/correct/generate/reflect/citation/done/error`。Go 对外增加 `run_id`、`request_id`、单调递增 `seq`，内部 `message_id` 映射成平台 ID。转发/解析 `generate` 时要验证旧实现的替换/重写语义，不能假定每个 delta 永远只追加。

## 6. 超时、断连和错误边界

- 使用复用连接的共享 `http.Client`，设置连接超时、响应头超时、响应体/事件大小限制。流的整体截止时间应覆盖 Agent 的最大运行时间，并预留 Go 转发时间，不能套普通短请求 timeout。
- 请求 context 从 Gin 传给 Agent 请求。浏览器断开时取消向下游传播；这只能保证 Go 停止等待/读取，不能在未验证 Python 取消行为前宣称模型计算一定停止。
- SSE headers 写出前：返回统一 JSON 错误和真实 HTTP 状态，例如 401、403、404、502、503、504。
- headers/首帧写出后：不能再改变 HTTP status，也不能附加 JSON body；用 SSE `error` event 结束，并把 run 标记 `failed`。
- 客户端中断或无终态 EOF：run 标记 `interrupted`。绝不自动重放用户问题。
- 中途发生写失败通常表示客户端已离开，避免尝试再写错误帧；记录状态并退出。

## 7. 迁移切换次序

1. 先实现 Go 平台会话映射、授权和 Agent 内部协议；未完成这些，不开放 `/api/v1/chat/sessions`。
2. 先迁移 GET/POST 会话并验证所有者隔离、Agent 创建未知结果和对账。
3. 再接入问答 SSE，验证半包、CRLF、多事件同块、Unicode、`done`、`error`、无终态 EOF、超时和客户端取消。
4. 前端改为只访问同源 Go `/api/v1`；检查 Vite 的 `API_PROXY_TARGET`，并确认浏览器网络面板没有 Python 请求。
5. 观测稳定后逐环境切流。出现问题时将前端流量切回上一版发布；Agent 和 Go 不可同时写同一 Agent 业务表，因此回滚不能通过让 Go 直接读写 Agent 表来实现。

## 8. 完成前检查

- OpenAPI 写明安全要求、请求/响应字段、HTTP 错误、SSE media type 和事件 schema/样例。
- 跨用户会话、跨 workspace KB 和附件访问均拒绝；空授权 KB 集合拒绝。
- SSE parser 有边界测试；端到端用真实 Agent 另行验证其取消和生成重写语义。
- Go `gofmt`、`go vet ./...`、`go test ./...`、`go build ./...` 通过；前端类型检查/测试/构建通过。
- 本章只是实施指导；在 Go 路由、Agent 内部认证和完整流式用例真正落地前，不得在 README 宣称真实 RAG 流已迁移完成。
