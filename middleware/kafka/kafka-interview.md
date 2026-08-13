#kafka

# Kafka 面试知识点

相关笔记：[[kafka-basics]] | [[producer-partition]] | [[producer-compression]] | [[zero-message-loss]] | [[cluster-config]]

## 1. Kafka 整体设计

> [!question]- 参考答案（点击展开）
>
> Kafka 将消息以 Topic 为单位归纳。发布消息的程序称为 **Producer**，消费消息的程序称为 **Consumer**。集群由一个或多个 **Broker** 组成，Producer 通过网络将消息发送到 Kafka 集群，集群向消费者提供消息。
>
> ### 核心组件
>
> | 组件 | 说明 |
> |------|------|
> | **Producer** | 消息生产者，发布消息到 Kafka 集群 |
> | **Broker** | Kafka 节点，多个 Broker 组成集群 |
> | **Topic** | 消息主题，Kafka 面向 Topic 进行消息组织 |
> | **Partition** | Topic 的物理分区，有序不可变的记录序列。单 Partition 内有序，跨 Partition 无序 |
> | **Consumer** | 消息消费者 |
> | **Consumer Group** | 消费者组，每条消息只能被组内一个 Consumer 消费，但可被多个组消费 |
> | **Replica** | Partition 的副本，保障高可用 |
> | **Controller** | 集群中负责 Leader Election 和 Failover 的 Broker |
> | **Zookeeper** | 存储集群 meta 信息（新版本逐步用 KRaft 替代） |
>
> > Broker 与 Partition 的分配关系：若 Topic 有 n 个 Partition、集群有 m 个 Broker，当 m >= n 时每个 Broker 最多存一个 Partition；当 m < n 时部分 Broker 会存多个 Partition（应尽量避免，会导致数据不均衡）。

## 2. Kafka 高性能原因

> [!question]- 参考答案（点击展开）
>
> ```mermaid
> graph LR
>     A[高性能] --> B[PageCache 缓存]
>     A --> C[磁盘顺序写]
>     A --> D[零拷贝技术]
>     A --> E[Pull 拉模式]
> ```
>
> 1. **PageCache 缓存**：利用操作系统页缓存，减少磁盘 I/O
> 2. **磁盘顺序写**：顺序写磁盘的速度接近内存随机写
> 3. **零拷贝（Zero-Copy）**：通过 sendfile 系统调用，数据直接从 PageCache 到网卡，避免用户态拷贝
> 4. **Pull 拉模式**：Consumer 主动拉取，按自身能力控制消费速率

## 3. Kafka 文件高效存储设计

> [!question]- 参考答案（点击展开）
>
> 1. Topic 的 Partition 大文件分成多个小 Segment 文件，便于定期清除已消费数据
> 2. 通过索引信息快速定位 Message
> 3. 索引元数据全部映射到 memory，避免 Segment 的磁盘 I/O
> 4. 索引文件稀疏存储，降低索引元数据空间占用

## 4. Kafka 的优缺点

> [!question]- 参考答案（点击展开）
>
> **优点**：
> - 高性能、高吞吐、低延迟：生产和消费速度达到每秒 10 万级
> - 高可用：消息持久化到磁盘，支持数据备份
> - 高并发：支持数千客户端同时读写
> - 容错性：允许 n-1 个节点失败（n 为副本数）
> - 高扩展性：支持热伸缩，无须停机
>
> **缺点**：
> - 没有完整的监控工具集
> - 不支持通配符主题选择

## 5. Kafka 应用场景

> [!question]- 参考答案（点击展开）
>
> | 场景 | 说明 |
> |------|------|
> | **日志聚合** | 收集各服务日志写入 Kafka 进行存储和分析 |
> | **消息系统** | 作为消息中间件在系统间传递消息 |
> | **系统解耦** | 重要操作完成后发消息，其他服务异步处理 |
> | **流量削峰** | 秒杀/抢购场景中缓冲高流量压力 |
> | **异步处理** | 消息入队后延迟处理 |

## 6. 分区概念

> [!question]- 参考答案（点击展开）
>
> 主题是逻辑概念，可细分为多个**分区（Partition）**。分区在存储层面是可追加的日志文件，消息被追加时分配唯一的 **offset（偏移量）**。
>
> > Kafka 保证的是**分区有序**而不是主题有序，offset 不跨分区。
>
> 分区引入了多副本（Replica）机制：
> - **Leader 副本**：负责读写
> - **Follower 副本**：被动同步 Leader，不对外服务
> - 副本分布在不同 Broker 上，Leader 故障时从 Follower 中选举新 Leader

