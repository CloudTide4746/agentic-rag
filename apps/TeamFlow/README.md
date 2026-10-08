# TeamFlow

TeamFlow 是一个使用 Go 构建的团队任务协作后端，围绕用户、团队、项目和任务提供完整的协作流程。项目在常规 REST API 之外，还实现了 JWT 鉴权、分层权限控制、Redis 在线状态、WebSocket 实时连接，以及基于 RabbitMQ 的可靠异步通知链路，适合作为 Go 企业级后端架构与消息可靠性实践项目。

> 当前仓库以后端实现和配套学习资料为主。根路径提供简单的演示页面，但尚未包含完整的生产级前端或容器编排文件。

## 核心能力

- **用户与认证**：用户注册、登录、个人资料、密码修改、头像上传和 JWT 鉴权。
- **团队协作**：创建与维护团队、成员加入或退出，以及 Owner、Admin、Member 等角色约束。
- **项目管理**：团队下的项目管理、项目成员维护和项目统计。
- **任务管理**：任务增删改查、筛选与分页、负责人分配、优先级和截止时间管理。
- **任务状态机**：约束 `todo → in_progress → review → done` 的流转，并支持指定的回退路径。
- **评论与附件**：任务评论、附件上传、列表、下载和删除。
- **缓存与在线状态**：使用 Redis 管理缓存、Token 和用户在线状态。
- **实时通信**：使用 WebSocket 维护连接、心跳和在线状态，并推送通知。
- **可靠消息**：RabbitMQ 手动 ACK、QoS、延迟重试、死信/停车场队列、发布确认、Outbox 和消费幂等。
- **工程基础设施**：统一响应与错误模型、参数校验、结构化日志、异常恢复和 GORM 自动迁移。

## 技术栈

| 领域 | 技术 |
| --- | --- |
| 语言与 Web 框架 | Go 1.26、Gin |
| 数据访问 | GORM、MySQL |
| 缓存与在线状态 | Redis |
| 消息系统 | RabbitMQ / AMQP 0-9-1 |
| 实时通信 | Gorilla WebSocket |
| 身份认证 | JWT、bcrypt |
| 配置管理 | Viper、godotenv |
| 日志 | Zap、Lumberjack |
| 数据校验 | go-playground/validator |
| 数据库迁移 | GORM AutoMigrate |

## 架构概览

常规 HTTP 请求采用分层结构：

```text
Client
  │
  ▼
Gin Router → Middleware → Controller → Service → Repository → MySQL
                    │           │                        └→ Redis
                    │           └→ Publisher → RabbitMQ → Consumer
                    │                                      │
                    └──────────────── WebSocket Hub ← Notification Service
```

任务分配会生成领域事件。事件通过 RabbitMQ 解耦发布和处理，消费者完成通知落库与 WebSocket 推送；Outbox 与消费幂等模型用于降低“数据库成功但消息丢失”以及重复投递带来的风险。

## 目录结构

```text
TeamFlow/
├── cmd/                         # 应用入口与 WebSocket 路由
├── config/                      # YAML 配置、环境变量绑定及配置测试
├── internal/
│   ├── cache/                   # Redis 缓存、Token 与资源缓存
│   ├── controller/              # HTTP 请求处理
│   ├── database/                # Redis 初始化与 GORM 自动迁移
│   ├── dto/                     # 请求、分页和响应 DTO
│   ├── event/                   # 事件信封、拓扑声明与事件处理器
│   ├── middleware/              # 认证、权限、错误处理和恢复
│   ├── model/                   # 领域模型与任务状态机
│   ├── mq/                      # RabbitMQ 连接、消费、重试和幂等
│   ├── publisher/               # 事件发布与发布确认
│   ├── repository/              # 数据持久化实现
│   ├── router/                  # Gin 路由注册
│   ├── service/                 # 业务逻辑
│   └── ws/                      # WebSocket Hub、Client 与消息模型
├── pkg/                         # 通用错误、日志、响应、校验与工具包
├── storage/                     # MySQL 连接；运行时日志和上传文件目录
├── scripts/                     # Redis 缓存诊断与压测脚本
├── docs/                        # 设计、研究、实施计划与专题文档
├── TeamFlow企业级开发教程/      # RabbitMQ 企业级实战教程
└── web/                         # 简单演示页面
```

## 本地运行

### Swagger 接口文档

启动服务后打开 `http://localhost:8080/swagger/index.html`。Swagger JSON 由 `swag` 从 Gin 控制器注释生成，提交在 `docs/swagger.json`，路由直接使用 `swaggo` 注册的文档，避免从进程工作目录读取静态文件。

安装锁定版本的生成器并导出 JSON/YAML：

```powershell
go install github.com/swaggo/swag/cmd/swag@v1.16.6
./scripts/generate-swagger.ps1
```

