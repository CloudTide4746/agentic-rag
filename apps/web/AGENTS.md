# Web 前端 AI 开发约束

本文件适用于 `apps/web/**`。开始任何前端工作前，必须同时读取仓库根 `AGENTS.md` 与 `docs/frontend-engineering-standards.md`。

## 1. 架构硬约束

- MUST 按业务能力放入 `src/features/<feature>`，而不是按 React 技术类型堆成全局 `hooks/`、`services/`、`pages/`。
- MUST 保持依赖方向：`app -> features -> shared`。feature 之间禁止深层导入；跨 feature 复用只能通过显式公开入口，或下沉为无业务偏向的 shared 模块。
- MUST 让 `app` 只负责组合：路由、Provider、错误边界、全局布局和启动配置。业务规则不得写进 `App.tsx`。
- MUST 区分状态所有权：远端状态归 TanStack Query；流式临时交互归 Zustand；组件私有状态归组件；可分享筛选归 URL。
- MUST 使用 TypeScript strict。禁止 `any`、双重断言、`@ts-ignore`、未说明的非空断言；边界数据必须运行时校验或由可信生成器产生类型。
- MUST 将 HTTP/SSE、时间、存储、浏览器 API 放在适配层；展示组件不得直接调用 `fetch`、读环境变量或操作全局存储。
- MUST 保持组件可组合且职责单一。禁止“万能组件”、布尔参数爆炸和复制粘贴业务规则。

## 2. 模块公开面

每个 feature 应以 `index.ts` 作为对外入口，只导出真正供外部使用的页面、组件、类型或工厂。禁止从其他 feature 导入其 `internal`、私有 hooks、测试工具或具体请求实现。循环依赖为构建阻断问题。

新增抽象必须通过复用门槛：至少两个已存在且语义一致的调用方，或一个明确的外部边界需要测试替身。仅代码长得相似不能证明应抽象。

## 3. 数据与副作用

- Query key 必须包含工作空间/用户作用域与全部查询条件；切换身份或工作空间时取消旧请求并清除敏感缓存。
- mutation 成功后的缓存更新必须集中在 feature 数据层，不得散落于展示组件。
- effect 只用于与 React 外部系统同步，必须可清理、可重复执行；禁止用 effect 派生可在渲染期计算的数据。
- 请求必须支持取消；组件卸载后不得继续写状态。计费、上传和创建类动作不得在挂载 effect 中自动触发。
- SSE 必须正确处理半包、多事件、CRLF、多行 data、Unicode、未知事件、`done`、`error` 与无终态 EOF。

## 4. 安全与可访问性

- 禁止在浏览器持久化刷新令牌或敏感正文；禁止输出令牌、完整问题、答案或文档到日志。
- 禁止未经清洗渲染 HTML；链接协议必须白名单；文件名不得直接拼路径。
- 权限由后端强制，前端只能改善体验，不能把隐藏按钮当授权。
- 所有交互必须支持键盘；使用语义 HTML；表单具备 label、错误关联和焦点处理；动态状态通过合适的 live region 告知。
- loading、empty、error、disabled、success 与断网/重试状态必须明确，不得只实现成功路径。

## 5. 测试与门禁

- 纯函数和状态转换做单元测试；组件从用户行为与可访问语义测试；接口边界使用可控适配器；关键旅程使用 Playwright。
- 禁止测试实现细节、快照替代行为断言、任意 sleep、真实生产服务或仅为测试暴露私有实现。
- 修复缺陷必须先有可失败的回归测试；权限、缓存隔离、SSE 终态与错误恢复属于强制测试。
- 完成前执行 `npm run typecheck && npm test && npm run build`，用户旅程变化执行 `npm run test:e2e`。

详细目录、接口和审查清单见 `docs/frontend-engineering-standards.md`。
