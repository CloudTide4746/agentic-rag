# FastAPI 迁移到 Go：详细接口文档

> 迁移设计稿。FastAPI 现状按运行时 Swagger 与源码整理；Go 路由是目标映射，不表示已实现。
> 基线：`http://localhost:8000/docs` 和 `http://localhost:8000/openapi.json`，采集日期 2026-10-08。OpenAPI 标题 Agentic RAG API，版本 1.0.0，共 18 个路径、23 个操作。

## 1. 边界与通用约定

- Go 是唯一公开业务入口，路径前缀 `/api/v1`；浏览器不直连 Python。
- Go 负责身份、工作空间和资源授权、平台 ID、任务/运行状态。Python 继续唯一写入 Agent 会话、消息、决策日志、文档、切片、向量和索引。Go 通过 `internal/agentclient` 调用 Python，不跨库查写 Agent 表。
- Go 对外 ID 为平台字符串，时间为 UTC RFC3339；Agent 数字 ID 只保留在内部映射。
- 当前 FastAPI Swagger/handler 没有鉴权。Go 必须逐请求授权；管理员查看他人问答正文另需权限。
- JSON 成功/错误采用 `{data,request_id}` / `{error:{code,message},request_id}`；SSE、图片/文件流不套 envelope。
- 浏览器不传 Agent ID 或 `skip_user_persist`。Go 从调用者权限映射出 Agent `kb_ids`；授权 KB 集合为空时拒绝，不得传 null。
- 上传返回 202 只代表接受处理。FastAPI BackgroundTasks 不保证可靠队列、幂等或进程重启恢复。

`apps/api/README.md` 说明 Go 当前只有健康检查骨架，没有业务 `/api/v1` 路由。本文件是迁移说明，不是已交付 API。

### Swagger 漂移

以本次运行时 `localhost:8000/openapi.json` 为准。仓库 `apps/api/openapi.yaml/json` 不完全同步：旧 YAML 声称 ask 有 409 活动运行冲突，但运行时 Swagger 只列 200/422，Python handler 也没有 active-run 锁。Go 如加同会话并发控制，是需实现验证的新行为。

README 称 Vite 默认代理 8000，但 `apps/web/vite.config.ts` 默认目标为 8080（`API_PROXY_TARGET` 可覆盖）；联调须核实代理目标。

## 2. 路由迁移表

| FastAPI 当前 | Go 目标或处理 |
| --- | --- |
| GET/POST `/api/chat/sessions` | GET/POST `/api/v1/chat/sessions`，限定调用者范围 |
| GET `/api/chat/sessions/{session_id}/messages` | `/api/v1/chat/sessions/{session_id}/messages`，授权后 AgentClient 读取 |
| DELETE `/api/chat/sessions/{session_id}` | 同路径迁移到 `/api/v1`，授权删除并处理附件 |
| DELETE `.../last-assistant` | 首版不公开，旧重答两请求存在竞态 |
| GET `/api/chat/images/{name}` | 改 Go 受保护附件读取并校验会话权限 |
| POST `/api/chat/sessions/{id}/ask` | POST `/api/v1/chat/sessions/{id}/runs`，Go 授权并转发 SSE |
| GET/POST `/api/admin/kb` | GET/POST `/api/v1/workspaces/{workspace_id}/knowledge-bases` |
| GET/POST `/api/admin/kb/{kb_id}/documents` | GET/POST `/api/v1/knowledge-bases/{knowledge_base_id}/documents` |
| DELETE `/api/admin/kb/{kb_id}/documents/{doc_id}` | 同文档资源改 `/api/v1`，设计部分失败补偿 |
| POST `.../documents/{doc_id}/retry` | 建议同文档资源 `/retry`，有幂等 job/attempt |
| GET `.../documents/{doc_id}/chunks` | 建议同文档资源 `/chunks`，逐层授权 |
| GET `/api/admin/chunks/by-citation` | 不复刻全局文件名查询；使用稳定文档/切片 ID |
| GET/PUT `/api/admin/datasources` | 暂不作公开 RAG API；先定权限、白名单、审计 |
| GET `/api/admin/decision-logs[/{log_id}]` | 如保留，单独受控；正文内容权限需独立 |
| POST `/api/admin/retrieval-test/lane`、`/fuse` | 暂不公开；database lane 可能走 NL2SQL |
| GET `/api/health` | Go 运维健康端点；README 记为 GET `/healthz` |

仓库基线另列 `GET /api/v1/ingestion-jobs/{id}` 和 `GET /api/v1/chat/runs/{id}`；这两条是 Go 目标，不是现有 FastAPI 路由。

## 3. 字段、响应及状态码

