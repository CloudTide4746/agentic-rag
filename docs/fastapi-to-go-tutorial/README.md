# FastAPI 路由迁移到 Go 教程

这组教程指导把现有 FastAPI 路由逐项迁移到 `apps/api`，Go 成为唯一公开业务入口，Python Agent 在需要 RAG 能力时只作为 Go 调用的内部运行时。

## 教程顺序

1. [会话路由与 SSE 流式问答](./01-chat-sessions-and-sse.md)：以 `GET/POST /api/chat/sessions` 和 `POST .../ask` 为例，说明契约盘点、会话归属映射、AgentClient 和 SSE 转发。
2. 后续章节按知识库/文档、决策日志、配置与检索调试路由扩展。先完成一条端到端业务链路，再开始下一类路由。

## 开始前

- 阅读 [迁移接口基线](../fastapi-to-go-api-migration.md)、[总体架构](../开发规范与技术选型.md) 和 [Go 后端规范](../../apps/api/docs/backend-engineering-standards.md)。
- 以当前 FastAPI 源码和运行时 `/openapi.json` 为行为基线；旧 OpenAPI 可能漂移。
- 当前 Go API 只有 Auth/TeamFlow 路由。本文中的 `/api/v1/chat/*` 与 SSE 示例是待实施方案，不代表仓库已有这些能力。
- 不要让浏览器直连 FastAPI，也不要让 Go 查询或写入 Python Agent 的数据库表。

## 每条路由的迁移闭环

1. 盘点旧 Handler、调用方、请求/响应样本、状态码、副作用和权限缺口。
2. 定义 Go 公共契约及 OpenAPI，再定义内部 Agent 协议（若此路由需要 Python）。
3. 在 Go 中实现边界校验、授权、应用用例、适配器和失败映射。
4. 更新前端调用与类型，验证 Go 公开入口不可绕过。
5. 小流量或开发环境切换后观察错误和数据，再移除旧调用；保留明确的回退开关/发布步骤。

具体接口遇到数据所有权、认证、公开契约破坏性变化或异步语义变化时，先按仓库规则补 ADR。
