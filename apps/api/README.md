# Agentic-RAG 前后端接口契约

本目录包含当前 `apps/web` 所调用的 Python Agent HTTP API 契约，以及 TeamFlow Go API 的初始骨架。Go 服务目前只提供健康检查，尚未实现业务 `/api/v1`；本契约记录的是 Python Agent 现状，不代表 Go 已接管这些接口。接口路径、字段和行为以 `apps/web/src/lib/api` 及 `backend` 实际实现为依据。

## Go 服务骨架

在本目录运行服务前设置监听地址：

```powershell
$env:API_ADDR = ":8080"
go run ./cmd/api
```

健康检查地址为 `GET /healthz`。TeamFlow 核心模块目前只建立了 `handler`、`service`、`repository` 包结构，尚未注册业务路由或连接基础设施。

`cmd/worker` 当前仅用于保持仓库构建目标完整。运行时会明确报错并以非零状态退出；消息消费能力尚未实现。

## 文档

- `openapi.yaml`：OpenAPI 3.1 源文件，可供 Swagger UI、Redoc、客户端生成器导入。
- `openapi.json`：从 YAML 导出的 JSON 版本。
- 当前 Web 开发代理指向 Python FastAPI（默认 `localhost:8000`）；线上基址由部署代理提供。
- 浏览器当前使用旧版 `/api/chat`、`/api/admin` 路径。这些路由属于 Python Agent，不是未来规划的 Go `/api/v1`。

从仓库根目录运行以下命令可验证并重新导出 JSON：

```powershell
python -c "import json, yaml; p='apps/api/openapi.yaml'; d=yaml.safe_load(open(p, encoding='utf-8')); json.dump(d, open('apps/api/openapi.json','w',encoding='utf-8'), ensure_ascii=False, indent=2); print('OpenAPI', d['openapi'], 'paths:', len(d['paths']))"
```

SSE 接口以 `text/event-stream` 返回阶段事件；事件 data 负载由 Agent 生成，不使用固定结构。管理接口鉴权方式须根据实际 Python 配置部署，不在本契约中虚构认证方案。
