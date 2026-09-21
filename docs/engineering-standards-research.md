# 前后端生产级开发规范：官方资料研究基线

> 适用仓库：`apps/web`（React 19、TypeScript、Vite）与 `apps/api`（Go、Gin）。  
> 用途：供 `AGENTS.md` 及前后端详细规范引用。本文件只采纳项目官方文档、OWASP、OpenSSF/SLSA、OpenTelemetry、Twelve-Factor 等一手资料。  
> 规范词：`MUST`/必须是合并门禁；`SHOULD`/应当偏离时必须记录理由；`MAY`/可以是可选实践。

## 1. 总体结论

生产级约束需要同时覆盖代码边界和自动化门禁。目录结构本身不能保证低耦合；必须明确依赖方向、公共 API、状态所有权、外部副作用边界，并用静态检查、测试、构建、安全扫描和部署验证阻止违规变更合入。

本仓库建议采用以下边界：

```text
Web: app -> features -> components/lib/types
     feature A -X-> feature B 内部实现（跨 feature 只能依赖其公开入口）

API: transport(handler) -> application(service/use-case) -> domain
                                      |                    ^
                                      v                    |
                              ports/interfaces <- infrastructure adapters
```

依赖箭头指向被依赖方。后端 domain/application 不得导入 Gin、GORM、Redis、RabbitMQ 或具体 Agent SDK；适配器实现由使用方定义的窄接口，在 `cmd/*` 组合根注入。

## 2. 前端：模块化、高复用、低耦合

### 2.1 模块与依赖边界