`docs/swagger.json` 和 `docs/swagger.yaml` 是生成产物；API 变更时同时更新控制器注释并重新生成。

### 1. 环境要求

- Go `1.26.4`（以 [`go.mod`](./go.mod) 为准）
- MySQL 8.x
- Redis 6.x 或更高版本
- RabbitMQ 3.x

应用默认连接：

| 服务 | 默认地址 |
| --- | --- |
| HTTP | `http://localhost:8080` |
| MySQL | `localhost:3306`，数据库名 `teamflow` |
| Redis | `localhost:6379` |
| RabbitMQ | `amqp://guest:guest@localhost:5672/` |

RabbitMQ 地址目前在 `internal/mq/rabbitmq.go` 中使用本地默认值，因此启动应用前需要确保该地址可用。

### 2. 创建数据库

```sql
CREATE DATABASE teamflow
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;
```

应用启动时会通过 GORM 自动创建或更新用户、团队、项目、任务、评论、附件、通知、Outbox 和消费幂等相关表。

### 3. 配置环境变量

在项目根目录创建 `.env`；该文件已被 Git 忽略：

```dotenv
DB_PASSWORD=your_mysql_password
JWT_SECRET=replace-with-a-long-random-secret
```

其余默认配置位于 [`config/config.yaml`](./config/config.yaml)。如需使用其他配置文件，可设置：

```dotenv
CONFIG_PATH=config/config.production.yaml
```

Viper 也支持以 `TEAMFLOW_` 为前缀覆盖配置项，例如 `TEAMFLOW_REDIS_HOST`。

### 4. 安装依赖并启动

```bash
go mod download
go run ./cmd
```

启动顺序为：加载配置 → 初始化日志 → 连接 MySQL 并迁移 → 连接 Redis → 初始化 RabbitMQ → 启动消费者与 WebSocket Hub → 监听 `:8080`。

### 5. 验证项目

```bash
# 运行单元测试
go test ./...

# 编译全部包
go build ./...

# 连接真实 MySQL，执行配置、日志和数据库组件自检
go run ./cmd -test
```

组件自检会访问本地数据库，不适合在没有准备依赖服务的环境中直接执行。

## API 概览

REST API 使用 `/api/v1` 前缀。除注册和登录外，其余接口需要携带 JWT。

| 模块 | 主要端点 |
| --- | --- |
| 认证 | `POST /api/v1/auth/register`、`POST /api/v1/auth/login` |
| 用户 | `GET/PUT /api/v1/users/me`、修改密码、上传头像、查询在线状态 |
| 团队 | `/api/v1/teams` 下的团队与成员管理 |
| 项目 | `/api/v1/projects/:id` 下的详情、成员和统计 |
| 任务 | `/api/v1/tasks` 下的 CRUD、状态流转和负责人分配 |
| 评论 | `/api/v1/tasks/:id/comments` |
| 附件 | 任务附件上传、查询，以及 `/api/v1/attachments/:id` 下载或删除 |
| 在线状态 | `/api/v1/online/:userId` |
| WebSocket | `GET /ws?token=<JWT>` |

完整的字段、权限和错误响应请以 `internal/router`、`internal/controller` 与 DTO 定义为准。

## RabbitMQ 可靠性实践

仓库包含一套逐步演进的任务分配通知方案：

1. 声明持久化 Exchange、Queue 和 Routing Key。
2. 使用手动 ACK 与 QoS 控制消费者吞吐。
3. 通过分级延迟队列进行有限重试，超过阈值进入停车场队列。
4. 使用 Publisher Confirm 验证消息是否被 Broker 接收。
5. 使用 Outbox 将业务写入与待发布事件放入同一数据库事务。
6. 使用已处理消息记录保证消费者幂等。

学习顺序与实现细节见：

- [`TeamFlow企业级开发教程/RabbitMQ企业级实战/00-学习路线与最终架构.md`](./TeamFlow企业级开发教程/RabbitMQ企业级实战/00-学习路线与最终架构.md)
- [`docs/rabbitmq-task-assignment-pattern.md`](./docs/rabbitmq-task-assignment-pattern.md)
- [`docs/research/rabbitmq-official-notes.md`](./docs/research/rabbitmq-official-notes.md)

## 当前边界

- 仓库没有提交 Dockerfile 或 Compose 文件，MySQL、Redis 与 RabbitMQ 需要自行准备。
- RabbitMQ 连接地址暂未纳入统一配置。
- `web/` 仅用于简单演示和接口调试，项目主体是后端服务。
- 默认配置面向本地开发；生产环境应使用独立密钥、受限账号、TLS 和外部化配置。

## License

仓库当前未包含独立 License 文件。如需开源分发，请先补充明确的许可证。
