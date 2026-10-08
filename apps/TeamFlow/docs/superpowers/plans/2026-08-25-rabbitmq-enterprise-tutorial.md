# TeamFlow RabbitMQ 企业级实战教程 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在当前 TeamFlow 仓库中交付一套可让学习者从 RabbitMQ 零基础逐章实作到生产级可靠消息、可观测性和高可用架构的中文教程。

**Architecture:** 教程采用能力阶梯式路线，以任务通知事件为贯穿案例；先建立单节点 AMQP 心智模型，再逐步引入 Publisher Confirm、mandatory Return、手动 ACK、有限重试、幂等、Transactional Outbox、独立 Worker 与生产运维。所有代码都以教程片段呈现并适配当前项目，原 Redis Stream 教程和学习者现有 `internal/mq/` 文件保持不变。

**Tech Stack:** Go 1.26.4、Gin、GORM、MySQL、RabbitMQ 4.3.5、Docker Compose、`github.com/rabbitmq/amqp091-go` v1.14.0、Prometheus、Grafana、RabbitMQ Cluster Operator。

**Spec:** `docs/superpowers/specs/2026-08-25-rabbitmq-enterprise-tutorial-design.md`

## Global Constraints

- 教程验证基线固定为 RabbitMQ `4.3.5`、Docker 镜像 `rabbitmq:4.3.5-management` 和 Go 客户端 `github.com/rabbitmq/amqp091-go` `v1.14.0`，资料查阅日期为 `2026-08-25`。
- 新建 `TeamFlow企业级开发教程/RabbitMQ企业级实战/`，并从 `TeamFlow企业级开发教程/00-总目录.md` 增加入口；不删除、不重写 `第14章-消息队列`。
- 不修改学习者已有的 `internal/mq/stream.go`，不替学习者直接实现教程中的业务源码；所有源码以可复制、可逐章演进的 Markdown 代码块交付。
- 所有 Go import 使用当前模块路径 `teamflow/...`，跨章节的文件名、类型名、方法签名、交换机、队列和 routing key 必须一致。
- 可靠性承诺统一为 at-least-once + application idempotency；不得宣称 RabbitMQ、Confirm 或 Outbox 提供端到端 Exactly Once。
- 生产关键队列以 durable quorum queue 为主；本地实操使用 Docker Compose 单节点，三节点集群与 Kubernetes 只提供生产级设计、配置、验证与演练指南。
- 技术结论就近引用 RabbitMQ、`rabbitmq/amqp091-go`、Docker 或 Kubernetes 官方一手资料；不得以二手博客作为事实依据。
- 每篇正文都包含目标与前置条件、原理、文件变化、步骤与代码、验证和预期输出、失败实验、排障、生产说明、检查点和建议提交信息。
- PowerShell、容器内 Shell 和跨平台命令必须明确标注；真实密码、私钥、令牌和环境专用地址不得写入仓库。
- 故障实验必须写明影响范围、停止条件、恢复步骤；性能章节不得给出脱离硬件、消息大小和可靠性配置的固定 QPS 承诺。

---

## File Structure and Responsibilities

```text
docs/research/
  rabbitmq-official-notes.md             # 官方事实底稿与版本陷阱
docs/superpowers/plans/
  2026-08-25-rabbitmq-enterprise-tutorial.md
TeamFlow企业级开发教程/
  00-总目录.md                            # 保留原课程并新增专题入口
  RabbitMQ企业级实战/
    00-学习路线与最终架构.md              # 专题索引、路线、验收、最终架构
    01-为什么选择RabbitMQ.md              # 选型边界
    02-消息系统核心概念.md                # 可靠性与失败模型
    03-Docker-Compose从零部署.md          # 本地 Broker
    04-管理台与rabbitmqctl.md             # 观测与诊断工具
    05-AMQP-0-9-1工作模型.md              # 协议实体和消息路径
    06-四类Exchange实验.md                # 路由实验
    07-TeamFlow消息拓扑设计.md            # 命名、vhost、队列和绑定
    08-引入官方Go客户端.md                # 依赖、配置、连通性
    09-连接与Channel生命周期.md           # 所有权、并发、关闭
    10-声明生产级拓扑.md                  # 幂等声明与策略
    11-实现基础Publisher.md               # Event 与发布接口
    12-实现基础Consumer.md                # Handler 与消费循环
    13-Publisher-Confirm.md               # Broker 接管确认
    14-Mandatory与Return.md               # 不可路由检测
    15-Consumer-ACK与QoS.md               # ACK、NACK、prefetch
    16-重试与死信体系.md                  # 有界退避、DLX、停车场
    17-消费幂等.md                        # 唯一键与原子副作用
    18-消息契约与演进.md                  # Envelope 与兼容性
    19-Transactional-Outbox.md            # 业务事务与事件原子落库
    20-Outbox-Relay.md                     # 租约、Confirm 与恢复
    21-TeamFlow通知Worker.md               # 通知持久化和 WebSocket
    22-断线恢复与优雅停机.md              # 恢复状态机和 drain
    23-自动化测试.md                      # 单元、集成、E2E、故障测试
    24-可观测性.md                        # 日志、Trace、指标和看板
    25-告警与积压治理.md                  # SLO、阈值和处置流程
    26-安全加固.md                        # 用户、权限、vhost、TLS
    27-性能压测与容量规划.md              # PerfTest 与业务压测
    28-Quorum-Queue.md                    # Raft、多数派与毒消息
    29-集群与Kubernetes指南.md            # 三节点与官方 Operator
    30-故障演练.md                        # 可恢复的 chaos 手册
    31-生产上线检查表.md                  # 上线评审与回滚
    32-排障手册.md                        # 症状到证据到动作
    33-面试与架构复盘.md                  # 架构取舍表达
    附录-A-配置字典与命名规范.md
    附录-B-AMQP与管理命令速查.md
    附录-C-Redis-Stream迁移对照.md
    附录-D-阶段提交与全课程验收.md
```

