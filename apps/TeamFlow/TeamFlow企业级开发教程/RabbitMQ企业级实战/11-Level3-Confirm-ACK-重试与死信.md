# Level 3：Confirm、ACK、有限重试与死信

> 本章目标：在 Level 2 已经跑通的链路上，为 `task.assigned.v1` 增加一套能够解释失败、限制失败并保留失败现场的可靠投递机制。
>
> 完成后，系统应从“通常能收到消息”升级为“发布结果可判断、消费结果可确认、临时失败会有限重试、永久失败有停车场可查”。
>
> 本章仍然不解决 MySQL 与 RabbitMQ 的跨系统原子性，也不承诺消息只处理一次。生产者 Outbox 和消费者幂等留到 Level 4。

## 1. 先定义这一周真正要获得的保证

Level 2 打通了下面这条功能链路：

```text
AssignTask
  → Publish(task.assigned.v1)
  → RabbitMQ Queue
  → Notification Worker
  → notifications
  → WebSocket
```

它证明了组件可以协作，却没有回答失败时最重要的问题：

- Publisher 调用返回 `nil`，消息是否真的已经被 Broker 接管？
- Exchange 没有匹配的 binding 时，消息去了哪里？
- Worker 刚收到消息就崩溃，RabbitMQ 是否会重新投递？
- MySQL 短暂不可用时，应该立即重试、延迟重试，还是永久放弃？
- JSON 字段错误这种永久失败，是否应该无限循环？
- 重试很多次仍失败的消息，如何保留现场并受控重放？

Level 3 的核心不是堆叠 RabbitMQ API，而是让每一种失败都有明确归宿。

```text
发布端：Publish → Return? → Confirm ACK / NACK / Timeout
消费端：Delivery → Handle → ACK / Retry / Parking lot
```

本章增加的保证和仍然缺失的保证如下：

| 场景 | Level 3 的处理 | 是否完全解决 |
| --- | --- | --- |
| 消息无法路由到任何 Queue | `mandatory` + Return，发布失败可见 | 是，针对不可路由问题 |
| Broker 拒绝或未确认消息 | Publisher Confirm 返回 NACK 或超时 | 是，发布者知道结果不可靠 |
| Worker 在业务成功前崩溃 | manual ACK，未确认消息重新投递 | 是，但可能重复处理 |
| MySQL 短暂失败 | 延迟、有限次数重试 | 是，超过上限后进停车场 |
| 非法 JSON、未知版本 | 永久失败，直接进停车场 | 是，可人工诊断 |
| DB 已更新、消息发布前进程崩溃 | 本章无能为力 | 否，Level 4 用 Outbox |
| 通知落库后、ACK 前进程崩溃 | 消息会重复投递 | 否，Level 4 用消费幂等 |

因此，本章的准确语义是：

```text
RabbitMQ 链路内可观察的至少一次投递基础
```

而不是：

```text
端到端 Exactly Once
```

## 2. 完成后的消息状态图

先把整条消息可能经过的状态画清楚。后续每一段代码都应该能映射回这张图。

```text
TaskService
    │
    │ EventPublisher.Publish
    ▼
teamflow.events.v1 (topic exchange)
    │
    ├── 无匹配 binding ── mandatory Return ──► Publisher 记录失败
    │
    └── 路由成功
          │
          ▼
teamflow.notification.websocket.q
          │
          ▼
Notification Worker
    │
    ├── 成功 ───────────────────────────────► ACK
    │
    ├── 临时错误，未超过次数
    │       │
    │       ├── 发布到延迟重试队列并等 Confirm
    │       └── ACK 原消息
    │                  │ TTL 到期
    │                  ▼
    │          回到 teamflow.events.v1
    │
    ├── 永久错误 ───────────────────────────► NACK(requeue=false)
    │
    └── 临时错误，超过次数 ─────────────────► 发布到停车场并 ACK 原消息
                                               │
                                               ▼
                              teamflow.notification.parking.q
```

这里有三个容易混淆的动作：

1. `ACK` 表示这次投递已完成，RabbitMQ 可以删除该 Queue 中的这份消息。
2. `NACK(requeue=true)` 会把消息放回原 Queue，通常会形成高频死循环，本章禁止把它当通用重试。
3. `NACK(requeue=false)` 会丢弃消息；若原 Queue 配置了 DLX，则消息被 dead-letter 到停车场。

## 3. 本章范围和停止条件

本章只继续处理一个事件：`task.assigned.v1`。所有可靠性机制都先围绕通知 Queue 建立。

本章要做：

- durable exchange、durable queue、persistent message；
- Publisher Confirm；
- `mandatory=true` 与 Return；
- Consumer manual ACK；
- QoS / prefetch；
- 错误分类；
- 三档固定延迟重试；
- parking queue；
- 关键结构化日志；
- 可重复执行的故障实验。

本章不做：

- Transactional Outbox；
- `processed_messages` 幂等表；
- RabbitMQ 集群、Quorum Queue 和 Kubernetes；
- 动态任意重试时间；
- 管理后台“一键重放”；
- 同时接入邮件、审计、评论和任务状态事件；
- 把所有错误都封装成一个庞大的“万能 MQ 框架”。

