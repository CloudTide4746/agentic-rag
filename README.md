# Agentic RAG 企业知识库

依据 `docs/开发规范与技术选型.md` 初始化的 React + Go 工程基线。

## 目录

```text
apps/web/                          React 19 + TypeScript strict + Vite
apps/api/                          Go + Gin，API / Worker 入口
backend/                           原 Python Agent，保持原样
archive/frontend-legacy-20260921/   原 Vue 前端，完整本地归档
docs/                              技术选型与开发规范
```

归档目录默认被 Git 忽略，旧源码与依赖仍保留在磁盘中。不要删除唯一备份。

## Monorepo 与交付

根目录提供跨应用验证命令；每个部署单元仍独立拥有自己的依赖与构建配置：

```powershell
# 首次使用时先安装前端锁定依赖
cd apps/web
npm ci
cd ../..

npm run check
```

GitHub Actions 会在 `main` 的推送和拉取请求上运行 Web、Go API 与 Python Agent 门禁。
推送 `v*` 标签将构建并发布 GitHub Release 产物。完整约定见
[`docs/monorepo-and-delivery.md`](docs/monorepo-and-delivery.md)。

## 本地启动

环境：Node.js 22.12+（已验证 24.20.0）、Go 1.26+（已验证 1.26.4）。
以下命令均从项目根目录分别开启终端执行。

终端一：

```powershell
cd apps/api
go mod download
go run ./cmd/api
```

终端二：

```powershell
cd apps/web
npm ci
npm run dev
```

打开 http://127.0.0.1:5173，首页应显示 Go API 已连接。
Vite 将 `/api` 转发至 `http://127.0.0.1:8080`，无需开放跨域。
前端可复制 `.env.example` 为 `.env.local` 修改代理地址；Go 仅读取进程环境变量，
例如 `$env:HTTP_ADDR='127.0.0.1:8081'`，不会自动加载 `.env`。
生产静态文件由 `npm run build` 输出到 `apps/web/dist`；实际部署需配置同源 `/api` 反向代理。

## 已完成的初始化

- React Router 懒加载、TanStack Query Provider、统一 HTTP 响应类型与取消信号。
- Tailwind CSS 和基础样式；预留文档指定的业务模块目录。
- 安装并锁定 Zustand、ECharts、Vitest、React Testing Library、Playwright，尚未实现聊天状态或统计图表。
- Gin API、Zap 结构化日志、请求 ID、统一 JSON 错误、优雅关闭。
- GORM/MySQL、Redis、RabbitMQ 的依赖与注入边界，尚未建立连接。
- OpenAPI 健康检查契约、单元测试、浏览器冒烟测试及 CI。

`GET /api/v1/health/live` 返回 200；`GET /api/v1/health/ready` 当前固定返回 503，
因为业务依赖尚未接入。Worker 入口执行 `go run ./cmd/worker` 会明确提示未实现并退出，
不会消费消息。依赖版本见 `package.json` / `package-lock.json` 和 `go.mod` / `go.sum`。

## 验证

```powershell
cd apps/web
npm run typecheck
npm test
npm run build
# 首次运行浏览器测试需安装 Chromium
node node_modules/@playwright/test/cli.js install chromium
npm run test:e2e
```

```powershell
cd apps/api
go test ./...
go vet ./...
go build ./...
```

浏览器冒烟测试模拟健康检查响应，用于验证页面；后端测试验证真实路由的状态码与响应。
业务端到端流程尚未实现。后续按技术规范接入认证、资源映射、AgentClient/SSE、上传、
可靠任务和 Compose；不得将预留目录或已安装依赖理解为业务能力已完成。
Python 代码和依赖未改动，旧流程运行仍沿用 `backend` 的现有配置。

初始化验证结果：前端 3 项单元测试、TypeScript 检查和生产构建通过；Go 测试、
vet 和编译通过。本机 Playwright 缺少匹配版本的 Chromium，下载未获授权，
浏览器冒烟测试尚未通过验证。npm scripts 直接调用 Node 入口，以兼容本项目路径中的 `&&`。