跨章节保持以下接口与名字不变；后续章节只能扩充语义，不能无说明改名：

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

```text
vhost: /teamflow-dev
topic exchange: teamflow.events.v1
dead-letter exchange: teamflow.dlx.v1
routing keys: task.assigned.v1, task.status.changed.v1
primary queue: teamflow.notification.websocket.q
dead queue: teamflow.notification.dead.q
consumer name: notification-worker
```

### Task 1: Freeze the Official Fact Base and Editorial Contract

**Files:**
- Verify: `docs/research/rabbitmq-official-notes.md`
- Reference: `docs/superpowers/specs/2026-08-25-rabbitmq-enterprise-tutorial-design.md`

**Interfaces:**
- Consumes: RabbitMQ 官方版本、客户端、Confirm、ACK、Quorum、DLX、TTL、TLS、监控、集群与 Operator 资料。
- Produces: 全部章节共同采用的事实基线和禁用表述。

- [ ] **Step 1: 审核研究笔记的来源和基线**

确认笔记只引用 RabbitMQ 官方站、RabbitMQ 官方 GitHub、AMQP 规范和 Docker Official Image，并明确 `4.3.5`、`4.3.5-management`、`amqp091-go v1.14.0` 与 `2026-08-25`。

- [ ] **Step 2: 提取必须贯穿教程的版本陷阱**

写作时必须覆盖：Channel 非线程安全；Confirm 不证明路由成功；mandatory Return 与 Confirm 需关联但不能依赖跨 Go channel 到达顺序；4.3 `basic.nack` 不增加 quorum `delivery-count`；普通 DLX 内部重发默认无 Confirm；TTL 通常到队首才死信；Outbox 仍有重复窗口；两节点生产集群不可取。

- [ ] **Step 3: 验证无低信任来源**

Run: `rg -n "(csdn|juejin|zhihu|stackoverflow|medium\\.com)" docs/research/rabbitmq-official-notes.md`

Expected: 无输出。

- [ ] **Step 4: Commit**

```bash
git add docs/research/rabbitmq-official-notes.md
git commit -m "docs: research RabbitMQ official guidance"
```

### Task 2: Create the Tutorial Entry and Foundation Chapters

**Files:**
- Modify: `TeamFlow企业级开发教程/00-总目录.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/00-学习路线与最终架构.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/01-为什么选择RabbitMQ.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/02-消息系统核心概念.md`

**Interfaces:**
- Consumes: approved five-level maturity ladder and final TeamFlow event flow.
- Produces: reader navigation, vocabulary, selection rules, reliability model used by all later chapters.

- [ ] **Step 1: 在总目录新增独立专题入口**

在原课程目录之后新增“RabbitMQ 企业级实战专题”，链接到专题 `00`；明确原 Redis Stream 章节继续保留用于方案对照。

- [ ] **Step 2: 编写 `00` 的完整学习地图**

包含环境清单、34 篇正文和 4 个附录链接、五级能力阶梯、建议学习节奏、每阶段验收、最终目录树，以及 `HTTP -> MySQL transaction/outbox -> relay -> RabbitMQ -> worker -> notification/WebSocket` 架构图。明确初学者从 03 开始操作、架构评审读者可从 28 开始。

- [ ] **Step 3: 编写 `01` 的场景化选型**

对比 RabbitMQ、Redis Stream、Kafka 时使用“路由能力、消费模型、积压/回放、可靠性、运维成本、适用数据规模”而非虚构固定吞吐；给出 TeamFlow 选择 RabbitMQ 的触发条件，以及继续使用 Redis Stream 和选择 Kafka 的反例。

- [ ] **Step 4: 编写 `02` 的失败模型**

用消息状态时间线解释 at-most-once、at-least-once、重复、丢失、乱序、积压、背压、毒消息；用两处崩溃窗口证明为什么“Broker 持久化”与“Exactly Once”不是同义词，并建立后续 Confirm、ACK、幂等和 Outbox 的问题清单。

- [ ] **Step 5: 验证入口和章节模板**

Run (PowerShell): `rg -n -g '0[0-2]-*.md' "至少一次|Exactly Once|建议 Git 提交" TeamFlow企业级开发教程/RabbitMQ企业级实战; rg -n "RabbitMQ企业级实战/00-学习路线与最终架构.md" TeamFlow企业级开发教程/00-总目录.md`