停止条件是：本章列出的六类故障实验全部能解释并通过。完成后进入 Level 4，不继续在 Level 3 叠加新能力。

## 4. 第一步：统一可靠性拓扑

Level 2 只有业务 Exchange 和通知 Queue。本章增加 retry exchange、三条延迟 Queue、parking exchange 和 parking queue。

### 4.1 推荐名称

```text
业务 Exchange
  teamflow.events.v1                         topic, durable

主消费 Queue
  teamflow.notification.websocket.q          durable
  binding: task.assigned.v1

重试 Exchange
  teamflow.retry.v1                          direct, durable

延迟重试 Queue
  teamflow.notification.retry.5s.q           TTL=5s
  teamflow.notification.retry.30s.q          TTL=30s
  teamflow.notification.retry.5m.q           TTL=5m

停车场 Exchange
  teamflow.parking.v1                        direct, durable

停车场 Queue
  teamflow.notification.parking.q            durable
```

三条 retry queue 的消息 TTL 到期后，通过 dead-letter exchange 回到 `teamflow.events.v1`，routing key 固定为 `task.assigned.v1`。主 Queue 的永久失败通过 DLX 进入 parking exchange。

### 4.2 为什么选择固定档位，而不是每条消息随意设置 TTL

RabbitMQ Queue 中的过期消息并不等同于一个精确计时器。把大量不同 TTL 的消息混在一条 Queue 中，可能出现短 TTL 消息被前面的长 TTL 消息挡住的情况。固定 Queue 让顺序、容量和运维含义更明确：

| attempt | 延迟 | routing key | 适用含义 |
| --- | ---: | --- | --- |
| 1 | 5 秒 | `notification.5s` | 短暂连接抖动 |
| 2 | 30 秒 | `notification.30s` | 数据库短时不可用 |
| 3 | 5 分钟 | `notification.5m` | 给依赖恢复留出时间 |
| > 3 | 不再重试 | `notification.parking` | 进入停车场人工诊断 |

教学阶段固定三档已经足够。以后只有在真实业务指标证明不够时，才调整档位。

### 4.3 拓扑声明示意

建议继续在 `internal/messaging/rabbitmq/topology.go` 中集中声明。下列代码重点展示参数关系，可根据现有封装拆成小函数：

```go
const (
    EventsExchange  = "teamflow.events.v1"
    RetryExchange   = "teamflow.retry.v1"
    ParkingExchange = "teamflow.parking.v1"

    NotificationQueue   = "teamflow.notification.websocket.q"
    Retry5sQueue         = "teamflow.notification.retry.5s.q"
    Retry30sQueue        = "teamflow.notification.retry.30s.q"
    Retry5mQueue         = "teamflow.notification.retry.5m.q"
    NotificationParking = "teamflow.notification.parking.q"
)

func DeclareTopology(ch *amqp091.Channel) error {
    if err := ch.ExchangeDeclare(EventsExchange, "topic", true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare events exchange: %w", err)
    }
    if err := ch.ExchangeDeclare(RetryExchange, "direct", true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare retry exchange: %w", err)
    }
    if err := ch.ExchangeDeclare(ParkingExchange, "direct", true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare parking exchange: %w", err)
    }

    mainArgs := amqp091.Table{
        "x-dead-letter-exchange":    ParkingExchange,
        "x-dead-letter-routing-key": "notification.parking",
    }
    if _, err := ch.QueueDeclare(NotificationQueue, true, false, false, false, mainArgs); err != nil {
        return fmt.Errorf("declare notification queue: %w", err)
    }
    if err := ch.QueueBind(NotificationQueue, "task.assigned.v1", EventsExchange, false, nil); err != nil {
        return fmt.Errorf("bind notification queue: %w", err)
    }

    if err := declareRetryQueue(ch, Retry5sQueue, "notification.5s", 5_000); err != nil {
        return err
    }
    if err := declareRetryQueue(ch, Retry30sQueue, "notification.30s", 30_000); err != nil {
        return err
    }
    if err := declareRetryQueue(ch, Retry5mQueue, "notification.5m", 300_000); err != nil {
        return err
    }

    if _, err := ch.QueueDeclare(NotificationParking, true, false, false, false, nil); err != nil {
        return fmt.Errorf("declare notification parking queue: %w", err)
    }
    if err := ch.QueueBind(NotificationParking, "notification.parking", ParkingExchange, false, nil); err != nil {
        return fmt.Errorf("bind notification parking queue: %w", err)
    }
    return nil
}

func declareRetryQueue(ch *amqp091.Channel, name, bindingKey string, ttlMS int32) error {
    args := amqp091.Table{
        "x-message-ttl":             ttlMS,
        "x-dead-letter-exchange":    EventsExchange,
        "x-dead-letter-routing-key": "task.assigned.v1",
    }
    if _, err := ch.QueueDeclare(name, true, false, false, false, args); err != nil {
        return fmt.Errorf("declare retry queue %s: %w", name, err)
    }
    if err := ch.QueueBind(name, bindingKey, RetryExchange, false, nil); err != nil {
        return fmt.Errorf("bind retry queue %s: %w", name, err)
    }
    return nil
}
```

