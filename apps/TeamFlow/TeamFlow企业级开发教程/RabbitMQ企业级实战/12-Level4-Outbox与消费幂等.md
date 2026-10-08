# Level 4：用 Outbox 与消费幂等补上可靠消息闭环

> 本章目标：在 Level 3 的 Confirm、Return、manual ACK、有限重试和停车场之上，继续解决两个真正影响业务数据的问题：
>
> 1. 任务已经写入 MySQL，但应用还没有机会把事件发布到 RabbitMQ；
> 2. 通知已经写入 MySQL，但消费者还没有来得及 ACK 就崩溃，RabbitMQ 再次投递同一条消息。
>
> 本章最终形成的链路是：
>
> `业务事务 → outbox_events → Relay → RabbitMQ → 幂等 Consumer → 业务事务 → ACK`

## 1. 先从两个真实故障开始

Level 3 结束时，我们已经把 RabbitMQ 链路做得比较可靠了。发布端可以等待 Confirm，能够识别 Return；消费端使用手动 ACK，临时错误会进入延迟重试，永久错误会进入 parking queue。

但这些能力解决的主要是“消息已经进入 RabbitMQ 以后怎么办”。它们没有解决消息还没有进入 RabbitMQ 之前，以及业务写库和 ACK 之间的问题。

### 1.1 故障一：任务成功了，但没有通知

假设用户调用接口给任务分配负责人，代码大概是这样：

```go
func (s *TaskService) AssignTask(ctx context.Context, taskID, assigneeID uint) error {
    if err := s.taskRepo.UpdateAssignee(ctx, taskID, assigneeID); err != nil {
        return err
    }

    event := event.NewTaskAssigned(taskID, assigneeID)
    if err := s.publisher.Publish(ctx, event); err != nil {
        return err
    }
    return nil
}
```

这段代码看起来符合直觉：先改任务，再发消息。但它实际上做了两件分属不同系统的事情：第一件写 MySQL，第二件写 RabbitMQ。两者没有共同事务。

中间只要发生一次进程崩溃，就可能出现：

```text
MySQL：tasks.assignee_id 已经变更
RabbitMQ：没有 task.assigned.v1
```

用户看到任务已经分配成功，但被指派人永远收不到通知。更麻烦的是，接口返回给前端时，应用甚至不知道自己是在“数据库成功后崩溃”，还是在“RabbitMQ 发布失败后返回”。

### 1.2 故障二：通知写了两遍

再看消费端：

```go
func (w *NotificationWorker) Handle(ctx context.Context, d amqp091.Delivery) error {
    var e event.Envelope
    if err := json.Unmarshal(d.Body, &e); err != nil {
        return permanent("decode event", err)
    }

    if err := w.notificationService.Create(ctx, e); err != nil {
        return err
    }

    return d.Ack(false)
}
```

假设 `Create` 已经提交成功，通知表里出现了一条记录。紧接着进程在 `Ack` 之前被杀掉：

```text
通知表：已经有一条通知
RabbitMQ：因为没有收到 ACK，把原消息重新投递
Worker：再次执行 Create
通知表：可能出现第二条相同通知
```

这不是 RabbitMQ 把消息复制错了，而是手动 ACK 的正常语义：Broker 只能知道“这条消息是否确认”，不知道你的数据库事务是否成功。

### 1.3 Level 4 的核心答案

生产端不再在业务事务之后临时发布消息，而是把“将来需要发布的事件”一并写入业务数据库：

```text
更新 tasks
插入 outbox_events
提交同一个 MySQL 事务
```

只要事务提交成功，事件就已经持久化。即使 API 进程随后崩溃，单独运行的 Relay 也能在稍后把它发布到 RabbitMQ。

消费端则把“处理记录”和业务数据放进同一个事务：

```text
插入 processed_messages(event_id)
插入 notifications
提交同一个 MySQL 事务
收到事务成功结果后 ACK
```

同一条 `event_id` 第二次到达时，唯一索引会告诉我们“已经处理过”，这次重复投递就会变成安全的空操作。

## 2. 本章到底提供什么保证

先把目标说准确。工程上最危险的一句话是“保证消息绝不丢、绝不重”。在网络和多个系统参与的情况下，这个说法通常不严谨。

本章要实现的是：

```text
业务事务提交成功
    ⇒ 事件最终会被 Relay 反复尝试发布

事件至少到达一次消费者
    ⇒ 同一个 event_id 不会重复产生业务副作用
```

这通常被称为“至少一次投递 + 业务幂等”。这里的“至少一次”意味着为了避免丢失，某些故障窗口允许重复；“业务幂等”意味着重复到达不会再次创建通知、扣库存或执行其他副作用。

本章不承诺：

- API 返回成功后 RabbitMQ 立刻已经消费完成；
- 所有外部系统都参与同一个事务；
- WebSocket、邮件、短信等外部发送动作天然幂等；
- 出现网络分区时可以同时做到绝不丢、绝不重、立刻成功；
- 跨 MySQL、RabbitMQ、Redis 和第三方服务的 Exactly Once。

把保证边界讲清楚，比堆更多“可靠消息”名词更重要。

## 3. 先画出 Level 4 的完整数据流

### 3.1 生产侧

```text
POST /tasks/:id/assign
        │
        ▼
TaskService.AssignTask
        │
        ├─ 开启 MySQL 事务
        ├─ 更新 tasks
        ├─ 插入 outbox_events(status=pending)
        └─ 提交事务
                │
                ▼
         API 返回成功

Outbox Relay（独立循环）
        │
        ├─ 找到 pending 事件
        ├─ 抢占，避免多个 Relay 重复发送
        ├─ 发布到 RabbitMQ
        ├─ 等待 Confirm，并检查 Return
        └─ 标记 published，失败则安排下次重试
```

API 请求不需要等待通知消费者完成。它只需要保证任务更新和 outbox 事件一起成功落库。

### 3.2 消费侧

```text
RabbitMQ Delivery
        │
        ▼
Notification Consumer
        │
        ├─ 解码并校验 event_id
        ├─ 开启 MySQL 事务
        ├─ 插入 processed_messages(event_id)
        ├─ 已存在：回滚/结束为 duplicate
        ├─ 不存在：写 notifications
        └─ 提交事务
                │
                ▼
             ACK
```

注意顺序：ACK 永远在业务事务成功之后。否则会出现“消息被确认，但业务数据没有写入”的丢失问题。

## 4. Outbox 是什么：先不急着记名词

可以把 Outbox 理解成业务数据库旁边的一个“待发送信箱”。

