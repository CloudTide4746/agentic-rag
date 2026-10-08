# TeamFlow RabbitMQ 企业级实战教程设计

## 1. 背景

TeamFlow 是一个使用 Go、Gin、GORM、MySQL、Redis、JWT 和 WebSocket 构建的协作平台。仓库现有 `第14章-消息队列` 使用 Redis Stream，适合讲解轻量级异步处理，但不足以系统覆盖 RabbitMQ 的路由、发布确认、死信、Quorum Queue、权限隔离和生产运维。

本设计新增一套独立的 RabbitMQ 专题教程。原 Redis Stream 章节保持不变，学习者可以据此比较两种方案，也可以在完成专题后自行迁移现有通知链路。

## 2. 目标读者与成功标准

### 2.1 目标读者

- 能阅读基础 Go 代码，了解 `context.Context`、接口和 goroutine。
- 能启动当前 TeamFlow 项目，但不要求有 RabbitMQ 使用经验。
- 希望一边学习、一边在当前仓库完成真实业务代码。
- 开发环境以 Windows 和 Docker Desktop 为主，命令同时保持跨平台可理解性。

### 2.2 完成教程后的能力

学习者能够：

1. 使用 Docker Compose 启动、检查和持久化本地 RabbitMQ。
2. 解释 AMQP 0-9-1 中 Exchange、Queue、Binding、Routing Key、Connection 和 Channel 的职责。
3. 使用官方 Go 客户端实现可测试的 Publisher 和 Consumer。
4. 正确使用持久化、Publisher Confirm、mandatory Return、手动 ACK 和 QoS。
5. 设计有限重试、死信停车场、消费幂等和版本化消息契约。
6. 使用 Transactional Outbox 消除 MySQL 业务事务与消息发布之间的双写丢失窗口。
7. 将 TeamFlow API 与通知 Worker 分离部署，并完成优雅启停和断线恢复。
8. 建立日志、指标、告警、压测、容量评估和故障演练闭环。
9. 说明 Quorum Queue、RabbitMQ 集群和 Kubernetes Operator 的适用边界。
10. 使用生产检查表判断系统是否达到上线条件。

## 3. 范围

### 3.1 包含

- 新建 `TeamFlow企业级开发教程/RabbitMQ企业级实战/`。
- 在 `TeamFlow企业级开发教程/00-总目录.md` 增加专题入口。
- 教程代码直接适配当前 TeamFlow 的配置、日志、数据库、Service 和 WebSocket 结构。
- 本地环境使用 Docker Compose 完整实操。
- 生产级应用可靠性机制完整实操。
- 集群和 Kubernetes 提供架构、配置、操作步骤、上线要求和故障场景说明。
- 提供验证命令、自动化测试建议、故障实验、排障表和阶段验收清单。

### 3.2 不包含

- 不删除或重写原 Redis Stream 教程。
- 不在教程编写阶段直接替学习者实现全部 RabbitMQ 业务代码。
- 不实际搭建长期运行的三节点生产集群。
- 不承诺端到端 Exactly Once；教程采用至少一次投递和应用幂等。
- 不将 RabbitMQ 用作业务数据库、长期事件存档或大数据日志平台。
- 不展开与 TeamFlow 无关的 RabbitMQ 插件生态。

## 4. 教学方法

教程采用“能力阶梯式”路线，每一阶段都产生可运行、可观察、可验收的结果，而不是先给出一个难以理解的万能封装。

每篇文档固定包含以下栏目：

1. 本节目标与前置条件。
2. 原理和关键术语。
3. 当前项目需要新增或修改的文件。
4. 分步骤代码与设计解释。
5. 启动、验证和预期输出。
6. 刻意制造的失败场景。
7. 常见错误和排查方式。
8. 生产环境注意事项。
9. 本节检查点和建议 Git 提交信息。

概念第一次出现时先用最小示例解释，随后立即映射到 TeamFlow 的任务通知场景。代码在后续章节演进时，要明确指出被替换的旧实现，避免学习者同时保留两套冲突代码。

## 5. 教程阶段

### 5.1 阶段一：从零入门

使用 Docker Compose 启动 RabbitMQ，学习管理台和命令行，完成 Exchange、Queue、Binding 与四种 Exchange 的可视化实验。

阶段验收：学习者可以预测一条消息会进入哪些队列，并能从管理台定位不可路由和堆积问题。

### 5.2 阶段二：接入 TeamFlow

加入 RabbitMQ 配置，使用 `github.com/rabbitmq/amqp091-go` 封装连接、拓扑、Publisher 和 Consumer，完成任务通知的最小消息闭环。

阶段验收：TeamFlow 可以发布通知事件，独立 Worker 可以消费、写入通知记录并触发 WebSocket 推送。

