#borg
# Google Borg
## Borg 架构
Borgmaster 主进程:
- 处理客户端 RPC 请求，比如创建 Job，查询 Job 等。
- 维护系统组件和服务的状态，比如服务器、Task 等。
- 负责与 Borglet 通信。
Scheduler 进程:
- 调度策略
	- Worst Fit
	- Best Fit
	- Hybrid
- 调度优化
	- Score caching:当服务器或者任务的状态未发生变更或者变更很少时，直接采用缓存数据，避免重复计算,
	- Equivalence classes: 调度同- Job 下多个相同的 Task 只需计算一次。
	- Relaxed randomization: 引入一些随机性，即每次随机选择一些机器，只要符合需求的服务器数量达到一定值时，就可以停止计算，无需每次对 Cell 中所有服务器进行 feasibility checking。
Borglet :
	Borglet 是部署在所有服务器上的 Agent，负责接收 Borgmaster 进程的指令。
![[Borg架构.png]]
### Borgmaster
Borgmaster包含两类进程：主Borgmaster进程和分离的调度器进程。主Borgmaster进程处理客户端RPC；管理系统中所有对象Object的状态机，包括machines、tasks、allocs；与Borglet通信；提供webUI。  
Borgmaster逻辑上一个进程，但是拥有5个副本。每个Borgmaster副本维护cell状态的一份内存副本，cell状态同时在高可用、分布式、基于Paxos的存储系统中做本地磁盘持久化存储。一个单一的被选举的master既是Paxos leader，也是状态管理者。当cell启动或者被选举master挂掉时，系统会选举Borgmaster，选举机制按照**Paxos算法**流程进行。

> Borgmaster基于高可用、分布式、基于Paxos的存储系统进行元数据持久化和热Borgmaster备份，以此实现Borg系统的高可用。关于这个基于Paxos的存储系统，在Google内部应该就是Chubby，不过不知道为何这里没有提及，难道还有新的系统？开源界etcd最近比较火，但是etcd没有采用Paxos算法，而是使用更为简单且易于理解的**raft**。这里还是采用Paxos算法。

Borgmaster的状态会定时设置checkpoint，具体形式就是在Paxos store中存储周期性的镜像snapshot和增量更改日志。
### 调度Scheduling
当作业被提交，Borgmaster将其记录到Paxos store中，并将作业的任务增加到等待队列中。调度器异步浏览该队列，并将任务分配给机器。调度算法包括两个部分：可行性检查和打分。  
 可行性检查，用于找到满足任务约束、具备足够可用资源的一组机器；打分，则是在“可行机器”中根据用户偏好，为机器打分。用户偏好主要是系统内置的标准，例如挑选具有任务软件包的机器、分散任务到不同的失败域中（出于容错考虑）。  
Borg使用不同的策略进行打分。

> 实践中，E-PVN（worst fit）会将任务分散到不同的机器上；best fit，会尽量“紧凑”的使用机器，以减少资源碎片。Borg目前使用一种混合模型，尽量减少“受困资源”。
### Borglet
Borglet是运行在每台machine上的本地Borg代理，管理本地的任务和资源。Borgmaster会周期性地向每一个Borglet拉取当前状态。这样做更易于Borgmaster控制通信速度，避免“恢复风暴”。  
为了性能可扩展性，每个Borgmaster副本会运行一个无状态的link shard去处理与部分Borglet通信。当Borgmaster重新选举时，会重新分区。Borgmaster副本会聚合和压缩信息，仅仅向被选举master报告状态机的不同部分，以此减少更新负载。  
如果Borglet多轮没有响应资源查询，则会被标记为down。运行其上的任务会被重新调度到其他机器。如果恢复通信，则Borgmaster会通知Borglet杀死已经重新调度的任务，以此保证一致性。
## 可用性 Availability
（1）自动重新调度器被驱逐的任务；  
（2）为了降低相关失败，将任务分散到不同的失败域中；  
（3）限制一个作业中任务的个数和中断率；  
（4）限制任务重新调度的速率，因为不能区分大规模机器故障和网络分区；  
（5）避免引发错误的任务-机器匹配对；  
（6）关键数据持久化，写入磁盘。
## 利用率 Utilization
本节首先通过实验证明，混合部署（prod负载和non-prod负载）比独立部署具有更高的利用率。实验结果下图。
![[混合部署（prod负载和non-prod负载）.png]]
随后，结合实验说明，几种方法可以提高集群利用率，具体包括Cell sharing、Large cell、Fine-grained resource requests和Resource reclamation。前几种方法都比较直观，不做展开。Resource reclamation比较有意思，重点阐述。  
一个作业（job）可以定义一个资源上限（resource limit），资源上限用于Borg决定用户是否具有足够的资源配额（quota）来提交作业（job），并且用于决定是否具有足够的空闲资源来调度任务。因为Borg会kill掉一个尝试使用更多RAM和disk空间资源（相比于其申请的资源）的task，或者节流CPU资源（相比于其要求的），所以用户总是申请更多的资源（相比其实际所有的）。此外，一些任务偶尔会使用其所有资源，但大部分时间没有。
> 用户总是会处于“心理安全”和负载高峰波动等原因，申请较多的资源，但大部分时候，任务不会真正使用如此之多的资源。这就造成了资源浪费。

对于可以容忍低质量资源的工作（例如批处理作业），Borg会评估任务将使用的资源，并回收空闲资源。这个整个过程称为资源回收（resource reclamation）。评估过程称为任务预留（task reservation）。最初的预留值与其资源请求一致，然后300秒之后，会慢慢降低到实际使用率外加一个安全边缘。如果利用率超过资源预留值，预留值会快速增长。  
这里直观的表明实际使用资源、预留资源、资源上限的关系。
## 隔离 Isolation
安全性隔离:
- 早期采用 Chroot jail，后期版本基于 [[Namespace]]
性能隔离:
- 采用基于 [[Cgroup]] 的容器技术实现。
- 在线任务(prod)是延时敏感(latency-sensitive)型的，优先级高，而离线任务(non-prodBatch)优先级低。
- Borg 通过不同优先级之间的抢占式调度来优先保障在线任务的性能，牺牲离线任务
- Borg 将资源类型分成两类:
	- 可压榨的(compressible)，CPU 是可压资源，资源耗尽不会终止进程;
	- 不可压榨的(non-compressible)，内存是不可压资源，资源耗尽进程会被终止。
