# 旧版 Agentic RAG 前端迁移与手写 CRUD 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成旧 Vue Agentic RAG 前端在 `apps/web` React 应用中的页面与交互迁移，手写知识库/文档、数据源配置、会话的 CRUD 前端逻辑；除 Go health 外，业务 API 保留明确的待接入占位。

**Architecture:** 在现有 React 19/Vite 应用中补齐页面，不新建前端应用；旧归档只作为行为参考。保留当前 React 视觉稿，复刻旧 Vue 的交互；按 feature 放置 `api.ts` 和数据逻辑，TanStack Query 独占服务端状态，Zustand 仅持有活动聊天流的临时状态。当前只调用 Go `GET /healthz`；业务 API 文件只保留手写签名与明确的未接入错误，不访问 Go 业务路由或 Python。

**Tech Stack:** React 19、TypeScript strict、Vite、React Router、TanStack Query、Zustand、Tailwind CSS、Vitest、React Testing Library、Playwright；不新增依赖。

**Spec:** `docs/vue-to-react-migration-guide.md`；项目约束补充见 `docs/开发规范与技术选型.md` 与 `apps/web/docs/frontend-engineering-standards.md`。

## Global Constraints

- React 固定 19 大版本；TypeScript strict；Vite 构建。
- 模块依赖遵循 `app -> features -> shared`；feature 间只通过公开 `index.ts` 协作。
- `archive/frontend-legacy-20260921` 只读，作为行为参考；不得修改或复制其代码作为新实现模板。
- 只接入 Go `GET /healthz`；浏览器不得访问旧 FastAPI，也不得假装 Go 业务路由已存在。
- 本次不扩展资源操作契约；业务 `api.ts` 占位函数必须显式抛出“API 未接入”错误，不返回假数据或固定成功。
- 保留现有 React 视觉稿；交互流程和状态按旧 Vue 行为迁移。
- TanStack Query 持有服务端列表/详情；Zustand 只持有聊天流式草稿和连接状态；筛选/分页放入 URL。
- Access token 不写入 localStorage；HTTP/SSE 和文件输入都按不可信边界校验，支持请求取消。
- React CRUD 表单、请求、Query mutation 和错误状态全部手写；不使用 CRUD 代码生成器或通用全局业务 API/store。
- 只实现后端契约实际支持的操作；未知操作结果不得盲目重放，错误必须保留并可恢复。

## Review Focus

- API 尚未填充时，创建、上传、删除、重试等操作必须明确提示未接入，不能报告成功；由 feature API 占位与页面测试验证。
- 上传边界文件类型/大小无效、批量上传部分失败时，不能丢弃成功/失败明细；由 KnowledgePage 上传测试验证逐文件结果。
- 删除当前会话或切换会话时，迟到的消息/SSE 回调不能覆盖新会话；由 chat model 测试验证 session 标识与 AbortSignal 清理。
- SSE 解析遇到跨块 UTF-8、半包、协议错误、error 或无 `done` EOF 时，不能误判为成功；由 parser/state-machine 测试验证终态。
- 未来填入 API 后，数据源保存失败或服务端返回不完整配置时不能静默覆盖用户草稿；当前由 datasource UI 测试验证草稿保留及未接入提示。

---

## 已确认的架构决策

- **API 接入：** 当前只接入 Go `GET /healthz`。不接 Go 业务 API，不直连 Python。业务函数放在各 feature 自己的 `api.ts`，保留明确签名并抛出 `ApiNotConfiguredError`，后续填入实现。
- **资源契约：** 不增加知识库编辑/删除、会话重命名或数据源单项创建/删除。操作范围固定为知识库列表/创建、文档列表/上传/删除/重试/切片、数据源读取/整体保存、会话列表/创建/删除/消息/提问/重新生成、日志只读、检索显式执行。
- **界面边界：** 保留当前 React 视觉稿，迁移旧 Vue 已有的交互、状态转换和错误路径。

## File Map

