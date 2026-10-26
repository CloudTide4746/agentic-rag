# ADR-0004：模块 HTTP 处理器接管路由注册

- 状态：已采纳
- 日期：2026-10-18
- 决策范围：`apps/api` 中 HTTP 请求解析、路由绑定与响应写出的归属位置

## 背景

ADR-0002 把 Gin 解析/响应适配留在 `internal/platform`，各模块的 `handler` 包只承载 DTO 与接口。结果是 `internal/platform/router.go` 中同时存在依赖组装、中间件和 25 个逐路由的匿名函数（绑定 JSON、解析路径/分页、注入操作者 ID、写出成功/错误响应、转发 SSE）。模块的 HTTP 契约散落在组装层，单个端点的行为无法独立测试或定位。

## 决策

1. `platform` 继续独占依赖构造（DB/Redis/JWT/仓储/用例）、全局中间件、请求关联 ID 和统一响应与错误映射。
2. 共享 HTTP 工具放在无业务依赖的中立包 `internal/httpx`（`Success`、`Fail`、`WriteError`、`ActorID`、`PathID`、`ParsePage`、`ActorIDKey`、`RequestIDKey`、`LoggerKey`）。`platform` 与各模块 handler 都依赖它；若让 handler 直接依赖 `platform` 会形成 `platform → handler → platform` 循环依赖。
3. 每个业务模块在自己的 `handler` 包内提供 `Handler` 结构体，持有该模块用例接口，并暴露 `Register(group)` 自行声明路由。所有逐路由逻辑移入这些方法。鉴权中间件随 `auth` 模块的专属性质归入 `auth/handler`，由组装层安装到受保护路由组。
4. `platform` 只负责构造各模块 `Handler` 并调用 `Register`，不再持有任何业务路由的匿名函数或鉴权逻辑。
5. 用例接口、`service` 签名与 `handler` DTO 本次不变；依赖方向为 `platform → 模块 handler → 模块 service`，模块 handler 不感知 `platform` 的组装细节。

## 依赖方向与数据所有权

```text
cmd/api → platform（组装、全局中间件）
            ├── httpx（响应信封、错误映射、路径/分页解析）
            └── {auth,team,project,task,chat}/handler → 同名 service → repository
```

数据所有权不变。HTTP 边界校验（路径、查询、分页、请求体大小）仍在 handler 中完成，业务不变量仍由 service 强制。

## 安全影响

鉴权中间件仍由 `platform` 在业务执行前统一安装；`ActorID` 只从已认证的上下文读取，客户端无法提交操作者 ID。错误信封与状态码映射逻辑不变，内部错误细节仍不外泄。

## 备选方案

- 维持匿名函数集中在 `router.go`：拒绝，端点不可独立测试，模块契约散落。
- 把依赖组装也下放到各模块：拒绝，会破坏单一组装点并引入反向依赖。
- 本次同时改造 service 接口签名：拒绝，超出结构调整范围，留待单独 ADR。

## 迁移与回滚

路由路径、中间件顺序、响应信封与状态码保持不变，属于无契约变化的结构调整。回滚方式是还原本次提交；不涉及数据库或配置迁移。

## 验证证据

2026-10-18 在 `apps/api` 执行：`go vet ./...` 通过；`go test ./...` 全部通过（新增 `httpx`、`auth/handler`、`team/handler` 测试，`platform` 测试保留请求 ID/错误信封/崩溃恢复用例）；`go build ./...` 通过；本次新增/修改文件已 `gofmt` 校验。
