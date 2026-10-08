# Level 2：打通任务分配事件与通知消费者

> 本章目标：把 TeamFlow 的一次“指派任务”从同步 WebSocket 调用，改造成一条可观察的异步事件链路：
>
> `AssignTask → task.assigned.v1 → RabbitMQ → Notification Worker → 通知落库 / WebSocket 推送`
>
> 本章是**功能打通阶段**。它会正确分开业务事件和 RabbitMQ 技术细节，但还不解决 MySQL 与 RabbitMQ 的跨系统原子性；那个问题留到 Level 4 的 Transactional Outbox。

## 1. 完成后你应当能看到什么

一次 `POST /tasks/:id/assign` 成功后：

1. `tasks.assignee_id` 被更新；
2. API 进程发布一条 `task.assigned.v1`；
3. RabbitMQ 将消息从 `teamflow.events.v1` 路由到 `teamflow.notification.websocket.q`；
4. Notification Worker 消费消息，创建一条 `notifications` 记录；
5. 若被指派者在线，再通过 WebSocket 推送一条实时通知。

本章的验收重点不是“RabbitMQ 管理台里出现过一条消息”，而是能证明这五步按顺序发生。

```text
HTTP 请求
  │
  ▼
TaskService.AssignTask
  │  校验权限、项目成员资格并更新 tasks
  ▼
EventPublisher（接口）
  │
  ▼
teamflow.events.v1  -- task.assigned.v1 -->  teamflow.notification.websocket.q
                                                     │
                                                     ▼
                                           Notification Worker
                                             ├─ 创建 notifications
                                             └─ 在线时 WebSocket 推送
```

## 2. 先理解现有代码的问题

当前 `internal/service/task_service.go` 的 `AssignTask` 直接创建 AMQP Channel，并且在更新任务**之前**调用 `channel.Publish`。这会产生三个本章就必须修正的问题。

### 2.1 事件顺序错误

“任务已指派”只能在数据库更新成功后发生。若先发布消息、后更新数据库：

```text
消息发布成功 → 数据库更新失败
```

消费者会得到一个不存在的业务事实。Level 2 至少应做到：数据库更新成功后才尝试发布。

### 2.2 Service 知道太多 RabbitMQ 细节

`TaskService` 不应该知道 `amqp091.Channel`、exchange 名称、routing key 或 `amqp091.Publishing`。这些都是 RabbitMQ Adapter 的实现细节，而不是“分配任务”这个业务动作的接口。

Service 只需要表达：**任务已被指派，请发布这个业务事件。** 因此它依赖一个小的 `event.Publisher` 接口；RabbitMQ 是该接口的一个 Adapter。

### 2.3 同时存在两条通知路径

当前更新后还直接调用 `notifier.OnTaskAssigned`。若又新增 MQ Consumer，通知可能被同步推送一次、异步消费后再推送一次。Level 2 选择一条路径：

```text
TaskService 只发布事件
Notification Worker 统一负责通知落库和 WebSocket
```

## 3. 本章范围与明确不做的事情

本章故意保持小：只接入 `task.assigned.v1`，不要同时接任务状态、评论、邮件、审计或死信队列。

| 本章做 | 留到后续 Level |
| --- | --- |
| 固定 exchange、queue、binding | Confirm、mandatory Return（Level 3） |
| 结构化事件信封 | Retry、DLX、prefetch（Level 3） |
| 一个 Notification Worker | Outbox、投递幂等（Level 4） |
| DB 更新成功后再调用 Publisher | MySQL/MQ 双写原子性（Level 4） |

特别注意：即使把发布放到任务更新之后，仍存在“数据库已提交、进程在发布前崩溃”的窗口。因此本章的语义只是**功能已连通**，不是“事件一定会送达”。不要为了假装可靠而在这里提前写一个复杂的万能 MQ 客户端。

## 4. 第一步：定义不依赖 RabbitMQ 的事件

新增 `internal/event` 包。这个包定义的是业务语言；它不能 import `amqp091-go`，也不应知道 queue 名称。

`internal/event/event.go`：

