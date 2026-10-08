# 智能问答 Implementation Plan（评审草案）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在新 React Web 中交付一个真实可用的知识库问答闭环：用户在授权范围内管理会话、提交问题、接收流式回答、查看安全引用与受控处理阶段，并从持久化历史恢复结果。

**Architecture:** 浏览器只访问同源 Go `/api/v1`；Go 校验身份、会话所有权和知识库授权，再通过内部 AgentClient 调用现有 Python Agent。Web 的 chat feature 按组件拆分草案组织页面、feature 状态、API/SSE 适配和展示组件；服务端历史是最终消息事实源，Zustand 仅保存流式临时态。

**Tech Stack:** 沿用 React 19、TypeScript strict、Vite、React Router、TanStack Query、Zustand、Tailwind、Vitest、Testing Library、Playwright；后端沿用 Go/Gin，复用现有 Python Agent。此计划不引入新框架或依赖。

**Spec:** [`docs/产品需求文档-PRD.md`](../../产品需求文档-PRD.md)（FR-07～FR-10、§4.4、§4.5、§4.8）；[`docs/开发规范与技术选型.md`](../../开发规范与技术选型.md)（§3、§5～§7、§9、§11、§13）；[`apps/web/docs/component-splitting-guidelines.md`](../../../apps/web/docs/component-splitting-guidelines.md)。

## Global Constraints

- MUST 遵循 `app -> features -> shared`，feature 跨模块只经公开 `index.ts`。
- MUST 让 Go 成为唯一公开业务入口；Web 不得直接调用 Python Agent。
- MUST 由 Go 校验会话归属及知识库权限；空授权集合必须拒绝，不能退化为全库检索。
- MUST 由 TanStack Query 持有服务端会话/消息事实，Zustand 仅持有当前流式临时交互态。
- MUST 只有 SSE `done` 代表正常完成；无终态 EOF 是 `interrupted`；断网后不得自动重提问题。
- MUST 以完成后的持久化消息刷新为准，不把拼接 `generate.delta` 当作最终答案。
- MUST 渲染 Markdown 时禁用原始 HTML，引用经 Go 当前权限校验后才可打开。
- MUST 不向用户展示未经处理的内部推理文本、SQL、供应商错误或未授权内容。
- MUST 为请求贯穿取消；停止接收不等于 Agent 已停止推理或费用已停止。
- MUST 完成前运行前端 `npm run typecheck`、`npm test`、`npm run build`；旅程变更另运行 `npm run test:e2e`。
- MUST 在开始后端代码前读取实际存在的 `apps/api/AGENTS.md` 与 Go 后端工程规范；当前仓库扫描未发现 `apps/api`，因此 Go 任务只能在该前置条件满足后执行。

## Review Focus

- 空知识库授权、伪造知识库 ID 或撤权后访问：服务端拒绝且不触发 Agent 检索（后端问答服务/权限测试）。
- 同一会话已有活动运行时重复提交：返回 409，前端保留当前流并提示等待（后端并发测试、Web 行为测试）。
- SSE 数据跨 UTF-8/CRLF/半包边界且 EOF 无 `done`：保留正确正文、标记中断、不显示成功、不自动再次提问（SSE parser 与状态机测试）。
- Agent 最终持久化正文或引用与流式 delta 不同：完成后以历史响应为准更新 UI（chat flow 集成测试）。
- 引用文档同名、失效或用户刚被撤权：只显示安全的无法定位/无权提示，不泄露其他文档信息（Go 引用解析测试、前端引用行为测试）。

---

## 1. 目标、边界与前置条件

### 本计划范围

- P0 闭环：本人会话创建/列表/历史；用户选择授权知识库；流式提问；安全 Markdown 回答；引用核验；可折叠的安全处理阶段；刷新后恢复持久化历史；连接失败、服务错误、断流、并发冲突和用户停止接收的明确状态。
- 第一版只支持文本问题和现有知识库。知识库创建/上传/删除、成员管理、运行记录页、问答反馈、重新生成、图片输入、NL2SQL 与多模态均不在本计划内。
- 保留现有 Agent 算法和 prompt；除接入所需且单独审查批准的适配外不重构 Python RAG。

