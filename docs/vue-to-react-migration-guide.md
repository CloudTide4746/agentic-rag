# Agentic RAG 前端 Vue 到 React 迁移技术指导

版本：1.0  
适用范围：`archive/frontend-legacy-20260921` 到 `apps/web`   
目标技术栈：React 19、TypeScript strict、Vite、React Router、TanStack Query、Zustand、Tailwind CSS、Vitest、React Testing Library、Playwright  
迁移原则：先保持行为和协议，再逐步重组模块；不在迁移过程中重写 Go/Python Agent 的业务事实。

## 1. 迁移目标和边界

这不是把 `.vue` 文件机械改成 `.tsx`，而是将旧 Vue 应用的用户可见行为、接口契约和关键状态机迁移到正式的 `apps/web` 架构。

必须保留的行为：

- `/chat`：会话列表、创建/删除会话、消息历史、POST SSE 流式问答、停止回答、引用展示、决策链实时展示、历史决策日志回看和重新生成语义。
- `/admin/kb`：知识库列表、知识库选择、文档列表、上传、删除、重试、切片分页、处理中状态轮询。
- `/admin/datasources`：vector、keyword、database 三类检索源的加载、开关编辑和保存。
- `/admin/playground`：三路召回、单路结果、RRF 融合、Grader 评分和样例查询。
- `/admin/logs`：按会话/日期过滤、分页、详情抽屉、决策步骤播放。
- 主题、布局、按钮、抽屉、模态框、分页、空状态等公共 UI 的可访问行为。

不在本次前端迁移中自行改变：

- Python Agent 的提示词、检索算法、数据库表和 SSE 事件业务语义。
- Go/Python 的数据所有权。正式版本由 Go 作为公开入口，浏览器不应继续依赖旧的 FastAPI 直连地址。
- `archive` 目录中的文件。归档只用于行为参考，不作为新代码目录或模板。

## 2. 已确认的旧项目基线

### 2.1 目录和依赖

旧项目是 Vite + Vue 3 + Pinia + Vue Router + Tailwind 的 TypeScript 应用，主要文件如下：

```text
archive/frontend-legacy-20260921/src/
  App.vue                         # 根应用
  main.ts                         # createApp、Pinia、Router、主题样式
  router/index.ts                 # 布局路由和懒加载页面
  api/types.ts                    # 实体、SSE 事件和分页类型
  api/http.ts                     # fetch JSON 封装
  api/chat.ts                     # 会话、历史、POST SSE
  api/admin.ts                    # 知识库、数据源、日志、检索测试
  api/sseParser.ts                # 跨 ReadableStream chunk 的 SSE 解析器
  stores/chat.ts                  # 问答和决策链状态机
  stores/kb.ts                    # 知识库和文档轮询
  stores/datasource.ts            # 数据源配置
  stores/decisionLog.ts           # 日志过滤和分页
  stores/theme.ts                 # 主题状态
  components/layout/*             # AppLayout、TopNavBar
  components/ui/*                 # BaseButton、Modal、Drawer、Input 等
  components/chat/*               # 消息、决策侧栏、引用和输入框
  components/kb/*                 # 文档行、上传、切片抽屉
  components/logs/*               # 日志详情和步骤播放器
  components/admin/*              # 数据源卡片
  components/visual/*             # 粒子背景
  views/*                         # 五个页面
```

旧 `package.json` 的关键依赖是 `vue`、`vue-router`、`pinia`、`three`；没有 React Query、Zustand、Markdown 清洗器或测试脚本。因此迁移时不要假定旧代码已经满足正式前端规范。

### 2.2 路由基线

旧路由只有一个布局父路由，页面懒加载：

| 旧路径 | 页面 | React 目标 |
|---|---|---|
| `/` | 重定向 | `Navigate` 到 `/chat` |
| `/chat` | `ChatView.vue` | `features/chat/pages/ChatPage.tsx` |
| `/admin/kb` | `KbManageView.vue` | `features/knowledge/pages/KnowledgeBasePage.tsx` |
| `/admin/datasources` | `DatasourceView.vue` | `features/knowledge/pages/DataSourcesPage.tsx` |
| `/admin/playground` | `PlaygroundView.vue` | `features/playground/pages/PlaygroundPage.tsx` |
| `/admin/logs` | `DecisionLogView.vue` | `features/logs/pages/DecisionLogsPage.tsx` |

