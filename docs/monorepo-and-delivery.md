# Monorepo 与交付流程

## 工作区边界

本仓库采用单仓库、多可部署单元的布局：

```text
apps/web/   React 单页应用
apps/api/   Go API 与 Worker 两个入口
backend/    仅内部调用的 Python RAG Agent
docs/       架构、规范与运行文档
```

`apps/api` 是唯一公开业务 API；`backend` 不直接向浏览器暴露。根目录 `package.json` 只提供跨工作区命令，不引入根依赖或改变各应用现有锁文件。各单元继续独立拥有其运行时依赖与构建配置。

## 本地验证

先在 `apps/web` 执行一次 `npm ci`，再从仓库根目录运行：

```powershell
npm run check
```

该命令运行前端类型检查、单元测试和构建，Go API 测试、vet 和构建，以及 Python Agent 的语法编译检查。浏览器端到端测试因需要安装 Chromium 而单独运行：

```powershell
npm run web:test:e2e
```

## GitHub Actions

`.github/workflows/ci.yml` 在针对 `main` 的推送、拉取请求和 `v*` 标签上执行。它使用最小只读权限，并分别校验 Web、API 与 Agent。Playwright 浏览器由工作流显式安装，不依赖提交到仓库的浏览器缓存。

`.github/workflows/release.yml` 在推送 `v*` 标签时构建 Web 静态包和 Linux amd64 的 API/Worker 二进制包，随后创建同名 GitHub Release 并附加产物。也可以从 Actions 页面手动运行，并填写一个已存在的标签。该流程只发布 GitHub Release，不会部署到任何环境；部署目标、凭据和回滚策略尚未指定，不能假定存在。

发布示例：

```powershell
git tag v0.1.0
git push origin v0.1.0
```