- `apps/web/src/app/providers.tsx`（新增）：组装 QueryClient、错误边界和应用 Provider。
- `apps/web/src/app/App.tsx`、`apps/web/src/main.tsx`（修改）：接入 Provider，不在启动层放业务逻辑。
- `apps/web/vite.config.ts`（修改）：把 `/healthz` 代理到 Go；不配置 Python target。
- `apps/web/src/lib/http.ts`、`apps/web/src/features/overview/api.ts`（修改/新增）：只实现 Go health 请求的取消、解码和错误归一化；不放业务 endpoint。
- `apps/web/src/features/overview/Overview.tsx`（修改）：health 请求从 `/api/health` 改为 `/healthz`，清除 Python Agent 连接文案。
- `apps/web/src/features/knowledge/api.ts` 与 `model/queries.ts`、`model/mutations.ts`（新增）：知识库、文档、切片 API 占位与 Query 缓存规则。
- `apps/web/src/features/knowledge/types.ts`（新增）：知识库、文档和切片类型由知识库 feature 所有。
- `apps/web/src/features/knowledge/routes/KnowledgePage.tsx`（修改）：知识库/文档列表及创建、上传、删除、重试、切片分页交互；API 未填充前显式显示未接入状态。
- `apps/web/src/features/admin/api.ts`、`model/queries.ts`、`model/mutations.ts`（新增）：数据源读取和整体保存 API 占位；同时承载检索测试 API 占位。
- `apps/web/src/features/admin/types.ts`（新增）：数据源与检索测试类型由 admin feature 所有。
- `apps/web/src/features/admin/routes/DatasourcePage.tsx`、`ui/SourceCard.tsx`（修改）：草稿编辑、保存和错误状态；API 未填充前不显示假数据或保存成功。
- `apps/web/src/features/chat/api.ts`、`model/chatStream.ts`（新增/迁移）：会话 CRUD、历史、问答流 API 占位和流协议适配。
- `apps/web/src/features/chat/types.ts`（新增）：会话、消息、引用和 SSE 类型由 chat feature 所有。
- `apps/web/src/features/chat/model/queries.ts`、`model/chatStreamStore.ts`、`model/chatState.ts`（新增/迁移）：Query 服务端数据及独立可测的流状态机。
- `apps/web/src/features/chat/routes/ChatPage.tsx`、现有 `ui/*`（修改）：会话删除/选择/新建、消息、流式答案和引用视图。
- `apps/web/src/features/logs/api.ts`、`model/queries.ts`（新增）；`features/logs/routes/DecisionLogPage.tsx`（修改）：过滤、分页、详情和步骤播放 API 占位。
- `apps/web/src/features/logs/types.ts`（新增）：日志、步骤与查询类型由 logs feature 所有。
- `apps/web/src/features/admin/routes/PlaygroundPage.tsx`（修改）：显式执行检索测试并显示分路、融合和评分；请求使用该 feature 的 `api.ts` 占位。
- `apps/web/src/lib/api/admin.ts`、`src/lib/api/chat.ts`、`src/lib/api/http.ts`、`src/lib/api/sseParser.ts`、`src/store/*.ts`、`src/types/legacy.ts`（迁移完成后删除或收缩）：移除旧业务 API 汇总点、远端数据 Zustand 副本和全局遗留类型。
- `apps/web/src/features/**/**/*.test.ts(x)`、`apps/web/e2e/*.spec.ts`（新增/修改）：请求边界、CRUD 行为、流状态和关键旅程测试。
- 业务 API 占位不得修改 Go OpenAPI 或后端；只有未来明确批准接入业务 API 时再单独设计与扩展契约。

## Task 1: 固定前端边界与待接入 API 清单

**Files:**
- Read: `docs/vue-to-react-migration-guide.md`
- Read: `docs/开发规范与技术选型.md`
- Read: `apps/api/README.md`、`apps/api/internal/platform/router.go`
- Read: `archive/frontend-legacy-20260921/src/`（只核对交互行为，不拷贝代码）
- Create: `apps/web/src/lib/apiError.ts`
- Test: `apps/web/src/lib/apiError.test.ts`

**Interfaces:**
- Produces: 共享错误类型 `ApiNotConfiguredError`；每个业务 feature 的操作范围已由本计划固定，只有 health 调用可执行。
- Consumes: 旧 Agent 页面的操作行为只作参考；不得把旧 Python URL 放入 React 请求实现。

- [ ] **Step 1: 写 `ApiNotConfiguredError` 单元测试**，断言稳定 `name`、`code: 'API_NOT_CONFIGURED'` 和可直接展示的提示。
- [ ] **Step 2: 运行测试确认失败**：`npm test -- src/lib/apiError.test.ts`。Expected: 因错误类型未实现而 FAIL。
- [ ] **Step 3: 实现 `ApiNotConfiguredError`**；各 feature 后续占位函数复用该类型。
- [ ] **Step 4: 运行测试确认通过**：`npm test -- src/lib/apiError.test.ts`。Expected: PASS。
- [ ] **Step 5: Commit**：`git add apps/web/src/lib/apiError.ts apps/web/src/lib/apiError.test.ts && git commit -m "feat(web): define not-configured api error"`。