```go
package event

import (
    "context"
    "time"
)

const TaskAssignedV1 = "task.assigned.v1"

type Envelope struct {
    EventID       string      `json:"event_id"`
    EventType     string      `json:"event_type"`
    SchemaVersion int         `json:"schema_version"`
    OccurredAt    time.Time   `json:"occurred_at"`
    Payload       interface{} `json:"payload"`
}

// Publisher 是 TaskService 需要的全部消息能力。
// 测试可替换为内存 Fake，生产实现由 RabbitMQ Adapter 提供。
type Publisher interface {
    Publish(ctx context.Context, message Envelope) error
}
```

`internal/event/task_event.go`：

```go
package event

import (
    "time"
    "github.com/google/uuid"
)

type TaskAssignedPayload struct {
    TaskID     uint `json:"task_id"`
    ProjectID  uint `json:"project_id"`
    AssigneeID uint `json:"assignee_id"`
    OperatorID uint `json:"operator_id"`
}

func NewTaskAssigned(taskID, projectID, assigneeID, operatorID uint) Envelope {
    return Envelope{
        EventID:       uuid.NewString(),
        EventType:     TaskAssignedV1,
        SchemaVersion: 1,
        OccurredAt:    time.Now().UTC(),
        Payload: TaskAssignedPayload{
            TaskID: taskID, ProjectID: projectID,
            AssigneeID: assigneeID, OperatorID: operatorID,
        },
    }
}
```

如果项目暂未引入 `github.com/google/uuid`，可先选择一个 UUID 库并固定下来；不要用 task ID 代替 `event_id`，因为同一任务可能被重新指派多次。

## 5. 第二步：固定拓扑，而不是在请求里临时拼名称

本章使用一个 topic exchange、一条通知队列和一条 binding：

```text
exchange:    teamflow.events.v1              type=topic, durable=true
routing key: task.assigned.v1
queue:       teamflow.notification.websocket.q durable=true
binding:     task.assigned.v1
```

Topic exchange 的作用是让之后的消费者按业务类别订阅。例如未来审计 Worker 可以绑定同一个 `task.assigned.v1`，邮件 Worker 可以绑定 `task.#`；发布任务的一方不需要改代码。

在 `internal/messaging/rabbitmq/topology.go` 集中声明这些名字和参数：

```go
const (
    EventsExchange      = "teamflow.events.v1"
    NotificationQueue   = "teamflow.notification.websocket.q"
    TaskAssignedRouting = "task.assigned.v1"
)

func DeclareTopology(ch *amqp091.Channel) error {
    if err := ch.ExchangeDeclare(EventsExchange, "topic", true, false, false, false, nil); err != nil {
        return err
    }
    if _, err := ch.QueueDeclare(NotificationQueue, true, false, false, false, nil); err != nil {
        return err
    }
    return ch.QueueBind(NotificationQueue, TaskAssignedRouting, EventsExchange, false, nil)
}
```

把拓扑声明放在应用启动或 Worker 启动处，而不是 `AssignTask` 每次请求时声明。声明失败应让进程明确失败或变成不就绪，而不是在业务请求中静默吞掉。

## 6. 第三步：让 RabbitMQ Adapter 实现 Publisher

将当前的 `internal/mq/rabbitmq.go` 逐步迁到 `internal/messaging/rabbitmq`。不要继续提供按字符串名字保存 Channel 的 `map[string]*Channel`：当前 HTTP 并发场景下这会有竞态，而且关闭后的 Channel 仍可能留在 map 中。

Level 2 的简单做法是：应用启动时建立一个长期 Connection；每个 Publisher/Consumer owner 拥有自己的 Channel；该 Channel 不由多个 goroutine 同时使用。

Publisher 只负责把事件 JSON 化并投递到固定拓扑：

```go
type Publisher struct {
    ch *amqp091.Channel // 此 Publisher 是这个 Channel 的唯一 owner
}

func (p *Publisher) Publish(ctx context.Context, e event.Envelope) error {
    body, err := json.Marshal(e)
    if err != nil {
        return fmt.Errorf("marshal event: %w", err)
    }
    return p.ch.PublishWithContext(ctx,
        EventsExchange,
        e.EventType,
        false, false,
        amqp091.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp091.Persistent,
            MessageId:    e.EventID,
            Type:         e.EventType,
            Timestamp:    e.OccurredAt,
            Body:         body,
        },
    )
}
```