### 代码现状与阻塞

- `apps/web/src/features/chat/routes/ChatPage.tsx` 是静态样例；多个既有 chat UI 文件仍为 `LegacyPlaceholder`。
- `apps/web/src/store/chat.ts` 与 `apps/web/src/lib/api/chat.ts` 有旧会话/SSE 实现，但直接使用旧 `/api/chat/...` 协议，不能作为新平台公开 API 的实现依据。
- `backend/app/routers/chat.py` 暴露无平台身份/工作空间授权边界的旧 Agent 路由；不能由浏览器直接访问。
- 本次目录扫描未发现 `apps/api`，也未发现该目录下的后端工程规范。仓库 PRD 描述的 Go 健康检查基线与实际目录状态需要先核实。
- 因此，实现代码的前置门槛是确认 Go API 所在位置/创建边界、读取该模块规范，并确认认证、工作空间及授权知识库能力由哪个已完成阶段提供。不得在此之前把旧 Python 路由改造成浏览器正式入口。

### 本计划不默认已有的能力

认证登录、Owner/Member 和工作空间、知识库授权映射、平台会话映射、Go-AgentClient、问答运行持久状态及安全引用解析都必须在真实代码中核实。若缺失，应作为显式前置工作或在你批准扩大范围后纳入本计划，而不是用 mock/演示数据宣称闭环完成。

## 2. 组件边界建议

```text
app/router
  └── features/chat/routes/ChatPage
        ├── chat API + SSE adapter       # feature 私有协议边界
        ├── chat model/store             # 流状态转换与用户动作
        └── chat UI                      # SessionList / ChatInput / MessageList /
                                          # AnswerMessage / CitationList / RunStatus
shared: 既有 Button、Input、Modal、HTTP 基础能力及通用 SSE framing（确认可复用后）

Browser ── POST /api/v1/chat/sessions/{id}/runs ──> Go API ── internal AgentClient ──> Python Agent
         ├── GET /api/v1/chat/sessions
         ├── GET /api/v1/chat/sessions/{id}/messages
         └── GET /api/v1/chat/runs/{runId}/citations/{citationId}
```

页面负责数据边界和布局组合；展示组件只接收已验证的视图数据及意图回调；chat store 只持有当前运行的草稿、事件摘要和运行状态。样式继续与 chat UI 就近维护并使用既有 Tailwind token，不新建主题层或全局聊天 CSS。

## 3. 拟定文件边界

### Web

| 文件 | 责任 |
| --- | --- |
| `apps/web/src/features/chat/routes/ChatPage.tsx` | 路由级组合与 TanStack Query 数据边界；不拼 SSE 协议 |
| `apps/web/src/features/chat/api/chatApi.ts` | 新 `/api/v1` 会话/历史/引用请求及错误映射 |
| `apps/web/src/features/chat/api/chatStream.ts` | POST SSE 消费、AbortSignal、事件运行时校验与终态判定 |
| `apps/web/src/features/chat/api/chatContracts.ts` | 该 feature 的稳定 DTO/视图边界类型；最终由 OpenAPI 生成策略决定 |
| `apps/web/src/features/chat/model/chatStore.ts` | chat 运行临时态和动作；服务端消息不复制进 Zustand |
| `apps/web/src/features/chat/model/chatState.ts` | 可测试的事件→状态纯转换，如有明确复杂度再拆出 |
| `apps/web/src/features/chat/ui/SessionList.tsx` | 会话列表、选择和新建回调 |
| `apps/web/src/features/chat/ui/ChatInput.tsx` | 输入校验、提交/停止按钮及键盘行为 |
| `apps/web/src/features/chat/ui/MessageList.tsx`、`AnswerMessage.tsx` | 消息列表和回答展示 |
| `apps/web/src/features/chat/ui/CitationList.tsx`、`CitationChunkDrawer.tsx` | 安全引用入口和授权片段展示 |
| `apps/web/src/features/chat/ui/RunStatusPanel.tsx` | 可折叠的受控阶段摘要和运行状态 |
| `apps/web/src/features/chat/index.ts` | 仅公开路由/真实跨 feature 所需能力 |
| `apps/web/src/features/chat/**/*.test.ts(x)` | API 边界、状态转换、组件行为测试 |
| `apps/web/e2e/chat.spec.ts` | 登录后问答、引用、刷新历史和关键失败状态旅程 |

