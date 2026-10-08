# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 仓库用途

这是 **TeamFlow 企业级协作平台**的主项目仓库，目前包含完整的开发教程文档。实际的 Go 源代码将按照教程逐步在此目录下构建。

目标项目：一个类 Jira/飞书任务协作的 Go 后端，技术栈为 Gin + GORM + MySQL + Redis + JWT + WebSocket + Docker。

## 教程文档

`TeamFlow企业级开发教程/` 下有 87 个 Markdown 文档，入口为 `00-总目录.md`，分四个阶段：

- **第01-07章**（第一阶段）：项目准备、数据库设计、用户系统、团队/项目/任务 CRUD、统一处理 → 完成后可作为实习项目
- **第08-12章**（第二阶段）：RBAC 权限、Redis 缓存、文件存储、操作日志、Docker 部署
- **第13-19章**（第三阶段）：WebSocket 实时通知、消息队列、定时任务、中间件体系、Swagger、测试
- **第20章 + 附录**：AI 增强（可选）、面试题、简历写法

## 预期项目结构（按教程搭建后）

```
TeamFlow/
├── cmd/main.go              # 应用入口
├── config/config.yaml       # Viper 配置文件
├── internal/
│   ├── controller/          # Gin handler，只做参数绑定和调用 Service
│   ├── service/             # 业务逻辑层
│   ├── repository/          # GORM 数据库操作层
│   ├── model/               # GORM 模型（对应数据库表）
│   ├── middleware/          # JWT、日志、CORS、限流等中间件
│   ├── router/              # 路由注册
│   └── dto/                 # 请求/响应 DTO 结构体
├── pkg/
│   ├── database/            # MySQL 和 Redis 连接初始化
│   ├── jwt/                 # Token 生成与解析
│   ├── response/            # 统一响应格式 {code, message, data}
│   └── utils/               # 密码加密、文件上传等工具
└── storage/uploads/         # 本地上传文件（不提交 git）
```

## 常用命令（项目搭建后）

```bash
# 运行
go run cmd/main.go

# 构建
go build -o bin/teamflow cmd/main.go

# 测试
go test ./...
go test ./internal/service/... -v       # 指定包
go test ./... -cover                    # 覆盖率

# Swagger 文档生成（需安装 swag）
swag init -g cmd/main.go

# Docker 开发环境
docker-compose up -d

# 生产部署
docker-compose -f docker-compose.prod.yml up -d --build
```

## 架构关键约定

**数据流向**：`Router → Middleware → Controller → Service → Repository → DB/Redis`

- Controller 不包含业务逻辑，只做参数绑定、调用 Service、返回响应
- Service 层管理事务，调用 Repository；需要发送通知时通过 WebSocket Hub 的 goroutine 异步推送
- Repository 只封装 GORM 查询，不含业务判断
- 所有 HTTP 响应统一走 `pkg/response` 的 `Success/Error` 函数，格式为 `{code, message, data}`
- 软删除使用 GORM 的 `gorm.Model`（内含 `DeletedAt`）

**配置**：Viper 读取 `config/config.yaml`，支持环境变量覆盖（前缀 `TEAMFLOW_`）

**认证**：JWT 存于 `Authorization: Bearer <token>` Header，`middleware/auth.go` 解析后将 `userID` 注入 `gin.Context`

**错误处理**：业务错误使用自定义 `AppError{Code, Message, HTTPStatus}`，Recovery 中间件统一捕获 panic 并返回 500 JSON

## Agent skills

### Issue tracker

Issues 存放在 GitHub Issues，通过 `gh` CLI 操作。详见 `docs/agents/issue-tracker.md`。

### Triage labels

使用默认 label 名称：needs-triage / needs-info / ready-for-agent / ready-for-human / wontfix。详见 `docs/agents/triage-labels.md`。

### Domain docs

Multi-context 布局：根目录 CONTEXT-MAP.md 指向各 context 的 CONTEXT.md，架构决策记录在各自的 docs/adr/ 下。详见 `docs/agents/domain.md`。