### 5.3 阶段三：可靠消息

逐步加入 Publisher Confirm、mandatory Return、手动 ACK、QoS、有限重试、死信停车场、消费幂等、版本化契约和 Transactional Outbox。

阶段验收：在进程崩溃、Broker 重启、重复投递和临时数据库错误下，业务消息不静默丢失，最终结果不重复。

### 5.4 阶段四：生产工程

完成连接恢复、优雅启停、健康检查、结构化日志、Prometheus 指标、告警、安全配置、压测、容量规划和故障演练。

阶段验收：学习者能够用指标解释系统状态，给出容量依据，并按演练手册恢复常见故障。

### 5.5 阶段五：大规模架构

讲解 Quorum Queue、RabbitMQ 集群、故障域、滚动升级、Kubernetes Cluster Operator 和生产上线评审。

阶段验收：学习者能够设计三节点生产拓扑，说明节点、磁盘和网络故障时的行为，但本教程不要求实际维护一套长期运行的生产集群。

## 6. 章节目录

| 编号 | 文档 | 交付结果 |
|---|---|---|
| 00 | 学习路线与最终架构 | 环境、路线、完成标准和架构全景 |
| 01 | 为什么选择 RabbitMQ | 与 Redis Stream、Kafka 的选型边界 |
| 02 | 消息系统核心概念 | 投递语义、顺序、积压、重复和丢失模型 |
| 03 | Docker Compose 从零部署 | RabbitMQ、管理台、数据卷和健康检查 |
| 04 | 管理台与 rabbitmqctl | 连接、Channel、拓扑和消息状态诊断 |
| 05 | AMQP 0-9-1 工作模型 | Producer、Exchange、Binding、Queue、Consumer |
| 06 | 四类 Exchange 实验 | direct、topic、fanout、headers 路由实验 |
| 07 | TeamFlow 消息拓扑设计 | vhost、命名、Exchange、Queue、Routing Key |
| 08 | 引入官方 Go 客户端 | 依赖、配置模型和最小连接验证 |
| 09 | 连接与 Channel 生命周期 | 资源边界、超时、关闭和并发约束 |
| 10 | 声明生产级拓扑 | 幂等声明 durable 拓扑和参数 |
| 11 | 实现基础 Publisher | 事件信封、序列化、消息属性和接口 |
| 12 | 实现基础 Consumer | 消费循环、Handler、ACK 和错误返回 |
| 13 | Publisher Confirm | Broker 接管确认、超时与失败处理 |
| 14 | Mandatory 与 Return | 检测不可路由消息 |
| 15 | Consumer ACK 与 QoS | 手动确认、NACK、prefetch 和过载保护 |
| 16 | 重试与死信体系 | 退避、次数限制、DLX、停车场和重放 |
| 17 | 消费幂等 | `event_id`、唯一约束和事务性去重 |
| 18 | 消息契约与演进 | 类型、版本、时间、Trace ID 和兼容策略 |
| 19 | Transactional Outbox | 同一事务写业务数据与事件 |
| 20 | Outbox Relay | 租约领取、Confirm、状态更新和崩溃恢复 |
| 21 | TeamFlow 通知 Worker | 通知落库、WebSocket 推送和独立运行入口 |
| 22 | 断线恢复与优雅停机 | 重连、重新声明、停止接单和等待在途消息 |
| 23 | 自动化测试 | 单元、集成、端到端和故障测试 |
| 24 | 可观测性 | 日志、Tracing、Prometheus 和 Grafana |
| 25 | 告警与积压治理 | Ready、Unacked、速率和资源水位 |
| 26 | 安全加固 | vhost、最小权限、TLS、Secret 和网络隔离 |
| 27 | 性能压测与容量规划 | 吞吐、P95/P99、消息大小和恢复时间 |
| 28 | Quorum Queue | 多数派、leader、delivery limit 和边界 |
| 29 | 集群与 Kubernetes 指南 | 节点发现、故障域、升级和 Operator |
| 30 | 故障演练 | 重启、断网、宕机、数据库失败和积压恢复 |
| 31 | 生产上线检查表 | 可靠性、安全、监控、备份和回滚评审 |
| 32 | 排障手册 | 连接、路由、积压、资源和性能故障索引 |
| 33 | 面试与架构复盘 | 用 TeamFlow 解释架构取舍 |

附录提供配置字典、命名表、API 速查、管理命令、错误索引、Redis Stream 迁移对照、阶段提交建议和全课程验收清单。

## 7. 目标代码结构

教程最终引导学习者形成以下结构：