Expected: 总目录入口存在；三个章节均明确可靠性边界并给出建议提交信息。

- [ ] **Step 6: Commit**

```bash
git add TeamFlow企业级开发教程/00-总目录.md TeamFlow企业级开发教程/RabbitMQ企业级实战/00-学习路线与最终架构.md TeamFlow企业级开发教程/RabbitMQ企业级实战/01-为什么选择RabbitMQ.md TeamFlow企业级开发教程/RabbitMQ企业级实战/02-消息系统核心概念.md
git commit -m "docs: start RabbitMQ enterprise tutorial"
```

### Task 3: Build the Local Broker and AMQP Laboratories

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/03-Docker-Compose从零部署.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/04-管理台与rabbitmqctl.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/05-AMQP-0-9-1工作模型.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/06-四类Exchange实验.md`

**Interfaces:**
- Consumes: Docker Desktop and AMQP vocabulary from Task 2.
- Produces: running `/teamflow-dev` broker and observable exchange/queue/binding experiments.

- [ ] **Step 1: 编写可复现的 Docker Compose 部署**

给出最终可复制的 `docker-compose.rabbitmq.yml`，固定 `rabbitmq:4.3.5-management`、稳定 hostname、命名卷、5672/15672 端口、开发用户/vhost 和 `rabbitmq-diagnostics -q ping` 健康检查；分别给出 PowerShell 启动、状态、日志、停止和保留/清理数据命令，清理卷前明确破坏性影响。

- [ ] **Step 2: 编写管理台与 CLI 诊断路径**

让读者从 Overview、Connections、Channels、Exchanges、Queues 页面定位连接、Channel、Ready、Unacked 与速率，并用 `rabbitmqctl`/`rabbitmq-diagnostics`/`rabbitmq-queues` 在容器内验证 vhost、用户、权限和队列；解释管理 UI 不承担长期监控。

- [ ] **Step 3: 编写 AMQP 实体和声明等价性实验**

展示 Producer、Connection、Channel、Exchange、Binding、Queue、Consumer 的责任和完整消息路径；补充默认 Exchange 的特殊行为，以及同名队列参数不一致会触发 channel-level error、该 Channel 必须废弃的失败实验。

- [ ] **Step 4: 编写四类 Exchange 可视化实验**

为 direct、topic、fanout、headers 分别给出明确的 Exchange、Queue、Binding、测试消息和预期路由矩阵；实验结束解释为什么 TeamFlow 主事件总线选 topic，以及 headers 在本项目中不是默认选择。

- [ ] **Step 5: 验证部署和路由矩阵完整性**

Run (PowerShell): `rg -n -g '0[3-6]-*.md' "4\.3\.5-management|rabbitmq-diagnostics|direct|topic|fanout|headers|预期" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: 固定镜像、健康检查、四类 Exchange 和每个实验的预期结果均存在。

- [ ] **Step 6: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/03-Docker-Compose从零部署.md TeamFlow企业级开发教程/RabbitMQ企业级实战/04-管理台与rabbitmqctl.md TeamFlow企业级开发教程/RabbitMQ企业级实战/05-AMQP-0-9-1工作模型.md TeamFlow企业级开发教程/RabbitMQ企业级实战/06-四类Exchange实验.md
git commit -m "docs: add RabbitMQ local labs"
```

### Task 4: Design and Connect the TeamFlow Messaging Infrastructure

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/07-TeamFlow消息拓扑设计.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/08-引入官方Go客户端.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/09-连接与Channel生命周期.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/10-声明生产级拓扑.md`

**Interfaces:**
- Consumes: current `config/config.go`, `config/config.yaml`, `cmd/main.go`, module `teamflow`, and the canonical names in this plan.
- Produces: tutorial definitions for `RabbitMQConfig`, connection ownership and idempotent `DeclareTopology`.

- [ ] **Step 1: 固化 TeamFlow 拓扑**

定义 `/teamflow-dev`、`teamflow.events.v1`、`teamflow.dlx.v1`、两个 task routing keys、通知 quorum queue 和 dead queue；给出命名规则、发布者/消费者权限矩阵、拓扑所有权，以及为何队列属于消费者而事件属于生产者。

- [ ] **Step 2: 编写客户端依赖与配置演进**

使用 `go get github.com/rabbitmq/amqp091-go@v1.14.0`；为当前 `Config` 增加 URI、连接/握手超时、heartbeat、publish confirm timeout、prefetch、consumer concurrency 等字段，给出 `config.yaml` 的开发示例与环境变量覆盖方式，密码仅用明显占位示例值。

- [ ] **Step 3: 编写连接和 Channel 所有权模型**

给出 `connection.go` 教学代码，说明长连接、publisher/consumer 分离连接、Channel owner、`NotifyClose` 缓冲和关闭顺序；显式禁止 Gin goroutine 共享一个 Channel，并给出“单 owner goroutine + 请求通道”或有界 publisher worker 的选择理由。

- [ ] **Step 4: 编写幂等拓扑声明**

给出 `Topology` 常量/结构和 `DeclareTopology(ctx, ch)`，使用 durable topic exchange、durable quorum queue、绑定和 `noWait=false`；解释 policy 优于可变 `x-arguments`，并用重复声明成功、参数冲突关闭 Channel 两个验证用例。