RabbitMQ 对同名 Queue 的参数要求完全一致。如果 Level 2 已经声明过不带 DLX 参数的主 Queue，再用带 DLX 参数声明同名 Queue，会得到 `PRECONDITION_FAILED`。开发环境可明确删除并重建该 Queue；生产迁移则应使用新 Queue 名称、双绑定、排空旧 Queue 的兼容迁移策略，不能直接删队列。

## 5. 第二步：理解持久化的三层条件

很多项目只把 `DeliveryMode` 设置成 persistent，就宣称消息不会丢。RabbitMQ 的持久化需要至少同时满足：

1. Exchange 是 durable；
2. Queue 是 durable；
3. Message 是 persistent。

发布结构应至少包含：

```go
amqp091.Publishing{
    ContentType:  "application/json",
    DeliveryMode: amqp091.Persistent,
    MessageId:    envelope.EventID,
    Type:         envelope.EventType,
    Timestamp:    envelope.OccurredAt,
    Body:         body,
}
```

这三层只表示 Broker 会按持久化语义处理消息。Publisher 仍需要 Confirm 才知道 Broker 是否接管。即使收到 Confirm，也只解决 RabbitMQ 发布环节，不解决 Task 数据库事务与发布之间的空窗。

## 6. 第三步：为 Publisher 增加 Confirm

### 6.1 普通 Publish 返回 nil 到底代表什么

普通 `PublishWithContext` 返回 `nil`，主要表示客户端成功把帧写入了连接；它不等于 Broker 已经把消息接管并落到目标 Queue。连接可能在写入后立即断开，Publisher 无法仅靠函数返回值判断最终结果。

Confirm 模式让 Broker 对发布消息给出异步 ACK 或 NACK：

```text
Publisher ── publish ──► Broker
Publisher ◄── ACK ───── Broker   已接管
Publisher ◄── NACK ──── Broker   未接管
Publisher ◄── timeout ─ Broker   结果未知
```

先在 Channel 上开启 Confirm：

```go
if err := ch.Confirm(false); err != nil {
    return nil, fmt.Errorf("enable publisher confirm: %w", err)
}
```

### 6.2 ACK、NACK、超时必须分开处理

- Confirm ACK：Broker 确认接管消息。它不代表 Consumer 已处理。
- Confirm NACK：Broker 明确没有接管，发布失败。
- 超时或连接中断：结果未知。消息可能已经接管，也可能没有。

“结果未知”是可靠消息里非常重要的概念。此时直接重发可能产生重复，不重发可能丢失。Level 3 可以记录失败并让上层知道结果未知；Level 4 的 Outbox 会选择可恢复的重复发布，再依靠 Consumer 幂等消除副作用。

### 6.3 Channel 必须有唯一 owner

Confirm 和 Return 都是异步事件。如果多个 goroutine 直接共享一个 Channel 并各自等待结果，很容易把 A 消息的确认错误地交给 B 请求。

推荐让 Publisher 内部只有一个 goroutine 拥有 Channel：

```text
HTTP goroutine ─┐
HTTP goroutine ─┼── publishRequest channel ──► publisher owner goroutine
HTTP goroutine ─┘                                  │
                                                   ├─ Publish
                                                   ├─ 等 Return / Confirm
                                                   └─ 回传 result
```

对当前教学项目，如果暂时采取“每次发布独占一个 Publisher Channel 并串行加锁”，也能工作，但吞吐较低。不要把 `sync.Mutex` 包住一个会无限等待的 Confirm；等待必须有超时，连接关闭时所有 pending 请求都要结束。

### 6.4 Publisher 接口仍然不暴露 AMQP 类型

业务层接口可以保留为：

```go
type Publisher interface {
    Publish(ctx context.Context, message Envelope) error
}
```

RabbitMQ Adapter 内部再把错误分成可观测类型：

```go
var (
    ErrPublishNack       = errors.New("rabbitmq publish nack")
    ErrPublishReturned   = errors.New("rabbitmq message returned")
    ErrPublishUncertain  = errors.New("rabbitmq publish outcome uncertain")
)
```

TaskService 不需要知道 delivery tag、AMQP reply code 或 Channel 状态。它只处理“事件发布成功或失败”这个接口结果。

## 7. 第四步：用 mandatory + Return 捕获不可路由消息

Publisher Confirm ACK 只表示 Exchange 接管消息，并不保证消息进入了 Queue。如果 routing key 没有匹配任何 binding，默认情况下消息可能被静默丢弃。

将发布参数中的 `mandatory` 设置为 `true`：

```go
err := ch.PublishWithContext(
    ctx,
    EventsExchange,
    envelope.EventType,
    true,  // mandatory
    false, // immediate 已废弃，不使用
    publishing,
)
```

同时注册 Return 通道：

```go
returns := ch.NotifyReturn(make(chan amqp091.Return, 1))
```

若消息无法路由，Broker 会发送 Return。日志至少记录：

