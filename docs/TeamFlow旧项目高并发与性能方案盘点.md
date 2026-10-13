# TeamFlow 旧项目：高并发与性能方案盘点

> 盘点日期：2026-10-12  
> 范围：`apps/TeamFlow` 的 Go 后端代码、配置、诊断脚本，以及随仓库保存的 RabbitMQ 企业实战资料。  
> 阅读方式：本文将“当前代码可见的实现”“教程/设计方案”“尚未具备的能力”分开描述。仓库中的方案和样例不等于经过生产容量验证；当前资料没有给出可信的生产 QPS、延迟 SLO 或压测基准。

## 1. 总览

旧项目主要从减少同步请求链路压力、保护数据库/缓存资源、把通知副作用异步化、控制消息消费速度、隔离慢 WebSocket 客户端和避免并发写覆盖几个方向处理并发与性能问题。

| 领域 | 方案 | 代码状态 | 主要收益 | 主要边界 |
| --- | --- | --- | --- | --- |
| MySQL | GORM SQL 连接池参数 | 已配置 | 限制并复用数据库连接 | 参数没有压测调优依据 |
| Redis | Redis 客户端连接池与超时 | 已配置 | 复用连接，避免无限等待 | 连接池容量仍需与实例数和请求量核算 |
| 热点读 | 用户、任务、列表、统计等 Redis 缓存 | 部分路径已接入 | 降低重复读数据库的压力 | 缓存一致性、故障时回源压力仍需关注 |
| 缓存击穿/雪崩 | 任务列表进程内互斥、TTL 随机抖动、空结果缓存 | 已实现但有限 | 缓解单实例同 key 回源和集中失效 | Mutex 不是分布式锁；空值有固定 TTL |
| 查询 | 分页、上限、必要字段预加载、索引 | 多条路径已实现 | 限制单次读放大 | 深页 Offset、总数 Count 仍可能昂贵；索引需按查询验证 |
| 并发写 | Task Version 乐观锁 | 已实现 | 避免并发更新互相覆盖 | 调用方要处理冲突重试/提示 |
| 副作用 | RabbitMQ 异步通知 | 已实现通知消费者链路 | 把通知工作从用户请求中解耦 | 任务指派代码走 Outbox；其他场景可靠性不同 |
| 消息吞吐 | QoS/prefetch、手动 ACK、延迟重试、停车场队列 | API 与拓扑存在；启动配置需核实 | 限制未确认消息并控制失败重试 | 当前消费者循环为串行处理；不是并行 Worker 池 |
| 发布可靠性 | Persistent、mandatory、Publisher Confirm | 已实现 | 检查 Broker 接收、路由失败和不确定结果 | 单 Publisher Channel 上锁等待 Confirm，吞吐可能受串行化限制 |
| DB 与消息一致性 | Outbox、处理记录幂等 | 业务代码和模型已接入，Relay 需核实 | 降低提交 DB 后丢通知及重复处理风险 | 是否启动 Relay/Worker 以及多实例竞争策略需检查部署入口 |
| WebSocket | Hub、每连接发送缓冲、慢客户端断开、Ping/Pong | 已实现 | 慢客户端不无限阻塞广播，检测失活连接 | Hub 广播仍逐客户端遍历；单 Hub/进程内状态 |
| 诊断 | Redis 风险扫描、同 key 并发读压测脚本 | 已提供 | 可观察缓存 TTL 风险和 DB 回源次数 | 是手动诊断工具，不是持续监控或完整容量测试 |

## 2. 数据库与连接资源

### 2.1 MySQL 连接池

当前主程序调用 `apps/TeamFlow/storage/database.go` 的 `InitDB`，在 `database/sql` 池上硬编码 `MaxIdleConns=10`、`MaxOpenConns=100`，没有应用配置文件中的连接最大生命周期。仓库另有 `apps/TeamFlow/pkg/database/mysql.go`，会读取配置并设置生命周期，但主程序当前走 `storage.InitDB`，不能把另一个初始化器的配置说成运行时生效。

这是资源上限配置，不代表 100 个连接就是最佳值。容量估算应同时考虑应用副本数、MySQL `max_connections`、单请求持有连接时间、慢查询和事务时长。若请求超过可用连接，等待池连接本身会增加尾延迟。

### 2.2 Redis 连接池与超时

`apps/TeamFlow/internal/database/redis.go` 使用配置中的 `PoolSize`、`MinIdleConns`，并设置 Dial/Read/Write 5 秒、Pool 等待 4 秒超时。配置文件默认池大小 10、最小空闲连接 5。缓存 helper 中单次 Redis 操作也有 2 秒 context timeout（`apps/TeamFlow/pkg/utils/redis_handler.go`）。