```text
config/
  config.go
  config.yaml

internal/mq/rabbitmq/
  config.go
  connection.go
  topology.go
  publisher.go
  consumer.go
  message.go
  retry.go
  health.go

internal/event/
  task_event.go

internal/worker/
  notification_worker.go

internal/repository/
  outbox_repository.go
  processed_message_repository.go

internal/model/
  outbox_event.go
  processed_message.go

cmd/
  main.go
  worker/main.go

deploy/rabbitmq/
  definitions.json
  rabbitmq.conf
  enabled_plugins
  prometheus.yml

docker-compose.rabbitmq.yml
```

文件可以在教学过程中逐步出现。教程不得要求学习者在第一阶段一次性创建全部结构。

## 8. 模块边界

### 8.1 业务事件层

`internal/event` 定义与 AMQP 无关的事件信封和业务载荷。事件信封至少包含事件 ID、事件类型、契约版本、发生时间、关联 ID、因果 ID、Trace ID 和载荷。

### 8.2 消息基础设施层

`internal/mq/rabbitmq` 负责连接、Channel、拓扑、发布确认、消费投递和恢复。它不知道 Task、Notification 等具体业务模型。

### 8.3 业务层

Service 只依赖项目接口，不直接接触 `amqp091-go` 类型：

```go
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}

type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

type RetryClassifier interface {
	Classify(err error) FailureKind
}
```

### 8.4 Worker 层

Worker 负责将投递转换为业务事件，调用 Handler，并依据处理结果执行 ACK、重试或进入死信停车场。通知 Worker 将通知先持久化，再进行 WebSocket 推送。

### 8.5 Outbox 层

Outbox Repository 与业务 Repository 共享数据库事务。Relay 使用短事务和租约领取记录，避免在等待 RabbitMQ 网络确认期间长期持有数据库行锁。

## 9. 数据流与可靠性语义

### 9.1 正常链路

```text
HTTP 请求
  -> TeamFlow Service
  -> 同一 MySQL 事务写业务表和 Outbox 表
  -> Outbox Relay 领取事件
  -> RabbitMQ Publisher Confirm
  -> Quorum Queue
  -> Notification Worker
  -> 同一 MySQL 事务写通知和消费幂等记录
  -> 手动 ACK
  -> WebSocket 尽力实时推送
```

### 9.2 投递承诺

系统采用至少一次投递：

- Relay 在 RabbitMQ 已确认但 Outbox 尚未标记完成时崩溃，会再次发布。
- Consumer 在业务事务已提交但 ACK 尚未到达 Broker 时崩溃，会再次收到消息。
- 消费者使用 `event_id` 唯一约束，在同一数据库事务中完成业务写入和幂等登记。
- WebSocket 失败不回滚已经持久化的通知；客户端可重新查询通知列表。

### 9.3 发布可靠性

Publisher 同时使用：

- durable Exchange 和 durable Queue。
- 持久化消息属性。
- Publisher Confirm 判断 Broker 是否接管消息。
- mandatory publish 与 Return 处理检测不可路由消息。
- 明确的确认超时和可观测错误。

这些机制不替代 Outbox。没有 Outbox 时，数据库提交后、消息发布前仍存在进程崩溃窗口。

### 9.4 消费可靠性

- Consumer 只在业务事务成功后 ACK。
- 临时错误进入有限重试，永久错误直接进入停车场。
- 不使用无限 `NACK requeue`，避免快速重复投递拖垮 Broker 和下游。
- 重试延迟、消息 TTL 和死信策略必须说明队头阻塞、策略变更和插件依赖等边界。
- 人工重放生成新的操作审计信息，同时保留原事件 ID 和失败原因。

## 10. 生命周期与错误处理

### 10.1 连接恢复

应用负责监控连接和 Channel 关闭通知。恢复流程必须重新建立连接、Channel、Confirm 模式、Return 监听、拓扑声明和 Consumer 注册。恢复期间 Publisher 返回可分类错误，Outbox 保留事件等待下一轮发布。

### 10.2 优雅启动

启动顺序为配置和日志、数据库、RabbitMQ 连接、拓扑声明、Publisher/Consumer、健康状态、HTTP 服务。依赖未就绪时 Readiness 失败，但 Liveness 不因短暂 RabbitMQ 故障立即杀死进程。

### 10.3 优雅停机

Worker 先停止接收新投递，再等待有界时间完成在途 Handler 和 ACK，最后关闭 Channel 与 Connection。超时后未确认的消息由 RabbitMQ 重新投递。

### 10.4 错误分类

教程定义三类错误：

- 临时错误：数据库短暂不可用、下游限流、网络超时，可有限重试。
- 永久错误：消息格式错误、契约版本不支持、业务目标不存在，进入停车场。
- 未知错误：按保守的有限重试策略处理，超过上限后进入停车场并告警。

## 11. 测试与验证

### 11.1 单元测试

