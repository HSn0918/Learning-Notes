>设计一个系统，精准的目标是第一步。Kafka 官方在最开始的时候，对 Kafka 的设计理想是将其做成一个可以帮助大型公司应对各种可能的实时数据流处理的通用平台。这句话里边有几个重点：“大型公司”、“实时”、“通用”，对应到系统设计上，就是需要支持大量数据的低延迟处理，并且需要考虑各种不同的数据处理场景。在官方阐述中，Kafka 着眼于以下几个核心指标：
- **高吞吐量**：因为 Kafka 需要处理大量的消息；
- **低延迟**：消息系统的关键设计指标；
- **支持加载离线数据**：这是 Kafka 考虑的所谓“各种可能的”数据处理场景，支持从离线系统中加载数据，或者将数据加载到离线系统中，都是无法逃避的；
- **支持分区的、分布式的、实时的数据流处理以产生新的、派生的数据流**：这个指导了 Kafka 里 topic 分区模型以及消费者模型的设计；
- **容错与可靠性**：Kafka 作为消息中间件，核心场景之一就是作为系统间的连接器，需要保证整体业务的正常运作，可靠的消息投递机制以及应对节点故障的高可用设计等，必不可少。
## 发布-订阅模型
Kafka消息模型
发布订阅模型（Pub-Sub） 使用**主题（Topic）** 作为消息通信载体，类似于**广播模式**；发布者发布一条消息，该消息通过主题传递给所有的订阅者，**在一条消息广播之后才订阅的用户则是收不到该条消息的**。
**在发布 - 订阅模型中，如果只有一个订阅者，那它和队列模型就基本是一样的了。所以说，发布 - 订阅模型在功能层面上是可以兼容队列模型的。**
## 核心概念
Kafka 将生产者发布的消息发送到 **Topic（主题）** 中，需要这些消息的消费者可以订阅这些 **Topic（主题）**，如下图所示：
![[kafka消费-订阅模型.png]]

上面这张图也为我们引出了，Kafka 比较重要的几个概念：

1. **Producer（生产者）** : 产生消息的一方。
2. **Consumer（消费者）** : 消费消息的一方。
3. **Broker（代理）** : 可以看作是一个独立的 Kafka 实例。多个 Kafka Broker 组成一个 Kafka Cluster。

同时，你一定也注意到每个 Broker 中又包含了 Topic 以及 Partition 这两个重要的概念：

- **Topic（主题）** : Producer 将消息发送到特定的主题，Consumer 通过订阅特定的 Topic(主题) 来消费消息。
- **Partition（分区）** : Partition 属于 Topic 的一部分。一个 Topic 可以有多个 Partition ，并且同一 Topic 下的 Partition 可以分布在不同的 Broker 上，这也就表明一个 Topic 可以横跨多个 Broker 。这正如我上面所画的图一样。
## Kafka 如何保证消息的消费顺序？