## Task 2: 建立请求边界与 Query 服务端状态

**Files:**
- Create: `apps/web/src/app/providers.tsx`
- Modify: `apps/web/src/app/App.tsx`、`apps/web/src/main.tsx`、`apps/web/src/lib/http.ts`、`apps/web/vite.config.ts`
- Create/Modify: `apps/web/src/features/overview/api.ts`、`src/features/knowledge/model/queryKeys.ts`
- Test: `apps/web/src/lib/http.test.ts`、`apps/web/src/app/providers.test.tsx`、`apps/web/src/features/overview/api.test.ts`

**Interfaces:**
- Produces: `AppProviders({ children }: Props): JSX.Element`；health 请求支持 `AbortSignal`，返回已验证 health DTO 或可识别的 `ApiError`。
- Produces: `getHealth(signal?: AbortSignal): Promise<Health>` 使用真实 `GET /healthz`。
- Produces: `knowledgeKeys` 集中工厂，键包含所有者 scope、知识库 ID、文档 ID 和完整查询条件；后续 feature 使用同样规则各自维护 key 工厂。

- [ ] **Step 1: 写 health 请求测试**，覆盖 `GET /healthz` 成功、非 2xx、无效 JSON DTO、请求取消和错误 ID 展示；不向 `http.ts` 加入业务 API 路径。
- [ ] **Step 2: 运行这些测试确认缺少目标能力**：`npm test -- src/features/overview/api.test.ts`。Expected: health 请求/校验断言 FAIL，既有通用 HTTP 测试保持原结果。
- [ ] **Step 3: 实现 health API 与最小通用 HTTP 边界**；仅负责 health 请求需要的 base URL、取消、JSON 解码和错误归一化，不暴露 Agent 业务请求。
- [ ] **Step 4: 更新 Vite 代理和 health 页面**，让 `/healthz` 到达 Go `GET /healthz`，显示在线/错误/重试状态，并删除“连接 Python Agent”的旧文案。
- [ ] **Step 5: 写 Provider 测试**，确认应用子树可访问一个 QueryClient，测试间 QueryClient 隔离。
- [ ] **Step 6: 接入 QueryClient Provider**，为 health 请求提供缓存与错误状态，不在 `App.tsx` 写业务规则。
- [ ] **Step 7: 用 QueryClient 测试确认 key 稳定；业务 feature key 保持按资源和过滤条件生成，不创建服务端 store 副本。**
- [ ] **Step 8: Commit**：`git add apps/web/src/app apps/web/src/lib/http.ts apps/web/src/features/overview apps/web/vite.config.ts && git commit -m "feat(web): connect go health check"`。

## Task 3: 手写知识库与文档操作

**Files:**
- Create: `apps/web/src/features/knowledge/api.ts`
- Create: `apps/web/src/features/knowledge/types.ts`
- Create: `apps/web/src/features/knowledge/model/queries.ts`、`mutations.ts`
- Modify: `apps/web/src/features/knowledge/routes/KnowledgePage.tsx`
- Modify: `apps/web/src/features/knowledge/ui/UploadModal.tsx`、`DocumentRow.tsx`、`ChunkDrawer.tsx`、`StatusBadge.tsx`
- Test: `apps/web/src/features/knowledge/api.test.ts`
- Test: `apps/web/src/features/knowledge/routes/KnowledgePage.test.tsx`

**Interfaces:**
- Produces: `listKnowledgeBases(signal?: AbortSignal): Promise<KnowledgeBase[]>`
- Produces: `createKnowledgeBase(input: { name: string; description: string }, signal?: AbortSignal): Promise<KnowledgeBase>`
- Produces: `listDocuments(kbId: number, signal?: AbortSignal): Promise<KbDocument[]>`
- Produces: `uploadDocument(kbId: number, file: File, signal?: AbortSignal): Promise<KbDocument>`
- Produces: `deleteDocument(kbId: number, docId: number, signal?: AbortSignal): Promise<void>`
- Produces: `retryDocument(kbId: number, docId: number, signal?: AbortSignal): Promise<KbDocument>`
- Produces: `listDocumentChunks(kbId: number, docId: number, page: number, pageSize: number, signal?: AbortSignal): Promise<Page<KbChunk>>`