`apps/web/src/lib/api/chat.ts`、`apps/web/src/lib/api/sseParser.ts`、`apps/web/src/store/chat.ts` 属于旧实现。执行时先查明所有调用者：只有 chat feature 使用的部分迁入 feature；确有多个真实 feature 共用的无业务 SSE framing 才留在 shared/lib。迁移完成后删掉失去调用的旧入口，禁止双路 fallback。

### Go API（仅在该服务目录与工程规范确认后落文件）

目标模块按仓库既定 Go 分层实现：chat handler/application/repository、AgentClient 适配、公开 OpenAPI 与必要 migration。具体路径和类型签名须先以实际 `apps/api` 布局及后端规范校准；不在不存在的目录上臆造可直接执行的代码清单。

## 4. Implementation Tasks

### Task 1: 冻结问答契约与平台前置依赖

**Files:**
- Modify: `apps/api/api/openapi.yaml`（目录/文件确认存在后）
- Create/Modify: `apps/api/docs/adr/` 中的问答接入决策记录（按实际 ADR 约定）
- Modify: `docs/项目开发路线图.md`（仅当阶段依赖或完成状态需纠正时）
- Reference: `backend/app/routers/chat.py`、`backend/app/agent/orchestrator.py`、现有去敏 SSE 样本

- [ ] 确认 Go API 仓库位置、对应 `AGENTS.md`/后端规范、登录/工作空间/知识库授权前置能力及 Python Agent 连通方式；把缺失项标记为独立依赖，不把 TeamFlow 或旧 Python 路由直接当作新平台服务。
- [ ] 冻结 FR-07～10 的公开请求/响应、字符串平台 ID、错误码、权限约束与 SSE 事件 JSON Schema；浏览器只可传 `question` 与平台知识库 ID，内部 Agent ID 和 `skip_user_persist` 由 Go 决定。
- [ ] 在契约中定义最小行为：授权知识库为空时拒绝；同会话单活动运行；HTTP 建流前错误用标准 JSON；建流后错误用 SSE `error`；只有 `done` 完成；EOF 无终态为 interrupted；流完成后取持久化历史。
- [ ] 在 ADR 记录 Agent 取消能力未经证实时的承诺边界、无 `run_id` 恢复时的失败路径、引用从旧 `(doc_name, section_path, chunk_no)` 到授权资源映射的处理；明确不改 Agent 算法/不支持重答。
- [ ] 评审检查：契约测试样本覆盖正常、409、权限拒绝、error、done 和中断；OpenAPI/SSE 样例与 PRD 一致。

### Task 2: 建立/补齐 Go 侧安全问答垂直切片

**Depends on:** Task 1；Go API 服务位置与后端标准已确认；身份、工作空间、知识库授权能力可用或已另行纳入范围。

**Files:**
- Create/Modify: Go chat application/handler/repository，按已确认的 `apps/api` 目录结构
- Create/Modify: `apps/api/internal/agentclient/` 的 chat 调用适配（若当前没有）
- Modify: `apps/api/api/openapi.yaml`
- Create/Modify: chat migration，仅当平台会话映射/问答运行确无现成持久模型时
- Test: chat service、权限、并发、AgentClient contract/integration tests

