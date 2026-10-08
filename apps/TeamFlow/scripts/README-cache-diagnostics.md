# Redis 缓存风险检查脚本

这两个脚本只执行 `SCAN`、`PTTL`、可选的 `--hotkeys`，不会写入或删除 Redis 数据。默认连接与项目 `config/config.yaml` 一致：`localhost:6379`、DB `0`。

前提是本机安装了与 Redis 服务端兼容的 `redis-cli`，并能在终端执行 `redis-cli --version`。若它不在 PATH，设置 `REDIS_CLI` 为可执行文件的绝对路径即可：

```powershell
$env:REDIS_CLI = 'C:\path\to\redis-cli.exe'
```

```powershell
node scripts/check_cache_avalanche.js
node scripts/check_cache_breakdown.js --observe-seconds 30
```

若 Redis 有密码，优先以环境变量传递，避免密码出现在命令历史中：

```powershell
$env:REDIS_PASSWORD = 'your-password'
node scripts/check_cache_avalanche.js --pattern 'task:*'
```

常用选项：`--host`、`--port`、`--db`、`--pattern`、`--max-keys`。默认最多扫描 2,000 个 key；生产库较大时应先用业务前缀缩小范围，例如 `task:*`、`project:*` 或 `team:*`。

`check_cache_avalanche.js` 按 TTL 窗口汇总。某个窗口聚集很多 key，说明这些 key 可能同时失效；它是风险信号，不等于已发生雪崩。

`check_cache_breakdown.js` 在观察窗口的前后各采样一次，TTL 增大代表该 key 在窗口内被回填/重建。结合 `--hotkeys` 的高访问 key 才能判断其是否真是击穿风险。`--hotkeys` 依赖 Redis 的 LFU 淘汰策略；未启用时不能据此下结论。

## ListTask 并发验证

`load_list_task.js` 不访问 Redis，直接并发调用项目实际的 `GET /api/v1/tasks` 接口。它用于在你查看 MySQL 慢查询日志或 GORM SQL 日志时制造可控的同 key 流量：

```powershell
$env:API_TOKEN = '<登录后得到的 JWT>'
$env:PROJECT_ID = '1'
node scripts/load_list_task.js --requests 100 --concurrency 20 --db-metrics
```

脚本只发读请求，不会修改任务数据。`--db-metrics` 会在压测前后执行 `SHOW GLOBAL STATUS LIKE 'Questions'`；它需要本机可执行 `mysql`，并读取项目根目录 `.env`（或优先读取已设置的 `DB_PASSWORD`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_NAME` 环境变量）。它是实例全局计数，测试期间应避免其他业务流量，否则差值会被污染。

要验证互斥锁，请先让该任务列表缓存失效，再运行脚本；随后查看数据库日志中 `tasks` 列表 SQL 的次数。单实例服务预期只有一次列表回源；若启动了多个 Go 服务实例，当前 `sync.Mutex` 不跨实例生效。