- [ ] **Step 5: 检查代码连续性**

Run (PowerShell): `rg -n -g '0[7-9]-*.md' -g '10-*.md' "amqp091-go@v1\.14\.0|teamflow\.events\.v1|teamflow\.notification\.websocket\.q|x-queue-type|Channel.*线程安全|noWait" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: 版本、拓扑、quorum、Channel 并发约束和同步声明全部出现，名字完全一致。

- [ ] **Step 6: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/07-TeamFlow消息拓扑设计.md TeamFlow企业级开发教程/RabbitMQ企业级实战/08-引入官方Go客户端.md TeamFlow企业级开发教程/RabbitMQ企业级实战/09-连接与Channel生命周期.md TeamFlow企业级开发教程/RabbitMQ企业级实战/10-声明生产级拓扑.md
git commit -m "docs: design TeamFlow RabbitMQ infrastructure"
```

### Task 5: Deliver the Minimal Publisher and Consumer Loop

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/11-实现基础Publisher.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/12-实现基础Consumer.md`

**Interfaces:**
- Consumes: `EventPublisher`, `EventHandler`, canonical topology, and configuration from Task 4.
- Produces: `Event`, `Publisher.Publish`, delivery decoding and handler-driven manual acknowledgment loop.

- [ ] **Step 1: 编写稳定 Event Envelope 与基础 Publisher**

给出 `internal/event/task_event.go` 和 `internal/mq/rabbitmq/message.go`/`publisher.go` 的逐文件代码；Envelope 包含 event ID、type、schema version、occurred at、correlation/causation/trace IDs 和 `json.RawMessage` payload。Publish 设置 content type、message ID、timestamp、type 和 persistent delivery mode，并清楚标记本章尚未解决 Confirm/Return 双重结果。

- [ ] **Step 2: 编写基础 Consumer 与 Handler seam**

给出 `consumer.go` 的 `autoAck=false` 消费循环、JSON/Envelope 校验、context 超时、Handler 调用和 `Ack(false)`；错误先用简单分类占位为“本章策略”，下一章必须明确替换为有限重试而不是让两套策略并存。

- [ ] **Step 3: 接入任务指派的最小闭环**

展示 Task Service 依赖 `EventPublisher` 而不是 AMQP 类型、发布 `task.assigned.v1`，再由最小 worker 打印事件；给出 API 与 Worker 两个终端的启动方式、管理台验证、预期日志字段和故意错误 JSON 的失败验证。

- [ ] **Step 4: 验证接口和包路径**

Run (PowerShell): `rg -n -g '1[1-2]-*.md' "type EventPublisher interface|type EventHandler interface|json\.RawMessage|DeliveryMode|autoAck.*false|Ack\(false\)|teamflow/internal" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: 两个接口、Envelope、持久属性、手动 ACK 和 `teamflow/...` import 均可定位。

- [ ] **Step 5: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/11-实现基础Publisher.md TeamFlow企业级开发教程/RabbitMQ企业级实战/12-实现基础Consumer.md
git commit -m "docs: add RabbitMQ publish consume loop"
```

### Task 6: Make Publishing Outcomes Explicit

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/13-Publisher-Confirm.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/14-Mandatory与Return.md`

**Interfaces:**
- Consumes: `Publisher.Publish` and exclusive Channel owner from Tasks 4–5.
- Produces: bounded in-flight publish state keyed by publish sequence/message ID with `confirmed`, `nacked`, `returned`, `timeout`, and `connection_lost_unknown` outcomes.

- [ ] **Step 1: 编写 Confirm 状态机**

解释 `Publish` 返回 nil、broker ack 和持久化责任的区别；给出开启 confirm mode、注册 buffered `NotifyPublish`、维护有界 in-flight、超时和连接关闭归类的代码。先展示逐条等待的正确性版本，再说明批量/异步版本为何改善吞吐。

- [ ] **Step 2: 编写 mandatory Return 关联逻辑**

发布时设置 `mandatory=true`，持续消费 `NotifyReturn`；用 message ID/发布记录关联 Return 与 Confirm，不依赖两个 Go notification channel 的到达顺序。不可路由被归为永久拓扑错误，错误中保留 exchange、routing key、reply code/text 和 message ID。

- [ ] **Step 3: 编写三个发布失败实验**

分别验证正常路由得到 positive confirm、错误 routing key 同时产生 Return 和 confirm、Broker 在等待期间关闭产生 unknown；每个实验给出预期状态、是否重试、是否告警，且说明 Confirm 和 mandatory 都不能替代 Outbox。

- [ ] **Step 4: 验证关键反误区表述**

Run (PowerShell): `rg -n -g '1[3-4]-*.md' "Publish.*nil|NotifyPublish|NotifyReturn|mandatory.*true|不可路由|unknown|不能替代.*Outbox|到达顺序" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: 所有发布结果及 Confirm/Return 边界均明确。

- [ ] **Step 5: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/13-Publisher-Confirm.md TeamFlow企业级开发教程/RabbitMQ企业级实战/14-Mandatory与Return.md
git commit -m "docs: add reliable RabbitMQ publishing"
```

