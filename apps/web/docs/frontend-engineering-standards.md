# 前端生产级工程规范

版本：1.0  
适用范围：`apps/web`  
技术基线：React 19、TypeScript strict、Vite、React Router、TanStack Query、Zustand、Tailwind CSS、Vitest、Testing Library、Playwright

本文是可执行的前端工程参考。关键词 MUST、MUST NOT、SHOULD、MAY 分别表示必须、禁止、应当和可选。代码审查与 AI 生成代码都按这些等级执行。

## 1. 质量目标

前端必须具备以下属性：业务模块可独立理解和演进；跨模块依赖显式；远端数据只有一个事实源；副作用可取消和清理；接口失败可恢复；默认安全与可访问；生产构建可复现。

“复用”指稳定能力通过小接口服务多个真实调用方，不是把相似代码提前塞进通用目录。“低耦合”指调用方只依赖模块公开接口，不知道其请求、缓存、状态和组件内部结构。

## 2. 目录与模块边界

```text
src/
  app/                    # Router、Provider、全局错误边界、启动配置
  features/
    chat/
      api/                # DTO、请求与 query/mutation options
      model/              # feature 状态、纯业务转换
      ui/                 # feature 私有展示/容器组件
      routes/             # 路由入口
      internal/           # 绝不跨 feature 导入
      index.ts             # 唯一公开入口
  shared/
    api/                  # HTTP/SSE 基础适配器，不含具体业务
    ui/                   # 无业务偏向、可访问的设计系统组件
    lib/                  # 纯函数与平台适配
    config/               # 已校验的公开运行时配置
  styles/
  test/
```

当前仓库仍使用 `src/lib`、`src/types`；新功能按上述结构演进。迁移须随业务变更逐步进行，不得仅为改目录制造大范围 diff。

允许的依赖：

```text
app ──> feature public interface ──> shared
                │
                └──> feature internal implementation
```

强制规则：

- `shared` MUST NOT 导入 `features` 或 `app`。
- feature MUST NOT 深层导入其他 feature；只能从其 `index.ts` 导入。
- feature MUST NOT 共享可变 store。跨 feature 协调由路由、应用组合层或明确领域事件完成。
- `shared/ui` MUST NOT 包含工作空间、知识库、聊天等业务词和权限判断。
- 禁止循环依赖；禁止巨型 barrel 导出所有内部实现。
- 模块公开接口应稳定、少而深；内部文件可自由重构而不影响调用方。

### 2.1 抽象与复用准入

提取公共模块前必须回答：调用方是否至少有两个；语义和生命周期是否一致；变化原因是否相同；抽象是否减少调用方必须知道的概念。任一答案为否时，优先保留局部重复。

不得创建：只转发参数的 wrapper；以大量布尔参数控制完全不同布局的组件；承载无关函数的 `utils.ts`；包含所有请求的全局 `api.ts`；包含所有页面状态的全局 store。

## 3. 组件设计

- 页面/路由组件负责组合和数据边界；展示组件通过 props 接收已准备的数据与回调。
- 一个组件只承担一个主要变化原因。超过约 200 行或同时处理请求、权限、表单、布局和导航时必须评估拆分，但行数不是机械指标。
- props 表达领域意图，避免透传底层实现配置。互斥状态使用 discriminated union，不使用多个可能冲突的布尔值。
- 列表 key 必须稳定，禁止使用可重排数组索引。
- render 必须纯；禁止渲染期写 store、发请求、改 DOM 或读写持久化存储。
- ref 只用于焦点、测量和必要的第三方适配，不作为常规数据通道。
- 错误边界必须提供可恢复入口，并记录去敏后的关联 ID。

## 4. 状态所有权

| 状态 | 唯一归属 | 禁止做法 |
| --- | --- | --- |
| 服务端列表、详情、统计 | TanStack Query | 再复制到 Zustand/useState |
| 当前流式草稿、流连接状态 | chat feature 的 Zustand store | 与已持久化历史混为一体 |
| 表单输入、展开状态 | 组件或表单 hook | 无理由提升为全局状态 |
| 分页、筛选、选中标签 | URL search params | 只放内存导致刷新丢失 |
| 会话/身份 | 鉴权模块内存 + 后端 session | localStorage 保存敏感令牌 |

Query key 必须由工厂集中生成，并包含 `workspaceId`、资源 ID、分页、筛选和排序。mutation 后只失效或精确更新受影响缓存。乐观更新必须具备回滚快照和并发冲突策略；没有可靠策略时使用服务端确认后更新。

Zustand store 必须按 feature 建立，导出细粒度 selector，禁止组件订阅整个 store。action 表达业务动作，不暴露任意 `setState`。持久化必须列出字段白名单和版本迁移，敏感数据禁止持久化。

## 5. 数据访问与契约

- 公开接口以 `/api/v1` 为基址；DTO 与 OpenAPI 保持一致，优先生成类型。
- `shared/api` 负责 base URL、凭据、请求 ID、JSON 解码、错误归一化、取消和超时；feature API 负责业务 endpoint 与 DTO 映射。
- 组件不得直接处理 HTTP 状态码。适配层把响应映射成稳定的 typed error，UI 再按业务错误码展示。
- 对 `response.json()`、URL 参数、localStorage、postMessage 和第三方 SDK 返回值进行运行时验证。TypeScript 类型不能替代运行时验证。
- 只对安全且幂等的请求做有限重试，并加入退避/抖动。创建、上传、提问超时结果未知时不得自动重放。
- AbortSignal 必须贯穿 Query、HTTP 和 SSE。身份或 workspace 变化时取消旧作用域请求。

### 5.1 SSE