## 7. 分区策略

> [!question]- 参考答案（点击展开）
>
> 1. **指定 Partition**：直接使用指定值
> 2. **有 Key 无 Partition**：`hash(key) % partitionCount`
> 3. **无 Key 无 Partition**：Round-Robin 轮询
>
> 详细分区策略参见 [[producer-partition]]。

## 8. 为什么要分区

> [!question]- 参考答案（点击展开）
>
> 1. **扩展性**：每个 Partition 可独立存储，Topic 跨多个 Broker，适应任意数据量
> 2. **并发性**：以 Partition 为单位并行读写

## 9. 生产者运行流程

> [!question]- 参考答案（点击展开）
>
> ```mermaid
> graph LR
>     A[消息] --> B[ProducerRecord 封装]
>     B --> C[序列化]
>     C --> D[分区器<br/>选择 Partition]
>     D --> E[缓冲区<br/>按 Batch 组织]
>     E --> F[Sender 线程]
>     F --> G[Broker]
> ```
>
> ![[kafka生产者.png]]
>
> 1. 消息封装为 ProducerRecord 对象
> 2. 序列化处理（默认或自定义序列化器）
> 3. 分区处理，获取集群元数据决定目标 Partition
> 4. 消息放入生产者缓冲区，按 Batch 组织（默认 16KB）
> 5. Sender 线程从缓冲区获取可发送的 Batch
> 6. 批量发送到 Broker

## 10. 消息封装（Batching）

> [!question]- 参考答案（点击展开）
>
> Producer 将消息在内存中累积后批量发送，可从三个维度控制：
>
> | 维度 | 示例 |
> |------|------|
> | 消息数量 | 累积 500 条 |
> | 时间间隔 | 每 100ms |
> | 数据大小 | 达到 64KB |
>
> 增大 Batch 可减少网络请求和磁盘 I/O 频次，但需在**吞吐量和时效性**之间权衡。

## 11. 消费模式

> [!question]- 参考答案（点击展开）
>
> Kafka 采用 **Pull 模式**：Consumer 主动从 Broker 拉取消息。
>
> | 模式 | 优点 | 缺点 |
> |------|------|------|
> | **Push** | 实时性好 | Consumer 难以处理不同速率的推送 |
> | **Pull** | Consumer 自主控制速率和批量 | Broker 无消息时 Consumer 空轮询 |
>
> > Kafka 提供参数让 Consumer 在无消息时阻塞等待，避免空轮询。

## 12. 负载均衡与故障转移

> [!question]- 参考答案（点击展开）
>
> ### 负载均衡
>
> Kafka 通过智能化的**分区 Leader 选举**实现负载均衡，将各 Partition 的 Leader 均匀分散到所有 Broker 上。
>
> ### 故障转移
>
> 通过 **Zookeeper 会话机制**实现：Broker 启动后以会话形式注册到 Zookeeper，运转异常导致会话超时断连时，集群选举另一台 Broker 替代。

## 13. Zookeeper 的作用

> [!question]- 参考答案（点击展开）
>
> - Broker 启动时在 Zookeeper 注册
> - 统一协调管理各 Broker
> - 维护分区信息及与 Broker 的对应关系
> - 周期性提交 offset，节点失败时可从中恢复

## 14. 系统工具

> [!question]- 参考答案（点击展开）
>
> - **Kafka 迁移工具**：辅助 Broker 版本迁移
> - **MirrorMaker**：跨集群数据镜像同步
> - **消费者检查**：显示 Topic、Partition、Owner 信息

## 15. Consumer Group 与负载均衡

