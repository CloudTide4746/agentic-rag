# Go 后端生产级工程规范

版本：1.0  
适用范围：`apps/api`  
技术基线：Go、Gin、GORM、MySQL、Redis、RabbitMQ、Zap；内部调用现有 Python Agent

本文定义可执行的模块化单体与分层约束。MUST、MUST NOT、SHOULD、MAY 分别表示必须、禁止、应当和可选。

## 1. 架构目标

系统必须让业务规则脱离框架独立测试，让数据库、缓存、队列和 Agent 可通过适配器替换，让模块独立演进，并让所有副作用可追踪、可重试、可对账。

采用按业务能力组织的模块化单体，不按全局技术层建立巨型 `handlers/services/repositories`。业务模块示例：`auth`、`workspace`、`knowledge`、`chat`、`ingestion`、`analytics`。

## 2. 每个模块的分层

```text
internal/chat/
  domain/          # 实体、值对象、规则、领域错误、必要端口
  application/     # command/query/use case、授权、事务协调
  transport/http/  # Gin handler、DTO、response mapper
  infrastructure/  # GORM repo、Agent client adapter、Redis adapter
  module.go        # 对外应用接口与装配所需定义
```

允许的编译期依赖：

```text
transport ──> application ──> domain
infrastructure ─────────────> domain/application ports
cmd/api ──> all concrete adapters for composition only
```

### 2.1 Domain

Domain MUST 是纯 Go：不导入 Gin、GORM、Redis、AMQP、Zap、HTTP 客户端或配置包。实体维护不变量，值对象在构造时验证。状态变化通过有意图的方法完成，禁止任意 setter。领域错误可由 `errors.Is/As` 判断。

只有业务规则需要的 repository/clock/id generator 等端口才定义在内层，接口放在使用方。DTO、数据库 model 与 domain model 不得共用一个 struct。

### 2.2 Application

Application 每个用例有明确输入/输出，负责授权、幂等、事务边界和跨端口编排。它不解析 HTTP，不返回 Gin response，不拼 SQL，不依赖具体 GORM repository。

读用例与写用例分离到足以清晰表达性能和事务语义即可，不为形式引入复杂 CQRS。跨模块调用对方公开 application facade；需要最终一致性时使用持久化事件/outbox。

### 2.3 Transport

Handler 只做：解析 path/query/header/body；大小限制；结构校验；从认证中间件取 principal；调用一个用例；映射结果/错误。禁止写业务规则、开启事务、直接访问 DB/Redis/Agent 或在 handler 启动无主 goroutine。

请求 DTO 使用明确 validation；未知字段按契约拒绝或兼容；body 有上限。响应只暴露公开 DTO，不序列化数据库/domain 对象。错误返回稳定 code、用户可理解 message 与 request_id，不泄露内部 cause。

### 2.4 Infrastructure

适配器封装供应商细节和观测信息。GORM model、索引提示、Redis key、AMQP header、Agent 旧协议都停留在此层。适配器必须把外部错误分类为内层可判断错误，同时保留 cause 供日志记录。

`platform` 只容纳路由装配、配置、日志、连接生命周期、通用中间件等横切基础设施，禁止把业务逻辑丢进 `platform`。

## 3. 模块解耦与复用

- 一个模块拥有自己的业务表和写入规则。其他模块不得直接引用其 GORM model 或 repository。
- 跨模块同步查询使用小型 facade；跨模块副作用优先使用带 schema version 的领域/集成事件。
- 共享代码只有在语义、生命周期和变化原因一致时才进入 `internal/shared`。共享目录不得成为杂物箱。
- 接口保持最小；调用方不应了解分页 SQL、缓存 key、Agent URL 或队列拓扑。
- 只有真实变化点才建立 seam：生产适配器与测试替身、两个生产实现或明确外部边界。
- 禁止循环依赖、service locator、全局 DB、隐式 init 副作用和跨层 helper。

## 4. 配置与进程生命周期

- 配置由环境注入，在启动时一次性读取、解析和校验；缺少必要值立即失败。禁止运行中散落 `os.Getenv`。
- 秘密不得提交、打印或出现在 panic；`.env.example` 仅含占位值。
- `cmd/api`、`cmd/worker` 是 composition root。依赖构造顺序显式，构造失败立即退出。
- 服务实现 SIGTERM/SIGINT 优雅关闭：先停止接收、标记不就绪、取消根 context、等待在途工作、限时关闭依赖。
- liveness 只判断进程活性；readiness 检查提供服务所需的关键依赖，不把暂时依赖抖动变成进程重启风暴。

## 5. Context、并发与资源

- 数据库、缓存、队列、HTTP 与长任务的入口必须接收并传播 `context.Context`；不得存入 struct 或传 nil。
- 每个外部调用有截止时间。普通 HTTP 与 SSE/模型调用使用不同预算；子调用预算不得超过父级。
- goroutine 必须有 owner、取消、退出和 join 机制；禁止 fire-and-forget。并发上限必须显式，禁止无界 goroutine/队列。
- 通道由发送方关闭；接收方不关闭；发送必须可响应 context 取消。
- ticker、response body、rows、文件和 AMQP delivery 必须释放。defer 前检查资源创建是否成功。
- 共享内存必须同步或保持不可变；并发变更运行 `go test -race ./...`。