以前的做法是：任务改完以后，直接拿着事件跑去 RabbitMQ 投递。如果人刚走到 RabbitMQ 门口就摔倒，信就没送出去，而且没人知道这封信存在过。

Outbox 的做法是先把信放进数据库信箱：

```text
任务记录和信箱记录同时保存
```

只要数据库事务成功，信箱里一定有这封信。之后由 Relay 定期查看信箱，把信送到 RabbitMQ。Relay 失败了可以再来；Relay 进程重启了可以继续扫描；机器突然断电了，已提交的数据库记录仍然在。

因此，Outbox 并不是一个 RabbitMQ 特性，而是一种数据库事务与消息发布之间的衔接方式。

## 5. 设计 outbox_events 表

### 5.1 最小可用字段

建议先建立下面这些字段：

```sql
CREATE TABLE outbox_events (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    event_id CHAR(36) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id BIGINT UNSIGNED NOT NULL,
    routing_key VARCHAR(200) NOT NULL,
    payload JSON NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at DATETIME(6) NOT NULL,
    locked_at DATETIME(6) NULL,
    locked_by VARCHAR(100) NULL,
    published_at DATETIME(6) NULL,
    last_error VARCHAR(1000) NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_outbox_event_id (event_id),
    KEY idx_outbox_poll (status, next_attempt_at, id),
    KEY idx_outbox_lock (status, locked_at),
    KEY idx_outbox_aggregate (aggregate_type, aggregate_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

每个字段都有明确用途：

| 字段 | 用途 |
| --- | --- |
| `event_id` | 业务事件的全局唯一 ID，也是消费幂等的关键键 |
| `event_type` | 例如 `task.assigned.v1`，方便排查和路由 |
| `aggregate_type` | 事件属于哪种业务对象，例如 `task` |
| `aggregate_id` | 哪一个任务产生了它 |
| `routing_key` | Relay 发布时使用的 RabbitMQ routing key |
| `payload` | 发布时要发送的完整事件 JSON |
| `status` | 当前是待发送、发送中、已发布还是失败 |
| `attempts` | 已经尝试发布的次数 |
| `next_attempt_at` | 未到时间不再重复扫描，避免失败事件高速空转 |
| `locked_at/locked_by` | 多个 Relay 抢占时识别所有权 |
| `published_at` | Confirm 和 Return 都成功后记录完成时间 |
| `last_error` | 最后一次失败原因，给运维排查 |

`payload` 保存完整消息，而不是只保存 `task_id` 再到发布时重新查询任务。原因是：任务后续可能继续变化，Relay 重试时应该发布“当时产生的事实”，而不是重新拼出一个已经变了的事实。

### 5.2 状态不要设计得过多

教程阶段使用四个状态足够：

```text
pending    等待首次发布或等待重试
publishing 已被某个 Relay 抢占，正在发布
published  已确认发布成功
failed     超过策略或被人工标记失败
```

`publishing` 不是永久状态。Relay 崩溃后，其他 Relay 必须能够把超时的 `publishing` 记录重新捡回来。否则一次机器重启就会留下永久卡住的事件。

不要把 `published` 理解成“消费者已经处理”。它只表示：发布端已经获得 RabbitMQ 的 Confirm，并且没有收到不可路由的 Return。消费者是否成功属于另一条链路。

### 5.3 GORM 模型

可以在 `internal/model/outbox_event.go` 中定义：

```go
type OutboxStatus string

const (
    OutboxPending    OutboxStatus = "pending"
    OutboxPublishing OutboxStatus = "publishing"
    OutboxPublished  OutboxStatus = "published"
    OutboxFailed     OutboxStatus = "failed"
)