超时能限制请求被故障依赖拖住的时间，但 Redis 超时和 Redis 故障下的数据库回源仍可能把压力转移到 MySQL；应结合降级策略、并发回源控制和观测数据一起评估。

## 3. 缓存与热点读

### 3.1 缓存覆盖范围

`apps/TeamFlow/internal/cache/resource_cache.go` 提供项目、团队、任务详情、任务列表、项目列表、团队列表、任务评论列表及项目统计的缓存键和读写/失效辅助方法。详情缓存基础 TTL 为 5 分钟并叠加最多 5 分钟随机量；列表/聚合缓存基础 TTL 为 1 分钟并叠加最多 1 分钟随机量。

任务服务 `apps/TeamFlow/internal/service/task_service.go` 的列表路径会先读取 `task:list:project:{id}:page:{page}:size:{size}` 缓存，未命中后查库并回填；任务详情也先读缓存。用户资料缓存使用 `user:{id}`，TTL 由调用方传入（用户 Repository 的示例路径为 5 分钟）。

### 3.2 缓存击穿保护

- 任务列表缓存未命中时，`taskService` 获取一个服务实例内的 `sync.Mutex`，加锁后再次检查缓存，只有首个请求回源查库，其余等待并复用回填结果。
- 用户 Repository 对不存在用户写入短暂空值缓存（5 分钟），用于防止对不存在 ID 的重复查库。
- 资源缓存 TTL 加随机抖动，降低同一批缓存同时失效导致集中回源的可能。
- Redis 资源缓存读失败按 cache miss 继续访问数据库（见 `GetResourceCache` 的说明）。

边界与风险：任务列表的单把 Mutex 会让不同 project/page/size 的未命中也互相排队；其作用域只是单进程，多副本之间无法合并回源。当前代码没有看到按 key 的 singleflight 或 Redis 分布式互斥。缓存回源失败不应被误报为缓存命中；对于 Redis 故障造成的大规模回源，需要额外容量保护。

### 3.3 缓存失效与一致性

写任务后按任务 ID、项目统计及该项目任务列表前缀删除缓存；项目/团队变更也会清理关联列表。分页变体通过 Redis `SCAN` 分批找 key 后删除，而不是 `KEYS` 阻塞式扫描（`resource_cache.go`）。任务分配等关键路径在 DB 提交后失效任务缓存；部分普通写路径忽略失效错误，其他路径记录错误。缓存采用“DB 为事实源、缓存可丢弃”的模式。

需要注意：前缀失效的 key 数量增长会增加写路径耗时；DB 写成功而缓存删除失败会有短时旧数据窗口。更高负载下可考虑版本化命名空间、写后双删/延迟重删或可靠失效事件，但这些属于后续改进，本文不将其记作已有能力。

## 4. 查询路径与数据读取

### 4.1 分页和输入上限

任务服务将 page 小于 1 归一为 1，将 size 缺省设为 10，并把 size 上限限制为 100（`task_service.go`）。项目服务也限制 page size 最大 100。任务列表使用 `project_id` 过滤、按 `created_at DESC, id DESC` 排序、Offset + Limit 分页，并执行 Count（`internal/repository/task_repository_impl.go`）。评论列表和用户/项目成员查询也使用 Limit/Offset。

分页能限制单次返回量和序列化成本，但 Offset 越深仍需数据库跳过更多行；总数 Count 对大集合也可能成为成本。当前任务筛选查询的 `ListTasks` 带状态、优先级、负责人、截止日期筛选和排序字段白名单，但其查询实现没有清晰的项目范围约束，且先执行 Find 后 Count 的 query 复用行为应单独验证。它不应被视为已验证的高性能检索接口。

### 4.2 选择字段与预加载

评论列表用 GORM `Preload("User")` 并只选择用户 `id, username, avatar`，减少关联数据载荷；项目列表预加载 Owner，项目详情按需加载 Team/Owner/Members。此类按需预加载可以减少 N+1 查询和多余字段读取，但预加载集合仍应控制大小。

### 4.3 索引

模型通过 GORM tag 声明若干索引：任务 `project_id`、`assignee_id`，评论 `task_id/user_id/parent_id`，附件 `task_id`，项目 `team_id`，成员等关联表的外键/组合唯一约束，以及 Outbox 状态/事件 ID 等字段。实际索引以迁移后的 MySQL schema 为准。索引是否匹配过滤和排序需通过 `EXPLAIN`、慢查询日志验证；单列索引不一定能覆盖 `project_id + created_at + id` 组合排序。

