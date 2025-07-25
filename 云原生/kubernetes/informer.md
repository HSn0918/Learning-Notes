## **概述**
informer 是 Kubernetes client-go 提供的 List+Watch 资源监听与本地缓存机制，用于实时感知集群中对象的增删改事件并触发回调。它内部包含 Reflector（负责与 API Server 通信）、DeltaFIFO（事件队列）、Store/Indexer（本地缓存）和 Controller（带 WorkQueue 的事件消费者）等模块协同工作。通过 SharedInformerFactory，可让多个 Controller 共享同一套缓存与 Watch 连接，显著提升性能并降低 API Server 压力。

------
## **1️⃣ 什么是 Informer？**
> informer 是 Kubernetes client-go 库中用于监听资源对象变化（Add/Update/Delete）的机制。
> 它会先通过 List 获取全部对象，再通过 Watch 持续获取增量更新，将所有数据同步到本地缓存，并支持注册事件处理函数（AddFunc、UpdateFunc、DeleteFunc）。
------
## **2️⃣ 它能解决什么问题？**
- **性能优化**
  - 初始化时一次性 List 全量对象，后续通过 Watch 获取增量，避免频繁的 API Server 请求
  - 所有读取操作均可从本地缓存完成，延迟更低、压力更小
- **事件驱动**
  - 支持注册回调函数，当资源变化时自动触发，替代手动轮询
  - 实现低延迟、反应式的控制器逻辑

------
## **3️⃣ 底层工作流程**
### **3.1 Reflector**
- **List**：拉取资源全量快照，写入 DeltaFIFO
- **Watch**：在 List 后保持长连接，接收新增/修改/删除事件，持续写入 DeltaFIFO
### **3.2 DeltaFIFO（变更队列）**
- 维护按时间顺序的事件（Delta）列表
- 保证事件有序交付，通过 Pop() 提供给 Controller
### **3.3 Store / Indexer（本地缓存）**
- 以 map（key=namespace/name，value=完整对象）存储资源
- 支持自定义索引（Label、Field）查询
- 用户可通过 Lister 在本地缓存中高效检索，无需访问 API Server
### **3.4 Controller（工作线程）**
1. 从 DeltaFIFO Pop() 获取事件对应的 key
2. 将 key 放入带速率限制和重试能力的 WorkQueue
3. 启动多个 Worker goroutine：
   - Get() 拉取 key
   - 通过 Indexer 获取最新对象
   - 执行业务逻辑（syncHandler）
   - Done()／重试／速率限流

------
## **4️⃣ 使用方式（结合项目）**

```
factory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)

// 构建 Pod informer
podInformer := factory.Core().V1().Pods().Informer()
podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc:    onAdd,
    UpdateFunc: onUpdate,
    DeleteFunc: onDelete,
})

stopCh := make(chan struct{})
factory.Start(stopCh)
factory.WaitForCacheSync(stopCh)
```

- **SharedInformerFactory**：

  多个 Controller 调用同一资源的 Informer()，可复用 List/Watch 连接和本地缓存

- **ResyncPeriod**：

  周期性触发本地缓存内所有对象的 Update 事件（可用于重试和最终一致性）

------
## **5️⃣ 优化设计（加分项）**

- **SharedInformerFactory 共享**

  多个 Controller 复用 Reflector、DeltaFIFO 和 Store，降低连接数和内存开销

- **自定义索引**

  在 Indexer 上添加 Label/Field 索引，加速特定维度的查询

- **速率限制 & 重试**

  WorkQueue 内置 RateLimiter，支持指数退避、限流，防止突发重试过载

------

## **🔧 Bonus：面试延展问题及回答思路**

| **追问**                          | **回答思路**                                                 |
| --------------------------------- | ------------------------------------------------------------ |
| **Resync 是什么？**               | 周期性地将本地缓存中的对象重新推回 DeltaFIFO，触发 Update 事件，不会再次调用 API Server，确保最终一致性与重试机会。 |
| **informer 与 lister 有何关系？** | Lister 是基于 informer 本地缓存（Indexer）提供的查询接口，用户通过 List()/Get() 从缓存获取对象，无需访问 API Server。 |
| **本地存储 key 还是对象？**       | Store 以 namespace/name 为 key，value 为完整对象（如 *v1.Pod）。 |
| **Controller 如何处理并发？**     | 使用带速率限制和重试的 WorkQueue，启动多个 Worker goroutine，保证同一 key 不被并发处理，并提供限速和重试策略。 |

> **总结**：informer 是 Kubernetes 控制器模式的基础组件，凭借 List+Watch、DeltaFIFO、Indexer 和 WorkQueue，实现高性能、低延迟且可靠的事件驱动机制；通过 SharedInformerFactory 和自定义索引，可进一步优化多控制器场景下的资源共享与查询效率。