```text
event_id
event_type
exchange
routing_key
reply_code
reply_text
```

需要特别理解：同一条不可路由消息可能同时得到 Return 和 Confirm ACK。因为 Exchange 确实接管并检查了消息，只是没有 Queue 匹配。因此发布成功条件不能只看 Confirm ACK，而应是：

```text
Confirm ACK && 没有 Return
```

生产实现应通过 publisher owner 串行发布，或者使用 sequence number / message ID 正确关联 Return、Confirm 和调用方。不要用一个 `select` 随便等待其中一个事件后就返回。

## 8. 第五步：Consumer 必须使用 manual ACK

### 8.1 auto-ack 的风险

若 `Consume` 使用 `autoAck=true`，RabbitMQ 将消息交给客户端时就视为完成。此后发生以下任何错误，消息都不会重新投递：

- JSON 反序列化失败；
- 查询用户失败；
- 写 notifications 失败；
- Worker 进程崩溃；
- 机器断电。

因此本章统一使用：

```go
deliveries, err := ch.Consume(
    NotificationQueue,
    consumerName,
    false, // autoAck=false
    false,
    false,
    false,
    nil,
)
```

### 8.2 ACK 的业务边界

通知链路的成功条件应是通知记录已提交到 MySQL。WebSocket 是尽力实时体验，不应成为 ACK 条件：

```text
解析并校验事件
  → notifications 写库成功
  → ACK
  → WebSocket 尽力推送
```

也可以先推送后 ACK，但推送成功、落库失败会产生只在屏幕上出现却无法查询的幽灵通知。TeamFlow 应以数据库通知记录作为业务真相。

推荐让 Worker Handler 返回分类错误，消费循环统一决定 ACK、重试或停车：

```go
for delivery := range deliveries {
    err := handler.Handle(ctx, delivery.Body)
    switch {
    case err == nil:
        _ = delivery.Ack(false)
    case errors.Is(err, ErrPermanent):
        _ = delivery.Nack(false, false)
    default:
        retryOrPark(ctx, delivery, err)
    }
}
```

不要在 Repository、NotificationService 和消费循环三个层级都调用 ACK。ACK 是 RabbitMQ Delivery 的生命周期动作，只应由消费适配层负责。

### 8.3 redelivered 不是幂等依据

`delivery.Redelivered` 可以帮助日志判断消息是否被 Broker 重投，但它不能替代 `event_id` 幂等。网络断开、重新发布和 Retry Queue 都可能让重复消息以不同方式出现。Level 4 仍必须在数据库用唯一约束记录处理过的 `event_id`。

## 9. 第六步：使用 QoS 控制在途消息

manual ACK 后，如果不设置 prefetch，RabbitMQ 可能把大量消息提前推给一个 Worker。这些消息都处于 Unacked 状态：其他 Worker 无法处理；该进程崩溃后又会集中重新入队。

设置 QoS：

```go
if err := ch.Qos(
    20,    // prefetchCount
    0,     // prefetchSize，通常不使用
    false, // 当前 Channel / Consumer 范围
); err != nil {
    return fmt.Errorf("configure consumer qos: %w", err)
}
```

初始值不需要追求“最佳”。可以从下面的可解释关系开始：

```text
prefetch ≈ 单个 Worker 并发数 × 每个任务允许的在途倍数
```

如果当前 Worker 串行处理，先用 `prefetch=10` 或更小；如果有 4 个受控 handler goroutine，可以从 20 开始压测。不要创建无限 goroutine 消费 deliveries，否则 prefetch 的背压意义会被破坏。

管理台观察：

- Ready：仍在 Queue 中等待派发；
- Unacked：已交给 Consumer，尚未 ACK；
- Consumers：当前消费者数量；
- Deliver / Ack rate：交付和确认速率。

当 MySQL 变慢时，合理的现象是 Unacked 被 prefetch 限制，Ready 开始增长；而不是 Worker 内存持续增长。

## 10. 第七步：先分类错误，再决定重试

不是所有错误都值得重试。重试策略的第一步不是计算退避，而是判断失败是否可能自行恢复。

### 10.1 永久错误

在不修改消息或代码的情况下，再执行一次仍会失败：

- JSON 无法解析；
- `event_type` 不支持；
- `schema_version` 不支持；
- payload 缺少 `task_id` 或 `assignee_id`；
- 用户、任务等业务对象确定不存在；
- 事件违反不可变业务规则。

永久错误应直接进入 parking queue，避免消耗 CPU、日志和 Broker I/O。

### 10.2 临时错误

外部状态恢复后可能成功：

- MySQL 连接超时；
- 数据库临时不可用；
- Redis 或内部依赖短时断连；
- 网络超时；
- 下游返回明确可重试状态。

临时错误进入有限延迟重试。

### 10.3 未知错误

无法可靠分类时，不应无限重试。教学阶段可把未知错误按临时错误处理，但最多三次，之后停车并告警。日志中必须标记 `error_class=unknown`，便于之后补齐分类。

可以定义一个轻量错误类型：