- MUST 按业务能力组织 `features/auth|workspace|knowledge|chat|ingestion|analytics`；每个 feature 可含 `api/`、`components/`、`hooks/`、`model/`、`routes/`、`tests/`，并用单一公开入口导出稳定 API。
- MUST 禁止跨 feature 深层导入；跨模块只允许导入公开入口及稳定 ID/DTO。通用组件不得反向导入业务 feature。
- MUST 保持 `app/` 只负责启动、路由、Provider、错误边界和组合；不得把业务规则堆入全局 Provider。
- MUST 先在真实调用处证明复用，再提升到 `components/` 或 `lib/`；禁止以 `utils.ts`、`helpers.ts`、`common.ts` 形成无边界杂物包。
- SHOULD 将视觉复用放入无业务知识的 UI 组件，将行为复用放入聚焦具体用例的自定义 Hook。React 官方明确建议 Hook 聚焦具体高层用例，避免 `useMount` 一类生命周期包装器：[Reusing Logic with Custom Hooks](https://react.dev/learn/reusing-logic-with-custom-hooks)。
- MUST 保持组件渲染纯净：渲染期不得写缓存、发请求、修改 props/state/context 或执行其他外部副作用；React 说明纯组件可安全重渲染、缓存及中断：[Keeping Components Pure](https://react.dev/learn/keeping-components-pure)。
- SHOULD 优先组合、小型专用组件和显式 props；抽象若不能缩小调用方能力、减少重复或稳定依赖边界，则不引入。

### 2.2 状态与副作用

- MUST 为每份状态指定唯一所有者；同一服务端数据不得同时复制到 TanStack Query、Zustand 和组件 state。React 官方要求每一份状态有单一事实来源：[Sharing State Between Components](https://react.dev/learn/sharing-state-between-components)。
- MUST 使用 TanStack Query 持有服务端状态，Zustand 仅持有跨组件客户端交互/流式草稿，`useState` 持有局部 UI 状态，URL 持有可分享的筛选、分页和导航状态。
- MUST 不存储可由 props/state 在渲染期推导的数据，不复制或深度嵌套状态；依据：[Choosing the State Structure](https://react.dev/learn/choosing-the-state-structure)。
- MUST 只用 Effect 同步 React 外部系统；派生数据、用户事件和状态联动不得滥用 Effect。依据：[You Might Not Need an Effect](https://react.dev/learn/you-might-not-need-an-effect)。
- MUST 为 Effect 清理订阅、定时器、网络请求和观察器；异步请求使用 `AbortSignal`，路由切换、工作空间切换和卸载时取消。
- MUST 将 Query Key 包含 `workspace_id`、资源 ID 和全部查询条件；身份退出/租户切换时取消进行中请求并删除敏感缓存。

### 2.3 TypeScript 与接口契约

- MUST 保持 `strict: true`；官方说明它启用更强的正确性检查：[TSConfig strict](https://www.typescriptlang.org/tsconfig/strict.html)。
- MUST 增开 `noUncheckedIndexedAccess`、`exactOptionalPropertyTypes`、`noImplicitReturns`、`noFallthroughCasesInSwitch`、`noImplicitOverride`；可查阅 [TSConfig Reference](https://www.typescriptlang.org/tsconfig/) 与 [exactOptionalPropertyTypes](https://www.typescriptlang.org/tsconfig/exactOptionalPropertyTypes.html)。
- MUST 禁止无理由 `any`、非空断言和双重类型断言；网络、localStorage、URL、`postMessage`、环境变量等运行时输入进入系统时必须验证，类型断言不能代替验证。
- MUST 由 OpenAPI/Schema 生成或集中定义 HTTP/SSE DTO；API DTO 不直接充当视图模型，转换发生在 feature 边界。
- MUST 对联合状态使用可辨识联合并穷尽处理；不得用多个可能互相矛盾的布尔值表示请求/任务状态。
- MUST 不从 barrel 导出 feature 私有实现；公共导出缩到调用者实际需要的最小面。

### 2.4 浏览器安全与可访问性

- MUST 把前端当作不可信客户端；权限判断、对象归属、字段级授权和 Agent `kb_ids` 均由后端执行。OWASP API Security 强调对象级授权风险：[API Security Top 10](https://owasp.org/API-Security/editions/2023/en/0x11-t10/)。
- MUST 禁止将访问令牌、刷新令牌、密钥或敏感文档写入 localStorage、日志、埋点、URL；刷新凭据使用符合部署模型的 Secure/HttpOnly/SameSite Cookie。
- MUST 禁止未经净化的 HTML；Markdown 渲染采用允许列表，外链设置安全属性，CSP 禁止 `unsafe-inline`/`unsafe-eval`（确需例外须威胁建模）。参考 [OWASP XSS Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html) 与 [Content Security Policy Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html)。
- MUST 对所有交互控件提供语义元素、键盘操作、可见焦点、可访问名称和错误关联；目标采用 [WCAG 2.2](https://www.w3.org/TR/WCAG22/) AA。
- MUST 为页面/组件实现 loading、empty、error、disabled 和 retry 状态；错误边界不得泄露堆栈、令牌或响应正文。

### 2.5 前端测试门禁

- MUST 测试业务行为与可访问结果，不断言组件内部实现；纯转换/状态机做单元测试，feature 做集成测试，核心旅程做 Playwright E2E。
- MUST 覆盖权限拒绝、缓存隔离、请求取消、重复提交、超时、错误恢复，以及 SSE 的 UTF-8 半包、CRLF、多行 data、多事件同块、无 `done` EOF。
- MUST 每个 PR 通过：类型检查、lint、单元/集成测试、生产构建；涉及核心用户旅程时通过 E2E。
- SHOULD 设变更覆盖率门禁，覆盖率数字不能替代关键风险场景断言。

## 3. 后端：模块化单体与严格分层

### 3.1 分层职责及依赖方向

- MUST 按业务模块组织 `internal/auth|workspace|knowledge|chat|ingestion|analytics`，而非建立全局巨型 `handlers/services/repositories/models` 横切目录。
- MUST 将层职责固定为：Handler/Transport 解析协议、校验形状、调用用例并映射响应；Application/Service 执行业务流程、授权、事务边界和幂等；Domain 表达实体、值对象、策略和领域错误；Repository/Client Adapter 处理持久化和外部协议。
- MUST 禁止 Handler 直接访问 GORM、Redis、RabbitMQ、Agent；禁止 Repository 做权限或业务流程；禁止 Domain 返回 HTTP 状态码、GORM model 或 Gin context。
- MUST 在消费方定义小接口，只暴露用例所需方法；实现方返回具体类型。Go 官方审查指南明确建议接口属于使用方且不要为了 mock 预先定义接口：[Go Code Review Comments — Interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)。
- MUST 只在 `cmd/api`、`cmd/worker` 组合依赖；业务包禁止服务定位器、可变全局单例及包级数据库客户端。
- MUST 防止模块环依赖和跨模块访问私有数据库模型；跨模块协作通过显式 application API、稳定 ID 或事件。
- MUST 禁止 `util/common/misc/interfaces` 万能包。Go 官方解释这类包会失去边界并积累依赖：[Package names](https://go.dev/blog/package-names)。

### 3.2 Go API、并发、错误与数据

- MUST 将 `context.Context` 作为需要它的方法第一个参数，从 HTTP/任务入口贯穿数据库、缓存、消息和 Agent 调用；不得保存在 struct 中。依据：[Go Code Review Comments — Contexts](https://go.dev/wiki/CodeReviewComments#contexts)。
- MUST 为所有外部 I/O 配置明确超时/截止时间并传播取消；创建派生 context 后 `defer cancel()`。依据：[Canceling in-progress operations](https://go.dev/doc/database/cancel-operations)。
- MUST 为每个 goroutine 指定所有者、退出条件、取消路径和错误回收；不得 fire-and-forget。Go 官方要求 goroutine 生命周期清晰：[Goroutine Lifetimes](https://go.dev/wiki/CodeReviewComments#goroutine-lifetimes)。
- MUST 处理每个错误；使用 `%w` 保留错误链；领域错误映射为稳定机器码，客户端消息不暴露内部错误、SQL、路径或上游正文。
- MUST 所有查询带租户/工作空间作用域；资源操作执行对象级授权。空授权集合必须拒绝，不得降级为不加过滤条件。
- MUST 参数化查询、排序字段允许列表、分页上限；事务只包含必需数据库操作，外部网络调用不得置于长事务中。
- MUST 使用编号、可审查、可回滚/前向修复的 migration；生产启动不得 AutoMigrate。
- MUST 在数据库用唯一键、外键和条件更新兜底不变量；应用层预检查不能替代约束。
- MUST 为创建、消费和外部副作用定义幂等键与结果语义；结果未知时进入 reconciliation，不得盲目重试非幂等操作。
- MUST 显式设置 HTTP server 的 read/header/write/idle 策略、请求体上限、优雅关闭；SSE 长请求采用独立的合理截止时间。
- MUST 调优并观测数据库连接池；`sql.DB` 是连接池，官方说明了最大连接/空闲/生命周期设置与 `DB.Stats`：[Managing connections](https://go.dev/doc/database/manage-connections)。

### 3.3 API 与文件安全

- MUST 仅通过 TLS 暴露生产接口，严格允许 HTTP 方法和内容类型，使用真实状态码；参考 [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html)。
- MUST 在每个端点执行认证、功能级授权、对象级授权和属性级授权；不得信任客户端传入的 workspace、owner、role、内部 Agent ID。
- MUST 对所有输入施加长度、格式、枚举、数量、嵌套深度和业务范围限制；分页、批量、上传、搜索和 AI 请求均有资源预算及速率限制。
- MUST 文件上传采用允许列表、文件签名/实际类型校验、随机存储名、路径隔离、容量限制和恶意内容处置；不得使用用户文件名拼接路径。参考 [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html)。
- MUST Cookie 状态变更请求执行 CSRF 防护与 Origin 校验；CORS 采用具体 origin 允许列表，凭据模式不得配 `*`。
- MUST 密码使用现代自适应哈希；令牌可撤销、轮换并校验 issuer/audience/expiry/algorithm。认证建议见 [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html) 和 [Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)。
- MUST 以 OWASP ASVS 作为发布验证清单；索引覆盖编码、业务逻辑、前端、API、文件、认证、会话、授权等：[OWASP ASVS Cheat Sheet Index](https://cheatsheetseries.owasp.org/IndexASVS.html)。

### 3.4 后端测试门禁

- MUST 对 domain/application 做表驱动测试；Repository/Agent/队列做真实依赖集成测试；Handler 做协议契约测试。
- MUST 覆盖越权、租户隔离、事务回滚、唯一键竞争、重复投递、超时/取消、上游部分成功、worker 崩溃恢复、优雅关闭。
- MUST 在 CI 运行 `go test ./...`、`go vet ./...`、格式检查与编译；并在适用平台运行 `go test -race ./...`。Go 官方 race detector 用法：[Data Race Detector](https://go.dev/doc/articles/race_detector)。
- SHOULD 对 SSE/JSON/文件名/解析器/权限过滤边界做原生 fuzz；Go 官方说明覆盖引导 fuzz 可发现人类遗漏的安全边界：[Go Fuzzing](https://go.dev/doc/security/fuzz/)。
- MUST 测试不得依赖真实公网、共享数据库、顺序或墙钟偶然性；随机性、时钟、ID 和外部端口必须可注入。

## 4. 可观测性与运行约束

- MUST 使用结构化日志，固定时间、级别、服务、环境、版本、request/trace/span、workspace（不可逆/非敏感标识）、route、operation、result、error_code、duration 字段。OpenTelemetry 推荐结构化日志并支持 trace/span 关联：[OpenTelemetry Logs](https://opentelemetry.io/docs/concepts/signals/logs/)。
- MUST 在 HTTP、队列和 Agent 边界传播 W3C Trace Context；日志、trace、metric 使用一致资源属性和语义约定。
- MUST 记录 RED 指标（请求率、错误率、耗时）及队列深度、重试、死信、运行中任务、连接池；业务指标不得使用用户 ID、问题文本等高基数/敏感标签。
- MUST 禁止日志记录密码、令牌、Cookie、Authorization、完整问题/答案/文档、上传正文和原始数据库错误；采用允许列表字段和集中脱敏。
- MUST 区分 liveness 与 readiness；readiness 验证服务能否接流量，liveness 不因可恢复下游短暂失败触发重启风暴。
- MUST 为关键流程定义 SLI/SLO、告警阈值和 runbook；告警基于用户影响及持续窗口，避免单点瞬时噪声。
- OpenTelemetry 将 traces、metrics、logs、baggage 定义为关联信号：[Signals](https://opentelemetry.io/docs/concepts/signals/)。

## 5. 配置、交付与供应链

- MUST 配置来自环境/密钥系统，代码、镜像和前端 bundle 不含秘密；同一构建产物跨环境晋级。Twelve-Factor 要求配置与代码分离：[Config](https://12factor.net/config)。
- MUST 进程无状态、可快速启动并优雅终止；持久状态进入后端服务，日志写事件流。参考 [Processes](https://12factor.net/processes)、[Disposability](https://12factor.net/disposability)、[Logs](https://12factor.net/logs)。
- MUST 提交并冻结 `package-lock.json`、`go.mod`、`go.sum`；CI 使用确定性安装，禁止 `latest` 镜像，生产镜像固定不可变 digest。
- MUST 每个 PR 做 npm/Go 漏洞检查；Go 使用 `govulncheck`，其基于可达调用降低噪声：[Go Vulnerability Management](https://go.dev/doc/security/vuln/)。
- MUST 使用最小运行镜像、非 root 用户、只读根文件系统（需要写的路径显式 volume/tmpfs）、删除构建工具与缓存、限制 Linux capabilities。
- MUST 生成并保存 SBOM、构建来源证明及签名；发布流水线向 [SLSA](https://slsa.dev/spec/v1.2/) 的隔离、可验证 provenance 要求演进。
- MUST 开启受保护分支、强制评审、最小权限 CI token、固定第三方 CI Action/工具版本；定期用 [OpenSSF Scorecard](https://scorecard.dev/) 检查危险工作流、分支保护、依赖更新、固定版本和发布实践。
- MUST 扫描源码秘密、依赖漏洞、容器漏洞和许可证；阻断存在可利用 critical/high 漏洞且无到期风险接受记录的发布。
- SHOULD 保存可复现构建所需的工具链版本、构建参数、SBOM、provenance、测试报告与镜像 digest，确保部署可追溯到审查过的提交。

## 6. 建议的硬性 CI 合并门禁

```text
web:
  clean install -> format/lint -> typecheck -> unit/integration -> build -> risk-based e2e
api:
  gofmt check -> go vet -> go test -> race test -> build -> govulncheck
cross-cutting:
  OpenAPI compatibility -> migrations validation -> secret scan -> dependency/container scan
  -> SBOM/provenance -> policy evaluation
```

任何门禁失败必须阻止合并。跳过检查只能通过有责任人、原因、风险、补救措施和到期日的书面例外；禁止在代码中长期保留 `nolint`、跳过测试、宽泛安全忽略规则而无对应记录。

## 7. 可直接转入 AGENTS.md 的执行约束

- 开始编码前，AI MUST 读取根 `AGENTS.md`、所在前端/后端 `AGENTS.md` 与对应 `/docs` 规范；冲突时以路径更近且更严格的规则为准，并报告冲突。
- AI MUST 先确定修改所属模块、依赖方向、数据/状态所有者和授权边界；无法满足边界时先写 ADR，不得用跨层调用赶工。
- AI MUST 保持最小变更，不顺手重构无关文件，不复制现有能力，不新建 `utils/common` 垃圾抽象。
- AI MUST 为行为变化补与风险匹配的测试并运行规定门禁；未运行或失败必须明确报告，禁止声称“已通过”。
- AI MUST 不削弱类型、安全、权限、租户过滤、超时、取消、幂等、日志脱敏和供应链门禁；任何临时例外必须显式、可搜索且有到期日。
- AI MUST 以实际源码、Schema 和测试为事实来源；文档与实现不一致时停止扩散错误，修正文档或提出明确差异。

## 8. 资料使用说明

以上链接均为规范制定者或项目官方维护的一手资料。React/Go/TypeScript 文档提供语言与框架事实；OWASP 提供安全验证基线；OpenTelemetry 提供遥测语义；Twelve-Factor 提供运行配置原则；OpenSSF/SLSA 提供供应链评估和来源证明框架。本文将其转换为本仓库的工程门禁，但具体威胁模型、SLO 数值、性能预算与漏洞修复 SLA 仍应由项目在 ADR/运行手册中量化。