- [ ] 为会话创建/列表/历史提供作用域化接口；每次查询均校验当前用户和工作空间，会话映射由 Go 持有，历史消息通过 AgentClient 读取。
- [ ] 实现 `POST /api/v1/chat/sessions/{session_id}/runs`：验证输入长度（暂按 4,000 字符上限提案）、会话归属和已选知识库权限；拒绝空授权集合；用数据库约束/事务保证同会话一个活动 run；不得对未知提问结果盲目重试。
- [ ] 通过 AgentClient 向 Python 发送已映射授权 KB ID，过滤客户端不可控内部字段；保持代理流可取消并把 Agent 事件转为公开契约，增加 run/request 标识和递增 seq（如契约冻结要求）。
- [ ] 完成引用映射端点：每次打开时重新检查用户、run、知识库和文档权限；重名/无法唯一映射返回安全的不可定位结果；不向客户端暴露全局按文档名查询接口。
- [ ] 对 Go 做权限矩阵（含空集合、伪造 ID、他人 session、撤权）、并发 409、Agent 启动前超时、流中错误/断流和引用歧义测试；执行 `gofmt`、`go vet ./...`、`go test ./...` 与契约校验。

### Task 3: 为 Web 建立 feature 私有 API 与可验证 SSE 客户端

**Depends on:** Task 1 的公开契约；Task 2 至少提供可替身的接口实现。

**Files:**
- Create: `apps/web/src/features/chat/api/chatApi.ts`
- Create: `apps/web/src/features/chat/api/chatStream.ts`
- Create: `apps/web/src/features/chat/api/chatContracts.ts`（或采用确认后的 OpenAPI 类型生成输出）
- Test: 同目录 API 与 SSE parser tests
- Migrate/Delete: 旧 `src/lib/api/chat.ts`、`src/lib/api/sseParser.ts` 的 chat 专用部分（确认调用者后）

- [ ] 实现会话列表、创建、历史与引用请求，使用 `AbortSignal`、标准 `{data, request_id}`/错误响应并将服务端错误映射为稳定类型；不从展示组件调用 `fetch`。
- [ ] 实现 POST + `ReadableStream` 事件 framing 和运行时 schema 校验，覆盖 UTF-8 跨块、CRLF 边界、多行 data、多个事件同块、半包、未知事件、畸形数据、`error`、`done`、无终态 EOF。
- [ ] 保证解析器错误不会被静默吞掉；未知但合法的扩展事件按契约忽略/记录，非法终态转为协议失败。
- [ ] 将已解析事件以 `onEvent(event: ChatRunEvent)` 输出，并以显式终态结果区分 `completed | failed | interrupted | cancelled`；只有收到 `done` 返回 `completed`。
- [ ] 用确定性字节块 fixture 验证分块组合、错误与取消；运行相关 Vitest。

### Task 4: 建立 chat feature 的状态转换和查询所有权

**Depends on:** Task 3。

**Files:**
- Create: `apps/web/src/features/chat/model/chatStore.ts`
- Create: `apps/web/src/features/chat/model/chatState.ts`（只在纯状态转换确有单独边界时保留）
- Create: `apps/web/src/features/chat/model/*.test.ts`
- Modify/Delete: `apps/web/src/store/chat.ts`（迁移现有调用者后）

- [ ] TanStack Query 持有会话、历史、当前用户可选知识库和授权引用片段；query key 纳入 workspace/user scope、session ID 和查询条件；缓存失效集中在 feature 数据层。
- [ ] Zustand 只保存当前活动 run 的临时草稿、状态、已解析安全阶段摘要和必要 seq；不复制已持久化历史、授权知识库列表或最终答案。
- [ ] 将状态机固定为 `idle -> connecting -> streaming -> completed | failed | interrupted | cancelled`；`completed` 仅由 `done` 进入，`cancelled` 只表示停止客户端接收；过期流/旧 session 回调不得覆写当前 session。
- [ ] 发送动作先校验非空、长度、会话与至少一个授权知识库；同一运行中不可重复提交；409 不清空当前草稿/流状态。
- [ ] 每个终态取消/释放 reader 与 controller；切换会话或组件卸载时清理旧流；不得将超时或 abort 误判为 Agent 已停止。
- [ ] 用单元测试覆盖各状态转换、重复/倒退 seq、旧会话迟到事件、取消、EOF 无 done 和 completed 后刷新历史。

### Task 5: 按拆分草案接通问答界面

**Depends on:** Task 4。