- [ ] **Step 1: 写 API 占位测试**，断言 `listKnowledgeBases`、`createKnowledgeBase`、`listDocuments`、`uploadDocument`、`deleteDocument`、`retryDocument`、`listDocumentChunks` 均拒绝为 `ApiNotConfiguredError`。
- [ ] **Step 2: 运行 API 测试确认占位失败行为**：`npm test -- src/features/knowledge/api.test.ts`。Expected: 新增用例在实现前 FAIL。
- [ ] **Step 3: 手写 feature API 函数签名与显式未接入错误**；保留未来上传使用 `FormData` 的输入类型，但当前不发网络请求、不伪造成功。
- [ ] **Step 4: 写 KnowledgePage 用户行为测试**，覆盖加载/空/错误/重试、创建表单校验、切换知识库、文件大小/类型校验、多文件逐项结果、删除确认、重试、切片分页和处理中状态；通过测试替身验证成功 UI，不把替身写进生产 bundle。
- [ ] **Step 5: 实现 Query 查询与 mutations**；服务端数据由 Query 持有。真实占位 API 被调用时展示未接入错误并保留输入；处理中轮询只在替换为真实 API 后启用。
- [ ] **Step 6: 接通页面及已有业务组件**；保留 React 当前样式，移除硬编码演示数组和布局预览说明；对话框、选择、验证、分页、确认和错误恢复行为与 Vue 一致。
- [ ] **Step 7: Commit**：`git add apps/web/src/features/knowledge && git commit -m "feat(web): handwrite knowledge and document interactions"`。

## Task 4: 手写数据源配置读取与更新

**Files:**
- Create: `apps/web/src/features/admin/api.ts`
- Create: `apps/web/src/features/admin/types.ts`
- Create: `apps/web/src/features/admin/model/queryKeys.ts`、`queries.ts`、`mutations.ts`
- Modify: `apps/web/src/features/admin/routes/DatasourcePage.tsx`、`ui/SourceCard.tsx`
- Test: `apps/web/src/features/admin/api.test.ts`
- Test: `apps/web/src/features/admin/routes/DatasourcePage.test.tsx`

**Interfaces:**
- Produces: `listDatasources(signal?: AbortSignal): Promise<DatasourceItem[]>`
- Produces: `replaceDatasources(items: DatasourceItem[], signal?: AbortSignal): Promise<{ updated: number }>`
- Produces: feature Query key `datasourceKeys.list(scope)`。

- [ ] **Step 1: 写 API 占位测试**，断言 `listDatasources` 和 `replaceDatasources` 拒绝为 `ApiNotConfiguredError`。
- [ ] **Step 2: 运行测试确认占位失败行为**：`npm test -- src/features/admin/api.test.ts`。Expected: 新增用例在实现前 FAIL。
- [ ] **Step 3: 手写 datasource API 函数签名和显式未接入错误**；数据源仍是固定 `vector/keyword/database` 类型的整体编辑，不新增单项增删。
- [ ] **Step 4: 写页面行为测试**，覆盖加载/错误/重试、开关编辑、保存中禁重提、测试替身成功反馈、未接入错误和失败时保留草稿。
- [ ] **Step 5: 接通 `DatasourcePage` 和 `SourceCard`**；编辑草稿只存在页面/表单局部状态，服务端列表仍由 Query 持有。
- [ ] **Step 6: 真实占位 API 被调用时展示未接入提示；保留后续 Query mutation 成功缓存同步的所有者位置，不显示保存成功状态。**
- [ ] **Step 7: Commit**：`git add apps/web/src/features/admin && git commit -m "feat(web): handwrite datasource configuration interactions"`。

## Task 5: 手写会话 CRUD 并迁移问答状态所有权

**Files:**
- Create: `apps/web/src/features/chat/api.ts`、`model/chatStream.ts`
- Create: `apps/web/src/features/chat/types.ts`
- Create: `apps/web/src/features/chat/model/queryKeys.ts`、`queries.ts`、`chatStreamStore.ts`、`chatState.ts`
- Modify: `apps/web/src/features/chat/routes/ChatPage.tsx`、`ui/SessionList.tsx`、`MessageList.tsx`
- Migrate/Delete: `apps/web/src/store/chat.ts`
- Test: `apps/web/src/features/chat/api.test.ts`、SSE parser/state-machine and `ChatPage.test.tsx`

