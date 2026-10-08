# React 组件拆分与样式组织规则（草案）

状态：供团队讨论与修改  
适用范围：`apps/web/src/**`  
关联规范：[前端生产级工程规范](./frontend-engineering-standards.md)、[`apps/web/AGENTS.md`](../AGENTS.md)

> 本文聚焦“组件何时拆、拆成什么、样式放哪里”。它补充现有前端规范，不改变既定技术栈、状态所有权或模块依赖方向。团队确认后可将其中 MUST/MUST NOT 条款纳入正式规范。

## 1. 核心原则

1. **按变化原因拆分，不按代码行数拆分。** 组件负责一个主要 UI 职责；超过约 200 行只是检查信号，不是强制拆分线。
2. **业务归 feature，通用能力归 shared。** 页面业务组件默认留在 `features/<feature>/ui`；只有至少两个真实调用方且语义一致时，才考虑提升到 `shared/ui`。
3. **行为与呈现分层，样式与组件就近。** 数据获取、业务规则和状态所有权放在页面/feature model/hook；视图通过 props 接收数据和动作。组件样式默认与组件同目录维护，不要求单独建立全局样式文件。
4. **拆分必须降低理解成本。** 若抽出子组件后需要传递大量临时 props、暴露内部实现或增加调用方概念，优先留在原组件。
5. **先组合，后抽象。** 组件优先通过明确 props 和 children 组合，不用 `isX`、`hasY` 等布尔开关拼出多种无关布局。

## 2. 拆分决策

遇到复杂组件时，按以下顺序判断：

### 应拆分

- 页面/路由同时承担请求编排、业务转换和大块视图：将业务数据与动作留在页面或 feature hook，将可独立表达的区域提为 UI 子组件。
- 某个区域有独立的交互状态、键盘/焦点行为或生命周期：提为具名子组件或自定义 hook，使其行为可独立测试和复用。
- 一段 UI 在同一 feature 中重复出现且语义、变化原因和交互一致：在该 feature 内提取私有组件。
- 某个组件有独立业务概念或被多个页面以同样方式使用：考虑 feature 的公开组件或稳定的 `shared/ui` 组件。
- 一段纯业务转换可脱离 React 输入输出：提为 feature `model` 中的纯函数。

### 暂不拆分

- 只为让文件变短而抽取只有一处使用、没有清晰语义名称的 JSX 片段。
- 抽取后只剩一层参数透传，调用方仍须理解原组件的所有内部细节。
- 多个页面看起来相似，但文案、状态、交互或变化原因不同。
- 为尚未发生的主题切换、皮肤替换或布局变化预建通用配置系统。

### 快速检查问题

拆分前回答：

1. 新组件有清楚、稳定的名字吗？名字能说出其 UI/业务职责吗？
2. 它是否有独立变化原因、交互边界或至少两个真实调用方？
3. 它需要的 props 是否少且表达业务意图？
4. 拆分后父组件是否更容易读懂？子组件是否可以独立理解？
5. 它放在当前 feature、公开入口或 `shared/ui` 的理由是什么？

如果多数答案是否，先不拆。

## 3. 推荐组件层次

| 层次 | 责任 | 典型位置 | 可以做什么 | 不应做什么 |
| --- | --- | --- | --- | --- |
| 路由/页面 | 组合页面、连接数据边界 | `features/<feature>/routes` | 调用 feature 数据能力，处理路由参数，组合页面区域 | 承载整页所有 JSX、重复实现通用视觉规则 |
| Feature 容器 | 连接业务状态与视图 | `features/<feature>/ui` 或 feature 私有目录 | 选择数据、协调动作、处理业务状态 | 直接实现通用 HTTP/SSE 协议 |
| Feature 展示组件 | 表达该业务的 UI | `features/<feature>/ui` | 接收已准备数据、触发回调、呈现业务状态 | 自行请求、访问全局 store 或其他 feature 私有文件 |
| Shared 基础组件 | 提供无业务偏向的通用交互/视觉能力 | `shared/ui` | Button、Input、Modal 等；保证语义和可访问性 | 判断知识库/工作空间等业务规则 |
| 自定义 hook | 封装有完整语义的状态/副作用行为 | 所属 feature 的 `model` 或 `hooks`（沿用现有结构） | 处理可复用行为，明确输入、输出和清理 | 仅把几行代码改名包装，或成为无边界全局工具 |

层次不是必须逐级创建。简单页面可直接使用展示组件；不要为了符合表格而增加空壳组件。

## 4. 功能与样式如何分工

### 功能侧

- 服务端数据由 TanStack Query 所有；组件不复制一份到本地 state。
- feature 业务状态和纯转换归 feature；短暂 UI 状态（弹层开关、局部展开）归拥有它的组件。
- 展示组件通过 props 获取数据和回调，不直接 `fetch`、读环境变量、访问 localStorage 或操作全局 store。
- 回调名称描述意图，如 `onRetry`、`onSelectSource`，避免向展示组件传入整套底层服务对象。
- 加载、空、错误、禁用和成功状态应作为明确 UI 状态呈现；互斥模式优先用 discriminated union，而非可冲突布尔值。

### 样式侧