运行时 Swagger 对多数数据库字典响应没有完整 response schema；下列字段由 Python handler 和 `apps/web/src/types/legacy.ts` 补充确认，不是 Swagger 对响应类型的完整保证。

- Session：`id,title,created_at,updated_at`。
- Message：`id,session_id,role,content,citations,images,decision_log_id,created_at`；role 为 user/assistant。
- KnowledgeBase：`id,name,description,doc_count,chunk_count,created_at,updated_at`。
- Document：`id,kb_id,file_name,file_type,status,chunk_count,error_msg,created_at,updated_at`；状态为 parsing/chunking/embedding/ready/failed。
- Chunk：`id,doc_id,chunk_no,section_path,content,token_count`。
- 旧分页：`{items,total,page,page_size}`；chunks 和 decision logs 默认 page_size=8，源码上限 50。Go 基线建议默认 20、上限 100。
- Datasource：`source_type,enabled,config,updated_at`；source 为 vector/keyword/database。
- DecisionLog：`id,session_id,message_id,question,final_answer,steps,total_rounds,elapsed_ms,created_at`；step 包含 step/type/detail/payload/elapsed_ms。
- Retrieval hit：`title,snippet,score,meta`；lane：`source,latency_ms,hits`；融合项：`key,rrf,from_lanes,hit`；grade：`score,verdict,reason`。

目标状态码：200 读取/更新/删除；201 同步创建；202 已接受异步任务；400 请求/业务前置错误；401 未认证；403 无权限；404 不存在或不可见；409 唯一性/状态/并发冲突（仅在 Go 实际实现后声明）；413 超限；422 可作为字段验证状态但须写入 OpenAPI；500/502/503/504 为 Go/Agent 错误、上游不可用或超时。FastAPI 旧错误体通常是 `{"detail":"..."}` 或校验错误 `{"detail":[...]}`；迁移统一 error envelope 会影响前端。

文件上传仅允许 md/txt/pdf/docx，最大 20 MiB（20*1024*1024 bytes）。图片每条最多 4 张，单图解码后最大 5 MiB，允许 PNG/JPEG/WebP/GIF 并校验文件头。Go 限制取 Go/Agent 较小值，验证内容，文件名不得作为磁盘路径。

## 4. Chat 接口

### 创建会话

旧 `POST /api/chat/sessions` 无 body，201 返回 Session。Go 为 `POST /api/v1/chat/sessions`；身份决定 owner/workspace，不接收 Agent session ID。Agent 创建结果未知时记录 pending 并对账，不盲目重试。

### 会话列表

旧 `GET /api/chat/sessions` 返回数组，按 updated_at DESC、id DESC 排序，无分页。Go 为 `GET /api/v1/chat/sessions?page=1&page_size=20`，只返回调用者可访问记录。

### 消息历史

旧 `GET /api/chat/sessions/{session_id}/messages` 返回消息数组，不存在 404。Go 保持路径语义但先授权，再经 AgentClient 读取；管理员看他人正文需独立权限。

### 删除会话

旧 `DELETE /api/chat/sessions/{session_id}` 成功返回 `{"deleted":<旧整数ID>}`，不存在 404；旧实现还清理消息引用图片。Go 对外只返回平台 ID/明确成功体，清理失败需可恢复或对账。

### 删除最近回答（不迁移）

旧 `DELETE /api/chat/sessions/{session_id}/last-assistant` 返回 `{"removed":boolean}`。无 assistant 返回 false；当前 handler 未先检查 session 是否存在。Go 首版不公开，因为“先删除、再发问”是两个请求，会有竞态。

### 读取图片

旧 `GET /api/chat/images/{name}` 返回图片二进制；无效名或文件不存在 404。Go 以受保护附件资源读取，校验当前用户与会话权限，不能按文件名直接访问文件系统。

### Ask / Runs

旧 `POST /api/chat/sessions/{session_id}/ask` 请求 AskRequest，成功 200 text/event-stream。Go 目标 `POST /api/v1/chat/sessions/{session_id}/runs`；浏览器使用 fetch + ReadableStream。

| 旧请求字段 | 类型 | 必填/默认 | 说明 |
| --- | --- | --- | --- |
| question | string | 必填，minLength 1 | Agent 前 strip；Go 应拒绝纯空白 |
| kb_ids | integer[] 或 null | 可选 | Agent 内部 KB ID；Go 从授权映射生成 |
| skip_user_persist | boolean | 默认 false | 旧重答字段，不进入 Go 公共 DTO |
| images | string[] 或 null | 可选 | data URL 或旧站内图片 URL |

Go body 建议：

~~~json
{
  "question": "如何申请设备？",
  "knowledge_base_ids": ["kb_platform_id"],
  "images": []
}
~~~