## 5. 并发写冲突控制

Task 模型有 `Version` 字段。`taskRepository.UpdateFieldsTx` 执行 `WHERE id = ? AND version = ?` 的条件更新，并将 version 自增；影响行数为 0 时服务返回 `ErrTaskConflict`。任务分配把相同版本检查放进事务，避免两个并发写者互相覆盖。

这是乐观锁：不提前锁住记录，冲突较少时并发度较好；冲突高时调用方会收到冲突，需要刷新或按业务规则重试。当前实现可见冲突错误建模，但不能据此假设客户端已有自动重试策略。

## 6. 异步消息与通知链路

### 6.1 解耦耗时副作用

任务分配请求更新任务并记录待发事件，消费者再负责通知落库/推送（架构见 `apps/TeamFlow/README.md` 和 `docs/rabbitmq-task-assignment-pattern.md`）。将非请求关键路径的通知异步处理，可以缩短用户请求与通知处理耦合时间，并允许消费者独立扩展；它不会自动增加数据库本身的写入能力。

### 6.2 发布端控制

`internal/publisher/publisher.go` 使用持久化消息、`mandatory=true`、Publisher Confirm/Return，并对 Confirm 等待设截止时间。Channel 由一个 Publisher 独占，发送时用 Mutex 串行化 Confirm 对应关系，避免并发确认错配。可靠性更强，但单 Channel 的同步确认和全局锁会形成发布吞吐上限；要提升吞吐需在正确关联 Confirm 的前提下评估多 Channel 或批量/窗口确认。

### 6.3 消费端背压与失败隔离

- `RabbitMQ.SetQos` 可设置 prefetch count，限制未确认投递量；需要在 Worker 启动时实际调用并确认配置值。
- Consumer 使用手动 ACK；处理成功后 ACK。临时失败按 5 秒、30 秒、5 分钟延迟队列重试，超过重试次数或永久错误进入 parking 队列。
- 持久化 Exchange/Queue 和消息（Publisher）降低 Broker 重启导致的消息丢失风险。
- 当前 `Consumer_Start` 从 delivery channel 逐条同步执行 Handler，然后 ACK/重试；因此一个消费者通道上的处理是串行的。水平扩展消费者副本可以增加并行度，但需检查启动方式、幂等性、数据库连接容量和队列竞争。

### 6.4 Outbox 和消费幂等

任务分配代码在同一 DB 事务中做版本条件更新并写入 `OutboxEvent`，确保业务提交和待发事件同时成功或回滚。`internal/mq/outbox.go` 已有 Relay 的单条 claim/发布/状态更新实现，模型有唯一 EventID、状态、尝试次数、下次尝试时间等字段；通知服务将 `processed_messages` 写入业务事务用于消费幂等。教程文档进一步描述了更完整的批量 claim、租约和 Worker 运行方式。

可靠投递通常是至少一次，重复投递仍然可能发生，故幂等是必要条件。当前 `cmd/main.go` 启动消费者，但没有看到创建并运行 `OutboxRelay.DispatchOnce` 的循环；因此任务分配写入的 Outbox 记录在现有主启动流程中缺少自动发布者。Relay 代码的多实例 claim 原子性、清理/保留策略也应结合实现与集成测试核实。Outbox/幂等解决正确性，不等同于吞吐优化。

## 7. WebSocket 长连接与慢消费者

`internal/ws/Hub` 用单独事件循环管理连接，连接映射由 `RWMutex` 保护；广播 channel 缓冲 256 条。每个 Client 有独立发送 channel，写入使用非阻塞 select；广播时客户端缓冲满则关闭并移除该客户端，避免慢客户端把广播生产者无限拖住。客户端读写分离为 readPump/writePump，写方向 10 秒 deadline、Ping 每 30 秒、Pong/read deadline 60 秒、最大入站消息 512 字节。

这种设计将慢连接的压力隔离到单个连接，并及时清理失活连接。限制是广播依旧逐个遍历所有连接，开销随连接数线性增长；单 Hub 事件循环和进程内连接表不是多实例共享方案。`SendToUser` 当前在发送缓冲满或用户不在线时仍返回 nil，调用者无法可靠区分丢弃，应视为需要改进的失败可见性问题。

## 8. 诊断和验证工具