### Task 7: Make Consumption Bounded, Retryable, Idempotent, and Evolvable

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/15-Consumer-ACK与QoS.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/16-重试与死信体系.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/17-消费幂等.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/18-消息契约与演进.md`

**Interfaces:**
- Consumes: `EventHandler.Handle`, manual ACK loop, event ID, schema version.
- Produces: `FailureKind`, `RetryClassifier`, retry headers/policies, `processed_messages` uniqueness, version dispatch rules.

- [ ] **Step 1: 编写 ACK、delivery tag 与 QoS 规则**

说明副作用事务提交后才 `Ack(false)`、delivery tag 只能在原 Channel 使用、并行 worker 不得随意 `multiple=true`；用 `Qos(prefetchCount, 0, false)` 建立未确认窗口，并用“慢 Handler + 两个 consumer”观察公平性和 Unacked。

- [ ] **Step 2: 编写有限重试、死信和停车场设计**

定义临时、永久、未知三类错误和 `RetryClassifier`；采用明确次数和档位的延迟重试，禁止无限 `Nack(requeue=true)`。同时讲清 RabbitMQ 4.3 `basic.nack` 不增加 quorum `delivery-count`、TTL 队首限制、普通 DLX 无内部 Confirm，以及 quorum at-least-once dead lettering 的前置条件和资源代价。

- [ ] **Step 3: 编写事务性消费幂等**

给出 `processed_messages(consumer_name, message_id, processed_at)` 模型、复合唯一索引和完整 GORM transaction 代码：插入去重键与通知写入同事务；唯一冲突视为已完成并 ACK，其他 DB 错误不 ACK 并进入有限重试。用并发重复投递测试证明只生成一条通知。

- [ ] **Step 4: 编写消息契约与版本路由**

定义 Envelope 校验、`task.assigned.v1` payload、兼容新增字段、破坏性变化新版本、未知版本永久失败；说明 correlation/causation/trace ID 的生成和传播规则，禁止消费者反序列化到数据库 Model。

- [ ] **Step 5: 验证可靠消费不变量**

Run (PowerShell): `rg -n -g '1[5-8]-*.md' "Ack\(false\)|Qos\(|Nack\(requeue=true\)|delivery-count|dead-letter-strategy|processed_messages|unique|schema_version|未知版本" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: ACK/QoS、4.3 重试陷阱、DLX 边界、幂等唯一键和契约版本均出现。

- [ ] **Step 6: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/15-Consumer-ACK与QoS.md TeamFlow企业级开发教程/RabbitMQ企业级实战/16-重试与死信体系.md TeamFlow企业级开发教程/RabbitMQ企业级实战/17-消费幂等.md TeamFlow企业级开发教程/RabbitMQ企业级实战/18-消息契约与演进.md
git commit -m "docs: add reliable RabbitMQ consumption"
```

### Task 8: Close the Database–Broker Dual-Write Gap

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/19-Transactional-Outbox.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/20-Outbox-Relay.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/21-TeamFlow通知Worker.md`

**Interfaces:**
- Consumes: Event envelope, explicit publish outcomes, consumer idempotency, current Task/Notification service boundaries.
- Produces: `outbox_events` state model, lease-based relay, independent `cmd/worker/main.go`, transactional notification Handler.

- [ ] **Step 1: 编写 Outbox 表与业务事务**

给出 `outbox_events` 字段：event ID、aggregate type/ID、event type、schema version、routing key、payload、status、attempts、available_at、lease_owner、lease_until、published_at、last_error 和 timestamps；定义索引与 pending/publishing/published/failed 状态机。展示 Task 更新和 Outbox insert 在一个 GORM transaction 内完成。

- [ ] **Step 2: 编写短事务租约 Relay**

分为“短事务领取并提交租约 -> 事务外发布并等 Confirm/Return -> 短事务标记 published 或重新调度”三段，严禁等待网络时持有行锁；给出多实例 owner token、lease 过期恢复、批量上限、退避、永久 Return 失败和安全停止领取的代码与 SQL/GORM 要点。

- [ ] **Step 3: 用崩溃窗口证明语义**

验证业务事务回滚无事件、提交后 Relay 最终发布、Confirm 后标记前崩溃会重发、租约领取后崩溃可恢复；明确结论是最终至少一次，消费者幂等仍不可删除。

- [ ] **Step 4: 编写独立通知 Worker**

给出 `cmd/worker/main.go` 组装顺序、`notification_worker.go` Handler、通知与 `processed_messages` 同事务写入、事务提交后 WebSocket 尽力推送；WebSocket 失败不回滚通知，客户端通过通知查询补偿。

- [ ] **Step 5: 验证双写边界和锁边界**

Run (PowerShell): `rg -n -g '19-*.md' -g '2[0-1]-*.md' "outbox_events|lease_until|事务外|Publisher Confirm|至少一次|processed_messages|WebSocket.*不回滚|cmd/worker/main.go" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: Outbox schema、lease、网络等待不持锁、重复窗口和 Worker 事务边界均明确。

- [ ] **Step 6: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/19-Transactional-Outbox.md TeamFlow企业级开发教程/RabbitMQ企业级实战/20-Outbox-Relay.md TeamFlow企业级开发教程/RabbitMQ企业级实战/21-TeamFlow通知Worker.md
git commit -m "docs: add RabbitMQ outbox and worker"
```