页面必须挂在 `AppLayout` 下；路由页面使用 `React.lazy` + `Suspense`，避免首屏加载所有管理端代码。页面标题由路由配置统一设置，不在每个页面散落 `document.title`。

### 2.3 领域类型和协议

旧 `api/types.ts` 已经定义了 `KnowledgeBase`、`KbDocument`、`KbChunk`、`ChatSession`、`ChatMessage`、`DecisionLog`、`DatasourceItem`、`RetrievalLane`、`FusedItem`、`Page<T>` 等类型。

SSE 事件类型必须保留：

```text
router | rewrite | retrieve | grade | correct |
generate | reflect | citation | done | error
```

目标公开接口应遵循仓库架构约定的 `/api/v1`、snake_case、字符串平台 ID 和统一错误结构。旧 `/api/chat/*`、`/api/admin/*` 路径只能在适配层兼容，不应被 React 组件直接调用。

## 3. 目标目录和依赖方向

目标目录采用仓库规定的按业务能力组织方式：

```text
apps/web/src/
  app/
    App.tsx
    router.tsx
    providers.tsx
    error-boundary.tsx
  features/
    auth/
    chat/
      api/
      components/
      hooks/
      pages/
      store/
      types.ts
      index.ts
    knowledge/
    playground/
    logs/
    analytics/
  components/
    layout/
    ui/
  lib/
    http.ts
    sse.ts
    query-client.ts
    markdown.ts
    browser.ts
  styles/
    globals.css
    tokens.css
  types/
    api.ts
```

依赖只能向下：`app -> features -> components/lib`。例如聊天页可以使用 `chat` 的公开入口，但知识库页面不能深层导入 `chat/store/internal.ts`。跨 feature 需要的纯类型下沉到 `types`，无业务倾向的 UI 下沉到 `components/ui`。

## 4. Vue 概念到 React 概念映射

| Vue | React 迁移方式 | 约束 |
|---|---|---|
| `createApp(App).use(...)` | `createRoot` + Provider 组合 | Provider 放 `app/providers.tsx`，业务规则不写进 `App.tsx` |
| `.vue` SFC | `.tsx` 组件 + CSS/Tailwind | 组件保持单一职责，避免一个页面复制所有模板 |
| `props` | `type Props` + 函数组件参数 | 不使用 `any`，边界数据先校验 |
| `defineEmits` | 回调 props | 命名为 `onClose`、`onChange` 等，不模拟事件总线 |
| `ref` | `useState` 或 `useRef` | 需要渲染的值用 state；瞬时值/DOM 引用用 ref |
| `computed` | 渲染期间直接派生，必要时 `useMemo` | 简单布尔值不要为了形式引入 effect 或 memo |
| `watch` | `useEffect` | 仅同步外部系统；必须有依赖和清理函数 |
| `onMounted/onBeforeUnmount` | `useEffect` 返回清理函数 | 轮询、事件、粒子动画、AbortController 都要清理 |
| Pinia store | TanStack Query + Zustand 的组合 | 远端事实只在 Query；流式临时态才进 Zustand |
| `v-if/v-else` | 三元表达式或显式早返回 | 避免把 `condition && 0` 等非布尔内容误渲染 |
| `v-for :key` | `array.map` + 稳定 key | 不用数组下标作为会增删排序的业务实体 key |
| `v-model` | `value` + `onChange` | 表单双向绑定拆成显式读写，保留受控状态 |
| `provide/inject` | Context | 只用于稳定的依赖注入，不替代 Query/Zustand |
| `<slot>` | `children` 或命名 props | 插槽复杂时拆成明确的组合组件 |
| `<Teleport>` | Portal | Modal/Drawer 挂到 `document.body`，并管理焦点 |
| Vue Router meta | 路由对象的 `handle`/配置 | 页面标题、权限和懒加载统一处理 |

## 5. 分阶段迁移路线

### 阶段 0：冻结基线和建立证据

1. 在迁移分支记录当前归档版本、Node/npm 版本和后端地址。
2. 运行旧项目并截取五条核心旅程：登录/进入聊天、流式回答、上传文档、检索测试、日志回放。
3. 保存真实 API 请求、响应和 SSE 样本，尤其是 Unicode 跨块、`generate.delta` 的追加行为、`done` 和 `error`。
4. 把旧路径与新 `/api/v1` 路径的映射写进接口契约；不要让页面在迁移期间同时维护两套请求函数。
5. 建立迁移验收表：每个路径、状态、错误分支和键盘操作都有对应测试或手工证据。

