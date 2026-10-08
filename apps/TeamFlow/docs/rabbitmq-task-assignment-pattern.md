# RabbitMQ 任务指派设计模式笔记

这份笔记只记录最终设计模式，不代表所有步骤已经在当前代码中完成。

## 一句话

业务数据和待发送事件先在同一个 MySQL 事务中提交；Relay 再把事件可靠发布到 RabbitMQ；Consumer 在同一个事务中完成幂等登记和业务处理，成功后才 ACK。

```text
HTTP 请求
  -> TaskService
  -> MySQL 事务：更新 tasks + 写 outbox_events
  -> Outbox Relay
  -> RabbitMQ（Confirm + mandatory + Return）
  -> Consumer
  -> MySQL 事务：写 processed_messages + 写 notifications
  -> ACK
```

## 一、生产端：AssignTask

```text
1. Controller 接收 task_id、assignee_id、operator_id
2. Service 查询任务并校验权限
3. 开启 MySQL 事务
4. 更新 tasks.assignee_id，并递增 version
5. 创建完整事件 Envelope
   - event_id
   - event_type = task.assigned.v1
   - schema_version
   - occurred_at / created_at
   - payload（task_id、project_id、assignee_id、operator_id）
6. 将完整事件 JSON 写入 outbox_events
   - status = pending
   - routing_key
   - attempts = 0
   - next_attempt_at
7. 提交事务
8. 事务成功后 API 返回成功
```

关键点：第 4 步和第 6 步必须属于同一个事务。不能先提交任务，再单独 Publish 或单独写 Outbox。

## 二、Outbox Relay

```text
1. 定时扫描 pending 且 next_attempt_at <= now 的事件
2. 回收超时的 publishing 事件
3. 通过短事务抢占事件，写入 locked_by / locked_at
4. 使用已有 Publisher 发布：
   - persistent message
   - mandatory = true
   - 等待 Publisher Confirm
   - 监听 Return
5. Confirm 成功且没有 Return：标记 published
6. Confirm 超时、NACK、连接失败：增加 attempts，安排下次重试
7. 超过最大次数或明确永久失败：标记 failed，进入人工处理
```

Relay 在“发布成功、标记 published 前”崩溃时，恢复后可能再次发布同一个事件。这是允许的，重复由 Consumer 幂等处理。

## 三、RabbitMQ 路由

```text
events exchange
  -> task.assigned.v1
  -> notification queue

retry exchange
  -> 5s / 30s / 5m 延迟队列
  -> 原业务队列

parking exchange
  -> parking queue（永久失败或超过重试次数）
```

`mandatory + Return` 只负责发现消息没有被任何 Queue 路由，不能替代 Outbox，也不能保证业务事务和 RabbitMQ 原子提交。

## 四、消费端：Notification Consumer

```text
1. 收到 Delivery（autoAck = false）
2. 解码并校验 Envelope、event_id、event_type
3. 开启 MySQL 事务
4. 插入 processed_messages(event_id)
5. 如果 event_id 已存在：判定为 duplicate，提交/结束事务
6. 如果是新事件：创建 notifications 记录
7. 提交事务
8. 事务成功后 ACK 原消息
9. WebSocket 推送放在落库之后，失败不影响 ACK
```

`processed_messages.event_id` 必须有唯一索引。幂等记录和通知记录必须在同一个事务中，不能先写幂等表再在事务外写通知。

## 五、失败处理顺序

```text
处理成功
  -> ACK

临时错误
  -> 发布到 retry exchange
  -> 重试副本发布成功后 ACK 原消息
  -> 重试副本发布失败则 NACK/requeue 原消息

永久错误 / 重试次数耗尽
  -> 发布到 parking queue
  -> 停车副本发布成功后 ACK 原消息
  -> 停车副本发布失败则 NACK/requeue 原消息
```

不要无限 `NACK(requeue=true)`，否则会形成高速重投循环。

## 六、必须记住的三个故障窗口

```text
任务更新成功、直接 Publish 前崩溃
  -> 只有 Outbox 能补上这个丢失窗口

RabbitMQ 已 Confirm、Relay 标记 published 前崩溃
  -> 可能重复发布，Consumer 幂等兜底

通知事务提交成功、ACK 前 Consumer 崩溃
  -> RabbitMQ 会重投，processed_messages 识别 duplicate
```

## 七、最终保证

- 业务事务提交成功后，事件最终可被 Relay 反复尝试发布。
- RabbitMQ 消息至少一次到达 Consumer。
- 同一个 `event_id` 不会重复产生通知副作用。
- 系统保证的是“至少一次投递 + 业务幂等”，不是跨 MySQL 和 RabbitMQ 的 Exactly Once。

## 八、最小落地顺序

```text
1. Publisher：persistent + Confirm + mandatory + Return
2. Consumer：manual ACK + 有限重试 + parking
3. Consumer 幂等：processed_messages + event_id 唯一索引
4. Outbox：任务事务写 tasks 和 outbox_events
5. Relay：抢占、发布、确认、重试、恢复
```

每一步都保持现有事件 Envelope 和 Consumer ACK/Retry/Parking 逻辑，不额外引入分布式事务或复杂消息平台。