> [!question]- 参考答案（点击展开）
>
> Consumer Group 是 Kafka 的可扩展容错消费机制：
>
> ```mermaid
> graph TB
>     subgraph Topic A
>         P0[Partition 0]
>         P1[Partition 1]
>         P2[Partition 2]
>         P3[Partition 3]
>     end
>
>     subgraph Consumer Group 1
>         C1[Consumer 1]
>         C2[Consumer 2]
>     end
>
>     subgraph Consumer Group 2
>         C3[Consumer 3]
>     end
>
>     P0 --> C1
>     P1 --> C1
>     P2 --> C2
>     P3 --> C2
>
>     P0 --> C3
>     P1 --> C3
>     P2 --> C3
>     P3 --> C3
> ```
>
> - 组内 Consumer 共享一个 Group ID
> - 每个 Partition 只被组内一个 Consumer 消费
> - Consumer 数量不应超过 Partition 数量（否则有空闲 Consumer）
> - Consumer 加入或退出时触发 **Rebalance**，重新分配 Partition
>
> > Consumer 订阅的是 Partition 而非 Message，同一时间同一 Partition 只能被同一 Group 内的一个 Consumer 消费。
>
> ### Rebalance 机制
>
> Consumer 周期性向 Coordinator 发送 heartbeat，若超时未收到，Coordinator 认为该 Consumer 退出，触发 Rebalance 将其 Partition 分配给组内其他 Consumer。

## 16. 消息偏移量（Offset）

> [!question]- 参考答案（点击展开）
>
> Offset 是分区中消息的**唯一顺序 ID**，主要作用：
> - 唯一标识分区内的每条消息
> - Kafka 存储文件按 offset 命名

## 17. QueueFullException 处理

> [!question]- 参考答案（点击展开）
>
> **触发条件**：Producer 发送速度 > Broker 处理速度
>
> **解决方案**：
> 1. 降低 Producer 生产速率
> 2. 增加 Broker 节点分担负载
> 3. 设置 `queue.enqueue.timeout.ms = -1`，生产者阻塞等待而非丢弃消息
> 4. 容忍消息丢弃

## 18. Consumer 消费指定分区

> [!question]- 参考答案（点击展开）
>
> Consumer 向 Broker 发出 `fetch` 请求时，可通过指定 offset 从特定位置开始消费。Consumer 拥有 offset 的控制权，可以向后回滚重新消费。
>
> 也可使用 `seek(TopicPartition, long offset)` 指定消费位置。

## 19. Replica、Leader 和 Follower

> [!question]- 参考答案（点击展开）
>
> > Kafka Partition 的副本（Replica）用于实现高可用，防止数据丢失。
>
> ```mermaid
> graph TB
>     subgraph Partition 0
>         L[Leader<br/>读写]
>         F1[Follower 1<br/>同步]
>         F2[Follower 2<br/>同步]
>     end
>
>     P[Producer] -->|写| L
>     C[Consumer] -->|读| L
>     L -->|同步| F1
>     L -->|同步| F2
> ```
>
> - **Leader**：负责读写，Producer 写消息到 Leader，Consumer 从 Leader 读消息
> - **Follower**：被动同步 Leader，不与客户端交互。向 Leader 请求最新消息以保持同步
> - 副本分布在不同 Broker，Leader 异常时从 Follower 中选举新 Leader

## 20. Replica 的重要性

> [!question]- 参考答案（点击展开）
>
> Replica 确保发布的消息不丢失，保证 Kafka 高可用。在机器错误、程序错误、软件升级、扩容等场景下都能正常使用。

## 21. Geo-Replication

> [!question]- 参考答案（点击展开）
>
> Kafka 官方提供 **MirrorMaker** 组件实现跨集群数据同步：
>
> ```mermaid
> graph LR
>     subgraph 源集群
>         S[Source Cluster]
>     end
>
>     MM[MirrorMaker<br/>Consumer + Producer]
>
>     subgraph 目标集群
>         T[Target Cluster]
>     end
>
>     S --> MM --> T
> ```
>
> 原理：从源集群消费消息，然后将消息生产到目标集群。适用于：
> - 主动/被动模式的备份和恢复
> - 跨数据中心的数据就近访问
> - 数据本地化合规需求

## 22. AR、ISR、OSR

> [!question]- 参考答案（点击展开）
>
> | 概念 | 全称 | 说明 |
> |------|------|------|
> | **AR** | Assigned Replicas | 分区中所有副本 |
> | **ISR** | In-Sync Replicas | 与 Leader 保持同步的副本集合（含 Leader） |
> | **OSR** | Out-of-Sync Replicas | 与 Leader 滞后过多的副本 |
>
> 关系：`AR = ISR + OSR`

## 23. 副本从 ISR 中剔除的条件

