# 中间件 (Middleware) · 学习索引

中间件学习围绕跨进程与跨服务协作展开：消息怎样传递、状态怎样持久化、失败怎样恢复、数据怎样保持最终一致。本索引以 Kafka、NATS、Canal 和分布式事务为主线，逐步建立吞吐、可靠性与一致性的权衡能力。

> ⬆ 返回 [知识库首页](../README.md)

## 🧭 推荐学习顺序

**入门**：Kafka 基础概念与设计理念 → Kafka 面试知识点 → NATS 基础与核心概念 → Canal 数据同步中间件

**进阶**：生产者分区机制 → 消息压缩机制 → Kafka 集群参数配置 → NATS JetStream 持久化

**深入**：Kafka 零消息丢失保证 → 分布式事务方案（2PC/TCC/Saga）

## 📚 笔记清单

### Kafka

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Kafka 基础概念与设计理念](kafka/kafka-basics.md) | 讲 Kafka 的设计目标、Pub-Sub 模型与 Producer/Broker/Topic 核心架构，是入门 Kafka 的起点。 | 入门 |
| [Kafka 面试知识点](kafka/kafka-interview.md) | 汇总 Kafka 核心组件、Broker/Partition 分配关系与高性能原因（PageCache、顺序写）等高频面试题。 | 入门 |
| [Producer 分区机制](kafka/producer-partition.md) | 讲 Topic-Partition-Message 三级结构与轮询/Key 等分区策略，解决负载均衡与顺序保证问题。 | 进阶 |
| [Kafka 消息压缩机制](kafka/producer-compression.md) | 讲 Producer/Broker 端压缩、V1/V2 消息格式演进与压缩粒度，用于节省磁盘和带宽。 | 进阶 |
| [Kafka 零消息丢失保证](kafka/zero-message-loss.md) | 讲 committed 语义、acks 配置与 ISR 副本同步，剖析消息丢失场景及可靠投递的端到端保证。 | 深入 |
| [Kafka 集群参数配置](kafka/cluster-config.md) | 梳理 Broker/Producer/Consumer 三端关键参数（日志、副本、min.insync.replicas），用于稳定运维调优。 | 进阶 |

### NATS

| 笔记 | 简介 | 难度 |
|------|------|------|
| [NATS 基础与核心概念](nats/nats-basics.md) | 讲云原生轻量消息系统 NATS 的定位、与 Kafka 对比及 Subject 路由模型，适合微服务通信场景。 | 入门 |
| [NATS JetStream 持久化与流处理](nats/nats-jetstream.md) | 讲 JetStream 持久化层的 Stream/Consumer、消息回溯、Ack 语义与背压，补齐 Core NATS 的持久化能力。 | 进阶 |

### Canal

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Canal 数据同步中间件](canal/canal.md) | 讲 Canal 伪装 Slave 解析 MySQL Binlog 实现增量订阅，用于缓存刷新、索引构建与异构数据同步。 | 入门 |

### 分布式事务

| 笔记 | 简介 | 难度 |
|------|------|------|
| [分布式事务方案（2PC/TCC/Saga）](distributed-transaction/distributed-transaction.md) | 对比 2PC/XA、TCC、Saga 等分布式事务方案，剖析跨库一致性与可用性的权衡取舍。 | 深入 |

---
共 **10** 篇 · 入门 4 / 进阶 4 / 深入 2