```go
type ErrorClass string

const (
    ErrorPermanent ErrorClass = "permanent"
    ErrorTransient ErrorClass = "transient"
    ErrorUnknown   ErrorClass = "unknown"
)

type HandlingError struct {
    Class ErrorClass
    Op    string
    Err   error
}

func (e *HandlingError) Error() string { return e.Op + ": " + e.Err.Error() }
func (e *HandlingError) Unwrap() error { return e.Err }
```

错误分类应靠类型、错误码和可验证状态，不要通过字符串包含 `timeout` 来猜测。

## 11. 第八步：实现有限延迟重试

### 11.1 为什么不使用无限 NACK requeue

下面这种写法看似“保证不丢”，实际会制造毒消息循环：

```go
delivery.Nack(false, true)
```

消息可能立刻回到同一 Consumer：

```text
失败 → requeue → 立即收到 → 失败 → requeue → ...
```

它没有退避、没有上限，还可能阻塞后续正常消息。数据库故障期间，它会快速放大流量和日志。

### 11.2 重试消息需要保留哪些元数据

重试发布时保留原始消息体和关键属性，并更新 headers：

```text
x-retry-count       已执行的重试次数
x-original-queue    teamflow.notification.websocket.q
x-last-error-class  transient / unknown
x-last-error        截断后的错误摘要
x-first-failed-at   第一次失败时间
x-last-failed-at    最近失败时间
```

`event_id` 必须保持不变。重试是同一个业务事件的再次投递，不是创建新事件。

不要把完整堆栈、SQL、凭据或超长错误写进 header。RabbitMQ header 会占用消息和内存，应只保留诊断摘要，详细堆栈放结构化日志。

### 11.3 重试档位选择

```go
func retryRoute(attempt int) (routingKey string, ok bool) {
    switch attempt {
    case 1:
        return "notification.5s", true
    case 2:
        return "notification.30s", true
    case 3:
        return "notification.5m", true
    default:
        return "", false
    }
}
```

### 11.4 最关键的顺序：先确认新消息，再 ACK 原消息

临时失败时，不能先 ACK 原消息再发布 retry；否则进程在两者之间崩溃，消息永久丢失。

正确顺序：

```text
1. 发布副本到 retry exchange
2. 等待 retry publish Confirm，并确认没有 Return
3. ACK 原始 Delivery
```

如果第 2 步失败或结果未知，不 ACK 原消息，让连接关闭或 Consumer 重启后由 Broker 重投。这个顺序存在一个允许的重复窗口：retry 副本已确认，但 ACK 原消息前进程崩溃，于是原消息也会重投。Level 3 接受至少一次；Level 4 用幂等消除重复通知。

重试流程示意：

```go
func (c *Consumer) retryOrPark(ctx context.Context, d amqp091.Delivery, handleErr error) error {
    attempt := readRetryCount(d.Headers) + 1
    route, retry := retryRoute(attempt)
    if !retry {
        return c.publishParkingThenAck(ctx, d, attempt, handleErr)
    }

    msg := cloneForRetry(d, attempt, handleErr)
    if err := c.retryPublisher.PublishRaw(ctx, RetryExchange, route, msg); err != nil {
        // 不 ACK 原消息。上层关闭/重建异常 Consumer Channel，
        // Broker 会重新投递尚未确认的原消息。
        return fmt.Errorf("publish retry copy: %w", err)
    }
    return d.Ack(false)
}
```

`PublishRaw` 同样必须启用 Confirm 和 mandatory。可靠性不能只用于业务首次发布，却在 retry 和 parking 转移时退回普通 Publish。

## 12. 第九步：停车场不是垃圾桶

parking queue 用来保存无法自动处理的消息及诊断信息。它不是“没人管的死信堆”。

消息进入停车场的条件：

- 永久错误；
- 临时或未知错误超过三次；
- 重试策略明确判定停止；
- 运维主动隔离的异常消息。

停车消息至少应能关联：

```text
event_id
event_type
original_exchange
original_routing_key
original_queue
retry_count
error_class
last_error
first_failed_at
last_failed_at
trace_id / correlation_id
```

停车后的操作流程：

1. 查看消息元数据和对应应用日志；
2. 判断是代码缺陷、契约不兼容、脏数据还是依赖故障；
3. 修复根因；
4. 在测试环境验证原消息可成功处理；
5. 使用受控工具重放少量消息；
6. 观察通知唯一性和消费结果；
7. 再逐批恢复其余消息。

禁止直接把整个 parking queue 一次性重新投回主 Exchange。若根因未修复，只会再次制造积压；即使根因已修复，也可能瞬间压垮数据库。

开发阶段可以先通过管理台手动 Get Message 检查，不需要在本章做重放后台。也不要随手 Purge，除非这些测试消息的范围已被明确确认。

## 13. 第十步：调整 Notification Worker 的职责边界

建议将 Worker 拆成三层，避免业务处理和 AMQP 生命周期混在一起：

