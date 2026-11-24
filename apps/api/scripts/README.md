# 本地开发种子数据

给 `apps/api`（Go 后端）准备可联调的数据。数据严格按现有迁移与代码写，不新增表、不改 schema。

## 1. 前提

| 组件 | 要求 | 说明 |
| --- | --- | --- |
| MySQL | 8.4 / 9.7（本机已验证 8.4.10） | 需要 `CHECK` 约束，5.7 不适用 |
| Redis | 7.x | `apps/api` 启动时强制 `PING` |
| 开发库 | `agentic_rag_dev` | ADR-0001 要求独立开发库，不能复用旧 TeamFlow 库或 Python Agent 库 |

`.env` 里的 `DATABASE_DSN` 指向 `agentic_rag`，那是 Python Agent 的库，**没有** Go 迁移创建的表。种子数据全部写入 `agentic_rag_dev`。

## 2. 一次性准备

```powershell
# 开发库与应用账号（root 口令自行替换）
mysql -uroot -p -e "CREATE DATABASE IF NOT EXISTS agentic_rag_dev CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE USER IF NOT EXISTS 'app'@'localhost' IDENTIFIED BY 'change-me';
GRANT ALL PRIVILEGES ON agentic_rag_dev.* TO 'app'@'localhost'; FLUSH PRIVILEGES;"

# 建表（顺序不可颠倒）
cd apps/api
mysql -uapp -pchange-me -h127.0.0.1 agentic_rag_dev < migrations/0001_teamflow_core.up.sql
mysql -uapp -pchange-me -h127.0.0.1 agentic_rag_dev < migrations/0002_chat_platform_sessions.up.sql

# Redis（容器示例，必须映射端口）
docker run -d --name agentic-rag-redis -p 6379:6379 redis:7-alpine
```

## 3. 写入种子数据

```powershell
# MySQL：可重复执行（显式主键 + ON DUPLICATE KEY UPDATE）
mysql --default-character-set=utf8mb4 -uapp -pchange-me -h127.0.0.1 agentic_rag_dev < apps/api/scripts/seed_dev.sql

# Redis：每次启动前跑一次（令牌 2 小时过期）
python apps/api/scripts/seed_redis.py
```

`seed_redis.py` 只依赖标准库。它读取 `apps/api/.env` 的 `JWT_SECRET`、`REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB` 现签 HS256 令牌，再写成 `auth:user:<user_id>`。

之所以必须现签：`internal/auth/repository/auth_repository.go` 的 `Matches` 会把 Redis 里的值与请求令牌做字符串全等比较，`internal/infrastructure/security/token_signer.go` 又用 `JWT_SECRET` 校验签名与 `iss=agentic-rag`。写死令牌既会签名不符，也会两小时后失效。

## 4. 启动

```powershell
npm run go:dev
```

启动配置已迁入 `apps/api/config.yaml` 和 `internal/infrastructure/config/config.go`，由 Go 读取 `.env` 中的真实凭据，使用 YAML 的 `127.0.0.1:7800` 和 `agentic_rag_dev`。进程环境变量优先于 YAML 和 `.env`；显式进程 `DATABASE_DSN` 不会被改写库名。

旧命令 `pwsh apps/api/scripts/run-dev.ps1` 仍可使用，但仅转发到同一 Go 入口。配置字段、生产启动和回退方式见 [API README](../README.md#配置加载)。

## 5. 种子内容

**账号**：口令统一 `DevPassw0rd!`（bcrypt cost 10）。

| id | username | 昵称 | 状态 |
| --- | --- | --- | --- |
| 1 | alice | Alice Chen | 正常（团队 1 owner） |
| 2 | bob | Bob Liu | 正常（团队 1 admin、团队 2 owner） |
| 3 | carol | Carol Wang | 正常（团队 2 admin、团队 3 owner） |
| 4 | dave | Dave Zhang | 正常（团队 1 / 项目 1 member） |
| 5 | erin | Erin Sun | 正常（团队 2 / 项目 3 member） |
| 6 | frank | Frank Wu | **禁用**（登录返回 401，用于验证 status 过滤） |

**团队 3 个**：Platform Engineering(1)、Data Intelligence(2)、Mobile Experience(3)。

**项目 6 个**：团队 1 下 3 个（含 1 个 `archived`）、团队 2 下 2 个、团队 3 下 1 个。

**任务 16 个**：覆盖 `todo / in_progress / review / done` 与 `low / medium / high / none`，包含无负责人（5、9）与无截止时间（6、12、16）的记录，`version` 分布在 0–6，便于验证 409 并发冲突。

**平台会话 6 个**：覆盖 `active`（4 个）、`pending`（1 个，`AGENT_TIMEOUT`）、`failed`（1 个，`AGENT_REJECTED`），`public_id` 为 `cs_dev_*` 形式的字符串。

## 6. 验证

```powershell
TOKEN=$(docker exec agentic-rag-redis redis-cli get auth:user:1 | tr -d "`r`n")

curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:7800/api/v1/teams
curl -H "Authorization: Bearer $TOKEN" "http://127.0.0.1:7800/api/v1/teams/1/projects"
curl -H "Authorization: Bearer $TOKEN" "http://127.0.0.1:7800/api/v1/tasks?project_id=1&page=1&page_size=20"
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:7800/api/v1/chat/sessions
```

预期：alice 看到 1 个团队、3 个项目（团队 1）、6 个任务（项目 1）、3 个会话。

授权边界（已实测）：

| 场景 | 结果 |
| --- | --- |
| alice 读自己会话 | 200 |
| bob 读 alice 的会话 | 404（他人资源统一 404） |
| erin 读团队 1（非成员） | 403 |
| 用口令登录后旧令牌 | 401（同一用户仅保留最近一次登录） |
| frank（status=0）登录 | 401 |

## 7. 边界与注意事项

- **只写 Go 归属的表**。`chat_session`、`chat_message`、`agent_decision_log`、`kb_*` 属于 Python Agent（ADR-0001 / ADR-0003），Go 不跨库读写，因此本种子不含这些数据。RAG 页面目前是静态预览，不调用业务 API，也不需要这些数据。
- **重跑 `seed_dev.sql` 会覆盖种子行的内容列**（`updated_at`、`title`、`status` 等），不会删除你后来新增的行。
- **会话令牌会在 2 小时后过期**，重跑 `seed_redis.py` 即可，无需重新登录。
- **不要对 `teamflow` 库执行**：那是 TeamFlow 旧项目的库，表结构由 GORM 生成。
- 种子口令与 `change-me` 数据库口令仅用于本地，不得进入生产配置。