type OutboxEvent struct {
    ID            uint64       `gorm:"primaryKey"`
    EventID       string       `gorm:"size:36;not null;uniqueIndex:uk_outbox_event_id"`
    EventType     string       `gorm:"size:100;not null"`
    AggregateType string       `gorm:"size:50;not null"`
    AggregateID   uint64       `gorm:"not null;index:idx_outbox_aggregate"`
    RoutingKey    string       `gorm:"size:200;not null"`
    Payload       datatypes.JSON `gorm:"type:json;not null"`
    Status        OutboxStatus `gorm:"size:20;not null;index:idx_outbox_poll"`
    Attempts      int          `gorm:"not null;default:0"`
    NextAttemptAt time.Time    `gorm:"not null;index:idx_outbox_poll"`
    LockedAt      *time.Time
    LockedBy      *string      `gorm:"size:100"`
    PublishedAt   *time.Time
    LastError     *string      `gorm:"size:1000"`
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

实际使用时，根据项目的 GORM 版本调整 `datatypes.JSON` 的导入和 tag。重点不是模型写法本身，而是：`event_id` 唯一、待发送索引存在、状态和锁信息可查询。

## 6. 生产业务：把任务更新和 Outbox 插入放在同一个事务

### 6.1 错误写法：事务提交以后才补 Outbox

下面这样仍然有问题：

```go
err := db.Transaction(func(tx *gorm.DB) error {
    return tx.Model(&task).Update("assignee_id", assigneeID).Error
})
if err != nil {
    return err
}

return db.Create(&outbox).Error
```

任务更新已经提交后，进程可能在 `db.Create(&outbox)` 之前崩溃，缺口和直接 Publish 没有本质区别。

### 6.2 正确顺序

```go
func (s *TaskService) AssignTask(
    ctx context.Context,
    taskID uint,
    assigneeID uint,
    operatorID uint,
) error {
    now := time.Now().UTC()
    assigned := event.NewTaskAssigned(taskID, 0, assigneeID, operatorID)

    payload, err := json.Marshal(assigned)
    if err != nil {
        return fmt.Errorf("marshal task assigned event: %w", err)
    }

    err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        var task model.Task
        if err := tx.First(&task, taskID).Error; err != nil {
            return fmt.Errorf("find task: %w", err)
        }

        if err := s.checkCanAssign(ctx, tx, task, assigneeID, operatorID); err != nil {
            return err
        }

        if err := tx.Model(&task).Updates(map[string]interface{}{
            "assignee_id": assigneeID,
            "updated_at":  now,
        }).Error; err != nil {
            return fmt.Errorf("update task assignee: %w", err)
        }

        assigned.Payload = event.TaskAssignedPayload{
            TaskID: task.ID,
            ProjectID: task.ProjectID,
            AssigneeID: assigneeID,
            OperatorID: operatorID,
        }
        payload, err = json.Marshal(assigned)
        if err != nil {
            return fmt.Errorf("marshal final task event: %w", err)
        }

        outbox := &model.OutboxEvent{
            EventID: assigned.EventID,
            EventType: assigned.EventType,
            AggregateType: "task",
            AggregateID: uint64(task.ID),
            RoutingKey: assigned.EventType,
            Payload: datatypes.JSON(payload),
            Status: model.OutboxPending,
            NextAttemptAt: now,
            CreatedAt: now,
            UpdatedAt: now,
        }
        if err := tx.Create(outbox).Error; err != nil {
            return fmt.Errorf("create outbox event: %w", err)
        }
        return nil
    })
    if err != nil {
        return err
    }

    // 到这里，任务和待发送事件已经同时提交。
    // 不在这里直接调用 RabbitMQ，避免业务请求重新承担双写窗口。
    return nil
}
```

这里最需要理解的是事务边界：

1. 先查任务并校验权限；
2. 修改 `tasks`；
3. 把同一个业务事实序列化成 Outbox 记录；
4. `tx.Create` 成功后，由数据库一次性提交两张表的变化；
5. 只有事务返回 `nil`，API 才返回成功。

如果第 2 步失败，第 3 步不会单独留下事件；如果第 4 步提交失败，任务和 Outbox 一起回滚。只要事务提交成功，即使进程紧接着崩溃，Relay 仍能找到这条事件。

### 6.3 为什么不能把发布动作放进数据库事务

有人会想到：在 `Transaction` 回调里直接 Publish，成功后再提交数据库。这样也不是真正的双系统事务：

```text
RabbitMQ 发布成功
    → MySQL 提交失败
    → 消费者收到一个数据库里不存在的事件
```

RabbitMQ 不会因为 MySQL 回滚而撤回已经发布的消息。正确做法是让数据库事务只负责保存“应该发布什么”，把真正的网络发送交给事务外的 Relay。

## 7. Relay：把数据库信箱里的事件送出去

Relay 可以是 API 进程里的后台 goroutine，也可以是独立 Worker。生产环境通常更建议独立进程，因为发布事件和处理 HTTP 请求的故障、吞吐、部署节奏不同；教程阶段两种方式都可以，但代码边界要清楚。

Relay 主要做五件事：

```text
查询 → 抢占 → 发布 → 判断结果 → 更新状态
```

### 7.1 定义 Relay 接口

```go
type EventPublisher interface {
    PublishConfirmed(ctx context.Context, routingKey string, body []byte, eventID string) error
}

type OutboxRelay struct {
    repo      OutboxRepository
    publisher EventPublisher
    workerID  string
    batchSize int
    clock     Clock
}
```

Relay 不应该直接拼 SQL、直接操作 `amqp091.Channel` 和直接决定业务事件内容。它只负责把一条 Outbox 记录交给可靠 Publisher。

### 7.2 扫描条件

查询待发送事件时，条件至少包括：

```sql
status = 'pending'
AND next_attempt_at <= NOW(6)
```

还要回收长时间停留在 `publishing` 的记录：

```sql
status = 'publishing'
AND locked_at < NOW(6) - INTERVAL 2 MINUTE
```

两分钟只是示例。它应该明显大于一次正常 Publish Confirm 的超时时间，否则慢一点的事件会被两个 Relay 同时抢占。

### 7.3 抢占不是为了追求绝不重复

多个 Relay 可能同时运行。如果它们都读到同一条 `pending` 记录，就可能同时发布两份。抢占的目标是减少这种重复，而不是宣称从此绝不重复。

一种兼容 MySQL 8 的实现是使用 `FOR UPDATE SKIP LOCKED`：

```go
func (r *OutboxRepository) ClaimBatch(
    ctx context.Context,
    workerID string,
    limit int,
    now time.Time,
) ([]model.OutboxEvent, error) {
    var events []model.OutboxEvent

    err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        query := tx.Clauses(clause.Locking{
            Strength: "UPDATE",
            Options:  "SKIP LOCKED",
        }).Where(`
            (status = ? AND next_attempt_at <= ?)
            OR (status = ? AND locked_at < ?)
        `,
            model.OutboxPending,
            now,
            model.OutboxPublishing,
            now.Add(-2*time.Minute),
        ).Order("id ASC").Limit(limit)

        if err := query.Find(&events).Error; err != nil {
            return err
        }

        for i := range events {
            lockedAt := now
            if err := tx.Model(&events[i]).Updates(map[string]interface{}{
                "status":    model.OutboxPublishing,
                "locked_at": lockedAt,
                "locked_by": workerID,
                "updated_at": now,
            }).Error; err != nil {
                return err
            }
        }
        return nil
    })
    return events, err
}
```

这段代码的流程是：先锁住本批记录，再把它们标记为当前 Worker 正在处理，事务提交后释放行锁。其他 Relay 看到这些记录是 `publishing`，不会立即再次领取。

但有一个重要事实：数据库状态更新和 RabbitMQ Publish 仍然不在同一个事务里。所以即使有抢占，下面这个窗口仍然存在：

```text
Relay 抢占成功
Relay 发布成功
Relay 还没来得及标记 published 就崩溃
```

恢复后记录会被重新领取，RabbitMQ 可能收到第二份。生产者侧也必须接受这个重复，并依靠消费者幂等兜底。

### 7.4 发布并更新状态

```go
func (r *OutboxRelay) ProcessOne(ctx context.Context, item model.OutboxEvent) {
    err := r.publisher.PublishConfirmed(
        ctx,
        item.RoutingKey,
        item.Payload,
        item.EventID,
    )
    if err == nil {
        if updateErr := r.repo.MarkPublished(ctx, item.ID, r.workerID, r.clock.Now()); updateErr != nil {
            // 发布已经成功，状态更新失败不能把事件当作未发送而静默丢弃。
            // 后续可能重复发布，消费者幂等负责把重复变成空操作。
            r.logger.Error("mark outbox published failed", "id", item.ID, "event_id", item.EventID, "error", updateErr)
        }
        return
    }

    attempts := item.Attempts + 1
    if attempts >= r.maxAttempts || IsPermanentPublishError(err) {
        _ = r.repo.MarkFailed(ctx, item.ID, r.workerID, attempts, ShortError(err))
        return
    }

    next := r.clock.Now().Add(Backoff(attempts))
    _ = r.repo.MarkRetry(ctx, item.ID, r.workerID, attempts, next, ShortError(err))
}
```

成功分支尤其重要：不能因为 `MarkPublished` 失败就认为 RabbitMQ 没收到消息。此时事实是“消息大概率已经发布，数据库状态没有跟上”。更安全的处理是记录告警，让它稍后重试；即使重复发布，消费幂等也会保护业务数据。

### 7.5 状态更新必须带上 worker 条件

```go
func (r *OutboxRepository) MarkPublished(
    ctx context.Context,
    id uint64,
    workerID string,
    at time.Time,
) error {
    result := r.db.WithContext(ctx).Model(&model.OutboxEvent{}).
        Where("id = ? AND status = ? AND locked_by = ?", id, model.OutboxPublishing, workerID).
        Updates(map[string]interface{}{
            "status":       model.OutboxPublished,
            "published_at": at,
            "locked_at":    nil,
            "locked_by":    nil,
            "updated_at":   at,
        })

    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected != 1 {
        return fmt.Errorf("outbox event %d ownership lost", id)
    }
    return nil
}
```

为什么要检查 `locked_by`？因为旧 Worker 可能已经超时，新的 Worker 重新领取了这条记录。如果旧 Worker 回来后无条件执行 `UPDATE outbox_events SET status='published'`，就可能覆盖新 Worker 的状态。状态变更必须证明自己仍然拥有这条记录。

## 8. Publisher 仍然必须使用 Confirm 和 Return

Outbox 只能保证事件存在数据库，不能替代 RabbitMQ 的发布确认。Relay 发布时仍然应沿用 Level 3 的可靠 Publisher：

```go
func (p *Publisher) PublishConfirmed(
    ctx context.Context,
    routingKey string,
    body []byte,
    eventID string,
) error {
    if err := p.ch.Confirm(false); err != nil {
        return fmt.Errorf("enable publisher confirm: %w", err)
    }

    confirms := p.ch.NotifyPublish(make(chan amqp091.Confirmation, 1))
    returns := p.ch.NotifyReturn(make(chan amqp091.Return, 1))

    if err := p.ch.PublishWithContext(ctx, rabbitmq.EventsExchange, routingKey, true, false,
        amqp091.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp091.Persistent,
            MessageId:    eventID,
            Type:         routingKey,
            Body:         body,
        }); err != nil {
        return fmt.Errorf("publish outbox event: %w", err)
    }

    select {
    case returned := <-returns:
        return fmt.Errorf("event returned: code=%d text=%s", returned.ReplyCode, returned.ReplyText)
    case confirmation := <-confirms:
        if !confirmation.Ack {
            return fmt.Errorf("broker nack delivery tag=%d", confirmation.DeliveryTag)
        }
        return nil
    case <-ctx.Done():
        return fmt.Errorf("wait publish confirmation: %w", ctx.Err())
    }
}
```

教程代码需要根据具体 Publisher 的生命周期完善 Notify channel 管理。生产代码不能每条消息都随意重复调用 `Confirm` 或创建无法回收的监听器；更稳妥的方式是 Publisher 启动时进入 Confirm 模式，使用发布序号维护待确认消息，并由一个确认循环分发结果。

这里先关注逻辑：

- Outbox 记录没有发布 Confirm，状态不能标记 `published`；
- `mandatory=true` 的消息收到 Return，不能标记成功；
- Confirm ACK 和 Return 的结果要一起判断；
- Confirm 超时不能直接删除 Outbox，应该保留并重试；
- 重试可能产生重复，所以必须有 `event_id`。

## 9. 消费幂等：先理解“挡住第二次副作用”

RabbitMQ 不知道“哪一条消息已经给某个用户创建过通知”。它只负责投递和确认。幂等必须由业务数据库做。

最简单的思路是：每次消费前，先把 `event_id` 写入一张处理记录表。因为 `event_id` 有唯一索引，同一事件第二次插入会失败。关键是：处理记录和业务写入必须在同一个事务里。

### 9.1 processed_messages 表

```sql
CREATE TABLE processed_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    event_id CHAR(36) NOT NULL,
    consumer_name VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    processed_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_processed_event_consumer (event_id, consumer_name),
    KEY idx_processed_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`consumer_name` 是否放进唯一键，要看业务含义：

- 如果每个事件全局只能被处理一次，使用 `UNIQUE(event_id)`；
- 如果通知消费者、审计消费者、搜索索引消费者都要各自处理，就使用 `UNIQUE(event_id, consumer_name)`。

TeamFlow 中 `task.assigned.v1` 可能同时被通知和审计订阅，因此本章使用组合唯一键。它表示“某个消费者对某个事件是否已经完成”，不会阻止另一个消费者处理同一事件。

### 9.2 不要只用内存 map

下面这种方案只能挡住当前进程生命周期内的重复：

```go
var seen sync.Map

if _, loaded := seen.LoadOrStore(eventID, struct{}{}); loaded {
    return nil
}
```

进程重启后 map 清空；多副本之间也各自有一份 map；内存中的“已处理”还可能在数据库事务失败时提前记录。因此它不能承担业务幂等，只能作为可选的短期性能优化，不能替代数据库唯一约束。

## 10. 把处理记录和通知写入同一个事务

### 10.1 需要区分三种结果

消费者业务处理最好不要只返回 `error`，因为以下三种情况语义不同：

```text
Processed：第一次处理，通知已写入
Duplicate：之前已经处理成功，本次不再写入
Retryable：本次没有完成，允许进入 Level 3 retry
Permanent：消息本身无法修复，进入 parking
```

可以定义：

```go
type HandleResult int

const (
    ResultProcessed HandleResult = iota
    ResultDuplicate
)

type BusinessError struct {
    Class string
    Err   error
}

func (e *BusinessError) Error() string { return e.Err.Error() }
func (e *BusinessError) Unwrap() error { return e.Err }
```

实际项目可以沿用 Level 3 已有的 `permanent`、`transient` 错误类型，不必为了本章重复创建一套冲突的错误体系。

### 10.2 推荐的事务流程

```go
func (w *NotificationWorker) Handle(
    ctx context.Context,
    e event.Envelope,
) (HandleResult, error) {
    if e.EventID == "" {
        return 0, permanent("missing event_id", errors.New("event_id is required"))
    }
    if e.EventType != event.TaskAssignedV1 {
        return 0, permanent("unsupported event type", fmt.Errorf("type=%s", e.EventType))
    }

    payload, err := decodeTaskAssigned(e.Payload)
    if err != nil {
        return 0, permanent("decode task.assigned", err)
    }

    var duplicate bool
    err = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        record := &model.ProcessedMessage{
            EventID: e.EventID,
            ConsumerName: w.consumerName,
            EventType: e.EventType,
            Status: "processing",
            CreatedAt: w.clock.Now(),
        }

        if err := tx.Create(record).Error; err != nil {
            if isDuplicateKey(err) {
                duplicate = true
                return nil
            }
            return transient("insert processed message", err)
        }

        notification := &model.Notification{
            UserID: payload.AssigneeID,
            Type: "task_assigned",
            Title: "你收到了一项新任务",
            Body: buildTaskAssignedBody(payload),
            ReferenceType: "task",
            ReferenceID: payload.TaskID,
            CreatedAt: w.clock.Now(),
        }
        if err := tx.Create(notification).Error; err != nil {
            return classifyDatabaseError("create notification", err)
        }

        processedAt := w.clock.Now()
        if err := tx.Model(record).Updates(map[string]interface{}{
            "status":       "processed",
            "processed_at": processedAt,
        }).Error; err != nil {
            return transient("mark processed message", err)
        }
        return nil
    })
    if err != nil {
        return 0, err
    }
    if duplicate {
        return ResultDuplicate, nil
    }

    // 数据库事务成功后，实时推送只是尽力而为。
    // WebSocket 失败不能让已经提交的通知重新走一遍创建逻辑。
    if err := w.realtime.PushIfOnline(ctx, payload.AssigneeID); err != nil {
        w.logger.Warn("realtime notification failed", "event_id", e.EventID, "error", err)
    }
    return ResultProcessed, nil
}
```

逐步看这段代码：

1. 先检查 `event_id` 和事件类型。消息格式都不合法时，不应反复重试；它属于永久错误。
2. 开启数据库事务。
3. 先插入 `processed_messages`，状态暂时是 `processing`。
4. 如果唯一键冲突，说明这个消费者已经成功处理过同一事件。将本次视为重复消息，事务正常结束，不创建新通知。
5. 如果不是重复键，而是数据库连接超时，则返回临时错误，不执行 ACK，让 Level 3 的重试接管。
6. 第一次插入成功后，再写入 `notifications`。
7. 通知写成功后，把处理记录改成 `processed`。
8. 事务提交成功，才返回成功结果。
9. WebSocket 放在事务外执行。它失败只影响实时体验，不影响数据库中已经存在的通知。

### 10.3 为什么重复分支可以 ACK

第二次收到同一事件时，`processed_messages` 唯一键冲突。这里不能把它当成业务失败，否则同一条消息会不断进入重试队列。

重复意味着：

```text
之前那次已经提交了业务结果
本次没有新的业务工作需要做
```

因此重复分支应返回成功，让 Consumer 调用 `Ack(false)`。ACK 的含义是“这次 Delivery 不需要再投递”，并不是“我刚刚一定新写了一行通知”。

## 11. 一个非常容易犯的错误：先标记已处理，再写业务表

下面的顺序是错误的：

```go
tx.Create(&processedMessage)
tx.Commit()