### Task 9: Engineer Lifecycle and Automated Proof

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/22-断线恢复与优雅停机.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/23-自动化测试.md`

**Interfaces:**
- Consumes: publisher/consumer owners, topology initializer, relay and handler seams.
- Produces: recovery/drain state machines and a four-layer test matrix.

- [ ] **Step 1: 编写恢复状态机**

解释 `amqp091-go v1.14.0` 的可选 auto-recovery 与应用消息语义恢复的区别；恢复步骤必须重建 connection、Channel、Confirm、Return listener、topology、consumer，并将未确认发布归为 unknown。Readiness 在依赖不可用时失败，Liveness 不因短暂 RabbitMQ 故障立刻失败。

- [ ] **Step 2: 编写有 deadline 的优雅停机**

Publisher/Relay 先停止接单或 claim，等待 Confirm/Return 后关闭 Channel/Connection；Consumer 先 Cancel/取消 context、drain deliveries、等待 Handler 与 ACK，deadline 后关闭连接让 Broker requeue。给出 SIGTERM、WaitGroup 和 shutdown context 的完整组装代码。

- [ ] **Step 3: 编写可执行自动化测试金字塔**

给出单元测试表和 Go test 示例：Envelope 校验、版本分派、fake publisher、retry classifier、幂等 Handler、Outbox 状态机；给出真实 RabbitMQ/MySQL 的集成测试启动/隔离/清理方式，覆盖 Confirm、Return、ACK、prefetch、DLX、并发幂等和 lease；再给 E2E 与 crash-window 故障测试。

- [ ] **Step 4: 验证测试覆盖映射**

Run (PowerShell): `rg -n -g '2[2-3]-*.md' "auto-recovery|Readiness|Liveness|Cancel\(|SIGTERM|单元测试|集成测试|端到端|故障测试|Confirm|Return|lease" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: 恢复、停机和四类测试均有可执行验证。

- [ ] **Step 5: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/22-断线恢复与优雅停机.md TeamFlow企业级开发教程/RabbitMQ企业级实战/23-自动化测试.md
git commit -m "docs: add RabbitMQ lifecycle and tests"
```

### Task 10: Add Production Observability, Alerts, Security, and Capacity

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/24-可观测性.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/25-告警与积压治理.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/26-安全加固.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/27-性能压测与容量规划.md`

**Interfaces:**
- Consumes: event IDs, publish outcome states, queue names, relay/worker metrics.
- Produces: log schema, Prometheus metrics, actionable SLO alerts, permission/TLS design, reproducible capacity worksheet.

- [ ] **Step 1: 编写三层可观测性**

定义应用结构化日志字段、Trace 传播和指标名；启用 `rabbitmq_prometheus` 的 `:15692/metrics`，解释 aggregated、per-object 和 detailed 指标代价；给出 API/Relay/Worker 与 Broker 的 Grafana 面板布局和一次 Return 的追踪示例。

- [ ] **Step 2: 编写带窗口和 Runbook 的告警**

覆盖 outbox oldest age、ready 增长、unacked 高位、returned/nack、dead queue、consumer 消失、resource alarm、quorum 成员与 confirm p99；每条规则写清触发条件、持续窗口、级别、首要证据、止损动作和恢复判定，避免只给静态队列长度。

- [ ] **Step 3: 编写最小权限和 TLS 加固**

区分 API publisher、Worker consumer、topology initializer、monitor 和 operator 用户；给出 configure/write/read 正则和 vhost 隔离，删除/禁用远程 guest，使用 `amqps://`、CA 与 hostname verification；Secret 只通过环境/平台注入，并说明管理端口、AMQP、metrics 和集群内部端口的网络边界。

- [ ] **Step 4: 编写容量实验**

先用 RabbitMQ PerfTest 得到 Broker 基线，再用 TeamFlow 链路测端到端；矩阵必须包含 quorum 副本、persistent、Confirm window、mandatory、消息大小、prefetch、consumer 数、Handler 延迟、TLS 与故障态。报告记录吞吐、P50/P95/P99、Ready/Unacked、CPU、内存、磁盘 I/O、网络和 drain 时间，并用生产峰值、增长率、故障冗余推导容量。

- [ ] **Step 5: 验证生产工程覆盖**

Run (PowerShell): `rg -n -g '2[4-7]-*.md' "15692/metrics|oldest|持续|configure|write|read|amqps|hostname|PerfTest|P50|P95|P99|故障态" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: 指标、可执行告警、权限/TLS 和可靠配置下的压测矩阵均存在。

- [ ] **Step 6: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/24-可观测性.md TeamFlow企业级开发教程/RabbitMQ企业级实战/25-告警与积压治理.md TeamFlow企业级开发教程/RabbitMQ企业级实战/26-安全加固.md TeamFlow企业级开发教程/RabbitMQ企业级实战/27-性能压测与容量规划.md
git commit -m "docs: add RabbitMQ production engineering"
```

