# Go API AI 开发约束

本文件适用于 `apps/api/**`。开始任何后端工作前，必须同时读取仓库根 `AGENTS.md` 与 `docs/backend-engineering-standards.md`。

## 1. 分层与依赖方向

后端采用按业务模块组织的模块化单体。每个模块内部使用以下职责：

`transport -> application -> domain <- infrastructure`

- transport：Gin/HTTP/SSE/消息入口，只做解析、认证上下文提取、校验、调用应用用例和响应映射。
- application：用例编排、授权、事务边界、幂等协调；依赖领域定义的端口。
- domain：实体、值对象、领域规则、领域错误和必要端口；必须是纯 Go，不依赖 Gin、GORM、Redis、RabbitMQ、Zap 或供应商 SDK。
- infrastructure：GORM repository、Redis、RabbitMQ、Agent HTTP client 等适配器；实现内层端口。

MUST 保持依赖只向内。禁止 Handler 直连 GORM/Redis/Agent，禁止领域类型携带 JSON/GORM/HTTP 框架语义，禁止跨模块直接访问对方表或 repository。

## 2. 模块与接口约束

- 业务模块位于 `internal/<business>`；`platform` 仅做进程组合和横切适配，不成为业务杂物箱。
- 跨模块调用通过小而稳定的应用接口或领域事件；不得导入对方 infrastructure。
- 接口定义在使用方，并保持最小。只有存在真实替换点、外部边界或测试替身需求时才创建接口。
- 构造函数显式注入依赖；禁止包级可变状态、service locator 和在业务函数中创建数据库/HTTP 客户端。
- `cmd/api`、`cmd/worker` 是 composition root，只负责配置、装配、生命周期与优雅关闭。

## 3. 数据、安全与可靠性

- 所有资源查询必须携带 workspace/owner 范围；先校验成员关系，再校验资源权限。空授权集合必须拒绝，不得退化成全量查询。
- Go 只写 platform 数据；不得直接读取或修改 Python Agent 表。
- 事务在 application 层定义，保持短小；外部 HTTP/队列调用不得位于数据库事务内。需要原子发布时使用 transactional outbox。
- 创建、上传、消费与外部副作用必须定义幂等键、唯一约束、重试分类和结果未知时的对账路径。
- SQL 参数化；排序字段白名单；分页有上限；生产结构变化只通过编号 migration，禁止 AutoMigrate。
- 密码、令牌、Cookie、上传、URL、模型输出和错误信息按详细规范处理。日志禁止包含秘密或默认记录完整业务正文。

## 4. Go 代码硬约束

- 所有可能阻塞的调用第一个参数传 `context.Context`，并传播取消和截止时间。
- 错误必须保留因果链并映射为稳定业务错误；禁止字符串匹配错误、吞错、panic 处理普通失败或泄露内部错误。
- goroutine 必须有所有者、退出条件、取消路径和回收等待；通道关闭权属于发送方。
- HTTP 客户端必须复用并配置连接、响应头与整体超时；SSE 使用独立的长时限策略。
- 时间内部使用 UTC；外部时间为 RFC3339；ID、金额、页码与大小显式建模。
- 任何权限判断、状态迁移、幂等判断和数据不变量必须由数据库约束与测试共同保护。

## 5. 契约、测试与门禁

- 接口实现前先更新 `api/openapi.yaml`；响应、状态码、错误码、权限与示例必须一致。SSE 事件需独立 schema/示例。
- 单元测试覆盖领域和应用规则；repository 用真实数据库集成测试；HTTP 用 `httptest`；外部 Agent/队列通过端口替身测试；关键竞争路径运行 race detector。
- 修复缺陷必须包含回归测试。权限隔离、事务回滚、重复投递、取消、超时、SSE 断流与优雅关闭为强制场景。
- 完成前执行 `gofmt`、`go vet ./...`、`go test ./...`；并发或异步变更执行 `go test -race ./...`。

详细分层模板、错误/事务/安全/可观测性规则见 `docs/backend-engineering-standards.md`。