```text
RabbitMQ Consumer Adapter
  ├─ Consume / QoS
  ├─ ACK / NACK
  ├─ retry / parking publish
  └─ 连接关闭处理
           │
           ▼
NotificationWorker.Handle(ctx, envelope)
  ├─ 校验契约
  ├─ 调 NotificationService 创建通知
  └─ 尝试实时推送
           │
           ▼
Repository / MySQL / WebSocket Hub
```

推荐接口：

```go
type Handler interface {
    Handle(ctx context.Context, envelope event.Envelope) error
}
```

AMQP Adapter 负责从 `amqp091.Delivery` 解出 Envelope，并决定 RabbitMQ 动作；业务 Worker 不应该调用 `Ack`。这样单元测试 Handler 时只传业务事件，不需要构造 AMQP Channel。

通知成功边界建议为：

```go
func (w *NotificationWorker) Handle(ctx context.Context, e event.Envelope) error {
    payload, err := decodeAndValidateTaskAssigned(e)
    if err != nil {
        return permanent("validate task.assigned.v1", err)
    }

    notification, err := w.notifications.CreateTaskAssigned(ctx, payload)
    if err != nil {
        return classifyStorageError("create notification", err)
    }

    // WebSocket 失败只记录，不让已经落库的通知进入重试。
    if err := w.realtime.PushIfOnline(ctx, notification); err != nil {
        w.logger.Warn("notification realtime push failed", ...)
    }
    return nil
}
```

如果让 WebSocket 推送失败触发整条消息重试，会重复创建数据库通知。Level 4 虽会加入幂等，但实时推送仍应保持尽力而为。

## 14. 第十一步：连接断开与优雅停机的最低要求

完整自动重连属于 Level 5，但 Level 3 至少不能在连接断开时静默退出。

需要监听：

```go
connClose := conn.NotifyClose(make(chan *amqp091.Error, 1))
channelClose := ch.NotifyClose(make(chan *amqp091.Error, 1))
cancelled := ch.NotifyCancel(make(chan string, 1))
```

出现以下情况应记录明确日志并让进程进入不就绪或退出，由进程管理器重启：

- Connection 关闭；
- Channel 因协议错误关闭；
- Broker 取消 Consumer；
- deliveries 通道关闭。

优雅停机顺序：

```text
收到 SIGTERM
  → 停止接受新的 HTTP 请求或新的发布请求
  → Cancel Consumer，停止获取新 Delivery
  → 等待当前 Handler 在有限时间内结束
  → 成功则 ACK，超时则不 ACK
  → 关闭 Channel
  → 关闭 Connection
```

不要在程序刚收到退出信号时立即关闭 Connection。否则正在等待 Confirm 的发布和正在提交事务的 Consumer 都会突然变成结果未知。

## 15. 第十二步：日志必须能回答“这条消息去哪了”

本章暂时不要求完整 Prometheus 指标，但结构化日志必须覆盖消息生命周期。

发布日志字段：

```text
component=event_publisher
event_id
event_type
exchange
routing_key
confirm=ack|nack|timeout
returned=true|false
duration_ms
trace_id
```

消费日志字段：

```text
component=notification_consumer
event_id
event_type
queue
consumer
delivery_tag
redelivered
attempt
error_class
action=ack|retry_5s|retry_30s|retry_5m|park
duration_ms
trace_id
```

错误日志不要只写：

```text
consume failed
```

它至少要让你搜索 `event_id` 后还原：首次发布、路由、每次消费、每次重试和最终归宿。

注意敏感信息：不要记录 RabbitMQ URI 密码、Authorization header、完整用户隐私字段或原始 SQL 参数。

## 16. 代码落地顺序

不要一次改完再启动。按下面的提交与验证顺序更容易定位问题。

### 16.1 拓扑与持久化

文件建议：

```text
internal/messaging/rabbitmq/topology.go
internal/messaging/rabbitmq/topology_test.go
```

先完成 exchange、main queue、retry queues、parking queue 的声明和重复声明测试。启动两次应无冲突；管理台参数应与代码一致。

### 16.2 Publisher Confirm 和 Return

文件建议：

```text
internal/messaging/rabbitmq/publisher.go
internal/messaging/rabbitmq/publisher_test.go
```

先串行保证正确，再考虑吞吐。至少覆盖：正常 Confirm ACK、Return、超时、Channel 关闭。

### 16.3 Consumer ACK 和 QoS

文件建议：

```text
internal/messaging/rabbitmq/consumer.go
internal/messaging/rabbitmq/consumer_test.go
```

让消费 Adapter 统一决定 ACK。测试成功 ACK、永久错误 NACK、临时错误进入 retry。

### 16.4 Retry 与 Parking

文件建议：

```text
internal/messaging/rabbitmq/retry.go
internal/messaging/rabbitmq/retry_test.go
```

把档位和次数做成明确常量，不提前设计复杂配置中心。测试 1、2、3 次对应的路由，以及第 4 次进入 parking。

### 16.5 Worker 错误分类

文件建议：

```text
internal/worker/notification_worker.go
internal/worker/notification_worker_test.go
```

Handler 只关心业务结果；AMQP 决策留在 Adapter。测试非法事件是永久错误，数据库超时是临时错误，WebSocket 失败不影响业务成功。

## 17. 自动化测试建议