> [!question]- 参考答案（点击展开）
>
> Leader 动态维护 ISR 列表。当 Follower 满足以下条件时被移出 ISR：
> - 落后 Leader 过多
> - 超过一定时间未发起数据复制请求
>
> > 只有当 ISR 中所有 Replica 都向 Leader 发送 ACK 时，Leader 才 commit 消息。

## 24. ISR 为空时的 Leader 选举

> [!question]- 参考答案（点击展开）
>
> 通过 `unclean.leader.election.enable` 配置：
>
> | 值 | 行为 | 风险 |
> |----|------|------|
> | `true` | 允许 OSR 成为 Leader | 消息可能不一致（OSR 数据滞后） |
> | `false` | 等待旧 Leader 恢复 | 降低可用性 |

## 25. 判断 Broker 是否有效

> [!question]- 参考答案（点击展开）
>
> 1. Broker 必须维持与 Zookeeper 的连接（心跳检测）
> 2. 如果是 Follower，必须能及时同步 Leader 的写操作，延时不能过大

## 26. 最大消息大小

> [!question]- 参考答案（点击展开）
>
> 默认 **1000000 字节（约 1MB）**，通过 `message.max.bytes` 修改。
>
> > 注意：`message.max.bytes` 必须小于消费端的 `fetch.message.max.bytes`（默认 1MB），否则 Broker 会因消费端无法读取消息而挂起。

## 27. ACK 机制

> [!question]- 参考答案（点击展开）
>
> | acks 值 | 行为 | 延迟 | 可靠性 |
> |---------|------|------|--------|
> | **0** | 不等待确认，发完即走 | 最低 | 最差，可能丢失 |
> | **1** | Leader 确认即返回（默认） | 较低 | 较好，Leader 宕机时可能丢失 |
> | **-1 (all)** | ISR 所有副本确认 | 最高 | 最好 |
>
> 详细的零丢失配置参见 [[zero-message-loss]]。

## 28. Consumer 消费数据方式

> [!question]- 参考答案（点击展开）
>
> Consumer 与 Broker 建立连接后，主动 Pull（Fetch）消息：
> - 按自身消费能力控制拉取速率
> - 控制消费进度（offset）
> - 控制每次消费数量，实现批量消费

## 29. Kafka Consumer API

> [!question]- 参考答案（点击展开）
>
> | API | 特点 | 适用场景 |
> |-----|------|----------|
> | **Simple API** | 底层 API，维持与单个 Broker 的连接，完全无状态，每次需指定 offset | 需要完全控制消费逻辑 |
> | **High-level API** | 封装集群访问，自动维护消费状态，支持 Consumer Group | 大多数场景 |
>
> High-level API 中：
> - 相同 Group Name -> 队列模式，均衡消费
> - 不同 Group Name -> 广播模式，每组都收到全量消息

## 30. Partition 数据存储

> [!question]- 参考答案（点击展开）
>
> Topic 的多个 Partition 以**文件夹**形式保存到 Broker，每个分区序号从 0 递增。
>
> Partition 文件下包含多个 Segment（`xxx.index` + `xxx.log`）：
> - 默认单个 Segment 大小 1GB
> - 超过 1GB 时滚动新 Segment，以上一个 Segment 最后一条消息的 offset 命名

## 31. 分区放置策略

> [!question]- 参考答案（点击展开）
>
> Kafka 创建 Topic 时的分区放置规则：
>
> 1. 副本因子不能大于 Broker 个数
> 2. 第 0 个分区的第一个副本随机选择 Broker
> 3. 后续分区的第一个副本依次往后移位
> 4. 剩余副本位置由 `nextReplicaShift`（随机数）决定

## 32. 日志保留与清理策略

> [!question]- 参考答案（点击展开）
>
> **保留期**：默认 7 天，通过 `log.retention.hours/minutes/ms` 配置。
>
> **清理策略**：
>
> | 策略 | 配置 | 说明 |
> |------|------|------|
> | **删除** | `log.cleanup.policy=delete` | 默认策略，过期后标记删除，经过 `log.segment.delete.delay.ms` 后真正删除 |
> | **压缩** | `log.cleanup.policy=compact` | 只保留每个 Key 最后一个版本。需设置 `log.cleaner.enable=true` |

## 33. 日志 Message 格式