完成标准：旧应用可重复启动，关键流程有样本，协议字段和数据所有权已确认。

### 阶段 1：建立 React 壳和设计令牌

1. 在 `apps/web` 固定 React 19、TypeScript strict、Vite 和锁定版本。
2. 加入 React Router、TanStack Query、Zustand、Tailwind、Vitest、Testing Library、Playwright。
3. 先迁移 `theme.css` 中的颜色、字号、间距、边框和圆角为 Tailwind CSS 变量；不在组件里重新发明颜色。
4. 实现 `AppProviders`：QueryClient、错误边界、主题 Context/存储适配器。
5. 实现 `AppLayout`、TopNav、全局 loading/error/empty 基础状态。
6. 使用路由级 `lazy`，每个页面有自己的 Suspense fallback。

不要在这个阶段迁移业务请求；壳能启动、路由能切换、旧视觉基线能对比即可。

### 阶段 2：先迁移类型、HTTP 和 SSE 适配层

这是所有页面共用的稳定锚点，应先完成。

`lib/http.ts` 至少包含：

- 统一 base URL 和凭据策略。
- `AbortSignal` 透传。
- 非 2xx 解析为带 `code/message/request_id` 的可识别错误。
- JSON 响应运行时校验；不要把 `as T` 当作验证。
- 文件上传使用 `FormData`，不要手动设置 multipart boundary。
- 不记录 token、完整问题、答案或文档正文。

`lib/sse.ts` 必须沿用旧 `SseParser` 的正确性要求：

1. `ReadableStream` 每次 `read()` 只是一段字节，不等于一个事件。
2. `TextDecoder.decode(value, { stream: true })` 保留跨块 UTF-8 字符。
3. 归一化 CRLF，缓存半包，支持同一块多个事件。
4. 支持多行 `data:`，按 SSE 规则拼接后再 JSON 解析。
5. 未知事件记录诊断信息但不让整条流静默成功。
6. `done` 才是 completed；`error` 是 failed；无终态 EOF 是 interrupted。
7. 解析失败要抛出可展示错误，不能 `catch { return null }` 后假装请求成功。

建议接口：

```ts
export type StreamTerminal = 'done' | 'error' | 'eof'

export async function consumeSse<TEvent>(
  response: Response,
  onEvent: (event: TEvent) => void,
  signal?: AbortSignal,
): Promise<StreamTerminal> {
  // 具体实现放在 lib，不让组件接触 reader/decoder/buffer。
}
```

### 阶段 3：迁移服务端状态

将旧 Pinia store 拆为 Query hooks 和 mutation：

| 旧 store | React 归属 | Query key 示例 |
|---|---|---|
| `chat.sessions` | `useChatSessionsQuery` | `['chat-sessions', workspaceId]` |
| `chat.messages` | `useMessagesQuery` | `['messages', workspaceId, sessionId]` |
| `kb.kbs` | `useKnowledgeBasesQuery` | `['knowledge-bases', workspaceId]` |
| `kb.documents` | `useDocumentsQuery` | `['documents', workspaceId, kbId]` |
| `decisionLog.logs` | `useDecisionLogsQuery` | `['decision-logs', workspaceId, filters]` |
| `datasource.items` | `useDataSourcesQuery` | `['data-sources', workspaceId]` |

每个 key 必须包含身份/工作空间和全部筛选条件。切换工作空间或退出登录时：取消旧请求、清除敏感 Query cache，并重置聊天流式状态。

文档处理中轮询改用 Query 的 `refetchInterval`，只在存在 `parsing/chunking/embedding` 文档时启用；组件卸载或切换知识库自动停止。不要把整个 `documents` 列表复制到 Zustand。

### 阶段 4：迁移聊天状态机

旧 `stores/chat.ts` 是最复杂的迁移点。不要把它直接翻译成一个巨大的 React Context；拆成：