当前 apps/web 的 chat client 发送 question、skip_user_persist、images，但省略 kb_ids；旧 FastAPI 没有按用户授权过滤 KB。迁移后 Go 必须从登录身份和平台知识库映射计算权限。Go 校验会话、附件和每个 KB 的 workspace 权限；授权集合空时直接拒绝。发出响应头前可返回统一 JSON 错误，之后用 SSE error。收到 done 才是 completed；无终态 EOF 记为 interrupted；解析失败为 failed。客户端断开传播取消，但未验证 Python 停止模型调用前不承诺模型计算已停止。无续传或盲目重放；结果未知需对账。

## 5. SSE 事件

旧帧为 event 名及 JSON data；公共字段 step、ts（毫秒 Unix）、detail。运行时 Swagger 未展开 payload schema，以下事件字段据前端类型和 orchestrator 源码整理。

| event | data 主要字段 |
| --- | --- |
| router | intent（chitchat/kb_qa/data_query/summarize）、confidence |
| rewrite | rewritten_query、sub_queries、hyde_used |
| retrieve | source（vector/keyword/database）、hit_count |
| grade | score、verdict（pass/partial/fail）、reason |
| correct | action（rewrite_retry/give_up）、round |
| generate | delta |
| reflect | result（pass/fail）、round、missing_info? |
| citation | citations；单项旧字段为 doc_name、section_path、chunk_no，可选 source_type、sql、rows（数据库证据） |
| done | message_id、elapsed_ms、token_usage（prompt/completion） |
| error | code、message |

~~~text
event: generate
data: {"step":6,"ts":1791448200000,"detail":"","delta":"答案片段"}

event: done
data: {"step":0,"ts":1791448200500,"detail":"","message_id":123,"elapsed_ms":8500,"token_usage":{"prompt":100,"completion":80}}

~~~

Go 公开流加入 run_id、request_id、递增 seq，并把内部 message_id 映射为平台 ID。关闭代理缓冲、及时 flush。解析器支持 UTF-8 跨块、CRLF、多事件同块、多行 data 和半包；生成回流时不能假设 delta 只追加。未知 event 可记录后忽略；解析失败结束运行并标 failed。当前运行时 Swagger 只声明 200/422；仓库 YAML 的 409 与 Python 实现不符。

## 6. 知识库、文档与入库

### 知识库

- 列表：旧 GET /api/admin/kb 返回数组；doc_count/chunk_count 统计 ready 文档。Go 目标 GET /api/v1/workspaces/{workspace_id}/knowledge-bases，授权后分页；Agent KB ID 只用于内部映射。
- 创建：旧 POST /api/admin/kb 成功 201。handler 从 dict 读取 name/description 并 trim；name 不能为空，重名 409。运行时 Swagger 未完整描述 dict；仓库 KbCreate 模型未被此 handler 使用，其长度约束不是当前行为。Go 目标 POST /api/v1/workspaces/{workspace_id}/knowledge-bases，明确长度、唯一性和 Owner/Admin 权限。

### 文档端点

| FastAPI 当前 | 当前行为 | Go 目标 |
| --- | --- | --- |
| GET /api/admin/kb/{kb_id}/documents | 返回文档数组，按创建时间/ID 倒序；含类型、状态、切片数、错误和时间 | GET /api/v1/knowledge-bases/{knowledge_base_id}/documents，授权后查询并分页 |
| POST 同一路径 | multipart 必填 file；成功 202；格式/大小错误 400，未知 KB 404 | /api/v1 同资源路径；授权并创建 Go 入库任务 |
| DELETE /api/admin/kb/{kb_id}/documents/{doc_id} | 成功回 deleted ID；KB/文档不匹配 404 | /api/v1 同资源路径；定义部分失败对账 |
| POST /api/admin/kb/{kb_id}/documents/{doc_id}/retry | 202；仅 failed 可重试；其他状态 400；不存在目前映射为 400 | 建议同文档资源 /retry，幂等 attempt/job |
| GET /api/admin/kb/{kb_id}/documents/{doc_id}/chunks | page 默认 1、page_size 默认 8；只允许 ready；非 ready 400；源码将 page 限至少 1、page_size 限 1..50 | 建议同文档资源 /chunks，校验 workspace/KB/document 权限 |

上传支持 md、txt、pdf、docx，最大 20 MiB（20*1024*1024 字节）。当前用 FastAPI BackgroundTasks，进程重启可能中断；202 只表示 accepted，不代表已索引。仓库目标 GET /api/v1/ingestion-jobs/{job_id} 是新的 Go 能力，不是当前 Python 接口。可靠处理需要另行实现 Python job 幂等、Outbox/Worker 和对账。

