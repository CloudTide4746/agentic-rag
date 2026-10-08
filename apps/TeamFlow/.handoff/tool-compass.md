# 工具罗盘
本项目适用的 Skills 和工具。由强模型在项目启动时填写，发现新工具时更新。

## 推荐 Skills
| Skill 名称 | 适用场景 |
|-----------|---------|
| `using-superpowers` | **任何对话开始前必调** — 强制检查是否有相关 skill 适用 |
| `writing-plans` | 接到 spec/需求时，动手前先写多步实现计划（已用于 3 个 user-* plan） |
| `subagent-driven-development` | 按现有 `docs/superpowers/plans/*.md` 计划逐任务执行（推荐） |
| `executing-plans` | 同上，但保持单线执行（不派 subagent） |
| `requesting-code-review` | 写完一段代码后请求评审 |
| `qa` | 用户报告 bug / 想要交互式提单 |
| `systematic-debugging` | 排查具体技术 bug |
| `test-driven-development` | 写新业务逻辑时先写测试 |
| `finishing-a-development-branch` | 收尾：merge / cleanup / PR |
| `git-guardrails-claude-code` | Git 提交安全保护（防止 hard reset / 误删） |
| `triage` | Issues 分类（needs-triage / ready-for-agent 等 label 已在 docs/agents/triage-labels.md 定义） |
| `smart-handoff` | **本次会话已用** — 弱模型扫描、强模型直接读简报 |
| `handoff` | 上下文接力（与 smart-handoff 互补） |

## 关键命令
- 编译验证：`go build ./...`
- 依赖整理：`go mod tidy`
- 运行：`go run cmd/main.go`
- 测试（待写）：`go test ./...`
- 测试 main 中的演示代码：`go run cmd/main.go -test`（需 MySQL 可用）
- Swagger（待装 swag 后）：`swag init -g cmd/main.go`

## 注意事项
- **DB 入口不统一**：`pkg/database/mysql.go` 是空文件，实际 DB 在 `storage/database.go`（包名 storage）。所有 Repository/Service 直接用 `storage.DB`。CLAUDE.md 描述的 `pkg/database/` 是预期结构但尚未迁移。
- **AutoMigrate 未启用**：`cmd/main.go` 注释掉了 `storage.InitDB` 调用，验证完 `database.AutoMigrate` 后再启用。
- **.env 中 DB_PASSWORD 已硬编码**（`"140322Bk"`），生产前需迁移到密钥管理。
- **设计文档约定**：所有 plan 写到 `docs/superpowers/plans/YYYY-MM-DD-<name>.md`，对应 spec 写到 `docs/superpowers/specs/YYYY-MM-DD-<name>-design.md`。已有 3 套成对文件。
- **教程是真理源**：业务方向不明确时优先翻 `TeamFlow企业级开发教程/00-总目录.md` 对应章节。
- **空指针陷阱**：Team/Project/Comment/Attachment/Notification 的 Repository/Service/Controller 都不存在，强模型接手时不要假设这些模块的脚手架已经就绪。