**Files:**
- Modify: `apps/web/src/features/chat/routes/ChatPage.tsx`
- Implement: chat `ui/SessionList.tsx`、`ChatInput.tsx`、`MessageList.tsx`、`AnswerMessage.tsx`、`CitationList.tsx`、`CitationChunkDrawer.tsx`、`RunStatusPanel.tsx`
- Modify: `apps/web/src/features/chat/index.ts`
- Test: 各 UI 的 `.test.tsx` 与 `ChatPage.test.tsx`

- [ ] 将静态会话/问答/决策演示替换为真实 Query 数据和 feature 动作；页面负责组合，子组件只依赖有意图的 props/回调。
- [ ] 完成创建/选择会话、键盘提交、输入校验、连接/流式/停止接收、空/加载/错误/409/中断/完成状态；不要显示不支持的删除/重答/图片按钮。
- [ ] 消息 Markdown 禁用原始 HTML并校验链接 scheme；流式文本增量展示，完成后重新加载持久化历史作为最终正文与引用。
- [ ] 引用编号只有经过本次 run 返回且由 Go 校验的对象才可交互；点击后读取授权片段。过期、撤权或歧义显示安全状态，不以名称匹配猜测资源。
- [ ] 处理摘要仅展示白名单事件（如检索、证据检查、生成、自检）的安全摘要/耗时/轮次；不展示原始 prompt、SQL、供应商错误或未经清洗 payload。
- [ ] 组件专属 Tailwind 样式与组件就近维护；不因拆分重排全局 CSS。组件测试以 role、label、可见文本及用户动作断言 loading、empty、error、disabled、streaming、引用抽屉和键盘交互。

### Task 6: 验证完整用户旅程和交付门禁

**Depends on:** Tasks 1–5；真实或契约兼容的测试依赖可用。

**Files:**
- Create: `apps/web/e2e/chat.spec.ts`
- Modify: `apps/web/playwright.config.ts` 仅在当前配置无法连接可控 Go/Agent test fixture 时
- Update: API/Agent 去敏合同样本及局部运行说明（按实际文档位置）

- [ ] E2E 覆盖登录上下文中选择授权知识库→新建会话→提交问题→接收正文与安全处理摘要→打开引用→刷新页面恢复最终消息。
- [ ] E2E/集成负向覆盖空授权知识库、他人会话/引用拒绝、409、SSE `error`、无 `done` 断流、停止接收后查询状态；断网恢复不自动重提问题。
- [ ] 对照 PRD §4.8 留存测试结果与受阻项；分别验证模拟 Agent 转发和真实 Agent 端到端，不能用模拟结果冒充真实 Agent 验收。
- [ ] 运行 `npm run typecheck`、`npm test`、`npm run build`、`npm run test:e2e`；Go 改动运行 `gofmt`、`go vet ./...`、`go test ./...`、OpenAPI/契约检查。
- [ ] 验收 FR-07～10：授权正确、会话历史持久化、done/错误/断流区分、引用安全、处理信息受控；记录尚未验证的上游 Agent 取消、撤权流中生效时延和真实依赖限制。

## 5. 重要决策检查表（请先审阅）

下列标为“建议”的内容是本计划的默认假设，不表示已经冻结。请逐项确认、改写或指出暂缓项；如选择会改变模块边界/公开契约/数据归属，先按仓库要求落 ADR，再进入实现。