## 6. 错误模型

稳定错误分类建议：validation、unauthenticated、forbidden、not_found、conflict、rate_limited、dependency_unavailable、timeout、internal。

- 使用 `%w` 包装 cause，使用 `errors.Is/As` 判断；禁止依赖字符串匹配。
- 一个错误只在有处理决策的层记录一次。底层返回上下文，边界统一记录，避免重复日志。
- 正常业务失败不用 panic。recover 只作为进程保护并产生关联 ID，不替代错误处理。
- 客户端 message 不含 SQL、路径、栈、供应商响应或内部 ID。详细 cause 进入去敏日志。
- SSE 发送 header 前可用标准 JSON 错误；发送后以 error 事件终止，并持久化运行终态。

## 7. 数据库与事务

- Repository 的每个查询都接收作用域对象（至少 workspace/owner）并使用 `WithContext`。禁止先按 ID 查出再在业务层补租户判断。
- 数据库约束保护唯一性、外键、非空、状态和幂等；应用校验用于友好错误，不能替代约束。
- 事务由 application use case 控制。事务短小，不包含远程 HTTP、模型调用、队列 publish 或文件解析。
- 防止 N+1；列表字段、排序列白名单；page_size 默认 20、上限 100；大结果使用游标或稳定分页。
- 禁止字符串拼接用户输入到 SQL。原生 SQL 必须参数化并有 repository 测试。
- schema 只通过编号 migration 修改；禁止生产 AutoMigrate。迁移必须可重复部署、考虑锁表、回填和向后兼容。
- 高风险迁移使用 expand/migrate/contract：先兼容新增，再回填切流，最后删除旧结构。down migration 会丢数据时必须明确，不伪造安全回滚。

## 8. 一致性、Outbox 与消息

数据库状态与消息发布需要原子一致时，必须同事务写业务记录和 outbox。发布器 claim 记录、发布、等待 broker confirm，再标记完成；失败可重试。

消费者采用至少一次语义：

- message 带 `event_id`、`job_id`、`workspace_id`、`schema_version`、`trace_id`、`created_at`。
- 消费前通过幂等表/业务唯一约束判重；业务副作用和幂等标记尽可能同事务。
- 重试区分 transient/permanent/unknown outcome，指数退避加抖动，有最大次数与 DLQ。
- 外部 Agent 已成功但 ACK 前崩溃时，必须能用 job/idempotency key 查询并复用结果。
- handler 在持久化完成后 ACK；panic/进程退出不能吞消息；消息正文不放文件或秘密。
- schema 只做兼容演进；破坏性变化使用新版本并支持滚动升级窗口。

## 9. HTTP 与公开契约

- OpenAPI 是公开契约的源之一，代码变更必须同步。接口前缀 `/api/v1`，JSON 使用 snake_case，时间 UTC RFC3339，ID 对外为字符串。
- GET/PUT/DELETE 的幂等语义明确；创建类接口支持幂等键时定义作用域、TTL、请求指纹与冲突行为。
- 所有列表有分页与稳定排序；所有写操作有大小限制；Content-Type/Accept 显式验证。
- CORS 使用明确 allowlist；不得在带凭据时使用 `*`。反向代理信任列表显式设置。
- 超时分层：server read header、read body、idle、普通 handler、依赖调用分别配置；流式响应不能套普通写超时。
- 限流按身份/workspace/资源成本设计；登录、上传、问答和管理接口使用不同策略。Redis 故障时的 fail-open/fail-closed 行为必须记录。

## 10. 认证、授权与多租户隔离

- 认证只证明身份；每个资源操作仍执行授权。默认拒绝，权限检查在 application 层集中完成。
- 先验证 workspace membership，再验证资源归属与动作权限。管理员读取员工正文必须有独立权限。
- 浏览器只提交平台资源 ID；Go 将其转换为已授权 Agent ID。空 `kb_ids` 立即拒绝，绝不转换为 nil/null。
- Access JWT 短期有效，验证签名算法、issuer、audience、exp、nbf；拒绝算法降级。刷新凭据只存哈希、可轮换、可撤销、检测重用。
- Cookie 使用 HttpOnly、Secure、合适 SameSite、窄 Path/Domain；状态变更结合 CSRF token 与 Origin 校验。
- 密码使用当前批准的慢哈希参数；登录错误不枚举账号；敏感动作审计但不记录秘密。
- 当前 Python Agent 未证明多租户隔离，因此不得宣称多租户生产安全；上线前必须完成全链路隔离测试或部署级隔离。

## 11. 外部 Agent 与 SSRF 防护