这里的 persistent 只是为后续可靠性做正确准备；本章没有 publisher confirm，因此仍不能把 `PublishWithContext` 返回 `nil` 解释为“Broker 已可靠保存”。

## 7. 第四步：收紧 TaskService 的接口

构造函数从“注入具体 RabbitMQ”变成“注入事件 Publisher”。推荐同时给 `AssignTask` 增加 `context.Context`，让 HTTP 请求的超时和取消可以传到发布动作。

```go
type taskService struct {
    repo      repository.TaskRepository
    publisher event.Publisher
    mu        sync.Mutex
}

func NewTaskService(repo repository.TaskRepository, publisher event.Publisher) TaskService {
    return &taskService{repo: repo, publisher: publisher}
}
```

`AssignTask` 的关键顺序如下。保留你现有的参数校验、项目成员校验、管理员权限校验和乐观锁更新；只替换“直接操作 Channel”的部分。

```go
// 1. 完成现有业务校验
// 2. updateFields(task, {"assignee_id": assigneeID})
if err := s.updateFields(task, map[string]interface{}{"assignee_id": assigneeID}); err != nil {
    return nil, err
}

// 3. 重新读取，避免把旧 task（AssigneeID / Version 未刷新）返回或写进事件。
updated, err := s.getTask(id)
if err != nil {
    return nil, err
}

// 4. 数据库成功后才发布事件。
message := event.NewTaskAssigned(updated.ID, updated.ProjectID, assigneeID, operatorID)
if err := s.publisher.Publish(ctx, message); err != nil {
    // Level 2 的策略：记录错误；不能回滚已经完成的任务指派。
    // Level 4 会改成同事务写 Outbox，从根本上消除该丢失窗口。
    log.Printf("publish task assigned event failed: %v", err)
}
return updated, nil
```

这一步还应删除 `TaskService` 对 `NotificationService` 的直接调用。否则同一个任务分配会绕过消费者先推一次，Worker 消费后又推一次。

对于“发布失败要不要让 HTTP 返回 500”，本章建议**不返回 500，也不回滚任务**：任务的写入已经成功，返回失败会诱导客户端重试，从而重复执行赋值。日志、指标和 RabbitMQ 管理台用于发现问题；Level 4 再用 Outbox 确保恢复。

## 8. 第五步：实现 Notification Worker

Worker 不是 Controller，也不处理 HTTP 参数。它只消费一条已验证的事件，将其转换为通知业务动作。

```go
func (w *NotificationWorker) Handle(delivery amqp091.Delivery) error {
    var envelope event.Envelope
    if err := json.Unmarshal(delivery.Body, &envelope); err != nil {
        return fmt.Errorf("decode envelope: %w", err)
    }
    if envelope.EventType != event.TaskAssignedV1 {
        return fmt.Errorf("unexpected event type: %s", envelope.EventType)
    }

    payloadBytes, _ := json.Marshal(envelope.Payload)
    var payload event.TaskAssignedPayload
    if err := json.Unmarshal(payloadBytes, &payload); err != nil {
        return fmt.Errorf("decode task assignment payload: %w", err)
    }

    notification, err := w.notifications.CreateNotification(
        payload.AssigneeID,
        payload.OperatorID,
        model.NotifyTaskAssigned,
        "任务分配通知",
        fmt.Sprintf("您已被分配任务 %d", payload.TaskID),
        payload.TaskID,
    )
    if err != nil {
        return err
    }
    return w.pushIfOnline(notification)
}
```

Level 2 可以使用 auto-ack 让第一条链路尽快跑通，但应把它视为教学临时状态。更推荐从第一天就使用 `autoAck=false`：仅在 `Handle` 成功后 `delivery.Ack(false)`；失败时记录错误并由 Level 3 引入有限重试和死信策略。

Worker 在本章可以先作为 API 进程启动的 goroutine，便于调试；生产形态应拆为 `cmd/worker/main.go`，并使用独立的 Consumer Connection。无论在哪运行，都不要在 Controller 内启动 Consumer。

## 9. 第六步：在 Composition Root 组装依赖