### Task 11: Explain High Availability and Run Safe Fault Drills

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/28-Quorum-Queue.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/29-集群与Kubernetes指南.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/30-故障演练.md`

**Interfaces:**
- Consumes: single-node labs, production topology, capacity and alert design.
- Produces: three-node production blueprint, Operator boundary, reversible failure exercises.

- [ ] **Step 1: 编写 Quorum Queue 的真实保证和代价**

解释 Raft leader/follower、多数派 Confirm、三成员容忍一节点、为什么两节点不合格、为何奇数成员更合理；列出不适用场景。专门解释 4.3 delivery-limit/delivery-count、at-least-once DLX 前置条件和 delayed retry 对 `basic.nack` 的限制。

- [ ] **Step 2: 编写三节点与升级蓝图**

给出节点/可用区/磁盘/网络/leader 分布、客户端多个 endpoint、容量 N-1 验收和网络分区行为；升级路径明确 `4.2 -> 4.3`、3.13 先到 4.2、stable feature flags、rolling 与 blue-green、RabbitMQ 不支持把 downgrade 当正式回滚。

- [ ] **Step 3: 编写 Kubernetes Operator 指南**

以官方 Cluster Operator 和 Messaging Topology Operator 为唯一推荐路线；提供 RabbitmqCluster 关键字段示例，解释 PVC、反亲和、PDB、requests/limits、termination grace、Secret、监控和 topology ownership；说明 Operator 不会替应用完成 AMQP 语义恢复，也不会自动升级已有集群。

- [ ] **Step 4: 编写安全故障演练矩阵**

覆盖 Broker restart、短时断网、Worker 在事务前后崩溃、Relay 在 Confirm 前后崩溃、MySQL 不可用、不可路由、毒消息、DLX 目标缺失、积压恢复和单节点失效。每项包括前提、影响、观测信号、注入步骤、停止条件、恢复命令、数据不变量和清理动作；生产禁止直接照搬破坏性命令。

- [ ] **Step 5: 验证 HA 事实和演练安全栏**

Run (PowerShell): `rg -n -g '2[8-9]-*.md' -g '30-*.md' "三节点|两节点|多数派|basic\.nack|stable feature flags|blue-green|Cluster Operator|停止条件|恢复步骤|数据不变量" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: HA 边界、升级、Operator 和每个演练的安全要素均明确。

- [ ] **Step 6: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/28-Quorum-Queue.md TeamFlow企业级开发教程/RabbitMQ企业级实战/29-集群与Kubernetes指南.md TeamFlow企业级开发教程/RabbitMQ企业级实战/30-故障演练.md
git commit -m "docs: add RabbitMQ HA and fault drills"
```

### Task 12: Finish Go-Live Review, Troubleshooting, and Architecture Narrative

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/31-生产上线检查表.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/32-排障手册.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/33-面试与架构复盘.md`

**Interfaces:**
- Consumes: all previous chapter invariants, metrics, failure modes and deployment boundaries.
- Produces: auditable release gate, symptom-based runbook and accurate project narrative.

- [ ] **Step 1: 编写可签字的上线检查表**

按版本/升级、拓扑、发布、消费、Outbox、数据一致性、容量、监控告警、安全、备份恢复、演练、变更与回滚分类；每项包含证据、责任人角色、通过条件和阻断级别。回滚不能写成 RabbitMQ downgrade，必须选择应用回退或 blue-green。

- [ ] **Step 2: 编写证据优先的排障手册**

以症状索引组织：连不上、Channel 关闭、消息 Return、Confirm 超时、Ready 增长、Unacked 增长、重复通知、重试热循环、dead queue 增长、磁盘/内存 alarm、quorum 不可用、恢复抖动。每项按“先看哪些指标/日志 -> 常见原因 -> 验证命令 -> 安全止损 -> 根治”展开。

- [ ] **Step 3: 编写面试与架构复盘**

用 2 分钟、10 分钟和深挖三种粒度讲 TeamFlow 链路；准备至少一次错误方案到成熟方案的演进，覆盖选型、Confirm+Return、ACK、幂等、Outbox、重试、Quorum、容量和 Exactly Once 反问。所有表达都限定条件，不使用虚构 QPS 或“绝不丢”。

- [ ] **Step 4: 验证终局一致性**

Run (PowerShell): `rg -n -g '3[1-3]-*.md' "证据|阻断|blue-green|Channel|Ready|Unacked|Outbox|Exactly Once|绝不丢" TeamFlow企业级开发教程/RabbitMQ企业级实战`

Expected: 上线证据、主要症状和面试可靠性边界可定位；“绝不丢”只出现在禁止/纠错语境。

- [ ] **Step 5: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战/31-生产上线检查表.md TeamFlow企业级开发教程/RabbitMQ企业级实战/32-排障手册.md TeamFlow企业级开发教程/RabbitMQ企业级实战/33-面试与架构复盘.md
git commit -m "docs: finish RabbitMQ operations guide"
```

### Task 13: Add Appendices and Perform the Full Editorial Verification

**Files:**
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/附录-A-配置字典与命名规范.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/附录-B-AMQP与管理命令速查.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/附录-C-Redis-Stream迁移对照.md`
- Create: `TeamFlow企业级开发教程/RabbitMQ企业级实战/附录-D-阶段提交与全课程验收.md`
- Modify: all files in `TeamFlow企业级开发教程/RabbitMQ企业级实战/`