- `apps/TeamFlow/scripts/load_list_task.js` 可对真实任务列表 API 发起指定总量和并发数的读请求，并统计响应延迟/状态；可选读取 MySQL 全局 `Questions` 计数差。文档指出测试前应清理目标缓存，并在 SQL 日志中核对回源次数；单实例场景预期同 key 列表只产生一次 DB 回源，多个服务实例时进程 Mutex 不生效。
- `check_cache_avalanche.js` 通过只读 Redis SCAN/PTTL 汇总 TTL 窗口，找同时到期风险；`check_cache_breakdown.js` 采样 TTL 变化并可参考 LFU 热 key。它们只是风险信号，不会自动证明故障。
- RabbitMQ 教程设计了 Ready/Unacked、Confirm、死信队列、延迟、资源告警以及 PerfTest/端到端容量矩阵等观测方法；不能据此宣称仓库已配置生产告警或已完成容量压测。

## 9. 明确的能力边界与风险

1. 仓库是应用代码和学习/演进资料，不是经过压测验收的生产高可用部署。README 说明没有提交 Docker/Compose 编排；当前配置含本地默认地址。
2. 没有找到全局 HTTP 限流中间件的有效实现证据；响应码中有 TooManyRequests 不等于请求限流已生效。
3. 任务列表防击穿锁为单实例、跨 key 共用，服务扩容后无法跨进程合并回源。
4. 任务列表使用 Offset 和 Count；缺少明确的复合索引基准、慢查询数据及大页测试结论。
5. RabbitMQ 消费启动流程确实调用 `consumerCh.Qos(20, 0, false)`，限制该消费者通道未确认投递数为 20；但 `Consumer_Start` 的处理循环逐条同步运行，实际 Handler 并发仍为 1。
6. Publisher 单 Channel 串行等待确认是安全但可能限制吞吐的实现选择；尚无确认窗口或吞吐数据。
7. WebSocket Hub 广播对连接数线性遍历，在线状态 Redis 与连接映射需在多实例场景下进一步统一设计。
8. 没有可引用的真实生产并发数、QPS、P95/P99、故障恢复时间或容量上限数据；不能用配置数字替代性能结论。

## 10. 建议的性能验证顺序

以下是后续验证建议，不代表已在旧项目中完成：

1. 为任务列表设定真实数据规模和请求分布，分别测冷缓存、热缓存、缓存失效、Redis 故障和多实例。
2. 同时记录 API P50/P95/P99、成功率、MySQL QPS/慢查询/连接池等待、Redis 延迟/命中率/连接池等待，以及 Go CPU、堆和 goroutine。
3. 用 `EXPLAIN ANALYZE` 检查任务列表过滤与排序，比较现有索引和组合索引；测深页 Offset 与游标分页方案。
4. 检查 DB/Redis 池等待和超时，按“副本数 × 单实例连接上限”校验依赖服务连接预算。
5. 用 RabbitMQ Ready/Unacked、消费者吞吐、Handler 延迟、Confirm 延迟和 Outbox 最老记录年龄判断瓶颈，再逐步调整消费者副本与 prefetch。
6. 对 WebSocket 测连接数、广播扇出、慢连接比例、发送队列积压和断连恢复；核实 `SendToUser` 的丢弃是否可观测。
7. 用故障注入检查 Redis 不可用、Broker 重启、Confirm 超时、重复投递及进程在 DB 提交后退出时的恢复行为。

## 11. 主要源码与资料索引

- 配置与连接：`apps/TeamFlow/config/config.yaml`、`config/config.go`、`storage/database.go`、`internal/database/redis.go`
- Redis 缓存：`internal/cache/resource_cache.go`、`internal/cache/user_cache.go`、`pkg/utils/redis_handler.go`、`internal/repository/user_repository.go`
- 任务服务与并发控制：`internal/service/task_service.go`、`internal/repository/task_repository_impl.go`、`internal/model/task.go`
- 查询与索引：`internal/repository/comment_repository.go`、`internal/model/`
- RabbitMQ：`internal/mq/consumer.go`、`internal/mq/rabbitmq.go`、`internal/event/topology.go`、`internal/publisher/publisher.go`、`internal/model/outbox_event.go`、`internal/model/processed_message.go`
- WebSocket 与在线状态：`internal/ws/hub.go`、`internal/ws/client.go`、`internal/service/online.go`
- 诊断脚本：`scripts/load_list_task.js`、`scripts/README-cache-diagnostics.md`、`scripts/check_cache_avalanche.js`、`scripts/check_cache_breakdown.js`
- 消息模式说明：`README.md`、`docs/rabbitmq-task-assignment-pattern.md`、`TeamFlow企业级开发教程/RabbitMQ企业级实战/`
