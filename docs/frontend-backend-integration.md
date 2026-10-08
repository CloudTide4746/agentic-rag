# TeamFlow 前后端接口对接

本说明对应 `internal/router/router.go` 当前注册的 `/api/v1` 路由。完整机器可读契约由 Go handler 注释生成，Swagger UI 提供在线浏览与调试。

## 文档入口

- 启动后端后访问：`http://localhost:8080/swagger/index.html`（端口以 `config/config.yaml` 中的 HTTP 配置为准）。
- 生成的 Swagger 2.0 JSON：`docs/swagger.json`。
- 生成的 Swagger 2.0 YAML：`docs/swagger.yaml`。

## 基本约定

- API 前缀：`/api/v1`。
- 如尚未安装生成器，先运行 `go install github.com/swaggo/swag/cmd/swag@v1.16.6`。
- JSON 响应结构：`{ "code": 0, "message": "success", "data": ..., "timestamp": 0 }`。
- 除登录、注册外的接口使用 `Authorization: Bearer <JWT>` 请求头。登录成功响应中的 token 由前端保存，并在后续请求中发送。
- 业务错误可能仍返回 HTTP 200，调用方需要同时检查响应体中的 `code`；未认证、无权限、资源不存在和参数错误分别可能使用 HTTP 401、403、404、400。
- 上传头像、附件使用 `multipart/form-data`；附件下载是文件响应。

## 路由目录

| 功能 | 方法与路径 |
| --- | --- |
| 登录、注册 | `POST /api/v1/auth/login`、`POST /api/v1/auth/register` |
| 当前用户 | `GET/PUT /api/v1/users/me`；`PUT /api/v1/users/me/password`；`POST /api/v1/users/me/avatar` |
| 在线成员 | `GET /api/v1/online/users`；`POST/DELETE /api/v1/online/users/{userId}` |
| 团队 | `GET/POST /api/v1/teams`；`GET/PUT/DELETE /api/v1/teams/{id}`；团队成员通过 `/teams/{id}/members...` 管理 |
| 项目 | 团队项目通过 `/teams/{id}/projects` 创建和查询；项目详情、成员、统计使用 `/projects/{id}...` |
| 任务 | `GET/POST /api/v1/tasks`；任务详情、状态、负责人、附件和评论使用 `/tasks/{id}...` |
| 附件 | `DELETE /api/v1/attachments/{id}`；`GET /api/v1/attachments/{id}/download` |

参数、请求字段及响应 schema 以 Swagger 规范为准。添加或调整路由时，请同步维护 handler 注释并重新生成文档：

```powershell
swag init -g main.go --dir ./cmd,./internal/controller -o docs --parseInternal --parseDependency --outputTypes json,yaml
```
