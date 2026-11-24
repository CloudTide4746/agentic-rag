# Agentic RAG 企业知识库

依据 `docs/开发规范与技术选型.md` 初始化的 React + Go 工程基线。

## 目录

```text
apps/web/                          React 19 + TypeScript strict + Vite
apps/api/                          Go + Gin，API / Worker 入口
apps/TeamFlow/                     原 TeamFlow 独立 Go 模块，迁移参考
backend/                           原 Python Agent，保持原样
archive/frontend-legacy-20260921/   原 Vue 前端，完整本地归档
docs/                              技术选型与开发规范
```

归档目录默认被 Git 忽略，旧源码与依赖仍保留在磁盘中。不要删除唯一备份。

当前包路径、引用修复与后续目录建议见[目录结构与引用维护](docs/目录结构与引用维护.md)。

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
推送 `v*` 标签将构建并发布 GitHub Release 产物。当前执行配置见
[CI](.github/workflows/ci.yml) 与 [Release](.github/workflows/release.yml)。

## 本地启动

环境：Node.js 22.12+（已验证 24.20.0）、Go 1.26+（已验证 1.26.4）。
安装前端依赖、准备好 Go 的本地 `.env` 和开发数据库后，在根目录运行 `npm run dev` 可同时启动前后端；任一进程退出时会停止本次启动的另一进程，Ctrl+C 同时停止两者。也可分别开启终端执行以下命令。

终端一：

```powershell
cd apps/api
go mod download
go run ./cmd/api -config config.yaml
```

终端二：

```powershell
cd apps/web
npm ci
npm run dev
```

打开 http://127.0.0.1:5173（前端 Vite）。Go 开发配置监听 `127.0.0.1:7800`，可在根目录用 `npm run go:dev` 启动。
Vite 默认将 `/api` 转发至 `http://127.0.0.1:7800`；如需连接其他 Go 端口，在 `apps/web/.env.local`
设置 `API_PROXY_TARGET`，然后重启 Vite。
本机 `8080` 已被 `ApplicationWebServer` 占用，其 HTML 404 并非 Go 登录接口的响应。开发配置与 Vite 代理统一使用独立的 `7800`，避免 `localhost` 在 IPv4/IPv6 之间解析到不同服务。浏览器请求仍保持同源 `/api/v1/...`，由 Vite 转发。
Go 的 `apps/api/config.yaml` 指定加载本地 `.env` 并选择独立开发库；
进程环境变量优先级最高；如改动 `API_ADDR`，须同步修改 Vite 的 `API_PROXY_TARGET`。可用 `curl.exe -i -X POST http://127.0.0.1:5173/api/v1/auth/login -H 'Content-Type: application/json' --data '{}'` 验证代理：应收到 Go 的 JSON 400 参数校验错误，而非 HTML 404。
凭据设置、完整字段和生产启动方式见 [`apps/api/README.md`](apps/api/README.md#配置加载)。
生产静态文件由 `npm run build` 输出到 `apps/web/dist`；实际部署需配置同源 `/api` 反向代理。

## 核心功能演示（2026-11-24）

前端已接入当前 Go 公开核心接口：注册/登录、团队/项目/任务 CRUD、分页、任务版本冲突处理，以及聊天会话/历史/SSE。打开 `/login` 登录；`/app/teams` 依次进入团队→项目→任务，`/chat` 查看问答。令牌只在内存，刷新后重新登录。

Go 使用独立开发库 `agentic_rag_dev`，准备方式见 [`apps/api/scripts/README.md`](apps/api/scripts/README.md)。现有本地种子账号 `alice` / `DevPassw0rd!` 仅用于开发演示。根目录运行 `npm run dev`，或分别运行 `npm run go:dev` 和 `npm run web:dev`；Web 监听 `127.0.0.1:5173` 并代理 API 到 `127.0.0.1:7800`。聊天还需准备并启动 Python Agent 及 MySQL/ES/模型配置；依赖未就绪时会显示真实后端错误。

知识库、数据源、检索测试、历史日志页面保留静态预览，不计入本轮真实功能。当前后端也不提供成员管理、任务状态流转或负责人分配的公开接口。

真实 HTTP 与 React DOM 联调已覆盖协作主流程及 401/403/409；类型检查、生产构建通过。现有测试有 4 条旧静态路由断言失败；E2E 缺少匹配版本 Chromium，真实浏览器/模型问答未验收。完整证据与限制见 [`docs/TeamFlow前端功能实现完成清单.md`](docs/TeamFlow前端功能实现完成清单.md)。以下初始化记录属于历史基线，不代表当前全部功能状态。

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