**Interfaces:**
- Produces: `listSessions(signal?: AbortSignal): Promise<ChatSession[]>`
- Produces: `createSession(signal?: AbortSignal): Promise<ChatSession>`
- Produces: `deleteSession(sessionId: number, signal?: AbortSignal): Promise<void>`
- Produces: `getMessages(sessionId: number, signal?: AbortSignal): Promise<ChatMessage[]>`
- Produces: `purgeLastAssistant(sessionId: number, signal?: AbortSignal): Promise<void>` and `ask(sessionId: number, question: string, onEvent: (event: SseEvent) => void, signal: AbortSignal, skipUserPersist?: boolean, images?: string[]): Promise<'completed' | 'failed' | 'interrupted' | 'cancelled'>`
- Produces: Zustand stores only current run draft, parsed stage summaries, terminal state, and AbortController lifecycle; messages/sessions remain in Query.

- [ ] **Step 1: 写会话 API 占位测试**，断言 list/create/delete/history/purge-last-assistant/ask 函数拒绝为 `ApiNotConfiguredError`。
- [ ] **Step 2: 运行 API 测试确认占位失败行为**：`npm test -- src/features/chat/api.test.ts`。Expected: 新增用例在实现前 FAIL。
- [ ] **Step 3: 手写 chat API 函数签名与未接入错误**；页面不得 fetch、拼接 Python URL 或假装创建/删除成功。
- [ ] **Step 4: 写 SSE/parser/state-machine 测试**，覆盖 UTF-8 跨块、CRLF、多行 data、半包、多事件、未知类型、error、done、无终态 EOF、取消和旧会话迟到事件。
- [ ] **Step 5: 实现纯状态转换和测试用流消费器**；只有 `done` 为 completed，测试替身事件可验证与 Vue 相同的状态转换；生产 `ask` 占位抛出未接入错误。
- [ ] **Step 6: 将会话/历史查询和创建/删除 mutation 放入 TanStack Query**；API 未接入时提示错误，删除/创建表单不显示成功；切换会话时取消旧查询和旧流。
- [ ] **Step 7: 接通聊天页面**，验证新建/删除/选择、空/加载/错误、文本和图片输入预览、停止/断流、引用、决策摘要及重新生成交互；保留当前 React 样式。
- [ ] **Step 8: 删除已迁移的远端会话/消息 Zustand 状态**；保留并导出细粒度流状态 selector，不允许 UI 订阅整个 store。
- [ ] **Step 9: Commit**：`git add apps/web/src/features/chat apps/web/src/store/chat.ts && git commit -m "feat(web): handwrite chat session interactions"`。

## Task 6: 迁移日志只读页与检索执行页

**Files:**
- Create: `apps/web/src/features/logs/api.ts`、`model/queryKeys.ts`、`model/queries.ts`
- Create: `apps/web/src/features/logs/types.ts`
- Modify: `apps/web/src/features/admin/api.ts`
- Modify: `apps/web/src/features/logs/routes/DecisionLogPage.tsx`、`ui/LogDetailDrawer.tsx`、`ui/StepPlayer.tsx`
- Modify: `apps/web/src/features/admin/routes/PlaygroundPage.tsx`
- Migrate/Delete: `apps/web/src/store/decisionLog.ts`
- Test: `apps/web/src/features/logs/api.test.ts`、logs page tests, `apps/web/src/features/admin/api.test.ts` and Playground page tests

**Interfaces:**
- Produces: `listDecisionLogs(query: DecisionLogQuery, signal?: AbortSignal): Promise<Page<DecisionLog>>`
- Produces: `getDecisionLog(logId: number, signal?: AbortSignal): Promise<DecisionLog>`
- Produces: `testRetrieveLane(query: string, source: RetrievalSource, topK: number, kbIds: number[] | undefined, signal?: AbortSignal): Promise<RetrievalLane>`
- Produces: `fuseAndGrade(lanes: RetrievalLane[], query: string, signal?: AbortSignal): Promise<{ fused: FusedItem[]; grade: RetrievalGrade }>`