SSE 解析器作为独立深模块维护，并覆盖：UTF-8 跨块、CRLF/LF、空行分隔、多行 `data`、多个事件同块、半包、未知事件、递增 seq、error、done、EOF 无终态。一次 `ReadableStream.read()` 绝不等于一个事件。

状态机至少为 `idle -> connecting -> streaming -> completed|failed|interrupted|cancelled`。只有 `done` 可进入 completed；EOF 无 done 为 interrupted。重复或倒退 seq 必须按契约处理并记录。组件只消费已解析事件，不拼字符串协议。

## 6. React 与副作用

- effect 仅同步外部系统。可由 props/state 计算的数据直接计算；用户动作放事件处理器；服务端请求交给 Query。
- 每个订阅、timer、observer、事件监听和流都必须返回清理函数。
- effect 在 Strict Mode 重复执行时必须安全。禁止用 ref 标记绕过正确清理。
- hook 名以 `use` 开头且遵守 Hooks 规则；自定义 hook 隐藏完整行为，不只把几行代码换个名字。
- 延迟加载以路由或大功能边界为单位；不得为细碎组件制造瀑布式 chunk。

## 7. TypeScript 规则

- 所有代码通过 strict、noUnusedLocals、noUnusedParameters。
- 使用 `unknown` 接收边界数据，经 narrowing 后使用。禁止 `any`、`as unknown as T`、无说明 `!` 和 `@ts-ignore`。
- `@ts-expect-error` 仅用于已知编译器/第三方类型缺陷，并附问题链接和删除条件。
- 使用 discriminated union 表达加载/错误/成功、事件和领域状态；禁止靠可选字段猜状态。
- 领域 ID 可使用 branded type 防止混用。时间、字节、百分比等单位体现在名称或类型中。
- 对外导出函数注明返回类型；局部变量优先推断。类型放在所有者模块，不建立全局类型垃圾场。

## 8. 样式、响应式与可访问性

- 颜色、字号、间距、圆角、阴影和层级来自设计 token。禁止随意 magic value 和无限增长的 z-index。
- 组件必须在 320px 宽度、文本放大 200%、键盘模式及长中文/英文内容下可用。
- 优先原生语义元素。可点击 div、仅颜色表达状态、无 label 输入框、无 alt 信息图均禁止合并。
- 焦点可见；弹窗打开聚焦合理位置，关闭后归还触发点；焦点不得被困或丢失。
- 动画尊重 `prefers-reduced-motion`；自动更新不得造成布局跳动。
- 图表提供文本摘要/表格替代，颜色不是唯一编码。ECharts wrapper 负责初始化、resize、dispose 和主题。
- 目标基线为 WCAG 2.2 AA；关键旅程要有自动检查与人工键盘检查。

## 9. 前端安全

- React 默认转义文本；使用 `dangerouslySetInnerHTML` 必须经批准的 sanitizer、协议白名单和专项测试。
- Markdown 禁止原始 HTML，外链增加安全属性并校验 scheme。模型输出与文档内容始终是不可信内容。
- Access token 仅存内存；refresh credential 使用 HttpOnly/Secure/SameSite Cookie。不得在 URL、日志、错误监控或 localStorage 中放秘密。
- CSRF 按后端契约携带 token/header；敏感 mutation 校验 Origin。前端不自行发明认证方案。
- 上传前检查大小、扩展名和 MIME 只改善体验；服务端必须重新验证。
- 禁止把内部 Agent ID、未授权引用、堆栈和内部错误详情展示给用户。
- 第三方脚本和依赖最小化；锁文件必须提交，生产不得使用浮动版本。

## 10. 性能预算

- 路由级代码拆分；大型图表和编辑器按需加载。
- 禁止前端拉全量日志做聚合。分页、筛选与统计由服务端完成。
- 避免无依据 memo。先测量 React Profiler/浏览器性能，再优化。
- 长列表采用分页或虚拟化；图片限定尺寸并延迟加载；大文件不得完整读入内存预览。
- 新增依赖必须记录 gzip 体积影响和已有能力为何不足。重大体积增长需在 PR 中说明。

## 11. 测试策略

测试金字塔：纯函数/状态机单元测试最多；feature 集成测试覆盖真实用户行为；少量 Playwright 覆盖跨层关键旅程。

必须覆盖：

- 鉴权失效、workspace 切换和缓存隔离。
- loading/empty/error/success/disabled 与重试。
- 表单校验、重复提交和服务端字段错误。
- SSE 正常终态、协议异常、断流、取消和并发保护。
- 权限不可见仅为 UX，403 仍正确处理。
- 键盘导航、可访问名称和焦点恢复。

使用 role、label 和可见文本查询 DOM。禁止依赖类名、私有 state 或组件内部函数。网络通过边界 mock；测试必须确定性，不用任意 sleep。E2E 数据独立、可重复执行、失败保留 trace。

## 12. 交付门禁与审查清单

```powershell
npm ci
npm run typecheck
npm test
npm run build
npm run test:e2e   # 用户旅程、路由、鉴权或网络协议变化时
```

审查者必须确认：模块归属正确；没有跨 feature 深层导入；没有重复远端状态；边界有验证和取消；错误与空状态完整；无敏感持久化/日志；可访问性满足；测试验证行为；包体变化合理；文档和 OpenAPI 同步。

## 13. 例外流程

任何违反 MUST 的临时例外必须写 ADR/issue，记录负责人、原因、风险、补偿控制和到期日期。永久“临时例外”视为缺陷。代码注释不能代替例外记录。

## 14. 依据

本规范结合仓库架构和官方资料制定。详细链接与采用理由见仓库根目录 `docs/engineering-standards-research.md`，包括 React 组件纯度、状态与 Effect，TypeScript strict、OWASP 前端安全、WCAG、OpenSSF 与 SLSA 资料。