`cmd/main.go` 是唯一应知道具体 RabbitMQ Adapter 的地方。它负责：读取配置、创建 Connection、声明拓扑、创建 Publisher、启动 Consumer，并把接口传给 TaskService。

```go
conn, err := rabbitmq.Dial(cfg.RabbitMQ)
if err != nil { return err }
defer conn.Close()

publisher, err := rabbitmq.NewPublisher(conn)
if err != nil { return err }
if err := rabbitmq.DeclareTopology(publisher.Channel()); err != nil { return err }

taskService := service.NewTaskService(
    repository.NewTaskRepository(storage.DB),
    publisher, // 类型是 event.Publisher，不是 *amqp091.Channel
)

worker := worker.NewNotificationWorker(notificationService, hub)
go worker.Run(ctx, conn)
```

当前 `cmd/main.go` 已经创建了 `rmq`，但没有传入 `NewTaskService`；而当前构造函数又要求该参数，导致项目无法构建。按照本章重构后，这个问题自然消失：`main` 创建 `Publisher`，再把它作为接口依赖传入。

## 10. 手工验收脚本

1. 启动 RabbitMQ、MySQL、Redis 和 TeamFlow。
2. 在管理台确认 `teamflow.events.v1`、`teamflow.notification.websocket.q` 已存在，且 queue 有一条 `task.assigned.v1` binding。
3. 用管理员身份指派一个项目成员。
4. 检查 HTTP 返回的 `assignee_id` 是否正确。
5. 检查 RabbitMQ：若 Worker 在线，Ready 消息会迅速归零；若停掉 Worker，Ready 应增长。
6. 启动 Worker，检查 `notifications` 表产生一条 `task_assigned` 记录。
7. 让被指派者保持 WebSocket 在线，确认收到实时通知；让其离线重试，确认虽没有实时推送但数据库通知仍存在。

建议至少写两个测试：

```text
TaskService 单元测试：任务更新成功时，FakePublisher 收到 task.assigned.v1。
NotificationWorker 单元测试：给定合法事件，创建对应通知并调用在线推送。
```

不要在 TaskService 单元测试里连接 RabbitMQ；那是在测试 Adapter，不是在测试任务业务规则。

## 11. 常见错误清单

| 错误 | 为什么错 | 本章的修正 |
| --- | --- | --- |
| 在 `AssignTask` 内 `Dial` RabbitMQ | 每个请求建立 TCP Connection，慢且难恢复 | 启动时建立长期 Connection |
| Service 直接使用 `amqp091.Channel` | 业务模块绑定基础设施，测试困难 | 依赖 `event.Publisher` 接口 |
| publish 后再更新 DB | 可能发送不存在的业务事实 | DB 成功后再发布 |
| 仍直接调用 `OnTaskAssigned` | 同一事件可能通知两次 | 通知统一由 Worker 处理 |
| queue 名称就是 exchange 名称 | exchange 负责路由，queue 负责保存 | 固定 exchange、queue、binding |
| 认为 persistent 等于不丢消息 | 未确认、网络断开等仍存在不确定性 | Level 3 加 Confirm，Level 4 加 Outbox |
| 消费后立即 ACK | 后续落库失败时消息已丢 | 使用 manual ACK，成功后 ACK |

## 12. 本章完成定义与下一章入口

完成本章前，请能清楚回答：

- 为什么 `task.assigned.v1` 是事件，而不是“调用 WebSocket”的命令？
- 为什么 TaskService 只依赖 `event.Publisher`，而不 import RabbitMQ？
- exchange、routing key、queue、binding 各自负责什么？
- 为什么本章即便“数据库更新后发布”，仍不具备可靠消息保证？
- Worker 停止时，为什么 queue 的 Ready 数会增长？

完成后推荐提交：

```text
feat: consume task assignment events
```

提交说明要明确：本提交新增了任务分配事件、通知消费者和实时推送链路；**仍未保证 MySQL 与 RabbitMQ 的原子一致性，也未实现重复消息幂等、确认重试与死信**。

下一章进入 Level 3：为 Publisher 加 Confirm/Return，为 Consumer 加 QoS、manual ACK、有限重试和死信；Level 4 再用 Outbox 解决本章有意保留的双写窗口。