| ID | 需你确认的决策 | 本计划建议 | 备选/影响 |
| --- | --- | --- | --- |
| D1 | 交付边界：做可演示的前端接线，还是按正式产品边界完成 Web→Go→Agent 垂直闭环？ | 正式垂直闭环；不让浏览器直连旧 Python API。 | 前端演示范围较小，但不是安全可上线问答；Go 当前目录缺失，需要先核实/补齐服务。 |
| D2 | Go 鉴权、工作空间、知识库授权映射尚不存在或位置不明时，是否纳入本次计划？ | 作为明确前置依赖；如它们未完成，就扩大计划补齐最小依赖后再做聊天。 | 只做 UI mock/直连 Agent 会跳过项目定下的唯一公开入口和权限边界。 |
| D3 | 首版问答是否严格限定为 FR-07～10？ | 是：会话、授权知识库选择、文本流式问答、引用、受控处理阶段、历史恢复。 | 上传/删除、反馈、运行记录、重答、图片另列计划，避免问答切片膨胀。 |
| D4 | 知识库选择交互：单选还是多选？未选时如何处理？ | 支持多个显式授权知识库；零选择时禁用发送并提示选择，服务端同样拒绝空集。 | 单选实现与检索隔离更简单；多选符合跨资料查询，但影响 DTO、权限与引用映射。 |
| D5 | 用户“停止”语义 | UI 文案用“停止接收”，保留中断草稿并提供查询结果；在 Agent 取消能力验证前不承诺停止推理。 | 若需硬取消，必须另做 Go→Agent 取消契约与资源/费用验证。 |
| D6 | 处理过程展示范围 | 只展示白名单阶段摘要与耗时，不显示完整推理、SQL、prompt 或原始 Agent payload。 | 更详细的追踪需安全审核和面向用户的专门协议，不能直接暴露现有内部日志。 |
| D7 | 引用 ID/稳定性 | Go 生成 run 作用域 citation ID，每次读取重新鉴权；底层旧文档名歧义返回不可定位。 | 需先确认 Go 是否已有稳定 document/chunk 映射；否则引用体验受旧 Agent 契约限制。 |
| D8 | Web API 类型来源 | 优先从确认后的 OpenAPI 生成；若仓库暂无稳定代码生成流程，先评估现有工具，不在本计划擅自加依赖。 | 手写 DTO 可先行但易漂移；引入生成工具影响脚本、锁文件和 CI。 |
| D9 | 问题长度上限 | 先按 PRD 建议 4,000 字符，冻结前核对 Go、Agent、模型限制。 | 若模型/旧 Agent 上限更低，要统一向用户给出可理解提示。 |
| D10 | 用户/工作空间作用域和会话 URL | 只在登录及 workspace 上下文已有稳定契约后实现；默认 URL 可恢复当前会话但不泄露正文。 | 路由格式（如 `/chat/:sessionId` 或 query param）影响刷新恢复、导航与分享；需看现有路由规范后定。 |
| D11 | 旧 `src/store/chat.ts`、`src/lib/api/chat.ts` 的迁移方式 | 找齐引用后迁入 `features/chat` 并移除废弃直连路径；不做全站 store/API 大迁移。 | 若仍被历史页面调用，先列出兼容调用者，避免贸然删除或保留双协议。 |

### 已由项目规范确定、无需再做产品选择的事项

- Go 是唯一公开业务入口；Python Agent 仅内部复用。
- 权限必须由后端执行；浏览器发送平台知识库 ID，不能指定 Agent 内部 ID。
- 空授权集拒绝；同会话一次只允许一个活动运行；409 必须给出可理解提示。
- 只有 `done` 是正常完成；无终态 EOF 是中断；失败/超时后不自动重发。
- 生成结束后以持久化历史为准；引用每次重新鉴权；原始 HTML 和内部推理内容不得直接显示。
- 重新生成首版后置；不得让组件拆分引入无复用价值的 shared 抽象或分散样式系统。

## 6. 计划自检与未决项

- **范围追踪：** FR-07 会话/历史在 Task 2、4、5；FR-08 流式问答在 Task 1–6；FR-09 引用在 Task 1–3、5–6；FR-10 阶段摘要在 Task 1、4–6。
- **失败模式：** 权限、并发、协议分块/错误/终态、最终历史对齐、引用歧义均已分别关联到实现及测试任务。
- **依赖/边界：** 计划坚持 Web→Go→Python；Go 目录与后端标准缺失被标为真实前置，不用旧路由或 mock 掩盖。
- **范围比例：** 不含知识库上传、可靠任务队列、统计/运行记录、反馈、重答和多模态；若 D1/D2 扩大到平台底座全量建设，应拆出身份/工作空间/授权计划，再接续本计划。
- **需要你评审后才可进入实现：** D1–D11；尤其 D1/D2 决定当前缺失的 Go 与授权依赖属于本次范围还是已完成前置。