- `chat-api.ts`：会话、历史和 run 请求。
- `chat-stream-store.ts`：当前 run 的 `streamingText`、`cards`、`pendingCard`、`streamError`、`activeStage`、`doneStats`。
- `decision-machine.ts`：纯函数，输入一个 `SseEvent` 和当前状态，输出新状态。
- `useChatRun.ts`：创建 AbortController、消费 SSE、处理终态和刷新 Query。
- 展示组件：只读取状态并发出用户动作。

`decision-machine.ts` 必须保持以下不变量：

1. 普通决策事件会先结算上一个 pending 卡，再追加当前卡。
2. `generate` 事件只追加 `delta`，不能把整个答案重复拼接。
3. 生成卡的耗时从首个 generate 的 `ts` 计算到下一个结算事件。
4. `grade` 非 pass、`correct`、`reflect` fail、`error` 显示 warn。
5. `correct`/`reflect fail` 会推进回流轮次，不能只按事件数组长度推断。
6. 收到 `done` 后结算 pending、写入 token 统计，并刷新消息历史。
7. 用户主动 Abort 不显示成服务端错误；无 done 的网络断流仍标记 interrupted。
8. 流式草稿只在 Zustand；最终消息以 Query 重新读取的 Python 持久化记录为准。

示意：

```ts
type ChatStreamState = {
  cards: DecisionCard[]
  pendingCard: DecisionCard | null
  streamingText: string
  activeRound: number
  activeStage: number
  terminal: 'idle' | 'running' | 'completed' | 'failed' | 'interrupted'
}

function reduceSseEvent(state: ChatStreamState, event: SseEvent): ChatStreamState {
  // 纯函数：不调用 API、不读 window、不修改输入对象。
}
```

### 阶段 5：迁移页面和组件

推荐顺序：聊天垂直闭环 → 知识库 → 数据源 → Playground → 日志。

聊天闭环先只交付：选择会话、读取历史、发送问题、看到流式文本、收到 done 后刷新。决策侧栏、引用抽屉和重新生成在闭环稳定后接入。

知识库页面先交付列表和状态，再加入上传、删除、重试、切片 Drawer。上传 mutation 的未知结果不能盲目重试；界面显示处理中/对账中，而不是假装成功。

Playground 的三个独立召回请求可以使用 `Promise.all` 并行，随后再调用 fuse/grade；不得串行等待无依赖的 lane，避免瀑布请求。重型粒子背景、Markdown 渲染器、图表和日志播放器按需动态加载。

日志页面的筛选条件同步 URL；详情抽屉由 `logId` 驱动 Query，关闭时取消未完成请求。StepPlayer 的播放计时器必须在 effect cleanup 中清除。

## 6. 组件迁移细则

### 6.1 UI 基础组件

| Vue 组件 | React 目标 | 必须保留的行为 |
|---|---|---|
| `BaseButton` | `components/ui/Button.tsx` | disabled、loading、键盘 focus、语义 button |
| `BaseInput` | `Input.tsx` | label、错误关联、受控 value、Enter 行为 |
| `BaseModal` | `Modal.tsx` + Portal | Escape、遮罩关闭策略、焦点回收、`aria-modal` |
| `BaseDrawer` | `Drawer.tsx` | 侧滑、焦点管理、关闭按钮和响应式宽度 |
| `BasePagination` | `Pagination.tsx` | page 边界、可访问 label、稳定页码算法 |
| `BaseToggle` | `Switch.tsx` | `role=switch`、键盘 Space、disabled |
| `BaseEmpty` | `EmptyState.tsx` | 空态说明和可执行动作 |
| `BaseTag/StatusBadge` | `Tag.tsx`/`StatusBadge.tsx` | tone 映射和文本颜色对比度 |
| `Icon` | `Icon.tsx` | 统一图标来源，按钮提供 tooltip/aria-label |

不要用一个带几十个布尔 props 的“万能组件”代替这些组件；复杂行为用组合组件表达。

### 6.2 Vue 模板改写示例

Vue：

```vue
<BaseButton :loading="running" :disabled="!query.trim()" @click="run">
  {{ running ? '执行中' : '执行' }}
</BaseButton>
```

React：

```tsx
<Button
  type="button"
  loading={running}
  disabled={!query.trim()}
  onClick={run}
>
  {running ? '执行中' : '执行'}
</Button>
```

Vue 的 `v-for`、条件 class 和事件不要逐字翻译。先提取稳定的 view model，再渲染：

