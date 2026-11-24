# Agentic-RAG API 契约与 Go 服务

本目录包含内部 Python Agent HTTP API 契约，以及公开 Go API。Python Agent 契约和 Go TeamFlow 契约分别维护，避免把内部 Agent 接口误认为公开业务 API。

## Go 服务骨架

Go API 提供用户注册/登录、团队、项目和任务基础 CRUD。登录后使用 Bearer JWT 访问受保护路由。模块应用接口使用标准库 `context.Context`；各业务模块的 `handler` 负责 Gin HTTP 适配，`internal/infrastructure/platform` 负责依赖组装与路由接线，中间件实现集中在 `internal/infrastructure/middleware`。

先创建独立开发 MySQL 数据库，并按顺序执行 `migrations/*.up.sql`。不要对旧 TeamFlow 或 Python Agent 数据库直接执行迁移。从仓库根目录启动：

```powershell
# 首次使用：复制后填写本机凭据；已有 .env 时不要覆盖。
# Copy-Item apps/api/.env.example apps/api/.env
npm run go:dev

# 或在 apps/api 目录直接启动（Windows、Linux、macOS 均可）
cd apps/api
go run ./cmd/api -config config.yaml
```

完整环境变量样例见 `.env.example`。健康检查地址为 `GET /healthz`。API 只在 MySQL 与 Redis 均就绪、JWT 密钥满足长度要求时启动。用户密码使用 bcrypt；每个用户仅保留最近一次登录的 JWT，令牌与 Redis 会话均为两小时有效。

## 配置加载

`internal/infrastructure/config/config.go` 负责读取、合并和校验配置；`cmd/api` 把配置传给 `internal/infrastructure/platform` 组装层，再将所需值注入数据库、Redis、JWT 和 Agent 客户端。业务 Service 不依赖配置文件，也不新增全局配置单例或改写进程环境。

配置、通用错误、HTTP 工具、中间件、依赖组装与 JWT 实现分别位于 `internal/infrastructure/{config,faults,httpx,middleware,platform,security}`。Go import 必须使用完整的 `teamflow/internal/infrastructure/...` 路径；迁移前的 `teamflow/internal/{config,faults,httpx,platform,security}` 已不存在。目录检查结果与后续建议见[目录结构与引用维护](../../docs/目录结构与引用维护.md)。

全局中间件按 `middleware.Recovery(logger)` → `middleware.RequestID(logger)` → `middleware.BodyLimit(1<<20)` 安装；`middleware.Authenticate(authApp)` 仅安装到受保护的 `/api/v1` 路由组。认证中间件只消费 `Authenticate` 能力，JWT 与 Redis 会话规则仍由认证用例负责。现有测试所需的 `platform` 私有入口及 `auth.Handler.Middleware()` 保留为薄转发；新组装直接调用新包。迁移边界、失败行为与回滚见 [ADR-0005](../../docs/adr/0005-中间件职责拆分.md)。

配置优先级为：**进程环境变量 > `config.yaml` > YAML 指定的可选 `.env` 文件**。`env_file` 相对 YAML 所在目录解析；支持空行、注释行、`KEY=VALUE` 和成对外层引号，不执行变量插值或 Shell 语法。

| YAML 字段 | 进程环境变量 | 用途 |
| --- | --- | --- |
| `server.addr` | `API_ADDR` | HTTP 监听地址，开发默认 `127.0.0.1:7800`（前端 Vite 使用 `5173`） |
| `server.gin_mode` | `GIN_MODE` | `debug`、`release` 或 `test` |
| `database.dsn` | `DATABASE_DSN` | MySQL 连接串，优先通过 `.env` 或进程环境提供 |
| `database.name` | — | 覆盖文件配置中的数据库名，开发为 `agentic_rag_dev` |
| `redis.addr` / `redis.password` / `redis.db` | `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB` | Redis 连接配置 |
| `jwt.secret` | `JWT_SECRET` | 至少 32 字节的 JWT 密钥 |
| `agent.base_url` / `agent.internal_api_token` | `AGENT_BASE_URL` / `INTERNAL_API_TOKEN` | 内部 Agent 地址与令牌 |

本地 `config.yaml` 复用 `.env` 中数据库连接串的账号、密码和地址，仅把库名改为 `agentic_rag_dev`，避免使用 Python Agent 的库。**显式设置进程 `DATABASE_DSN` 时，使用其完整连接串，忽略 YAML 的 `database.name`**。启动不会创建数据库、执行迁移或写入种子数据。

修改端口和开发库名使用 `config.yaml`；改动端口时还需在 `apps/web/.env.local` 同步设置 `API_PROXY_TARGET` 并重启 Vite。开发默认避开被本机 `ApplicationWebServer` 占用的 `8080`，使用 `127.0.0.1:7800`。真实密码、JWT 密钥、内部令牌应留在被 Git 忽略的 `.env` 或由部署环境注入，不提交到 YAML；启动日志不打印连接串、密码或令牌。Agent 未配置时仍可启动协作 API，聊天请求按现有逻辑报告服务不可用。

