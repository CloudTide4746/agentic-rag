# 项目简报
生成时间：2026-07-03 21:42
生成模型：弱模型（扫描阶段）

## 项目基本信息
- **名称**：TeamFlow 企业级协作平台
- **类型**：Web 后端 API 服务（Gin + GORM + MySQL + Redis + JWT + WebSocket + Docker）
- **技术栈**：
  - 语言：Go 1.26.4
  - 框架/库：Gin（待引入）、GORM v1.31.2、go-sql-driver/mysql v1.8.1
  - 配置：Viper v1.21.0 + godotenv
  - 日志：Zap v1.27.0 + lumberjack
  - 加密：golang.org/x/crypto/bcrypt
  - **尚未引入**：Gin、Redis 客户端、jwt-go、gorilla/websocket
- **当前阶段**：开发中 — 第一阶段第03章（用户系统）末尾，下一步进入第04章（团队/项目）

## 当前任务状态
### 已完成
- 项目骨架：go.mod、目录结构、配置加载（Viper + YAML）、日志（Zap）
- DB 连接：`storage/database.go` 的 `InitDB(dsn)` 初始化 GORM + 连接池
- 9 个 GORM Model 全部定义（User/Team/TeamMember/Project/ProjectMember/Task/Comment/Attachment/Notification）
- User Repository：GetByID / GetByEmail / GetByUsername / ListUsers / ListUsersByTeamID / ListUsersByProjectID
- User Service：CreateUser / UpdateUser（空密码跳过）/ DeleteUser（软删除）+ isMySQLDuplicate(1062)
- User Model Hooks：BeforeCreate（密码 bcrypt + Nickname 兜底）、BeforeUpdate（Changed 守卫）
- `database.AutoMigrate()` 已写好但 `cmd/main.go` 中未启用
- 3 套 plan/spec 设计文档已沉淀到 `docs/superpowers/{plans,specs}/2026-07-01-user-*.md`

### 进行中
- 无（User 模块基本收尾，等待强模型决策下一步：跑编译验证？继续 Controller/JWT？）

### 下一步（按优先级）
1. **跑 `go build ./...` 验证现有代码可编译** — 0 风险必经步骤
2. 启动本地 MySQL（docker-compose），启用 `storage.InitDB` 和 `AutoMigrate`，写最小冒烟测试
3. 教程第03章剩余：DTO / Controller / 注册接口 / 登录接口 / JWT 中间件
4. 教程第04章：Team / Project / 成员关联 的 Repository → Service → Controller
5. 教程第05章：Task 模块
6. 教程第06章：Comment / Attachment
7. 教程第07章：`pkg/response` 统一响应 / `pkg/errors` 业务错误 / Validator 参数校验
8. 清理：`pkg/database/mysql.go` 是空文件，决定是否迁移 DB 入口到 `pkg/database/`

### 阻塞点
- 无技术阻塞
- **安全风险**：`.env` 中 `DB_PASSWORD="140322Bk"` 硬编码生产前必须换
- **DB 入口分裂**：`pkg/database/` 是空包，实际 DB 在 `storage/`，CLAUDE.md 与代码不一致需要决策

## 绝对不要尝试（失败记录）
- （暂无 — 当前会话未触发明确的失败方案）
- 已规避：**不要让 UpdateUser 二次哈希已加密密码**（用 `tx.Statement.Changed("Password")` + Service 空字符串判断双保险）
- 已规避：**不要在 DeleteUser 里混读 Error 和 RowsAffected 的顺序**（先 Error 后 RowsAffected）

## 关键决策摘要
- **Service 直连 storage.DB**：Create/Update/Delete 简单写入不经过 Repository 层
- **UpdateUser 空密码语义**：传 `""` 表示不改密码，不传值用 pointer 表达会更地道（当前用 string 是务实选择）
- **DeleteUser 报错"用户不存在"**：而非 silent success
- **MySQL 唯一冲突 1062 错误码**：用 `errors.As(&mysql.MySQLError{})` 翻译为业务错误"用户名或邮箱已存在"
- **教程驱动开发**：所有功能参考 `TeamFlow企业级开发教程/` 对应章节

## 本项目推荐 Skills
- `using-superpowers` — **任何对话开始前必调**，强制 Skill 检查
- `writing-plans` — 接需求时先写多步计划（项目惯例 `docs/superpowers/plans/`）
- `subagent-driven-development` — 按 plan 逐任务派 subagent 实施（项目主流模式）
- `executing-plans` — 不想派 subagent 时的单线执行替代
- `requesting-code-review` — 完成一段代码后评审
- `qa` — 用户报告 bug 时交互式提单
- `systematic-debugging` — 排查具体技术 bug
- `git-guardrails-claude-code` — Git 提交安全
- `smart-handoff` — 本次会话已在用，handoff 接力场景

## 关键文件路径速查
```
项目根：E:\Backend\Go\New_to_Go\Middle_Project
入口：    cmd/main.go
配置：    config/config.go + config/config.yaml
日志：    pkg/logger/logger.go
DB：      storage/database.go     （pkg/database/mysql.go 是空文件！）
Model:    internal/model/*.go     (9个全有)
Repo:     internal/repository/user_repository.go    (只有 User)
Service:  internal/service/user_service.go          (只有 User)
迁移：    internal/database/migrate.go              (未在 main 启用)
设计：    docs/superpowers/{plans,specs}/2026-07-01-user-*.md
教程：    TeamFlow企业级开发教程/00-总目录.md
全局CLAUDE.md：C:\Users\31338\.claude\CLAUDE.md
项目CLAUDE.md：E:\Backend\Go\New_to_Go\Middle_Project\CLAUDE.md
.handoff/：已创建 BRIEF.md + task-state.md + decision-log.md + failure-map.md + tool-compass.md
```