if err := db.Create(&notification).Error; err != nil {
    return err
}
```

如果通知写入失败，消息进入 retry。但下一次消费先看到 `processedMessage`，误以为已经处理成功，于是直接 ACK。结果是：

```text
processed_messages：有记录
notifications：没有记录
RabbitMQ：消息已 ACK，不会再来
```

这就把本来可以重试的临时失败变成了真正的数据丢失。

处理记录不能成为业务写入的“先行锁”。它必须和业务副作用处于同一个数据库事务，成功一起提交，失败一起回滚。

## 12. Consumer Adapter 如何决定 ACK、Retry 和 Parking

业务 Handler 只返回业务结果，RabbitMQ Adapter 统一决定动作：

```go
func (c *Consumer) consumeOne(ctx context.Context, d amqp091.Delivery) {
    envelope, err := decodeEnvelope(d.Body)
    if err != nil {
        c.park(ctx, d, permanent("decode envelope", err))
        return
    }

    result, err := c.worker.Handle(ctx, envelope)
    if err == nil {
        // Processed 和 Duplicate 都说明本条 Delivery 可以结束。
        if ackErr := d.Ack(false); ackErr != nil {
            c.logger.Error("ack notification failed", "event_id", envelope.EventID, "error", ackErr)
        }
        c.logger.Info("notification handled", "event_id", envelope.EventID, "result", result)
        return
    }

    var classified *BusinessError
    if errors.As(err, &classified) && classified.Class == "permanent" {
        c.park(ctx, d, err)
        return
    }

    if c.retryOrPark(ctx, d, envelope, err) {
        return
    }
    c.logger.Error("failed to transfer message", "event_id", envelope.EventID, "error", err)
}
```

需要特别注意：`retryOrPark` 或 `park` 只有在副本发布 Confirm 成功后才能 ACK 原消息。这是 Level 3 已经建立的规则，本章不改变它。

如果 retry copy 发布失败，原消息不能 ACK；否则主消息和 retry copy 都没有，消息就丢了。此时可以让原 Delivery 保持未确认，或者在连接关闭时由 Broker 重新投递。

## 13. Outbox Relay 的重试策略

### 13.1 为什么 Relay 不能死循环

如果 Relay 每秒扫描一次失败事件，失败后立刻再次扫描，可能造成：

```text
RabbitMQ 不可用
→ 同一事件每秒发布一次
→ 日志刷屏
→ 数据库持续被更新
→ 网络恢复时瞬间积压更多请求
```

因此 Outbox 也要有退避时间：

```go
func OutboxBackoff(attempt int) time.Duration {
    switch attempt {
    case 1:
        return 5 * time.Second
    case 2:
        return 30 * time.Second
    case 3:
        return 5 * time.Minute
    default:
        return 15 * time.Minute
    }
}
```

可以加入一点随机抖动，避免大量事件在同一秒同时重试：

```go
func OutboxBackoffWithJitter(attempt int, jitter time.Duration) time.Duration {
    base := OutboxBackoff(attempt)
    // 示例中省略随机数注入，生产代码应通过接口注入随机源方便测试。
    return base + jitter
}
```

### 13.2 永久发布错误与临时发布错误

以下情况通常值得短暂重试：

- Connection 暂时断开；
- Channel 关闭；
- Confirm 等待超时；
- Broker 暂时不可用；
- 网络连接被重置。

以下情况通常不是重试几次就能解决：

- Exchange 名称配置错误；
- routing key 永远没有 binding；
- 消息 JSON 永远无法序列化；
- Queue 或 Exchange 参数与已有拓扑冲突。

不过“不可路由”是否永久，要结合业务。发布到错误 routing key 一般是代码或配置错误，应进入 `failed` 并报警；临时删除 binding 做故障实验时，也可以把它作为临时故障。不要只凭错误字符串硬编码，最好把 RabbitMQ 错误映射到结构化错误类型。

## 14. 并发与数据库锁：先保证正确，再追求速度

### 14.1 一个事件只允许一个 Relay 处理吗

理想状态下，同一时间只有一个 Relay 持有一条 Outbox 记录。但由于网络和进程崩溃，实际仍然可能重复发送。因此设计目标是：

```text
正常情况下尽量不重复
异常情况下即使重复也不产生错误业务副作用
```

这两层分别由抢占和消费幂等负责。

### 14.2 不要在事务里等待 RabbitMQ

错误做法：

```text
开启数据库事务
→ 锁住 outbox_events
→ 等 RabbitMQ Confirm
→ 更新 published
→ 提交数据库事务
```

网络等待期间数据库行锁一直持有，Relay 并发升高后会造成锁等待；RabbitMQ 变慢时，数据库也被拖慢。更合适的方式是：短事务抢占，事务外发布，短事务更新状态。

### 14.3 批量大小不要盲目扩大

批量 100 或 500 看起来吞吐更高，但会带来更大的锁范围、更多内存占用和更长的恢复时间。初始可以使用 20 或 50，观察：

- Outbox 待发送数量；
- 单批处理耗时；
- 数据库锁等待；
- RabbitMQ Confirm 延迟；
- Relay 崩溃后的重复数量。

可靠消息首先要可解释。没有指标和故障实验之前，不要仅凭“批量越大越快”调整。

## 15. 事件结构的兼容性

Outbox 保存的是事件产生时的完整 JSON，因此后续代码升级时仍然要面对事件版本兼容。

例如 Level 4 当前使用：

```json
{
  "event_id": "9b8...",
  "event_type": "task.assigned.v1",
  "schema_version": 1,
  "occurred_at": "2026-08-28T10:00:00Z",
  "payload": {
    "task_id": 10,
    "project_id": 3,
    "assignee_id": 8,
    "operator_id": 2
  }
}
```

如果以后要增加字段，优先采用向后兼容的方式，例如新增可选字段；如果字段语义发生不兼容变化，创建 `task.assigned.v2`，不要直接让旧消费者接收无法理解的 JSON。

消费者要明确：

- 缺少必要字段属于永久错误；
- 多出的未知字段通常可以忽略；
- 不支持的 `schema_version` 不应无限重试；
- `event_id` 一旦产生就不能在重试时改变。

同一事件进入 retry queue、parking queue 或 Outbox 重发时，必须保留原始 `event_id`。如果每次重试都生成一个新 ID，数据库唯一索引就失去意义。

## 16. 清理策略：published 记录不能无限增长

Outbox 和 processed_messages 都是生产数据表，不能因为“功能完成”就永久保留所有历史记录而不考虑容量。

### 16.1 Outbox 清理

`published` 记录通常保留一段时间，例如 7 天或 30 天，供排查发布轨迹。清理条件可以是：

```sql
DELETE FROM outbox_events
WHERE status = 'published'
  AND published_at < NOW() - INTERVAL 30 DAY