```tsx
const visibleLanes = enabledSources.map((source) => ({
  source,
  meta: SOURCE_META[source],
  state: lanes[source],
}))

return visibleLanes.map(({ source, meta, state }) => (
  <RetrievalLaneCard key={source} source={source} meta={meta} state={state} />
))
```

## 7. 性能规则（迁移时必须执行）

本节采用 Vercel React Best Practices 的优先级，避免迁移后只追求“能运行”。

### 7.1 消除请求瀑布

- Playground 三路独立检索使用 `Promise.all`；融合必须等待 lanes，不能反过来阻塞配置读取。
- 页面所需的独立 Query 并行预取；只有确实依赖上一个结果的请求才串行。
- 对话页面先启动 sessions/messages 请求，再在分支中等待；不要在 effect 中层层触发请求。
- 重型组件使用懒加载和 Suspense，避免首屏把 Three.js、Markdown、日志播放全部打进主包。

### 7.2 控制包体

- 直接从具体模块导入，避免 barrel 导入导致无法 tree-shake。
- 管理端页面和 Playground 使用路由级动态 import。
- 第三方分析、日志、图表仅在真正需要时加载。
- 不把旧的 Three.js 粒子背景作为所有页面的同步首屏依赖；默认可降级为静态背景。

### 7.3 减少重复渲染

- `DecisionStepCard`、`MessageItem`、`RetrievalHit` 等昂贵列表项拆成 `memo` 组件，并保持 props 原始值稳定。
- 只订阅 Zustand 中当前组件使用的字段，不在所有组件中读取整个 store。
- 从 render 派生 `isProcessing`、`hasError` 等简单值，不使用 effect 写回派生 state。
- 输入框的瞬时光标/组合输入可用 ref；需要展示的 query 仍放 state。
- 高频流式 token 更新时，使用切分组件或 `useDeferredValue` 保持输入响应；不要对每个 token 触发整个页面重渲染。

### 7.4 正确处理 effect

- effect 只用于 SSE、轮询、键盘/窗口事件、动画和浏览器 API 等外部系统。
- 所有 effect 返回清理：`AbortController.abort()`、`clearInterval`、移除 listener、销毁 ECharts/Three 实例。
- 不在挂载 effect 中自动发起计费问答、上传或创建会话。
- 非紧急筛选/日志分页可使用 `startTransition`，但不能用 transition 掩盖错误状态。

## 8. 安全、权限和数据一致性

1. 浏览器只提交平台知识库 ID；不能提交 `skip_user_persist`、Agent 内部 ID 或任意 `kb_ids`。
2. 前端隐藏按钮只是体验，所有会话、文档、引用和日志访问必须由 Go 再校验工作空间和资源权限。
3. 不把刷新令牌、完整答案、文档正文持久化到 localStorage。主题等非敏感偏好才可存储。
4. Markdown 使用清洗后的渲染器；链接协议只允许 `https`、`http` 和必要的站内路径，禁止直接渲染未清洗 HTML。
5. 文件名永远不是磁盘路径；上传前校验扩展名、大小，服务端再次校验。
6. Query key 必须带 `workspace_id`；退出/切换空间时清理旧缓存，避免跨空间显示数据。
7. 引用按授权映射解析；同名文档不能猜测，无法唯一定位时展示“引用无法定位”。
8. SSE 错误要区分 HTTP 开始前 JSON 错误、开始后的 `error` 事件和无终态断流。

## 9. 测试迁移计划

### 9.1 单元测试

- `sseParser.test.ts`：半包、CRLF、多行 data、Unicode、多个事件同 chunk、未知事件、非法 JSON、无 done EOF。
- `decision-machine.test.ts`：pending 结算、generate 累积、回流轮次、warn 条件、done/error/interrupted。
- `pagination.test.ts`：第一页、最后一页、总数为 0、页码省略号。
- `citation.test.ts`：引用 URL 编码、重复文档名、无法定位。

### 9.2 组件测试

使用 Testing Library 的用户行为和可访问语义断言：

- Button loading 时不可重复提交；Modal Escape 和焦点回收。
- ChatInput 回车/组合键、停止按钮、错误重试提示。
- MessageList loading/empty/error；流式文本增量展示。
- UploadModal 文件类型/大小错误、上传中禁用、成功后刷新。
- DecisionSidebar timeline/pipeline 切换、warn 标签和日志回看。