切片返回 {items,total,page,page_size}；每项含 id、doc_id、chunk_no、section_path、content、token_count。仓库旧 YAML 写 page_size 最大 100，但运行时代码实际限制为 50。

引用接口 GET /api/admin/chunks/by-citation?doc_name=&chunk_no= 返回 {doc,chunk} 或 404。源码按去扩展名文件名全局查 ready 文档，重名时选最新记录，没有 KB/workspace 过滤。Go 不应复刻。应使用稳定的平台 document/chunk ID 并在获准 KB 内定位；无法唯一匹配时返回引用无法定位。

## 7. 管理、诊断和健康检查

### 数据源配置

GET /api/admin/datasources 返回已配置的 vector/keyword/database 项。PUT /api/admin/datasources 请求为 {"items":[{"source_type":"vector","enabled":true,"config":{}}]}；enabled 默认 true、config 默认空对象，返回 {"updated":n}，未知类型 400。虽然 Swagger 描述为 replace，源码实际只 upsert 请求中出现的项，不删除缺失项。Go 暂不公开；若保留需确认 PUT 语义、管理角色、配置白名单和审计。

### 决策日志

GET /api/admin/decision-logs 可按 session_id、start、end 筛选；page 默认 1，page_size 默认 8。源码限制 page 至少 1、page_size 为 1..50，按 ID 倒序返回分页结构。当前无效 start/end 会被静默忽略；Go 应返回 400。GET /api/admin/decision-logs/{log_id} 返回单条日志，不存在 404。若保留，先授权会话；查看 question/final_answer/steps 需要独立内容权限。

### 检索实验台

POST /api/admin/retrieval-test/lane 请求含 query、source、top_k（默认 5）、可选 kb_ids；source 为 vector、keyword 或 database。返回 {source,latency_ms,hits:[{title,snippet,score,meta}]}。旧 database lane 异常会被记录后转换为空 hits；Go 不应静默降级。

POST /api/admin/retrieval-test/fuse 请求含 query（默认空）和 lanes；lane 含 source、latency_ms（默认 0）、hits；hit 含 title、snippet、score（默认 0）、meta。返回 {fused:[{key,rrf,from_lanes,hit}],grade:{score,verdict,reason}}，并调用真实 grader。两接口暂不开放为公开 Go API；NL2SQL/database lane 需先评估安全和权限，调试路由默认应隔离或禁用。

### 健康检查

旧 GET /api/health 返回 status（ok/degraded）、version、model、mysql、elasticsearch。Go README 记为 GET /healthz。建议区分 liveness/readiness，只暴露部署所需状态，不向公网返回模型或依赖内部细节。

## 8. 迁移实施与验收

1. 实现前重新获取 /openapi.json，与 FastAPI 路由、models、Web client 对照并保存版本基线；不能只依赖仓库旧 YAML。
2. 补全 apps/api/openapi.yaml/json 中 Go 方法、字段约束、授权、安全、状态码、错误 envelope、分页、上传和 SSE 事件 schema。
3. 将旧 Agent URL、DTO 和 SSE 解析封装在 internal/agentclient；Go 不跨库访问 Agent 表。
4. Web 切换 Go 后确认不能绕过 Go 直连 Python，并核对 Vite API_PROXY_TARGET。
5. 验证跨 workspace/他人会话、文档、引用访问拒绝；空 KB 权限拒绝；SSE 半包、Unicode、error、无 done EOF；同会话并发；文件限制；未知结果对账；删除和索引一致性。
6. 可靠入库单独阶段验收。没有 Agent job_id 幂等提交/查询能力时，不声称 exactly-once 或可靠恢复。

需要和 Web DTO、发布说明同步的破坏性变化：旧路径改 /api/v1、响应 envelope、Agent 整数 ID 改平台字符串 ID、错误结构、分页限制、认证授权、移除 last-assistant 清理接口、改变 citation 定位方式。数据源、决策日志、检索实验台是否迁移及其权限模型尚未由基线确定。

## 9. 一手来源

- 当前 Swagger：http://localhost:8000/docs 和 http://localhost:8000/openapi.json
- Go 状态/契约：apps/api/README.md、apps/api/openapi.yaml、apps/api/openapi.json
- 架构/规范：docs/开发规范与技术选型.md、apps/api/docs/backend-engineering-standards.md
- FastAPI：backend/app/routers/chat.py、kb.py、datasources.py、decision_logs.py、retrieval_test.py、main.py、backend/app/models.py
- 入库/SSE/图片：backend/app/services/ingestion.py、backend/app/services/chat_images.py、backend/app/agent/orchestrator.py
- Web 调用/类型：apps/web/src/lib/api/chat.ts、apps/web/src/lib/api/admin.ts、apps/web/src/types/legacy.ts