LIMIT 1000;
```

要分批删除，避免长事务和大范围锁。不要直接按 `id < 某个值` 删除，因为事件 ID 和业务时间未必严格对应。

### 16.2 幂等记录清理

`processed_messages` 不能比可能重复到达的消息生命周期短。只要 RabbitMQ、retry queue、parking 重放工具还可能重新投递旧事件，幂等记录就必须保留。

如果只保留 7 天，却允许 30 天前的 parking 消息人工重放，那么旧事件会再次创建通知。清理策略必须和重放策略一致：

```text
允许重放多久
→ processed_messages 至少保留多久
```

对于需要永久防重的业务，不能简单删除处理记录；可以归档到历史表，或在业务表本身使用唯一业务键。

## 17. “event_id 幂等”不等于所有副作用都幂等

本章的数据库通知写入可以通过事务和唯一键做到幂等，但 WebSocket 推送是另一个问题。

推荐的业务边界是：

```text
数据库通知：必须可靠、幂等
WebSocket：数据库成功后尽力推送
```

用户即使错过 WebSocket，也可以在下一次请求通知列表时看到已经落库的通知。

如果业务要求外部邮件也不能重复，需要为邮件发送建立独立的发送记录和幂等键：

```text
notification_id + channel=email
```

邮件供应商是否支持幂等键、网络超时后是否可能已发送，也必须单独分析。不要因为 `event_id` 在本地数据库唯一，就认为第三方已经被保护。

## 18. 故障场景逐个推演

### 18.1 任务事务回滚

```text
更新 tasks 失败
→ outbox_events 插入也回滚
→ API 返回失败
→ Relay 没有事件可发送
```

这是正确结果，因为业务事实并没有成立。

### 18.2 任务事务成功，API 进程立即崩溃

```text
tasks 和 outbox_events 一起提交
→ API 崩溃
→ Relay 后续扫描到 pending
→ 发布并 Confirm
→ 通知消费者处理
```

这正是 Outbox 解决的第一个窗口。

### 18.3 Relay 发布成功，标记 published 前崩溃

```text
RabbitMQ 已 Confirm
→ Relay 崩溃
→ Outbox 仍是 publishing
→ 超时回收
→ 可能再次发布
→ Consumer 用 event_id 判重
```

不会保证 RabbitMQ 只有一份，但会保证通知业务不产生第二份副作用。

### 18.4 Consumer 写通知失败

```text
processed_messages 插入和 notifications 写入在同一事务
→ notifications 超时
→ 整个事务回滚
→ 不 ACK 原消息
→ Level 3 retry
```

下一次重试时没有成功的处理记录，因此可以正常再次处理。

### 18.5 Consumer 事务提交，ACK 前崩溃

```text
processed_messages=processed
notifications 已提交
→ Worker 崩溃
→ RabbitMQ 重新投递
→ 插入 processed_messages 触发唯一键冲突
→ 判定 Duplicate
→ ACK
```

这正是消费幂等解决的第二个窗口。

### 18.6 数据库提交成功，但 WebSocket 失败

```text
通知已落库
→ WebSocket 连接不存在或发送失败
→ 记录告警
→ ACK 消息
```

不能因为 WebSocket 失败而重新创建通知。实时推送是通知的一个加速通道，不是通知数据的唯一存储。

## 19. 测试应该怎么写

### 19.1 Outbox 事务测试

至少覆盖：

- 任务更新成功时同时有一条 `pending` Outbox；
- 任务更新失败时没有 Outbox；
- Outbox 插入失败时任务更新回滚；
- `event_id` 重复时事务失败而不是产生两条事件；
- Outbox payload 保存的是事件产生时的完整内容；
- API 事务成功后不直接调用 RabbitMQ。

伪代码：

```go
func TestAssignTaskCreatesTaskAndOutboxAtomically(t *testing.T) {
    db := newTestDB(t)
    service := NewTaskService(db, fakePublisherThatMustNotBeCalled{})

    require.NoError(t, service.AssignTask(ctx, 10, 8, 2))

    var task model.Task
    require.NoError(t, db.First(&task, 10).Error)
    require.Equal(t, uint(8), task.AssigneeID)

    var outbox model.OutboxEvent
    require.NoError(t, db.Where("aggregate_id = ?", 10).First(&outbox).Error)
    require.Equal(t, model.OutboxPending, outbox.Status)
    require.NotEmpty(t, outbox.EventID)
}
```

### 19.2 Relay 测试

- pending 事件会被领取并标记 `publishing`；
- 未到 `next_attempt_at` 的事件不会被领取；
- Confirm ACK 后标记 `published`；
- Return、NACK、Confirm timeout 会保留事件并安排重试；
- 达到最大次数后标记 `failed`；
- 旧 Worker 不能覆盖新 Worker 的状态；
- Relay 发布成功但更新状态失败时会记录告警；
- Relay 重启后能回收超时的 `publishing`。

### 19.3 Consumer 幂等测试

最关键的测试不是“正常消息能写入通知”，而是连续处理同一个事件两次：

```go
func TestNotificationConsumerIsIdempotent(t *testing.T) {
    worker := newNotificationWorker(t)
    e := validTaskAssignedEvent("event-1")

    result1, err := worker.Handle(ctx, e)
    require.NoError(t, err)
    require.Equal(t, ResultProcessed, result1)

    result2, err := worker.Handle(ctx, e)
    require.NoError(t, err)
    require.Equal(t, ResultDuplicate, result2)

    require.Equal(t, 1, countNotificationsByEventOrTask(t, e))
    require.Equal(t, 1, countProcessedMessages(t, e.EventID))
}
```

还要覆盖事务回滚：模拟通知写入失败，确认 `processed_messages` 也不会留下 `processed` 记录；随后解除故障再次处理，消息应能成功。

### 19.4 并发幂等测试

使用两个 goroutine 同时处理相同 `event_id`，最终应该满足：

```text
两个调用中最多一个真正创建通知
另一个调用返回 Duplicate，或在唯一键竞争后读取已处理结果
通知表最终只有一条
```

不要只在单线程测试里验证幂等。唯一索引和数据库事务的价值，正是在并发竞争时体现。

## 20. 六个建议执行的故障实验

### 实验一：事务提交后杀死 API

在事务提交后、HTTP 响应返回前临时终止 API。

预期：`tasks` 和 `outbox_events` 都存在；Relay 重启或下一轮扫描后能够发送通知。这个实验证明事件没有依赖 API 进程的内存。

### 实验二：RabbitMQ 暂时不可用

提交任务后停止 RabbitMQ。

预期：Outbox 仍然是 `pending` 或 `publishing`；Relay 按退避时间重试，不删除事件；RabbitMQ 恢复后最终变成 `published`。

### 实验三：Relay 在 Confirm 后、状态更新前退出

在 Publisher 返回成功后人为阻塞，再终止 Relay。

预期：Outbox 可能被重新领取并再次发布；消费者日志显示一次 `processed`、一次 `duplicate`；通知表只有一条。

### 实验四：Consumer 在事务提交后、ACK 前退出

让业务事务完成后暂停 ACK，终止 Worker。

预期：RabbitMQ 重新投递；第二次消费触发唯一键冲突，被识别为 Duplicate；不会创建第二条通知。

### 实验五：Consumer 在事务提交前退出

在写通知事务中途终止 Worker。

预期：数据库事务回滚；消息重新投递后可以完整执行；不会留下只有 `processed_messages` 没有通知的脏状态。

### 实验六：人工重放 parking 消息

修复一个永久错误后，复制一条停车消息重新发布，保持原始 `event_id`。

预期：如果它以前已经成功处理过，消费者返回 Duplicate；如果以前只是停车、从未成功处理，则正常创建通知。不要在重放时生成新 `event_id`，否则无法区分真正的新事件和旧事件重放。

## 21. 监控与日志：要能回答“现在卡在哪里”

Outbox 引入后，排查链路会多一个数据库阶段。结构化日志建议增加：

```text
component=outbox_relay
outbox_id
event_id
event_type
attempt
worker_id
action=claim|publish|published|retry|failed|recover
duration_ms
error_class
```

消费日志继续沿用 Level 3 字段，并增加幂等结果：

```text
component=notification_consumer
event_id
event_type
consumer_name
action=processed|duplicate|retry|park|ack
notification_id
attempt
duration_ms
```

关键指标包括：

- Outbox pending 数量；
- 最老 pending 事件年龄；
- publishing 超时数量；
- 发布 Confirm 延迟；
- 发布失败次数和失败类型；
- Consumer processed、duplicate、retry、park 数量；
- 幂等冲突比例；
- 通知事务耗时；
- ACK 失败数量。

如果 `duplicate` 数量突然升高，可能是 Relay 重复发布、网络抖动、Consumer 频繁在 ACK 前崩溃，或者上游重复生成了 `event_id`。幂等日志不仅是保护机制，也是定位系统问题的重要证据。

## 22. 常见错误与排查方式

| 症状 | 常见原因 | 检查顺序 |
| --- | --- | --- |
| 任务成功但没有 Outbox | Outbox 没加入同一事务，或插入失败被吞掉 | 查事务日志和 `outbox_events` |
| Outbox 一直 pending | Relay 未运行、扫描条件错误、时间字段时区不一致 | 查 Relay 心跳、`next_attempt_at`、数据库时区 |
| Outbox 一直 publishing | Relay 崩溃后没有回收超时锁 | 查 `locked_at`、回收条件和 Worker 日志 |
| published 数量增加但通知没有 | Consumer 未运行、路由错误、队列积压 | 查 Return、binding、Ready/Unacked |
| 同一事件通知两条 | 没有唯一键、event_id 在重试时改变、业务表不在事务内 | 查 `event_id`、事务边界和数据库约束 |
| processed 有记录但通知没有 | 处理记录先提交，通知写入在事务外 | 合并为同一个事务 |
| 重复消息一直 retry | 唯一键冲突被当作普通数据库错误 | 将重复键映射为 Duplicate 并 ACK |
| WebSocket 失败导致通知重复 | 把实时推送失败作为整条消息失败 | 数据库成功后将推送改为尽力而为 |
| 多个 Relay 反复发送 | 抢占条件或 worker ownership 检查不完整 | 查 `FOR UPDATE SKIP LOCKED` 和状态更新条件 |
| 清理后旧消息重放产生重复通知 | processed_messages 保留时间短于重放窗口 | 调整保留策略或限制重放范围 |

## 23. 不要过早引入的复杂方案

看到 Outbox 后，容易马上加入分布式锁、Kafka、事务消息、全局一致性平台和复杂调度中心。Level 4 当前不需要这些东西。

先把下面这条最小链路做正确：

```text
同一 MySQL 事务写业务表和 outbox_events
→ Relay 可靠发布
→ Confirm/Return
→ Consumer 事务写 processed_messages 和业务表
→ 成功后 ACK
```

当系统规模变大，再根据证据处理具体问题：

- 单表太大，再做归档和分区；
- 多 Relay 抢占竞争，再优化批量和索引；
- 发布吞吐不足，再引入发布序号和并发窗口；
- 跨多个业务库，再讨论更高层的事件架构；
- 外部邮件重复，再为邮件通道设计自己的幂等键。

不要用一个复杂组件替代对事务边界的理解。Outbox 的价值恰恰在于它用一个普通的关系型数据库表，把最关键的可靠性问题变得可查询、可重试、可审计。

## 24. 推荐的代码目录

本章落地时，可以按下面的边界组织代码：

```text
internal/
├── event/
│   ├── event.go
│   └── task_event.go
├── model/
│   ├── outbox_event.go
│   └── processed_message.go
├── outbox/
│   ├── repository.go
│   ├── relay.go
│   └── relay_test.go
├── messaging/rabbitmq/
│   ├── publisher.go
│   ├── consumer.go
│   ├── topology.go
│   └── publisher_test.go
└── worker/
    ├── notification_worker.go
    └── notification_worker_test.go