### 9.3 Playwright 核心旅程

至少覆盖：

1. 登录 → 进入聊天 → 发送问题 → 看到 generate 增量 → done → 历史消息出现。
2. 同一会话第二次提交被阻止或收到 409 提示。
3. 关闭流连接后显示 interrupted，不把它伪装成 completed。
4. 上传文档 → 轮询 parsing/chunking/embedding → ready/failed。
5. 只能打开授权知识库、会话、引用和日志。
6. 切换工作空间后旧 Query 不会短暂显示。

## 10. 迁移验收门槛

### 功能

- 五个旧路由均有 React 页面，默认 `/` 跳转 `/chat`。
- 聊天流保持事件顺序、增量文本、引用和终态语义。
- 知识库处理状态轮询只在需要时运行，上传/删除/重试错误可见。
- 日志筛选、分页和播放结果与旧行为一致。

### 架构

- 远端列表只有 TanStack Query 一个事实源；Zustand 只保存流式临时态和跨组件交互。
- 页面不直接 `fetch`，组件不读环境变量，不深层导入其他 feature。
- 每个 feature 有公开 `index.ts`；app 只组合路由、Provider 和布局。
- 所有 effect、SSE reader、轮询、计时器和图表实例都有清理路径。

### 安全和可靠性

- 权限测试覆盖他人会话/文档/引用、空权限集合和工作空间切换。
- 不保存敏感令牌/正文，不渲染未清洗 HTML，不把失败伪装成成功。
- 超时结果未知时进入对账或可重试状态，不能自动重复创建资源。

### 工程门禁

在 `apps/web` 执行并记录真实结果：

```powershell
npm ci
npm run typecheck
npm test
npm run build
npm run test:e2e
```

如果脚手架尚未提供某个脚本，应先补齐脚本或明确记录阻塞原因，不能在文档中声称通过。迁移完成还要检查 git diff，确认没有修改 `archive`、没有秘密、没有调试日志和未清理的订阅。

## 11. 建议提交顺序和回滚策略

按可独立验证的垂直切片提交：

1. `docs`: 契约、类型和迁移决策记录。
2. `app-shell`: React 壳、Provider、路由、主题。
3. `transport`: HTTP、SSE、错误模型和测试。
4. `chat-core`: Query、Zustand 流状态机、聊天闭环。
5. `knowledge`: 知识库、上传、轮询和切片。
6. `admin-tools`: 数据源、Playground、日志。
7. `qa`: Playwright、权限/故障注入证据和部署说明。

每一切片都应能独立回滚，不要把后端接口破坏性变更和页面重写混在一个提交。上线初期保留旧归档作为只读参考，使用功能开关或部署回滚恢复旧前端；不要让新旧前端同时写同一套临时状态。

## 12. 开发者逐项执行清单

### 开始前

- [ ] 读取根 `AGENTS.md`、`apps/web/AGENTS.md`、前端规范和本指南。
- [ ] 确认旧 API/SSE 样本和目标 `/api/v1` 契约。
- [ ] 确认数据所有者、权限边界和工作空间作用域。
- [ ] 检查工作区已有修改，不覆盖用户代码。

### 实现中

- [ ] 先完成类型、HTTP、SSE 和纯状态机，再写展示层。
- [ ] 远端数据进 Query，流式临时数据进 Zustand，表单进组件 state，筛选进 URL。
- [ ] 所有请求支持 AbortSignal；所有 effect 有清理。
- [ ] 独立请求并行，重型组件按需加载，避免 barrel import。
- [ ] loading、empty、error、disabled、success、interrupted 均有界面。

### 完成前

- [ ] 运行 typecheck、unit/component test、build 和相关 Playwright。
- [ ] 手工走登录→上传→处理完成→提问→引用→历史消息完整链路。
- [ ] 检查 SSE 半包、Unicode、错误、断流和并发。
- [ ] 检查权限、缓存清理、敏感信息、日志和可访问性。
- [ ] 检查 `archive` 未被修改，文档/OpenAPI/测试与行为同步。

迁移完成的判断标准不是“页面看起来像 Vue 版本”，而是 React 版本在相同协议、权限和故障条件下仍然保持同样的业务不变量，并满足 `apps/web` 的模块边界、状态所有权、性能和可验证性要求。