### 17.1 不连接 RabbitMQ 的单元测试

优先覆盖纯逻辑：

- `retryRoute(1)` 为 5 秒；
- `retryRoute(2)` 为 30 秒；
- `retryRoute(3)` 为 5 分钟；
- `retryRoute(4)` 停止重试；
- 非法 JSON 分类为永久错误；
- 不支持的 schema version 分类为永久错误；
- 数据库 timeout 分类为临时错误；
- WebSocket 失败时 Handler 仍返回成功；
- retry copy 保留原 `event_id`；
- 错误 header 被截断且不包含敏感数据。

### 17.2 连接真实 RabbitMQ 的集成测试

集成测试应使用独立 vhost 或带随机后缀的拓扑，避免污染开发 Queue。覆盖：

- durable topology 可重复声明；
- 合法 routing key 能进入主 Queue；
- 错误 routing key 产生 Return；
- 未 ACK 后关闭 Channel，消息会重新投递；
- retry queue 的 TTL 到期后回到主 Queue；
- `NACK(requeue=false)` 进入 parking queue；
- prefetch 限制 Unacked 数量。

测试结束只清理本测试创建的确定命名资源。不要对共享 vhost 执行全局清理。

## 18. 六个必须完成的故障实验

本章最有价值的部分是主动制造故障。只看正常日志不能证明可靠性。

### 实验一：制造不可路由消息

操作：临时将 routing key 改为 `task.assigned.invalid`，或在独立测试 Exchange 中不创建 binding。

预期：

- Publisher 收到 Return；
- Return 包含 `NO_ROUTE`；
- 即使随后收到 Confirm ACK，也把整体发布判为失败；
- 日志能通过 `event_id` 查到 routing key 和 reply text。

恢复：恢复正确 routing key，不删除业务 Queue。

### 实验二：Worker 在 ACK 前退出

操作：Handler 完成前设置断点或临时阻塞，确认管理台出现 1 条 Unacked，然后终止 Worker。

预期：

- Consumer 断开后，消息从 Unacked 回到 Ready；
- 新 Worker 启动后再次收到消息；
- `redelivered=true` 通常可观察到；
- 当前 Level 3 可能创建重复通知，这是预期缺口，记录下来供 Level 4 修复。

### 实验三：MySQL 短暂不可用

操作：消费前短暂停止 MySQL，发布一条合法任务分配事件，随后恢复 MySQL。

预期：

- 第一次失败分类为 transient；
- 原消息只有在 retry copy Confirm 后才 ACK；
- 消息依次经过 5 秒、30 秒等档位；
- MySQL 恢复后通知最终落库并 ACK；
- 没有高频 `NACK requeue` 循环。

### 实验四：发送非法 JSON

操作：向业务 Exchange 发布一条 routing key 正确、Body 为非法 JSON 的消息。

预期：

- 错误分类为 permanent；
- 不进入三档 retry queue；
- 消息直接进入 parking queue；
- parking message 能查到失败原因和原始 `event_id`；若没有合法 event ID，日志用 delivery 信息关联。

### 实验五：让临时错误超过上限

操作：保持数据库不可用，等待消息完成三次重试。

预期：

- attempt 依次为 1、2、3；
- 第 4 次不再回到 retry exchange；
- 消息进入 parking queue；
- 主 Queue 最终没有该消息的无限循环；
- 日志 action 最终为 `park`。

### 实验六：retry publish Confirm 后、原消息 ACK 前退出

操作：在 retry publish 已 Confirm 后、执行原 Delivery ACK 前主动终止 Worker。

预期：

- retry queue 中已有一份副本；
- 原主 Queue 消息因为未 ACK 也会重新投递；
- 最终可能处理两次；
- 你能解释这是至少一次投递的重复窗口，而不是 RabbitMQ 故障；
- Level 4 将用 `event_id UNIQUE` 让第二次处理变为安全空操作。

## 19. 常见错误与排查方式

| 症状 | 常见原因 | 检查顺序 |
| --- | --- | --- |
| Queue 声明时报 `PRECONDITION_FAILED` | 同名 Queue 的 durable、DLX、TTL 参数变了 | 对比管理台 Arguments 与代码；开发环境受控重建 |
| Publish Confirm ACK，但 Queue 没消息 | routing key 无 binding，或被 Consumer 立即取走 | 检查 Return、binding、Ready/Unacked、消费日志 |
| Return 收不到 | `mandatory=false` 或 NotifyReturn 注册太晚 | 启动时注册 Return，再接受发布请求 |
| Unacked 持续增长 | Handler 变慢、漏 ACK/NACK、并发失控 | 查 handler 时延、prefetch、每个分支是否结束 Delivery |
| 同一消息高速重复 | 使用 `NACK(requeue=true)` | 改为延迟 retry queue 和有限次数 |
| retry queue 有消息但不返回主 Queue | DLX 或 dead-letter routing key 配错 | 查 Queue Arguments、events exchange binding |
| 永久错误也重试三次 | 错误分类丢失或被统一 wrap | 使用 `errors.Is/As` 保留类型，不靠字符串 |
| parking queue 为空但日志说已停车 | parking publish 未 Confirm 就 ACK 原消息 | 转移同样执行 mandatory + Confirm |
| Worker 重启后重复通知 | 业务提交后 ACK 前崩溃 | Level 3 的已知缺口；Level 4 加幂等表 |
| Task 已指派但完全没有事件 | DB 提交后、Publish 前崩溃 | Level 3 的已知缺口；Level 4 加 Outbox |