- [ ] **Step 1: 写日志 API 占位测试**，断言 list/detail 函数拒绝为 `ApiNotConfiguredError`。
- [ ] **Step 2: 运行测试确认占位失败行为**：`npm test -- src/features/logs/api.test.ts`。Expected: 新增占位用例在实现前 FAIL。
- [ ] **Step 3: 手写日志 API 占位与 Query**；筛选、日期范围、分页同步到 URL search params，未接入时保留筛选并展示错误。
- [ ] **Step 4: 写日志页面测试**，覆盖过滤、分页、空态、未接入错误、详情抽屉开关和步骤播放键盘操作；测试替身只用于覆盖成功呈现。
- [ ] **Step 5: 接通日志页面**，移除静态布局预览和旧 Zustand 远端数据副本，保留 React 当前视觉。
- [ ] **Step 6: 写 Playground API 占位与 UI 测试**，断言 lane/fuse 函数返回未接入错误；测试显式提交后呈现 loading/error/测试替身成功，不在挂载 effect 自动触发。
- [ ] **Step 7: 手写检索测试 API 占位并接通 Playground**；每次执行绑定当前查询，旧响应不得覆盖新的查询结果。
- [ ] **Step 8: Commit**：`git add apps/web/src/features/logs apps/web/src/features/admin && git commit -m "feat(web): connect decision logs and retrieval playground"`。

## Task 7: 整合关键旅程、清理兼容代码并完成门禁

**Files:**
- Modify/Create: `apps/web/e2e/legacy-migration.spec.ts`
- Delete after migration: `apps/web/src/lib/api/admin.ts`、`apps/web/src/lib/api/chat.ts`、`apps/web/src/store/chat.ts`、`apps/web/src/store/kb.ts`、`apps/web/src/store/datasource.ts`、`apps/web/src/store/decisionLog.ts`（仅在所有调用迁移后）
- Delete after migration: `apps/web/src/lib/api/http.ts`、`apps/web/src/lib/api/sseParser.ts`、`apps/web/src/types/legacy.ts`（仅在调用和类型已迁移后）

**Interfaces:**
- Consumes: Task 1 的 API 占位错误与 Task 2–6 feature public API。
- Produces: React 关键交互的可复现验证；运行时仅 health 调用 Go，所有业务操作诚实显示未接入。

- [ ] **Step 1: 写 Playwright 交互旅程**：打开各旧版路由、检查 React 视觉稿仍在、打开知识库/数据源/会话操作、触发未接入提示、验证 health 状态显示；业务成功状态由 feature 组件测试中的测试替身覆盖。
- [ ] **Step 2: 验证 Playwright 断言真实用户可见导航、表单、确认、错误和重试行为**，不依赖 class 名或任意 sleep；不能伪造业务 API 成功。
- [ ] **Step 3: 搜索 `apps/web/src` 的旧 `/api/chat`、`/api/admin` 请求、Python base URL、硬编码 CRUD 演示数组和已废弃 store 导入；删除旧业务请求实现及状态副本，保留 `LegacyPlaceholder` 仅当仍被真实路由使用。**
- [ ] **Step 4: 执行前端门禁**：`npm run typecheck`、`npm test`、`npm run build`、`npm run test:e2e`；按仓库根 AGENTS 记录命令输出和任何阻塞。
- [ ] **Step 5: 检查手工键盘流程、焦点管理、320px 宽度、200% 文本缩放、错误恢复、请求取消和隐私日志；修复后复跑受影响门禁。**
- [ ] **Step 6: Commit**：`git add -A -- apps/web/e2e/legacy-migration.spec.ts apps/web/src/lib/api/admin.ts apps/web/src/lib/api/chat.ts apps/web/src/lib/api/http.ts apps/web/src/lib/api/sseParser.ts apps/web/src/types/legacy.ts apps/web/src/store/chat.ts apps/web/src/store/kb.ts apps/web/src/store/datasource.ts apps/web/src/store/decisionLog.ts && git commit -m "test(web): verify migrated agent interactions"`。

## Self-Review

- **范围覆盖：** `/chat` 会话/消息/SSE、`/admin/kb` 知识库和文档操作、`/admin/datasources` 配置、`/admin/logs` 只读日志、`/admin/playground` 检索执行均映射到 Task 3–6；集成和清理映射到 Task 7。
- **契约准确性：** 用户已确定本次不接入 Go 业务 API、不直连 Python、不扩展资源操作；只有 health 可实际调用，业务操作显式显示未接入。
- **所有权一致：** Query 管服务端数据，Zustand 只管活动流，表单草稿局部持有；迁移中的旧 store 有明确清理任务。
- **失败路径：** API 未接入错误、取消、处理中展示、断流状态、过期回调和未保存草稿均在对应任务里有可验证行为；真实服务端写入结果留待未来 API 接入时验证。
- **比例与边界：** 计划复用 `apps/web` 当前 React 壳和 UI 组件，不重构后端/Agent 算法，不修改归档目录，不引入新依赖；Go 变更仅在用户作出对应决策时进入范围。
