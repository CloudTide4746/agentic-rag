# ADR-0002：TeamFlow CRUD 模块接口与统一响应

- 状态：已采纳
- 日期：2026-10-11
- 决策范围：将 TeamFlow 用户认证及团队、项目、任务基础 CRUD 迁入 `apps/api`

## 背景

`apps/api` 已按 `auth`、`team`、`project`、`task` 预留模块目录，但只有路由健康检查和空包。`apps/TeamFlow` 中的 Gin 控制器直接依赖旧 service，旧 service 又访问全局 GORM DB；团队/项目接口与任务接口使用不同 JSON 响应形状，部分业务错误使用 HTTP 200。新项目规范要求模块接口不依赖 Gin、持久化经过所属模块 Repository、身份由认证边界可信注入，并以统一响应信封和真实 HTTP 错误状态对外。

## 决策

1. 本次迁入用户注册/登录/JWT 认证，以及团队、项目、任务实体的创建、读取、更新、删除和列表能力。任务更新需要提交当前 `version`，过期版本返回 409。团队/项目成员管理、任务状态与负责人专项操作、评论、附件、通知及 WebSocket 不在本次范围。
2. 保留 TeamFlow 已有 `/api/v1` 路由；响应统一为成功 `{"data":{},"request_id":"..."}` 与失败 `{"error":{"code":"...","message":"..."},"request_id":"..."}`。所有错误使用适当 HTTP 状态。公开平台 ID 使用字符串，JSON 字段使用 snake_case；OpenAPI 与实际 DTO 同步。
3. 业务模块通过使用 `context.Context` 的窄应用接口暴露用例；Gin 解析/响应适配留在 `internal/platform`，并从已验证身份注入操作者 ID。客户端不能提交操作者 ID。
4. `team`、`project`、`task` 各自拥有领域类型、应用规则和 Repository 边界。跨模块权限能力通过模块公开接口注入，不跨模块查表。GORM/MySQL 适配器由启动组装层构造；数据库结构采用有序 SQL migration，不使用 AutoMigrate。
5. Go 继续独占 TeamFlow 平台数据写入。认证/JWT 密钥及数据库、Redis 连接配置由环境变量提供；不把秘密提交到仓库。JWT 与登录会话有效期按已批准的两小时基线实现；同一用户再次登录会覆盖其 Redis 当前令牌，使旧令牌失效。

## 依赖方向与数据所有权

```text
Gin platform adapter → auth/team/project/task application interfaces
composition root → repositories (GORM/MySQL), JWT, Redis session adapter
project → public team membership capability
task → public project membership capability
```

用户、认证会话、团队、项目及任务仍由 Go/平台库唯一写入。RAG Python 数据所有权不变，也不参与本次数据访问。

## 安全影响

请求体、路径、查询参数和分页均在 HTTP 边界验证。受保护读写从 JWT 身份得到操作者，并在应用用例中重新校验团队/项目归属。密码使用 bcrypt；JWT 固定签名算法、校验过期时间；Redis 故障不得被当作有效认证。错误信封不包含 SQL、堆栈、令牌或密码。

## 备选方案

- 保留旧的多套响应：拒绝，会继续扩大客户端适配分支并违反新 API 的单一契约。
- 将 Gin 类型暴露给业务模块：拒绝，会使应用规则依赖 HTTP 框架。
- 继续使用全局 DB 与 AutoMigrate：拒绝，难以注入依赖、测试和审查数据结构变更。
- 本次一并迁移成员专项管理、任务流转及周边实时功能：拒绝，超出本次基础 CRUD 边界。

## 迁移与回滚

保留已有 `/api/v1` 路由，更新成功/失败响应为统一信封，并将公开 ID 编码为字符串；同步 OpenAPI/调用端类型。由于响应与 ID 表示是公开契约变化，随本次发布记录迁移说明；如调用端未能同步，可回滚 API 发布包和相应客户端版本。数据库采用仅新增的编号 migration，并提供对应 down 脚本；先在独立开发数据库验证，不能覆盖旧 TeamFlow 数据库或 Python Agent 数据。

## 验证证据

实现完成后记录本 ADR 对应的 `gofmt`、`go vet ./...`、`go test ./...`、`go build ./...` 结果，并校验 OpenAPI 与路由/DTO/错误状态一致。当前 ADR 本身不代表实现或数据库迁移已验证。