```

职责边界如下：

| 模块 | 只负责什么 |
| --- | --- |
| `event` | 业务事件结构和构造，不依赖 RabbitMQ |
| `model` | 数据库表映射和状态常量 |
| `outbox/repository` | 查询、抢占、状态更新，不决定业务事件内容 |
| `outbox/relay` | 调度 Outbox 发布和失败退避 |
| `messaging/rabbitmq` | Confirm、Return、Channel 和 Delivery 细节 |
| `worker` | 解码事件、业务事务、幂等和实时通知 |

这样拆分后，测试业务幂等不需要启动 RabbitMQ；测试 Relay 不需要执行 HTTP；测试 Publisher 也不需要了解任务权限规则。

## 25. 配置建议

```yaml
rabbitmq:
  publish_confirm_timeout: 5s
  prefetch: 20
  consumer_concurrency: 4

outbox:
  poll_interval: 1s
  batch_size: 20
  max_attempts: 10
  publishing_timeout: 2m
  published_retention: 720h

idempotency:
  consumer_name: notification-worker
  processed_retention: 720h
```

配置启动时应校验：

- `poll_interval` 大于 0；
- `batch_size` 在合理范围内；
- `max_attempts` 大于 0；
- `publishing_timeout` 大于 Confirm timeout；
- `consumer_name` 非空且稳定，不能每次启动随机生成；
- 清理时间不能短于允许的 retry 和 parking 重放窗口。

`consumer_name` 很重要。如果今天叫 `notification-worker`，明天改成 `notification-worker-v2`，而唯一键包含 consumer name，那么同一事件可能被新旧消费者各处理一次。升级消费者时，只有在确实需要两套独立业务处理记录时才改变名称；普通代码升级应保持名称不变。

## 26. 本章完成定义

完成下面清单后，Level 4 才算真正完成：

- [ ] `outbox_events` 表已创建，并有 `event_id` 唯一索引；
- [ ] 任务更新和 Outbox 插入处于同一个 MySQL 事务；
- [ ] API 不再依赖“更新成功后立刻 Publish”来保证事件存在；
- [ ] Outbox payload 保存完整事件，而不是只保存业务 ID；
- [ ] Relay 能扫描 pending 事件；
- [ ] Relay 能抢占事件，并回收超时的 publishing；
- [ ] Relay 发布使用 Persistent、Confirm 和 mandatory Return；
- [ ] Confirm ACK 且无 Return 后才标记 published；
- [ ] Confirm 超时、NACK、连接失败会保留事件并退避重试；
- [ ] Relay 状态更新带有 worker ownership 条件；
- [ ] `processed_messages` 有稳定的唯一幂等键；
- [ ] 幂等记录和通知业务写入处于同一个事务；
- [ ] 唯一键冲突被识别为 Duplicate，而不是无限重试；
- [ ] 数据库事务成功后才 ACK；
- [ ] WebSocket 失败不会重新创建通知；
- [ ] retry/parking 转移仍沿用 Level 3 的 Confirm 后 ACK 原消息规则；
- [ ] 已覆盖 Relay 崩溃、Consumer ACK 前崩溃和事务回滚测试；
- [ ] 已设置 Outbox 和 processed_messages 的保留与清理策略；
- [ ] 日志能用 `event_id` 串起业务事务、Outbox、RabbitMQ 和消费结果；
- [ ] 能明确说明系统仍然是至少一次投递，而不是跨系统 Exactly Once。

## 27. Level 3 到 Level 4，究竟增加了什么

把两个版本放在一起看，会更容易理解：

| 问题 | Level 3 的处理 | Level 4 的补强 |
| --- | --- | --- |
| Publish 是否到 Broker | Confirm | Relay 从持久化 Outbox 反复发布 |
| routing key 是否可达 | mandatory + Return | Return 失败时不标记 Outbox 成功 |
| Consumer 业务失败 | manual ACK + retry/parking | 事务回滚后继续使用 retry/parking |
| ACK 前进程崩溃 | 消息重新投递 | `event_id` 唯一幂等，重复变空操作 |
| DB 提交后 Publish 前崩溃 | 仍可能丢事件 | 业务表和 Outbox 同事务 |
| retry copy 后 ACK 前崩溃 | 可能重复 | Consumer 幂等兜底 |
| WebSocket 发送失败 | 可能触发重复业务处理 | DB 通知与实时推送分开，推送尽力而为 |

所以 Level 4 不是推翻 Level 3，而是把 Level 3 的可靠 RabbitMQ 传输接到业务数据库的两个关键边界上。

## 28. 最后用一句人话总结

生产端不要把“消息要发出去”只放在代码的下一行。先把这件事和业务变更一起写进数据库，Relay 再慢慢送。

消费端不要相信“这条消息只会来一次”。假设它一定可能再来，用同一个 `event_id` 和数据库唯一约束把第二次副作用挡住。

最终链路是：

```text
业务成功，事件就有记录
事件有记录，Relay 就能重试
消息重复，数据库能识别
业务事务成功，消息才 ACK
```

这四句话，就是 TeamFlow RabbitMQ 企业级实战从“能跑”进入“遇到故障也能解释、能恢复”的关键一步。