`-config <路径>` 可选择独立配置文件，指定文件缺失、YAML 字段拼错/重复、多文档、无效地址或弱密钥会使启动失败，不回退为假成功。未指定 `-config` 时尝试当前目录的 `config.yaml`；该文件不存在则保留仅靠进程环境变量启动的方式，不自动加载 `.env`。生产可从不含开发 YAML 的工作目录运行二进制并设置现有环境变量，或显式提供生产 YAML（不包含开发库名覆盖）。

旧 `scripts/run-dev.ps1` 和 `run-local.ps1` 保留为兼容入口，仅转发到 Go 并透传退出码。迁移不修改现有 `.env`、数据库或认证规则；回退时恢复原启动代码和命令即可，不涉及数据回滚。

YAML 解析复用仓库已经锁定的 `github.com/goccy/go-yaml v1.19.2`（MIT，Gin 现有依赖），仅提升为直接依赖；不引入 Viper、dotenv 框架或额外依赖版本。

公开 Go API 契约见 `api/openapi.yaml`。`/api/v1` 保留 TeamFlow 路径，成功响应统一为 `{"data": ..., "request_id": "..."}`，错误统一为 `{"error":{"code":"...","message":"..."},"request_id":"..."}`。对外 ID 为字符串。任务更新必须带上读取时的 `version`；冲突时返回 409。团队/项目成员管理、任务状态和负责人专项操作，以及评论、附件、通知、WebSocket 不在当前迁移范围内。当前提供的是代码与空库 schema 迁移，不会自动复制旧 TeamFlow 数据。

## 聊天会话与问答

`/api/v1/chat/*` 是 Go 公开入口，按 ADR-0003 的“双层会话”设计实现：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/chat/sessions` | 当前用户会话列表，分页 |
| `POST` | `/api/v1/chat/sessions` | 创建会话（先落平台映射，再建 Agent 会话） |
| `GET` | `/api/v1/chat/sessions/{id}` | 单个会话及其消息历史 |
| `GET` | `/api/v1/chat/sessions/{id}/messages` | 单个会话的消息历史 |
| `POST` | `/api/v1/chat/sessions/{id}/runs` | 提问，SSE 流式返回 |

- Go 独占 `chat_platform_sessions` 映射表，只保存平台归属、生命周期状态和幂等键；消息正文、引用和决策日志仍由 Python Agent 写自己的表。Go 不跨库读写 Agent 表。
- 对外只暴露平台生成的随机字符串 ID，Agent 的数字会话 ID 不出网；查询始终带 `owner_id` 过滤，他人会话统一返回 404。
- Agent 创建明确被拒 → 映射置 `failed`（HTTP 502）；连接不可达 → `failed`（503）；超时或响应丢失 → `pending`（504），留给对账处理，**不做盲目重试**，以免产生两个 Agent 会话。
- Agent 错误映射为稳定公开码：`AGENT_REJECTED`(502)、`AGENT_UNAVAILABLE`(503)、`AGENT_TIMEOUT`(504)。是否持久化用户消息由平台决定，浏览器不能传 `skip_user_persist`。
- 会话映射表由 `migrations/0002_chat_platform_sessions.up.sql` 创建。

存储层契约测试需要真实数据库（未设置时自动跳过）：

```powershell
$env:TEST_DATABASE_DSN = "root:change-me@tcp(127.0.0.1:3306)/agentic_rag_test?charset=utf8mb4&parseTime=True&loc=UTC"
go test ./internal/chat/repository/ -run TestSessionStore -v
```

`cmd/worker` 当前仅用于保持仓库构建目标完整。运行时会明确报错并以非零状态退出；消息消费能力尚未实现。

## 文档

- `openapi.yaml`：OpenAPI 3.1 源文件，可供 Swagger UI、Redoc、客户端生成器导入。
- `openapi.json`：从 YAML 导出的 JSON 版本。
- `api/openapi.yaml`：公开 Go `/api/v1` Auth/TeamFlow 契约。
- `migrations/`：有序 SQL 升级/回滚脚本；应用启动不自动修改数据库结构。
- 当前 Web 开发代理指向 Go（默认 `http://127.0.0.1:7800`），可通过 `API_PROXY_TARGET` 覆盖；线上基址由部署代理提供。
- 当前协作与聊天页面使用 Go `/api/v1/*`；知识库、数据源与日志页面保留静态预览。旧 Python 请求适配器仍有保留代码，不能据此将 Agent 管理接口视为公开业务入口。

从仓库根目录运行以下命令可验证 Python Agent 契约并重新导出 JSON：

```powershell
python -c "import json, yaml; p='apps/api/openapi.yaml'; d=yaml.safe_load(open(p, encoding='utf-8')); json.dump(d, open('apps/api/openapi.json','w',encoding='utf-8'), ensure_ascii=False, indent=2); print('OpenAPI', d['openapi'], 'paths:', len(d['paths']))"
```

SSE 接口以 `text/event-stream` 返回阶段事件；事件 data 负载由 Agent 生成，不使用固定结构。管理接口鉴权方式须根据实际 Python 配置部署，不在本契约中虚构认证方案。
