# RabbitMQ 官方资料研究笔记（TeamFlow / Go / Gin）

> 调研基准日：2026-08-25  
> 范围：只使用 RabbitMQ 官方文档、RabbitMQ 官方 GitHub 仓库、AMQP 0-9-1 规范，以及 Docker Official Image 的一手说明。本文是后续教程的事实底稿，不是最终实现步骤。

## 1. 先给结论：本项目应采用的基线

- 截至基准日，RabbitMQ 最新且仍受社区支持的系列是 **4.3**，最新补丁为 **4.3.5（2026-08-17）**；4.3 的社区支持截止 2026-11-30。4.2 的社区支持已经在 2026-07-31 结束，因此新项目不要从 4.2 起步。[RabbitMQ Release Information](https://www.rabbitmq.com/docs/versions)
- 本地开发镜像固定为 `rabbitmq:4.3.5-management`，不要用会漂移的 `latest`。Docker Official Image 已发布 `4.3.5-management` 和 `4.3.5-management-alpine` 标签；management 变体默认包含管理插件，AMQP 端口通常为 5672、管理 UI 为 15672。[Docker Official Image README](https://github.com/docker-library/docs/blob/master/rabbitmq/README.md) [RabbitMQ official image sources](https://github.com/docker-library/rabbitmq)
- Go 客户端选择 RabbitMQ 核心团队维护的 **`github.com/rabbitmq/amqp091-go`**，基准日最新版本为 **v1.14.0（2026-08-18）**；不要从旧教程复制已归档的 `streadway/amqp`。[官方客户端 README](https://github.com/rabbitmq/amqp091-go/tree/v1.14.0) [v1.14.0 release](https://github.com/rabbitmq/amqp091-go/releases/tag/v1.14.0)
- 可靠主链路应是：**持久化消息 + durable quorum queue + publisher confirms + `mandatory=true`/return 处理 + consumer manual ack + 明确 prefetch + 消费幂等**。其中任何一个机制都不能单独提供端到端 exactly-once；RabbitMQ 官方可靠性文档明确指出网络断开时在途消息需要重传，而消费者必须准备处理重投。[Reliability Guide](https://www.rabbitmq.com/docs/reliability) [Consumer Acknowledgements and Publisher Confirms](https://www.rabbitmq.com/docs/confirms)
- TeamFlow 已使用 MySQL/GORM，涉及“数据库更新成功且事件最终一定发出”的业务（如任务指派、状态变更）应使用 **transactional outbox**；这不是 RabbitMQ 自带事务，而是本地数据库事务 + 异步 relay + publisher confirm 的应用架构。confirm 与 DB 事务不能形成跨资源原子提交，relay 在“发布已成功、sent 状态尚未提交”之间仍可能崩溃，所以消费者仍必须幂等。这是根据 AMQP/RabbitMQ 的事务与故障边界作出的架构推论。[AMQP 0-9-1 specification](https://github.com/rabbitmq/amqp-0.9.1-spec) [Go client channel transaction API](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/channel.go) [Reliability Guide](https://www.rabbitmq.com/docs/reliability)

## 2. 版本、镜像与升级策略

### 2.1 新环境

本地教程应固定完整补丁版本 `4.3.5-management`。固定补丁版本能让学员、CI 和故障复现实验使用相同 broker；升级应是一次显式变更，而不是容器重建时意外跟随浮动标签。Docker 镜像的数据目录为 `/var/lib/rabbitmq`，数据路径包含节点名，因此容器要设置稳定 hostname，并挂载命名卷。[Docker Official Image README](https://github.com/docker-library/docs/blob/master/rabbitmq/README.md)

开发环境可通过 `RABBITMQ_DEFAULT_USER`、`RABBITMQ_DEFAULT_PASS`、`RABBITMQ_DEFAULT_VHOST` 初始化非 guest 用户和专用 vhost；这些仅适合开发便利，生产凭据应由 Secret 管理，且不应提交到仓库。[Docker Official Image README](https://github.com/docker-library/docs/blob/master/rabbitmq/README.md) [Access Control](https://www.rabbitmq.com/docs/access-control)

### 2.2 升级必须写进运维设计

- 官方推荐 rolling/in-place upgrade；逐节点升级前应核对版本可滚动升级、Erlang 要求、目标版本 release notes，并确保全部 stable feature flags 已启用、无 alarms、无队列/stream 副本同步、集群负载可承受少一台节点。安全要求更高或不存在滚动路径时使用 blue-green。[Upgrade Guide](https://www.rabbitmq.com/docs/upgrade)
- **4.3.x 只能从 4.2.x 直接升级**；3.13.x 必须先到 4.2.x，再到 4.3.x。升级前未启用所有 stable feature flags 可能直接导致升级失败。[Upgrade Guide](https://www.rabbitmq.com/docs/upgrade)
- RabbitMQ 不正式支持 downgrade，不能把回滚计划写成“装回旧包”；需要可回退性时用 blue-green，并保留旧集群直到迁移验证完成。[Upgrade Guide](https://www.rabbitmq.com/docs/upgrade)
- 三节点滚动升级期间至少始终保留两个节点可用，并预先证明剩余容量足够。升级前备份节点数据目录，但“备份存在”不等于 blue-green 的快速业务回切能力。[Upgrade Guide](https://www.rabbitmq.com/docs/upgrade)

## 3. AMQP 0-9-1 心智模型与 TeamFlow 拓扑

AMQP 0-9-1 的核心路径不是“生产者直接写队列”，而是：

```text
publisher -> exchange --binding/routing key--> queue -> consumer
```

exchange 负责路由，queue 保存消息，binding 定义二者的关系。常用 exchange 类型：direct 精确匹配 routing key；topic 按点分段模式匹配；fanout 广播且忽略 routing key；headers 按消息头匹配。默认 exchange 名称为空字符串，是预声明的 direct exchange，每个队列在声明时都会获得“routing key = queue name”的隐式绑定。[AMQP Concepts](https://www.rabbitmq.com/tutorials/amqp-concepts) [AMQP 0-9-1 specification](https://github.com/rabbitmq/amqp-0.9.1-spec)

面向 TeamFlow 的推荐起步拓扑（项目建议，不是协议要求）：

```text
topic exchange: teamflow.events.v1
  task.assigned.v1       -> teamflow.notification.websocket.q
  task.status.changed.v1 -> teamflow.notification.websocket.q
  task.*.v1              -> teamflow.audit.q（后续阶段）

DLX: teamflow.dlx.v1
  -> teamflow.notification.dead.q
```

事件 envelope 至少包含 `message_id/event_id`、`event_type`、`schema_version`、`occurred_at`、`producer`、`correlation_id/trace_id` 和 payload。事件名/version 一旦有消费者依赖就属于契约；破坏性 schema 变更用新版本而不是静默覆盖。

客户端应在 publish/consume 前声明它依赖的 exchange、queue 和 binding。相同声明是可重复的；若同名实体已经存在但 durable、type、arguments 等属性不等价，broker 会返回 channel-level error，该 Channel 随即失效，必须丢弃后新建。声明操作使用 `noWait=false` 才能在启动期看到失败。[Go client package documentation](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/doc.go) [Queues Guide](https://www.rabbitmq.com/docs/queues) [Exchanges Guide](https://www.rabbitmq.com/docs/exchanges)

应用声明与平台声明二选一作为 durable topology 的唯一权威源：小项目可由一个 initializer 声明；Kubernetes 上可由 Messaging Topology Operator 管理。不要让多个服务用不同参数争抢同名队列。DLX、TTL、length limit、delivery limit 等可变参数优先使用 policy，因为 RabbitMQ 官方强烈反对把可变策略硬编码成 `x-arguments`：后者通常要删队列、重部署才能修改。[Dead Letter Exchanges](https://www.rabbitmq.com/docs/dlx) [Optional Queue Arguments](https://www.rabbitmq.com/docs/queues#optional-arguments)

## 4. Go 客户端的连接、Channel 与恢复约束

### 4.1 连接与 Channel 所有权

- AMQP connection 应长连接复用，不要按请求或按消息创建。一个 TCP connection 多路复用多个 AMQP Channel。[Channels Guide](https://www.rabbitmq.com/docs/channels) [Networking Guide](https://www.rabbitmq.com/docs/networking)
- `amqp091-go` v1.14.0 明确写明：**Channel 不是线程安全的，不要在多个 goroutine 间共享同一个 Channel**；并发调用可能产生竞态或不可预测结果。工程上让每个 publisher worker/consumer owner 独占 Channel，或用有界 channel pool；不要只用一个 Channel 给所有 Gin 请求并发 Publish。[`Connection.Channel` source documentation](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/connection.go#L1139-L1147)
- publisher 和 consumer 最好使用分开的 connection。RabbitMQ/Go 客户端说明指出 broker 通常通过 TCP backpressure 限流；同一 connection 上的过量 publish 会拖慢其所有 Channel，甚至延迟 consumer ack。[Go client `Channel.Flow` / `Consume` documentation](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/channel.go) [Flow Control](https://www.rabbitmq.com/docs/flow-control)
- 任一 Channel 方法返回协议错误后，该 Channel 不再有效，应关闭/丢弃并新建；`NotifyClose`、`NotifyReturn`、`NotifyPublish` 等异步通知必须持续有接收者，否则会阻塞同步操作。通知 Go channel 应有足够 buffer。[Go client package documentation](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/doc.go)

### 4.2 自动恢复是“传输恢复”，不是消息语义恢复

`amqp091-go` v1.14.0 已支持可选的自动 connection/channel 重连和 topology/consumer recovery；`Config.Recovery == nil` 时默认关闭。恢复模式可以恢复全部已跟踪拓扑、仅 transient 拓扑，或禁用 topology recovery。[Go client `Config.Recovery`](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/connection.go) [Go client recovery implementation](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/recovery.go) [v1.14.0 release](https://github.com/rabbitmq/amqp091-go/releases/tag/v1.14.0)

仍不能把自动恢复当作“发布不丢不重”的保证：网络断开时，未收到 confirm 的 publish 到底有没有被 broker 接受存在不确定窗口；应用必须保留未确认消息并按策略重发，重发可能造成重复。未 ack 的消费会被 requeue/redeliver。因此恢复后仍需要 confirm 状态机、outbox 和消费幂等。[Reliability Guide](https://www.rabbitmq.com/docs/reliability) [Publisher Confirms](https://www.rabbitmq.com/docs/confirms)

注意 v1.14.0 是基准日前一周发布的新版本，且 release notes 包含多项 auto-recovery correctness 修复。教程应固定 v1.14.0、对 broker restart/网络中断做集成测试，不应把旧版客户端“完全不做自动恢复”的经验或新版 API 混写。[v1.14.0 release](https://github.com/rabbitmq/amqp091-go/releases/tag/v1.14.0)

## 5. Publisher：confirm、mandatory return 与持久性

### 5.1 正确的成功定义

`Publish` 返回 nil 只表示调用/写 socket 没有立即失败，不表示 broker 已接收、已路由或已持久化。进入 confirm mode 后，broker 以 `basic.ack/basic.nack` 异步确认发布；应用应维持有界 in-flight 窗口并持续消费 confirm，批量/异步等待比每条同步等待有更高吞吐。[Publisher Confirms](https://www.rabbitmq.com/docs/confirms) [Go client publish documentation](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/channel.go)

对 persistent message：只有消息投递到 durable queue，并且 publisher 收到 confirm，才能确认 broker 已完成该次持久化责任；对 quorum queue，confirm 要等消息被 quorum 副本接受。持久消息的磁盘刷写可能成批进行，confirm 延迟可能达到数百毫秒，故需要异步 confirm 或 batch，而不是把每条消息 RTT 串行化。[Publisher Confirms: confirmation timing](https://www.rabbitmq.com/docs/confirms#when-publishes-are-confirmed) [Quorum Queues](https://www.rabbitmq.com/docs/quorum-queues)

### 5.2 confirms 不能发现“路由到了零个队列”

不可路由的消息也可能收到 publisher ack。需要在 Publish 时设置 `mandatory=true`，并注册 `NotifyReturn` 处理 `basic.return`；否则 routing key 写错、binding 缺失可能静默丢业务消息。对 mandatory 且不可路由的消息，RabbitMQ 会先发 `basic.return` 再发 confirm，但 `amqp091-go` README 明确不保证分别投递到两个 Go channel 的 return/ack 跨 channel 顺序，因此业务不能靠接收顺序判断，应以自己的 publish sequence/message id 关联并让两个监听器始终运行。[Publisher Confirms](https://www.rabbitmq.com/docs/confirms#when-publishes-are-confirmed) [Go client README: non-goals](https://github.com/rabbitmq/amqp091-go/tree/v1.14.0) [Go client `NotifyReturn`](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/channel.go)

建议发布状态至少区分：`pending`、`confirmed`、`nacked`、`returned`、`connection_lost_unknown`。连接断开时所有尚未 confirm 的项进入 unknown/retry，而不是直接标记失败或成功。

## 6. Consumer：manual ack、prefetch、重投与幂等

- 生产业务应 `autoAck=false`，只有当所需副作用已经持久化/提交后才 `Ack(false)`。自动 ack 在消息写入网络前就视为成功，consumer 随后崩溃会丢消息，也没有未 ack 窗口限制，可能压垮客户端内存。[Consumer Acknowledgements](https://www.rabbitmq.com/docs/confirms#acknowledgement-modes)
- delivery tag 只在 Channel 内有效，必须在收到消息的同一 Channel ack；跨 Channel ack 会产生 `unknown delivery tag` 并关闭 Channel。并行处理时最简单可靠的是每条 `Ack(false)`；若要 `multiple=true` 批量 ack，必须由单一有序协调器串行维护连续完成水位，不能让 worker 并发 multi-ack。[Delivery Tags](https://www.rabbitmq.com/docs/confirms#delivery-identifiers-delivery-tags) [Go client delivery API](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/delivery.go)
- 手工 ack 模式下，connection/channel 关闭会自动 requeue 未确认消息，所以消费者必须按至少一次投递设计。`Redelivered`/计数头可用于诊断，但不能作为“没见过”证明；真正的防重依据应是稳定的 `message_id/event_id` 和原子去重记录。[Consumer Acknowledgements](https://www.rabbitmq.com/docs/confirms)
- 明确调用 `Qos(prefetchCount, 0, false)`，并在 Consume 前设置。RabbitMQ 对 AMQP `basic.qos` 做了每 consumer 扩展；`global=false` 是常用的每 consumer 限制，跨队列/Channel 的 global limit 协调成本更高。prefetch 不是越大越好：它界定 broker 允许的未 ack 窗口，也影响客户端内存、负载公平性和故障时的重复批量。[Consumer Prefetch](https://www.rabbitmq.com/docs/consumer-prefetch) [Go client `Channel.Qos`](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/channel.go)
- 不要对瞬时故障无限 `Nack(requeue=true)`：消息可能立即再次投递，形成高 CPU/网络占用的 requeue/redelivery loop。需要退避时转入延迟重试队列或重新发布时间，而不是原队列热循环。[Negative Acknowledgement and Requeueing](https://www.rabbitmq.com/docs/confirms#negative-acknowledgement-and-requeueing)

幂等实现建议（TeamFlow）：为每个消费业务建立 `processed_messages(consumer_name, message_id, processed_at)`，对 `(consumer_name, message_id)` 加唯一键；在同一个 MySQL 事务里“插入去重键 + 写 Notification/业务副作用”，唯一键冲突视为已完成并 ack。若副作用是外部 HTTP/邮件/WebSocket，仍需把可重试状态本地持久化，不能只靠内存去重。

## 7. Quorum queue、delivery-limit、DLX 与 TTL 的关键陷阱

### 7.1 Quorum queue 适用范围

quorum queue 基于 Raft，是需要复制、高可用和数据安全的长期队列的默认选择。publisher confirm 只有在消息复制到多数成员后才发出；consumer 应使用 manual ack。最小有实际容错意义的 group size 是 3，可容忍 1 个节点故障，并推荐奇数成员数。[Quorum Queues](https://www.rabbitmq.com/docs/quorum-queues)

它不适合临时/exclusive queue、高频创建删除、追求最低延迟、数据安全不重要、超长积压（官方提示 5M+ 消息通常应评估 stream）或大 fanout。quorum queue 总是 durable；RabbitMQ 4.0 起 classic queue mirroring 已移除。[Quorum Queues: when not to use](https://www.rabbitmq.com/docs/quorum-queues#when-not-to-use-quorum-queues)

### 7.2 RabbitMQ 4.3 的 delivery-limit 语义改变

- RabbitMQ 4.0 起 quorum queue 默认 delivery limit 为 20；超过限制的消息会丢弃，配置 DLX 时则 dead-letter。官方建议 quorum queue 配置 DLX，避免无意丢失。[Quorum Queues: poison message handling](https://www.rabbitmq.com/docs/quorum-queues#poison-message-handling)
- **4.3 起 delivery limit 根据 `x-delivery-count`（真实失败）而不是 acquired count 判定。AMQP 0-9-1 `basic.nack` 作为显式退回不再增加 delivery-count，而 `basic.reject`、客户端崩溃/连接丢失会增加。**因此不要假设默认 20 能截断 Go 代码的 `Nack(requeue=true)` 热循环；应用重试次数/延迟仍需显式设计。`x-acquired-count` 更适合观察“被分配给 consumer 的次数”，但分配不代表应用代码一定看到了消息。[RabbitMQ 4.3 poison message semantics](https://www.rabbitmq.com/docs/quorum-queues#poison-message-handling)
- `x-delivery-limit=-1` 可恢复无限失败重投，但官方不推荐，因为反复 requeue 会威胁 queue/cluster 稳定性。[Quorum Queues](https://www.rabbitmq.com/docs/quorum-queues#poison-message-handling)

### 7.3 DLX 默认并非可靠转移

默认 dead-letter 是 broker 内部无 confirms 的重新发布，源消息发出后立即移除；目标队列不可用时可能丢失。quorum queue 可启用 `dead-letter-strategy=at-least-once`，但还必须使用 `overflow=reject-publish`、配置 DLX，并满足相应 feature flag；官方还建议配置 max-length/max-length-bytes 防止目标异常时源队列无限堆积。[Dead Letter Exchanges: re-publishing](https://www.rabbitmq.com/docs/dlx#re-publishing-with-publisher-confirms) [Quorum Queues: at-least-once dead lettering](https://www.rabbitmq.com/docs/quorum-queues#activating-at-least-once-dead-lettering)

at-least-once DLX 会占更多 CPU/内存；目标 exchange 不存在、消息无法路由、任一目标队列不 confirm 时，源 quorum queue 会保留并周期重试，目标端仍可能看到重复。若 DLX 根本不存在，普通 DLX 路径会静默丢弃消息。源队列用户还需要对源 queue 有 read、对 DLX 有 write 权限。[Dead Letter Exchanges](https://www.rabbitmq.com/docs/dlx) [Quorum Queues: DLX caveats](https://www.rabbitmq.com/docs/quorum-queues#activating-at-least-once-dead-lettering)

RabbitMQ 会检测没有发生 rejection 的 dead-letter cycle，并丢弃循环消息；因此不能把循环 DLX 当无限重试器。[Dead Letter Exchanges: cycles](https://www.rabbitmq.com/docs/dlx#dead-letter-cycle)

### 7.4 TTL 延迟重试 caveat

TTL + DLX 可以实现固定档位延迟队列，但每消息 TTL 不是通用精确定时器：quorum queue 的过期消息到达队首时才 dead-letter；过期消息可能在未过期消息后面占用资源并仍计入队列统计。队列级 TTL 与消息级 TTL 同时存在时取更小值。[Time-to-Live](https://www.rabbitmq.com/docs/ttl)

RabbitMQ **4.3** 新增 quorum queue delayed retry，可通过 policy/queue arguments 设置 `delayed-retry-type`（`all`/`failed`/`returned`）、`delayed-retry-min`、`delayed-retry-max`，按 `min(min_delay * delivery_count, max_delay)` 做线性退避，而且 queue arguments 可由 AMQP 0-9-1 声明。这比原队列热 requeue 更合适。[Quorum Queues: Delayed Retry](https://www.rabbitmq.com/docs/quorum-queues#delayed-retry)

但 4.3 中 `basic.nack` 不增加 delivery-count，所以 nack 即使被 `all`/`returned` 延迟，也不会让后续延迟随失败次数增长，可能一直停留在最小 delay；`failed` 则不会覆盖 nack。教程应显式选择策略并用集成测试证明行为，不能从 4.2 或更旧教程推断。若需固定档位可继续用少量 TTL retry queues（如 10s/1m/10m），但不要为每条消息声明队列，也不要承诺毫秒级准时。最终失败进入独立 dead queue，配套告警和人工/自动 replay；replay 必须保留原始 event id 以维持幂等。[RabbitMQ 4.3 poison message semantics](https://www.rabbitmq.com/docs/quorum-queues#poison-message-handling)

## 8. Transactional outbox 的准确边界

RabbitMQ 的 AMQP Channel 事务只覆盖 broker 协议操作，不能与 MySQL/GORM 的本地事务组成原子提交；Go 客户端还注明跨多个队列的事务原子性未定义，queue declaration/binding 也不在事务中。publisher confirms 是更合适的 broker 接收证明，但同样不能提交 MySQL。[Go client transaction API](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/channel.go) [Publisher Confirms](https://www.rabbitmq.com/docs/confirms)

TeamFlow 的正确边界：

```text
Gin request
  -> MySQL transaction:
       update task
       insert outbox_event(status=pending, event_id=UUID, payload=JSON)
  -> commit

outbox relay
  -> claim pending row (支持并发/租约)
  -> publish persistent + mandatory
  -> 收到非 returned 的 positive confirm
  -> mark published
```

仍存在不可消除的 crash window：confirm 已到但 `mark published` 未提交，重启后会重发。因此 outbox 提供的是“业务写与待发事件不分裂 + 最终至少一次发布”，不是 exactly-once；consumer 必须用 event id 幂等。对永久 `basic.return`（配置/拓扑错误）不要无限重试，应进入 failed 并告警；对 connection-lost/timeout 则按未知结果重试。这一段是从官方确认与重投语义推导出的应用架构结论。[Reliability Guide](https://www.rabbitmq.com/docs/reliability) [Publisher Confirms](https://www.rabbitmq.com/docs/confirms)

## 9. 优雅停机

Publisher 停机顺序：停止接收新的发布请求/停止 outbox claim；等待正在处理的发布得到 return/confirm 或落回 pending；再关闭 Channel，最后关闭 Connection。官方 Go 客户端建议关闭前等完所有 confirms；确认通知 channel 的 buffer 至少覆盖最大未确认数并持续消费，避免死锁。[Go client package documentation](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/doc.go) [Go client confirm API](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/channel.go)

Consumer 停机顺序：先停止新 delivery（`Channel.Cancel(consumerTag, false)` 或取消 `ConsumeWithContext`）；继续 drain delivery channel；等待 worker 完成并 ack/nack 所有已接收消息；最后关闭 Channel/Connection。`noWait=true` 取消可能让在途消息在客户端被丢弃而未 ack，官方仅建议在确定没有需确认的在途 delivery 时使用。[Go client `Channel.Cancel` and `Consume`](https://github.com/rabbitmq/amqp091-go/blob/v1.14.0/channel.go)

必须给停机设置 deadline；deadline 到达时关闭连接，让 broker 自动 requeue 未 ack 消息，而不是错误地 ack 未完成工作。Kubernetes 的 `terminationGracePeriodSeconds` 要大于应用 drain deadline。

## 10. TLS、用户、权限与 vhost

- 生产删除默认 `guest` 用户；官方建议每个应用使用独立用户，便于审计、连接归属和最小权限。`guest` 默认只允许 localhost，但解决方式不是开放远程 guest，而是创建新用户。[Production Checklist](https://www.rabbitmq.com/docs/production-checklist) [Access Control](https://www.rabbitmq.com/docs/access-control)
- 每个环境至少独立 vhost，例如 `/teamflow-dev`、`/teamflow-staging`、`/teamflow-prod`。vhost 是资源和权限的逻辑隔离，不提供物理资源隔离；生产多租户仍需结合 per-vhost limits/operator policies，强隔离需求使用独立集群。[Virtual Hosts](https://www.rabbitmq.com/docs/vhosts)
- 权限是 vhost 内的 configure/write/read 三个正则。运行时用户只授予所需 exchange/queue 的 write/read；拓扑 initializer 若需要 declare 再给受限 configure；监控、运维用户分开。DLX 还要求源 queue read + DLX write。[Access Control](https://www.rabbitmq.com/docs/access-control) [Dead Letter Exchanges](https://www.rabbitmq.com/docs/dlx)
- 生产使用 `amqps://`，启用服务端证书链验证和 hostname verification；需要客户端证书身份时才启用 mTLS。RabbitMQ TLS listener 常用 5671，可完全关闭明文 listener。mTLS 必须 broker 与客户端两端共同正确配置，单边配置不构成双向验证。[TLS Support](https://www.rabbitmq.com/docs/ssl)
- 凭据、CA、证书和私钥由环境 Secret 注入，不进 YAML/仓库；管理 UI、Prometheus endpoint 和 inter-node/CLI 端口也要网络隔离、认证和 TLS。[Production Checklist](https://www.rabbitmq.com/docs/production-checklist) [TLS Support](https://www.rabbitmq.com/docs/ssl)

## 11. 监控、告警与 SLO

启用内置 `rabbitmq_prometheus` 插件；默认指标端点为 `:15692/metrics`。RabbitMQ 官方推荐 Prometheus + Grafana 做长期监控，management UI 只提供部分指标且不是长期采集方案。[Prometheus and Grafana](https://www.rabbitmq.com/docs/prometheus)

大规模环境默认使用 aggregated `/metrics`：响应大小不随连接/队列对象数线性膨胀。`/metrics/per-object` 或 `prometheus.return_per_object_metrics=true` 在大量对象时会产生巨大 payload 和 CPU 序列化开销；需要定位具体队列时优先按所需 metric group/filter 使用 `/metrics/detailed`，内存分解端点也应低频抓取。[Prometheus metrics aggregation](https://www.rabbitmq.com/docs/prometheus#metric-aggregation)

至少建立四层指标：

1. 基础设施：CPU、内存、磁盘容量/IO 延迟、网络 RTT/丢包、文件描述符。
2. RabbitMQ 节点：节点存活、memory/disk alarms、连接/Channel 数与 churn、Erlang process/file descriptor 使用率、cluster partition/leader election、quorum 可用成员。
3. 队列：ready、unacknowledged、publish/deliver/ack/redelivery rate、consumer 数/consumer capacity、队列增长斜率、dead queue 深度、unroutable/returned。
4. TeamFlow 应用：publish confirm latency、nack/return/unknown 数、reconnect/recovery 次数、consumer processing latency/error、outbox pending/oldest age/retry、幂等冲突数。

RabbitMQ 官方监控指南明确把 `messages_ready`、`messages_unacknowledged`、publish/delivery rate 等列为关键 queue 指标；监控必须同时覆盖基础设施、broker 和应用，单看“进程还活着”不足以判断服务健康。[Monitoring Guide](https://www.rabbitmq.com/docs/monitoring)

告警应基于趋势和业务时限，而不只是静态队列长度。例如：outbox oldest age 超过事件投递 SLO；ready 持续增长且 deliver < publish；unacked 长时间高位；任何 returned/nack；dead queue 非零；资源 alarm；quorum 失去成员；confirm p99 突升。健康探针不要频繁新建 AMQP 连接，否则会制造 connection churn 和额外负载。[Monitoring Guide](https://www.rabbitmq.com/docs/monitoring)

## 12. 容量规划与压测

官方生产清单给出的每节点最低参考是 4 CPU cores、4 GiB RAM，并强调 RabbitMQ 不应与其他磁盘/网络 IO 密集服务共置；这只是底线而不是本项目容量答案。生产至少准备 50K 文件描述符，官方估算方法是并发连接 p95 × 2 + queue 总数，再留余量。[Production Checklist](https://www.rabbitmq.com/docs/production-checklist)

使用 RabbitMQ 官方 **PerfTest** 做 broker 基线，再用 TeamFlow 自己的 Go producer/consumer 做端到端压测。[rabbitmq-perf-test](https://github.com/rabbitmq/rabbitmq-perf-test) [PerfTest documentation](https://perftest.rabbitmq.com)

压测矩阵必须与生产语义一致：quorum/classic 类型、副本数、消息大小分布、persistent、publisher confirms 的 in-flight 窗口、mandatory、consumer 数、prefetch、处理耗时/失败率、队列数量、TLS、跨 AZ RTT。至少记录吞吐、publish confirm p50/p95/p99、端到端 p99、ready/unacked、CPU、内存、磁盘 IO/fsync、网络、GC 和 redelivery；不能拿 transient/no-confirm 的峰值代表可靠链路容量。

容量验收还要做稳态和故障态：持续负载数小时；生产停发后 backlog drain；单节点宕机/恢复；leader re-election；broker 滚动重启；连接闪断；磁盘/内存 alarm；consumer 全停后恢复；DLX 目标不可用。官方强调升级时集群要能在少一节点时承载负载，quorum confirm 又必须等待多数副本，因此只测健康峰值不够。[Upgrade Guide](https://www.rabbitmq.com/docs/upgrade) [Quorum Queues](https://www.rabbitmq.com/docs/quorum-queues) [Production Checklist](https://www.rabbitmq.com/docs/production-checklist)

## 13. 集群与 Kubernetes 的适用边界

- 单节点适合本地学习/CI，不提供 broker 节点级 HA。生产关键队列用三节点集群 + 三成员 quorum queue 起步；两节点集群官方强烈反对，4/6 节点对共识可用性的提升与 3/5 类似，因此集群和 quorum 成员通常取奇数。[Clustering Guide](https://www.rabbitmq.com/docs/clustering) [Quorum Queues](https://www.rabbitmq.com/docs/quorum-queues)
- 集群共享 users、vhosts、queues、exchanges、bindings 等元数据；消息内容是否复制取决于 queue 类型，非复制 classic queue 不会因为“加入 cluster”自动获得 quorum 数据安全。[Clustering Guide](https://www.rabbitmq.com/docs/clustering)
- AMQP 客户端任一时刻只连一个节点，应提供多个 endpoint/DNS/LB 和重连能力；节点可以透明转发到 quorum leader。还要均衡 queue leader，否则负载集中。[Clustering Guide](https://www.rabbitmq.com/docs/clustering)
- RabbitMQ 4.3 官方仍推荐亚毫秒 RTT 的 LAN。1–10ms（典型同 region AZ）通常可行；10–100ms 可行但要接受 confirm/metadata 延迟并监控 leader election；超过 100ms 或明显丢包不建议做 stretched cluster，应使用 Federation/Shovel 连接独立集群。若目标是容忍整站点故障，两个数据中心不够，三站点才是实际最低拓扑。[Clusters Spanning Multiple Data Centers](https://www.rabbitmq.com/docs/clustering#clusters-spanning-multiple-data-centers)
- Kubernetes 只有在团队已经具备 K8s/PVC/调度/监控/升级能力时才带来净收益；不要为了“像大厂”在学习第一阶段就引入。需要上 K8s 时使用 RabbitMQ 官方 **Cluster Operator** 管理 RabbitmqCluster 和 day-2 operations，Messaging Topology Operator 管 vhost/user/queue 等拓扑，不手写一套脆弱的 StatefulSet 运维逻辑。[RabbitMQ Kubernetes Operators](https://www.rabbitmq.com/kubernetes/operator/operator-overview)
- Operator 不会因自身升级而自动把既有 RabbitmqCluster 改到新默认值或 RabbitMQ 最新版本；升级仍需显式计划。被删除的凭据 Secret 虽会被重建，新值不会自动同步到既有 RabbitMQ；不能把“Secret 自动重建”当凭据恢复方案。[Operator limitations](https://www.rabbitmq.com/kubernetes/operator/operator-overview#limitations)

## 14. 建议的教程成熟度阶梯与验证门槛

1. **Level 0：本地可运行**：固定 4.3.5 management 镜像、稳定 hostname/volume、专用 user/vhost、健康检查；能通过 UI/CLI 观察消息。
2. **Level 1：协议正确**：topic exchange、durable queue/binding、官方 Go 客户端、连接与 Channel 正确所有权、显式拓扑声明。
3. **Level 2：单服务可靠**：persistent、confirm、mandatory return、manual ack、prefetch、有界 worker、重连、优雅停机；用重启/断网证明不静默丢。
4. **Level 3：业务一致性**：MySQL outbox、relay claim/lease、幂等消费、版本化 envelope、分档 retry/dead queue/replay；用 crash-window 测试证明最终一致。
5. **Level 4：生产可运维**：TLS、最小权限、policy、Prometheus/Grafana、SLO/告警、容量与故障压测、runbook、升级演练。
6. **Level 5：HA/平台化**：三节点、quorum queue、故障域分布、滚动升级/blue-green；已有 K8s 能力时再采用官方 Operator。

每一级都必须有可执行验收，而不是只堆配置：正常 publish/consume、错误 routing key 返回、consumer 处理后崩溃、ack 前崩溃、confirm 前断网、broker 重启、重复事件、poison message、DLX 目标缺失、应用 SIGTERM、资源告警、单节点故障。

## 15. 最需要避免的过时教程/版本陷阱

1. 把 4.2 当当前社区支持版，或使用 `rabbitmq:latest`。
2. 直接从 3.13 升 4.3；漏启 stable feature flags；把 downgrade 当正式回滚。
3. 仍使用 classic mirrored queues；RabbitMQ 4.0 起已移除，复制关键队列应用 quorum queue。
4. 继续用 `streadway/amqp`，或忽略 `amqp091-go` v1.14.0 的可选 auto-recovery；反过来，也不要以为 auto-recovery 会替应用补发未知状态 publish。
5. 多 goroutine 共用单一 AMQP Channel；v1.14.0 官方源码明确说 Channel 非线程安全。
6. 只开 publisher confirms、不设 mandatory/不消费 returns；不可路由消息仍可被 ack。
7. consumer `autoAck=true`，或在副作用提交前 ack；会把应用崩溃转成业务丢失。
8. 无限 `Nack(requeue=true)` 并指望 quorum 默认 delivery-limit=20 截断；在 4.3 中 `basic.nack` 不再增加 delivery-count，delayed retry 的线性退避也不会因此自动逐次变长。
9. 把普通 DLX 当可靠转移；默认内部 republish 无 confirm，目标不可用会丢。
10. 把每消息 TTL 当精确定时器；过期消息通常要到队首才 dead-letter，并可能继续占资源。
11. 宣称 outbox 或 RabbitMQ 提供端到端 exactly-once；confirm 后更新 sent 前仍有重复窗口。
12. 两节点生产集群，或在高丢包/100ms+ 链路硬拉 stretched cluster。
13. 为追求“大厂感”过早手写 Kubernetes StatefulSet；已有 K8s 平台时也应优先官方 Operators。

## 16. 一手来源索引

- [RabbitMQ release information](https://www.rabbitmq.com/docs/versions)
- [RabbitMQ upgrade guide](https://www.rabbitmq.com/docs/upgrade)
- [AMQP 0-9-1 concepts](https://www.rabbitmq.com/tutorials/amqp-concepts)
- [AMQP 0-9-1 specification repository](https://github.com/rabbitmq/amqp-0.9.1-spec)
- [RabbitMQ official Go client v1.14.0](https://github.com/rabbitmq/amqp091-go/tree/v1.14.0)
- [Consumer acknowledgements and publisher confirms](https://www.rabbitmq.com/docs/confirms)
- [Reliability guide](https://www.rabbitmq.com/docs/reliability)
- [Consumer prefetch](https://www.rabbitmq.com/docs/consumer-prefetch)
- [Quorum queues](https://www.rabbitmq.com/docs/quorum-queues)
- [Dead letter exchanges](https://www.rabbitmq.com/docs/dlx)
- [TTL](https://www.rabbitmq.com/docs/ttl)
- [Production checklist](https://www.rabbitmq.com/docs/production-checklist)
- [TLS](https://www.rabbitmq.com/docs/ssl)
- [Access control](https://www.rabbitmq.com/docs/access-control)
- [Virtual hosts](https://www.rabbitmq.com/docs/vhosts)
- [Monitoring](https://www.rabbitmq.com/docs/monitoring)
- [Prometheus and Grafana](https://www.rabbitmq.com/docs/prometheus)
- [Clustering](https://www.rabbitmq.com/docs/clustering)
- [RabbitMQ PerfTest](https://github.com/rabbitmq/rabbitmq-perf-test)
- [RabbitMQ Kubernetes Operators](https://www.rabbitmq.com/kubernetes/operator/operator-overview)
- [Docker Official RabbitMQ image README](https://github.com/docker-library/docs/blob/master/rabbitmq/README.md)