> [!question]- 参考答案（点击展开）
>
> 每个日志文件是 log entry 序列：
>
> 1. 4 字节整型：Message 长度（值为 1+4+N）
> 2. 1 字节 magic：协议版本号
> 3. 4 字节 CRC32：校验值
> 4. N 字节消息数据，每条消息有 Partition 下唯一的 64 位 offset
>
> > 推荐单条消息不超过 1MB，通常在 1~10KB 之间。

## 34. 多租户隔离

> [!question]- 参考答案（点击展开）
>
> 通过配置 Topic 级别的生产/消费权限实现多租户隔离。管理员可对请求定义和强制配额（Quota），控制客户端使用的 Broker 资源。

## 35. 日志分段与刷新策略

> [!question]- 参考答案（点击展开）
>
> ### 日志分段（Segment）
>
> | 参数 | 默认值 | 说明 |
> |------|--------|------|
> | `log.roll.hours` | 168h（7天） | 强制滚动新 Segment 的周期 |
> | `log.segment.bytes` | 1GB | 单个 Segment 最大容量 |
> | `log.retention.check.interval.ms` | 60000ms | 日志片段检查周期 |
>
> ### 日志刷新
>
> 日志先写入缓存，按策略批量刷盘以提升吞吐：
>
> | 参数 | 默认值 | 说明 |
> |------|--------|------|
> | `log.flush.interval.messages` | 10000 | 达到消息数时刷盘 |
> | `log.flush.interval.ms` | null | 达到时间时强制刷盘 |
> | `log.flush.scheduler.interval.ms` | 很大的值 | 周期性检查是否需要刷盘 |

## 36. 主从同步

> [!question]- 参考答案（点击展开）
>
> Kafka 通过 `producer.type` 配置同步或异步模式（默认同步）。
>
> ### 同步复制
>
> ```mermaid
> sequenceDiagram
>     participant P as Producer
>     participant L as Leader
>     participant F as Follower
>
>     P->>L: 发送消息
>     L->>L: 写入本地 log
>     F->>L: Pull 消息
>     F->>F: 写入本地 log
>     F->>L: ACK
>     L->>P: ACK（所有 Follower 确认后）
> ```
>
> ### 异步复制
>
> Producer 异步发送时，消息先放入 `BlockingQueue`，由 `ProducerSendThread` 线程从队列取出后调用同步发送接口。
>
> > 内存缓存消息批量发送可提升网络效率，但 Producer 不可用时缓存数据会丢失。

## 37. 消息丢失/不一致场景

> [!question]- 参考答案（点击展开）
>
> ### 发送端
>
> | acks 值 | 丢失风险 |
> |---------|----------|
> | `0` | 不确认，网络异常或缓冲区满时丢失 |
> | `1` | Leader 确认但 Follower 未同步完，Leader 宕机时丢失 |
> | `-1` | 最安全，ISR 全部确认 |
>
> ### 消费端
>
> - **High-level API**：自动提交 offset 后消息未处理完就崩溃，导致数据丢失
> - **Low-level API**：自行维护 offset，可完全控制
>
> 详细的防丢失配置参见 [[zero-message-loss]]。

## 38. Kafka 作为流处理平台

> [!question]- 参考答案（点击展开）
>
> 特点：
> 1. 轻量级 Java 类库，可集成到任何 Java 应用
> 2. 无外部依赖（除 Kafka 本身），利用分区模型支持水平扩容和顺序性
> 3. 支持本地状态容错，快速有效的有状态操作
> 4. 支持 exactly-once 语义
> 5. 支持逐条记录处理，ms 级延迟

## 39. 活锁问题

> [!question]- 参考答案（点击展开）
>
> **活锁**：Consumer 持续维持心跳，但不处理消息。
>
> **解决方案**：利用 `max.poll.interval.ms` 检测机制。若 poll 调用频率低于最大间隔，Consumer 主动离开 Consumer Group，分区被其他 Consumer 接管。

## 40. 保证顺序消费

> [!question]- 参考答案（点击展开）
>
> Kafka 消费单元是 Partition，同一 Partition 内通过 offset 保证有序。要保证 Topic 级别的全局有序：
>
> **方案**：发送时指定 Message Key，同一 Key 的消息路由到同一 Partition。
>
> ```java
> // 同一订单的消息保证有序
> producer.send(new ProducerRecord<>("order-topic", orderId, message));
> ```