**Interfaces:**
- Consumes: every chapter, canonical code names, official source notes and original Redis Stream tutorial.
- Produces: navigation-complete, internally consistent and executable final course.

- [ ] **Step 1: 编写四个附录**

附录 A 汇总配置字段、默认开发值、生产建议、敏感级别和环境变量；附录 B 汇总 AMQP API、管理命令、常见错误码和“命令运行位置”；附录 C 按拓扑、消费者组、ACK、pending、重试、顺序、回放和运维对照 Redis Stream，给出渐进迁移和回滚边界；附录 D 给出每阶段建议提交、完整文件清单和 Level 0–5 验收清单。

- [ ] **Step 2: 检查文件数量与双向导航**

Run (PowerShell):

```powershell
$docs = Get-ChildItem 'TeamFlow企业级开发教程/RabbitMQ企业级实战' -File -Filter '*.md'
$docs.Count
```

Expected: `38`（34 篇编号正文 + 4 个附录）。`00` 链接全部 37 个后续文档，每篇末尾提供上一篇、目录、下一篇链接。

- [ ] **Step 3: 检查章节必需栏目**

Run (PowerShell):

```powershell
$required = @('本节目标','前置条件','文件变化','验证','失败','排查','生产','检查点','建议 Git 提交')
Get-ChildItem 'TeamFlow企业级开发教程/RabbitMQ企业级实战' -File -Filter '[0-3][0-9]-*.md' | ForEach-Object {
    $text = Get-Content -Raw $_.FullName
    $missing = $required | Where-Object { $text -notmatch [regex]::Escape($_) }
    if ($missing) { "{0}: {1}" -f $_.Name, ($missing -join ', ') }
}
```

Expected: 无输出。

- [ ] **Step 4: 检查版本、术语、包路径和危险表述**

Run:

```bash
rg -n "rabbitmq:(latest|4\.2)|streadway/amqp|Exactly Once|绝不丢失|百万级/s|万级/s|teamflow\.internal|internal/mq/stream\.go" TeamFlow企业级开发教程/RabbitMQ企业级实战
```

Expected: 不出现浮动/过时依赖、错误包路径或无条件可靠性/吞吐承诺；`Exactly Once` 只出现在否定和纠错语境；不要求改动 `internal/mq/stream.go`。

- [ ] **Step 5: 检查 Markdown 相对链接目标存在**

Run (PowerShell):

```powershell
$root = Resolve-Path 'TeamFlow企业级开发教程'
Get-ChildItem $root -Recurse -File -Filter '*.md' | ForEach-Object {
    $file = $_
    $text = Get-Content -Raw $file.FullName
    [regex]::Matches($text, '\[[^\]]+\]\((?!https?://|#)([^)#]+)(?:#[^)]+)?\)') | ForEach-Object {
        $target = Join-Path $file.DirectoryName ([uri]::UnescapeDataString($_.Groups[1].Value))
        if (-not (Test-Path -LiteralPath $target)) { "BROKEN {0} -> {1}" -f $file.FullName, $_.Groups[1].Value }
    }
}
```

Expected: 无 `BROKEN` 输出。

- [ ] **Step 6: 人工代码连续性审计**

从 08 到 23 逐章追踪 `RabbitMQConfig`、`Event`、`EventPublisher`、`Publisher`、`EventHandler`、`RetryClassifier`、`outbox_events` 和 `processed_messages`；确认每次替换旧代码都有明确完整修改点，所有 Go import 使用 `teamflow/...`，所有命令注明 PowerShell、容器内 Shell 或通用 Shell。

- [ ] **Step 7: Commit**

```bash
git add TeamFlow企业级开发教程/RabbitMQ企业级实战 TeamFlow企业级开发教程/00-总目录.md
git commit -m "docs: complete RabbitMQ enterprise course"
```

## Final Acceptance

执行完成后必须同时满足：

1. `TeamFlow企业级开发教程/RabbitMQ企业级实战/` 恰有 38 个 Markdown 文件，编号正文无缺号。
2. 原 `第14章-消息队列` 内容保持不变，学习者已有 `internal/mq/` 内容保持不变。
3. 教程从单节点 Docker 实验逐步演进到 Outbox、独立 Worker、观测、安全、容量、Quorum 和三节点设计，没有在前置知识出现前使用未解释抽象。
4. 发布链路同时覆盖 Confirm、mandatory Return 和 unknown outcome；消费链路覆盖事务后 ACK、prefetch、有限重试、停车场和事务幂等。
5. 4.3.5/v1.14.0 的版本特有行为与升级边界有官方来源，所有可能变化的默认值都注明验证日期。
6. 每篇都有可执行验证、失败场景、恢复/排查和建议提交，所有相对链接有效。
7. 全局未出现无条件 Exactly Once、“绝不丢”或脱离测试上下文的固定高吞吐承诺。
