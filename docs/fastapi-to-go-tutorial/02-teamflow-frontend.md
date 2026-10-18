# 教程 2：TeamFlow 协作前端（Platform → Team → Project → Task）

## 1. 第二步的目标

`01-chat-sessions-and-sse` 完成并验收后，下一步不是继续扩展聊天功能，而是把已经存在的 TeamFlow 业务能力组织成一个可操作的 React 页面闭环：

```text
Platform（工作台/当前用户）
  └── Team（团队与成员）
        └── Project（项目与项目成员）
              └── Task（任务、负责人、状态与版本）
```

本阶段的完成标准是用户可以从登录后的工作台进入团队，进入团队项目，进入项目任务，并完成真实的创建、查看、编辑、分配和删除操作。页面只能访问同源 Go `/api/v1`，不能直接访问 Python Agent 或绕过 Go 的权限检查。

## 2. 开始条件

开始写业务页面前，先完成以下短暂的接口准备工作：

1. 核对 Team、Project、Task Handler 的真实路由、请求体、响应 envelope、分页字段和错误码，并补齐 OpenAPI。
2. 确认登录、当前用户、当前团队/工作空间和受保护路由的稳定契约。刷新后按既定首版规则重新登录，不在浏览器持久化令牌。
3. 确认权限矩阵：团队 Owner/Admin/Member、项目成员权限、任务负责人资格，以及跨团队/跨项目访问的 403 行为。
4. 确认任务更新的 `version` 乐观并发语义；前端遇到冲突必须提示刷新并重新加载服务端数据，不得静默覆盖。

如果以上契约仍在变化，先完成接口审查和必要 ADR，再做页面；否则 UI 会把不稳定的后端字段固化成第二套协议。

## 3. 页面和路由切分

建议路由如下，ID 使用平台公开字符串，并把分页、筛选放在 URL：

| 页面 | 路由 | 主要职责 |
| --- | --- | --- |
| Platform 工作台 | `/app` | 当前用户、团队概览、最近项目、统一错误与空状态 |
| Team 列表 | `/app/teams` | 团队列表、创建团队、选择团队 |
| Team 详情 | `/app/teams/:teamId` | 团队信息、成员列表、成员角色、团队设置 |
| Project 列表 | `/app/teams/:teamId/projects` | 团队项目列表、创建项目、归档入口 |
| Project 详情 | `/app/teams/:teamId/projects/:projectId` | 项目概览、项目成员、任务入口 |
| Task 列表 | `/app/teams/:teamId/projects/:projectId/tasks` | 分页、状态/负责人筛选、创建与批量浏览 |
| Task 详情 | `/app/teams/:teamId/projects/:projectId/tasks/:taskId` | 编辑、状态流转、负责人、截止日期、冲突恢复 |

路由层只负责布局、参数和 feature 组合；业务请求、缓存失效和权限错误映射放在对应 feature 内部。

## 4. 前端模块边界

按现有前端规范新增公开入口：

```text
apps/web/src/features/platform/
apps/web/src/features/teams/
apps/web/src/features/projects/
apps/web/src/features/tasks/
```

每个 feature 至少包含 `api/`、`model/`、`ui/`、`routes/` 和 `index.ts`。远端列表、详情和统计由 TanStack Query 管理；表单草稿和展开状态属于组件；分页与筛选属于 URL。不要把 Team、Project、Task 数据复制到 Zustand，也不要创建一个跨业务的全局 `api.ts` 或 `store`。

推荐的实现顺序：

1. `platform`：受保护应用布局、侧边导航、当前用户和当前团队上下文。
2. `teams`：列表、创建、详情和成员管理。
3. `projects`：团队下的列表、详情、成员管理和归档。
4. `tasks`：项目下分页列表、详情、创建、编辑、分配和状态流转。
5. 统一权限、错误、空状态、加载状态和移动端导航。

## 5. 数据访问约定

为每个 feature 建立 typed API 适配器，统一使用 `shared/api` 的请求、取消、错误解码和运行时校验。Query key 必须包含作用域和查询条件，例如：

```text
['teams', actorId, page, pageSize]
['projects', actorId, teamId, page, pageSize]
['tasks', actorId, teamId, projectId, page, pageSize, filters]
```

创建、更新、删除成功后只在 feature 数据层失效或精确更新相关 Query。创建类请求超时后不得自动重放；未知结果要给出可重试前的明确提示。403、404、409 和网络失败必须分别映射为用户可理解的状态。

## 6. 关键用户旅程

先实现并验证这一条主旅程，再扩展高级交互：

```text
登录
  → Platform 工作台
  → 创建 Team
  → 添加成员
  → 创建 Project
  → 添加项目成员
  → 创建 Task
  → 分配负责人
  → 更新状态
  → 编辑时遇到 version 冲突并恢复
```

成员撤权、跨团队 URL 访问、非法负责人、空列表、重复提交、鉴权失效和网络断开属于同一阶段的必测失败路径。隐藏按钮只改善体验，后端 403 仍必须正确处理。

## 7. 分阶段交付

### 2A：应用壳与 Team

- 完成 `/app` 受保护布局、身份清理和 Team 路由。
- 接入 Team 列表、创建、详情、成员添加/移除/退出。
- 增加 Query 缓存隔离、403/404、加载/空/错误状态。

验收：新用户能创建团队并看到成员；切换用户后不残留上一用户数据。

### 2B：Project

- 完成团队项目列表、创建、详情、编辑、归档/删除。
- 完成项目成员管理和权限提示。
- 将 Team 上下文纳入所有 Project 请求和 Query key。

验收：非团队成员不能读取项目；项目撤权后刷新和直接访问都被拒绝。

### 2C：Task

- 完成项目任务分页列表和 URL 筛选。
- 完成任务创建、详情、编辑、删除、负责人和状态流转。
- 处理 `version` 冲突、非法状态和非法负责人。

验收：任务完整闭环可重复执行，冲突不会覆盖其他人的更新。

### 2D：整体验证与体验收口

- 补齐键盘导航、焦点、表单错误关联和响应式布局。
- 用 Playwright 覆盖注册/登录后的主旅程和越权旅程。
- 执行 `npm run typecheck`、`npm test`、`npm run build`、`npm run test:e2e`。

## 8. 本阶段不做的事情

- 不把聊天页面和 TeamFlow 数据复制到同一个全局状态容器。
- 不在前端实现权限规则或把权限判断当作授权。
- 不新增状态库、ORM、后端接口或实时协作基础设施。
- 不在本阶段扩展知识库、文档上传、RAG 统计或可靠异步任务；这些依赖 TeamFlow 基础页面和契约稳定后再安排。

## 9. 第二步的交付物

1. TeamFlow 前端路由和 feature 目录。
2. Team、Project、Task 的 typed API 适配层和 Query key 工厂。
3. 可重复执行的主用户旅程与越权/冲突测试。
4. 更新后的 OpenAPI、页面验收记录和实际检查结果。

完成本教程后，第三步再安排文档/知识库页面或可靠异步任务；选择哪一个取决于 TeamFlow 任务闭环的验收结果和真实产品优先级。