- 组件专属样式和 JSX 就近维护；沿用项目当前的 Tailwind 与全局 token，不为每个组件额外创建样式系统。
- 颜色、间距、字号、圆角、阴影、层级使用既有设计 token；组件状态样式和结构一起维护，避免散落到多个不相关全局选择器。
- `src/styles` 仅放全局 reset、字体、设计 token、全局基础样式；不放某个业务页面的局部布局规则。
- `shared/ui` 的样式只表达组件自身的视觉/交互状态（尺寸、变体、焦点、禁用等），不包含具体页面布局或业务语义。
- feature 页面可以组合 shared 组件并定义自己的布局与业务视觉，不反向修改 shared 组件来适配单个页面。
- 不以“将来换主题”为由预先加入复杂主题 provider、样式接口或 CSS 变量层；出现真实多主题需求时再单独设计。

## 5. 文件与目录约定

优先贴合现有 feature 结构，不为单个小组件制造过多目录。示例：

```text
src/features/knowledge/
  routes/
    KnowledgePage.tsx       # 页面组合与路由数据边界
  ui/
    KnowledgePageHeader.tsx # feature 私有展示区域
    SourceList.tsx
    SourceListItem.tsx
  model/
    formatSource.ts         # 可独立验证的纯业务转换
  index.ts                  # 只导出允许跨 feature 使用的公开能力
```

若确实需要独立的组件专属样式文件，按组件同名放置，并限制选择器作用域；Tailwind 为主的组件通常不需要额外 CSS 文件：

```text
SourceList.tsx
SourceList.css
```

约定：

- 私有组件默认不从 feature `index.ts` 导出。
- 跨 feature 使用只能经对方 `index.ts` 的公开 API；禁止深层导入。
- 不要创建只有一个文件且没有边界价值的 `components/`、`hooks/`、`utils/` 嵌套目录。
- 不要把组件与其纯业务规则、API 请求或测试夹具混进一个超大文件；各自放在真实所有者目录。

## 6. Props 与组合约定

- props 只暴露组件完成职责所需的数据和动作；优先传领域数据或展示模型，不传 QueryClient、store、fetcher 等底层对象。
- 组件需要不同展示模式时，用有意义的变体（如 `size`、`tone`）表达确实共享的视觉差异；完全不同的结构拆成不同组件或由父级组合。
- 避免多个可同时为真的布尔 props 控制复杂布局。状态互斥时使用判别联合；内容可组合时使用 `children` 或明确插槽。
- 只有在调用方需要真正自定义内部内容时才开放插槽，不把所有 DOM 区域都变成配置 API。
- 列表项使用稳定 key；不要让子组件通过索引推测业务身份。

示例：

```tsx
type LoadState<T> =
  | { status: 'loading' }
  | { status: 'error'; message: string; onRetry: () => void }
  | { status: 'success'; items: T[] };

function SourcePanel({ state }: { state: LoadState<SourceView> }) {
  if (state.status === 'loading') return <SourceListSkeleton />;
  if (state.status === 'error') {
    return <ErrorNotice message={state.message} onRetry={state.onRetry} />;
  }
  if (state.items.length === 0) return <EmptySources />;
  return <SourceList items={state.items} />;
}
```

## 7. 拆分后的验证

改动组件边界后，验证行为而非文件结构：

- 用户可见行为、键盘交互和可访问名称保持正确。
- loading、empty、error、disabled、success 等相关状态仍有明确表现。
- 组件卸载、路由切换或作用域变化后，订阅与请求按规范取消/清理。
- 测试通过 role、label、可见文本和用户动作观察结果，不依赖 CSS 类名或组件私有实现。
- 若只是纯重排 JSX 且无行为变化，不为拆分本身制造大量快照或重复测试。

涉及前端代码修改时，仍按仓库要求运行 `npm run typecheck`、`npm test`、`npm run build`；用户旅程变化时执行相关 Playwright 用例。仅新增/修改本规则文档时不需要运行前端构建。

## 8. Code Review 检查清单

- [ ] 拆分对应一个明确职责或独立变化原因，不是单纯追求小文件。
- [ ] 页面/feature/shared 归属合理，没有把业务组件误放入 shared。
- [ ] 展示组件不直接访问网络、全局存储或环境配置。
- [ ] 样式靠近所有者；全局样式没有混入单页规则。
- [ ] props 简洁表达意图，没有布尔参数爆炸或底层对象透传。
- [ ] 状态归属唯一；互斥状态表达清楚。
- [ ] 可访问性、错误状态和副作用清理没有因拆分退化。
- [ ] 新的跨 feature 能力只通过公开入口使用。

## 9. 待团队确认的选择

以下内容刻意留给团队修改后定稿：

1. 是否要求每个 feature 的展示组件都放入 `ui/`，还是允许页面同目录放置私有组件。
2. 项目以 Tailwind 为主时，是否仅在复杂样式场景允许组件专属 CSS 文件。
3. 是否把“至少两个真实调用方”作为 shared 提取的硬门槛，或允许一个明确的跨应用基础能力提前进入 shared。
4. 是否需要在组件 PR 模板中增加本文件的 review checklist。

