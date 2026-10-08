# 任务状态
最后更新：2026-07-03

## 已完成
- 项目骨架初始化：go.mod、目录结构、配置加载（Viper）、日志（Zap）
- 数据库连接初始化（`storage/database.go` 的 `InitDB`）
- 全部 9 个 GORM Model 定义（User/Team/TeamMember/Project/ProjectMember/Task/Comment/Attachment/Notification）
- `User.BeforeCreate` hook（密码 bcrypt 加密、Nickname 兜底）
- `User.BeforeUpdate` hook（仅 Password 字段 Changed 时重新哈希）
- `UserRepository`：GetByID / GetByEmail / GetByUsername / ListUsers / ListUsersByTeamID / ListUsersByProjectID
- `UserService`：CreateUser / UpdateUser（空密码跳过）/ DeleteUser（软删除）+ isMySQLDuplicate(1062) 辅助
- `database.AutoMigrate()` 已实现但 `cmd/main.go` 注释掉未启用

## 进行中
- 无

## 待处理（按优先级）
1. 跑 `go build ./...` 验证当前代码可编译（关键路径：cmd/main.go → config → logger → storage）
2. 对照教程第03章继续 User 体系：DTO、Controller、JWT 登录、注册接口
3. 第04章 Team/Project 模块：Repository → Service → Controller
4. 第05章 Task 模块
5. 第06章 Comment/Attachment 模块
6. 第07章 统一响应 / 错误处理 / 参数校验（pkg/response, pkg/errors）
7. 启动 MySQL+Redis docker-compose，跑通 `AutoMigrate()`，写 CRUD 接口集成测试
8. 删除空的 `pkg/database/mysql.go`（DB 实际在 `storage/database.go`），统一入口

## 阻塞点
- 无技术阻塞
- `.env` 中 `DB_PASSWORD="140322Bk"` 已硬编码密码，**生产前必须改密钥管理**（Vault / KMS）
- `cmd/main.go` 中 `storage.InitDB(...)` 仍被注释，需要在验证完迁移函数后再启用