## 20. 你必须能解释的三个“结果未知”窗口

### 20.1 首次发布超时

```text
Publisher 发出消息
  → Broker 可能已接管
  → ACK 在网络中丢失
  → Publisher 超时
```

重发可能重复，不重发可能丢。Level 4 的策略是允许 Outbox Relay 重发，并由消费者幂等兜底。

### 20.2 retry 转移的确认窗口

```text
retry copy 已 Confirm
  → Worker 崩溃
  → 原消息尚未 ACK
```

两份消息都可能被消费。这是为了避免丢失而接受的重复。

### 20.3 业务提交和 ACK 的窗口

```text
notifications 已提交
  → Worker 崩溃
  → ACK 未送达
```

Broker 重投，通知可能再创建一次。manual ACK 只避免“先 ACK 后业务失败”的丢失，不能自动实现业务幂等。

理解这三个窗口后，你就不会再把 Confirm、ACK 或持久化单独称为“绝不丢、绝不重”。

## 21. 配置建议

Level 3 可以把这些参数加入 RabbitMQ 配置，但保持默认值简单：

```yaml
rabbitmq:
  publish_confirm_timeout: 5s
  prefetch: 20
  consumer_concurrency: 4
  retry:
    max_attempts: 3
    delays:
      - 5s
      - 30s
      - 5m
```

注意：如果代码只支持三条固定 retry queue，配置也应只允许这三个档位，或者启动时根据配置声明完全一致的 Queue。不要让配置看似可以任意调整，实际 RabbitMQ 中旧 Queue 参数无法原地修改。

配置校验应在启动阶段完成：

- confirm timeout 必须大于 0；
- prefetch 必须大于 0，并设置合理上限；
- consumer concurrency 必须大于 0；
- retry attempts 与已声明档位数量一致；
- Queue 和 Exchange 名称不能为空。

## 22. 本章完成定义

只有下面各项都有证据时，Level 3 才算完成：

- [ ] exchange、main queue、retry queues、parking queue 均为 durable；
- [ ] 业务消息、retry copy、parking copy 均为 persistent；
- [ ] 所有发布路径启用 Publisher Confirm；
- [ ] 所有发布路径启用 mandatory 并处理 Return；
- [ ] NACK、Return、Confirm timeout 被区分记录；
- [ ] Consumer 使用 `autoAck=false`；
- [ ] 业务落库成功后才 ACK；
- [ ] WebSocket 推送失败不会导致通知重复重试；
- [ ] 设置 QoS，Unacked 数受 prefetch 限制；
- [ ] 永久、临时、未知错误有明确分类；
- [ ] 临时错误按 5 秒、30 秒、5 分钟有限重试；
- [ ] 永久错误和超限错误进入 parking queue；
- [ ] retry/parking 副本 Confirm 后才 ACK 原消息；
- [ ] 六个故障实验全部执行并记录结果；
- [ ] 能明确说出 Level 3 仍保留的两个重复/丢失窗口。

## 23. 建议拆分提交

如果希望提交容易复查，不要把整章代码压成一个巨大提交。建议：

```text
feat: declare notification retry topology
feat: confirm RabbitMQ event publishing
feat: manually acknowledge notification events
feat: retry transient notification failures
feat: park exhausted notification events
test: cover RabbitMQ delivery failure scenarios
```

完成第 4 周后的阶段提交可归纳为：

```text
feat: make notification delivery resilient
```

提交说明应明确新增保证：Publisher 可判断 Confirm/Return，Consumer 在落库后 ACK，临时错误有限重试，永久或超限消息进入停车场。

提交说明也必须明确仍不保证：任务事务与事件发布的原子性、重复事件只产生一条通知、跨系统 Exactly Once。

## 24. 下一阶段：为什么 Level 4 必须是 Outbox 与幂等

完成本章后，RabbitMQ 链路已经有较完整的失败处理，但系统仍有两个结构性缺口。

生产者缺口：

```text
tasks 更新成功
  → 进程在 Publish 前崩溃
  → RabbitMQ 中永远没有该事件
```

消费者缺口：

```text
notifications 写入成功
  → 进程在 ACK 前崩溃
  → 消息重新投递
  → 可能创建第二条通知
```

Level 4 将分别处理：

```text
生产者：tasks + outbox_events 在同一个 MySQL 事务提交
消费者：processed_messages(event_id UNIQUE) + notifications 在同一个事务提交
```

到那时，系统才形成完整的“至少一次投递 + 业务幂等”闭环。本章的 Confirm、Return、ACK、Retry 和 Parking 不会被推翻；它们会成为 Outbox Relay 与幂等 Consumer 的可靠传输基础。