- Agent URL 来自启动配置和固定 allowlist，用户不得控制 scheme/host/port/path。禁止自动跟随到非允许目标的重定向。
- 使用复用 Transport 的单例 HTTP client；设置 dial、TLS、response header、idle connection 和调用预算。
- 请求只传授权后的最小数据；内部服务凭据轮换并避免日志记录。
- 响应大小设限，JSON/SSE 严格解析；未知字段按版本策略处理，未知事件记录指标但不得导致权限绕过。
- 普通幂等读可有限重试；提问、上传和创建结果未知时进入对账，不得盲重试。
- Python 的错误、模型输出和引用均为不可信输入；公开前验证 document/kb 映射和权限。

## 12. 文件上传与内容处理

- 同时限制请求总大小、单文件大小、文件数、扩展名和检测后的 MIME；不能只信 Content-Type 或文件名。
- 存储名使用随机 ID，原文件名仅作元数据并清洗；防止路径穿越、覆盖、压缩炸弹和超大解析结果。
- 上传先进入隔离区；解析/扫描完成后再进入可用状态。下载强制鉴权并设置安全响应头。
- 解析器在受限资源和超时下运行；不执行宏、脚本或外部引用。失败时清理临时文件并保留可审计状态。

## 13. 缓存规则

- MySQL 是业务事实源；Redis 不作为唯一持久化事实。
- key 包含环境、模块、workspace、资源和 schema version，设置 TTL；禁止不同租户共享无作用域 key。
- cache-aside 必须定义失效顺序、陈旧容忍度和穿透/击穿保护。权限与撤销数据优先一致性。
- 缓存反序列化失败视为 miss 并记录指标，不得 panic；缓存内容同样按不可信数据处理。

## 14. 可观测性与审计

- 每个请求/任务贯穿 request_id、trace_id、run_id/job_id；日志为结构化字段，不拼接长文本。
- 记录操作名、稳定错误码、状态、耗时、重试次数和作用域化资源 ID。禁止记录密码、token、Cookie、完整问题/答案/文档、DSN 和模型密钥。
- 指标至少覆盖 RED（rate/errors/duration）、在途请求、SSE 活动数、依赖延迟、连接池、队列深度、重试/DLQ 和任务阶段。
- trace 跨 HTTP、数据库和消息传播；采样不得破坏错误可诊断性。高基数字段不能作为 metric label。
- 审计日志记录谁在何时对哪个作用域执行何动作及结果；只追加、限制访问并设保留期。

## 15. 测试策略

| 层 | 测试重点 | 不允许 |
| --- | --- | --- |
| domain | 不变量、状态机、值对象 | mock 框架依赖 |
| application | 权限、事务、幂等、端口交互 | 真实网络 |
| repository | SQL、作用域、约束、事务 | SQLite 冒充 MySQL 语义 |
| transport | 状态码、DTO、错误、大小限制 | 仅测试 Gin 实现细节 |
| contract | OpenAPI、Agent/SSE/消息 schema | 手写样例与代码漂移 |
| end-to-end | 登录→上传→入库→问答→引用 | 共享不可重置数据 |

必须包含负向和故障测试：越权/跨 workspace、空权限集、重复请求、重复消息、事务回滚、超时、取消、Agent 成功但响应丢失、SSE 无 done、队列重投、DLQ、优雅关闭与依赖恢复。时间、随机数与 ID 通过小端口控制，测试不依赖任意 sleep。

## 16. 性能与容量

- 优化前建立基线，记录硬件、数据量、并发、P50/P95/P99、吞吐和错误率。
- 数据库连接池、HTTP Transport、worker 并发、消息 prefetch 和 SSE 并发必须有上限并与下游容量匹配。
- 避免每 token 日志/数据库写入；流式转发批量与刷新延迟需测量。
- 所有无界输入、列表、缓存、队列和内存 buffer 必须设置上限或背压。
- 性能回归不能通过放宽超时或删除正确性检查掩盖。

## 17. 交付门禁

```powershell
gofmt -w <changed-go-files>
go vet ./...
go test ./...
go test -race ./...   # 并发、缓存、worker、SSE 或生命周期变化时
go build ./cmd/api ./cmd/worker
```

数据库变更还必须在空库和上一版本数据副本上验证 up migration，并验证应用滚动兼容；接口变更校验 OpenAPI；消息变更执行新旧 producer/consumer 兼容测试；安全关键变更包含越权与滥用测试。

审查者必须确认：依赖方向正确；数据所有者唯一；作用域在查询层强制；事务不含远程调用；幂等与未知结果有方案；context/资源正确释放；错误不泄露；日志去敏；契约/迁移/测试同步；没有假完成声明。

## 18. 例外和架构决策

偏离分层、跨模块直连、共享数据库写入、同步跨服务事务、新中间件或安全模型变化必须写 ADR。ADR 记录背景、决定、替代方案、数据与依赖方向、安全影响、故障模式、迁移/回滚、验证和复审日期。带到期日的例外到期后自动视为阻断缺陷。

## 19. 依据

本规范结合仓库现有总体设计和官方一手资料制定。来源与采用理由见仓库根目录 `docs/engineering-standards-research.md`，覆盖 Go Code Review Comments、Go context/database/sql、OWASP ASVS/API Security、OpenTelemetry、Twelve-Factor、OpenSSF 与 SLSA。
