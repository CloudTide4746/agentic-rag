# Agentic-RAG API 契约与 Go 服务

本目录包含 React 当前使用的 Python Agent HTTP API 契约，以及公开 Go API。Python Agent 契约和 Go TeamFlow 契约分别维护，避免把内部 Agent 接口误认为公开业务 API。

## Go 服务骨架

Go API 提供用户注册/登录、团队、项目和任务基础 CRUD。登录后使用 Bearer JWT 访问受保护路由。模块应用接口使用标准库 `context.Context`；Gin 只用于 `internal/platform` HTTP 适配。

先创建独立开发 MySQL 数据库，并按顺序执行 `migrations/*.up.sql`。不要对旧 TeamFlow 或 Python Agent 数据库直接执行迁移。设置环境变量后启动：

```powershell
$env:API_ADDR = ":8080"
$env:DATABASE_DSN = "app:change-me@tcp(127.0.0.1:3306)/agentic_rag?charset=utf8mb4&parseTime=True&loc=UTC"
$env:REDIS_ADDR = "127.0.0.1:6379"
$env:JWT_SECRET = "replace-with-at-least-32-random-characters"
go run ./cmd/api
```

完整环境变量样例见 `.env.example`。健康检查地址为 `GET /healthz`。API 只在 MySQL 与 Redis 均就绪、JWT 密钥满足长度要求时启动。用户密码使用 bcrypt；每个用户仅保留最近一次登录的 JWT，令牌与 Redis 会话均为两小时有效。

公开 Go API 契约见 `api/openapi.yaml`。`/api/v1` 保留 TeamFlow 路径，成功响应统一为 `{"data": ..., "request_id": "..."}`，错误统一为 `{"error":{"code":"...","message":"..."},"request_id":"..."}`。对外 ID 为字符串。任务更新必须带上读取时的 `version`；冲突时返回 409。团队/项目成员管理、任务状态和负责人专项操作，以及评论、附件、通知、WebSocket 不在当前迁移范围内。当前提供的是代码与空库 schema 迁移，不会自动复制旧 TeamFlow 数据。

`cmd/worker` 当前仅用于保持仓库构建目标完整。运行时会明确报错并以非零状态退出；消息消费能力尚未实现。

## 文档

- `openapi.yaml`：OpenAPI 3.1 源文件，可供 Swagger UI、Redoc、客户端生成器导入。
- `openapi.json`：从 YAML 导出的 JSON 版本。
- `api/openapi.yaml`：公开 Go `/api/v1` Auth/TeamFlow 契约。
- `migrations/`：有序 SQL 升级/回滚脚本；应用启动不自动修改数据库结构。
- 当前 Web 开发代理指向 Python FastAPI（默认 `localhost:8000`）；线上基址由部署代理提供。
- 浏览器当前使用旧版 `/api/chat`、`/api/admin` 路径。这些路由属于 Python Agent，不是未来规划的 Go `/api/v1`。

从仓库根目录运行以下命令可验证 Python Agent 契约并重新导出 JSON：

```powershell
python -c "import json, yaml; p='apps/api/openapi.yaml'; d=yaml.safe_load(open(p, encoding='utf-8')); json.dump(d, open('apps/api/openapi.json','w',encoding='utf-8'), ensure_ascii=False, indent=2); print('OpenAPI', d['openapi'], 'paths:', len(d['paths']))"
```

SSE 接口以 `text/event-stream` 返回阶段事件；事件 data 负载由 Agent 生成，不使用固定结构。管理接口鉴权方式须根据实际 Python 配置部署，不在本契约中虚构认证方案。