- 事件信封校验和版本分派。
- Service 通过假的 `EventPublisher` 验证事件生成。
- Handler 的成功、重复、临时失败和永久失败。
- 重试分类、次数计算和 Outbox 状态机。

### 11.2 集成测试

使用真实 RabbitMQ 和 MySQL 验证：

- 拓扑声明幂等性。
- Confirm 成功、超时和连接关闭。
- mandatory Return。
- 手动 ACK、NACK、prefetch、重投递标记。
- 重试、死信和停车场。
- 幂等唯一约束和并发消费。
- Outbox 并发 Relay 和租约恢复。

### 11.3 端到端测试

验证 HTTP 创建或变更任务后，事件依次经过 MySQL Outbox、RabbitMQ 和 Worker，最终生成一条通知并可通过 WebSocket 或查询接口获取。

### 11.4 故障测试

- Broker 重启和短时网络断开。
- Worker 在数据库提交前后分别崩溃。
- Relay 在 Confirm 前后分别崩溃。
- 数据库临时不可用。
- 不可路由消息和未知契约版本。
- 大量积压后的恢复速度。

### 11.5 性能验收

压测报告至少包含发布吞吐、消费吞吐、端到端 P50/P95/P99、Ready、Unacked、CPU、内存、磁盘 I/O、网络和积压恢复时间。教程不预设脱离硬件和消息大小的固定“高 QPS”结论。

## 12. 可观测性与安全

### 12.1 可观测性

- 日志字段：event ID、message ID、event type、routing key、queue、consumer、attempt、trace ID、耗时和结果。
- 应用指标：发布成功/失败/Return、Confirm 延迟、消费成功/失败/重试/死信、Handler 延迟和重连次数。
- Broker 指标：Ready、Unacked、发布/投递/确认速率、连接、Channel、消费者、内存、磁盘和文件描述符。
- 告警必须包含触发条件、持续窗口、严重级别和处置入口，避免只列指标名称。

### 12.2 安全

- 为 TeamFlow 使用独立 vhost。
- API Publisher、Worker Consumer 和运维账户按职责最小授权。
- 生产环境禁用默认 guest 远程访问，不在仓库保存真实密码。
- 生产连接使用 TLS，并说明证书校验和轮换。
- 管理端口与业务 AMQP 端口分离控制网络访问。

## 13. 集群与 Kubernetes 边界

- 生产持久关键队列以 Quorum Queue 为主要讲解对象。
- 集群章节说明多数派可用性、leader 分布、节点故障、网络分区和滚动升级。
- 不将 Kubernetes Service 当作 AMQP 连接自动恢复的替代品。
- Kubernetes 章节以官方 RabbitMQ Cluster Operator 为主，说明持久卷、反亲和、PodDisruptionBudget、资源请求和监控接入。
- 本教程提供可执行配置和验证方法，但不要求学习者实际维护长期三节点环境。

## 14. 资料与版本策略

- RabbitMQ 行为、配置和运维结论优先引用 RabbitMQ 官方文档。
- Go API 结论优先引用 `rabbitmq/amqp091-go` 官方仓库和 API 文档。
- Kubernetes 内容优先引用 RabbitMQ 官方 Cluster Operator 文档和 Kubernetes 官方文档。
- 版本相关命令必须标注教程验证版本和查阅日期。
- 对可能随版本变化的默认值，教程要求读者通过管理命令或官方文档确认，不把默认值伪装成永久规范。
- 每篇文档在关键结论附近给出来源链接，文末不堆砌未使用的参考资料。

## 15. 文档质量要求

- 所有代码片段使用当前项目真实包路径 `teamflow/...`。
- 文件路径、类型名和函数名跨章节保持一致。
- 每次演进都给出完整修改点，避免省略关键字段或错误处理。
- 命令必须区分 PowerShell、容器内 Shell 和通用命令。
- 预期输出只展示稳定的关键字段，不伪造随机 ID、时间和性能数字。
- 对危险或破坏性故障实验说明影响、停止条件和恢复步骤。
- 不使用“绝不丢失”“绝对可靠”“Exactly Once”等无条件表述。
- 不把管理台手工建拓扑作为生产方案；生产拓扑由声明代码或受版本控制的定义管理。

## 16. 交付顺序

1. 完成官方资料研究笔记。
2. 创建专题入口、学习路线和基础概念章节。
3. 完成本地部署与 AMQP 实验章节。
4. 完成 Go 接入和 TeamFlow 最小业务闭环章节。
5. 完成可靠消息、Outbox 和 Worker 章节。
6. 完成测试、可观测性、安全和压测章节。
7. 完成 Quorum Queue、集群、Kubernetes 和上线章节。
8. 全局检查链接、术语、代码连续性和验收步骤。

每个阶段写完后进行一次文档自检和代码片段一致性检查，防止错误在后续章节扩散。
