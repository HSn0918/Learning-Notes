#kubernetes
# kubernetes
## [[Google Borg]]
|                                           Workload                                           | Cell                                                                                                                                                                      |                                                           Job 和 Task                                                           |                                                           Naming                                                            |
|:--------------------------------------------------------------------------------------------:| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |:-------------------------------------------------------------------------------------------------------------------------------:|:---------------------------------------------------------------------------------------------------------------------------:|
| prod:在线任务，长期运行、对延时敏感、面向终端用户等，比如GmailGoogle Docs,WebSearch 服务等。 | 一个 Cell 上跑一个集群管理系统 Borg。                                                                                                                                     | 用户以 Job 的形式提交应用部署请求。一个 Job 包含一个或多个相同的 Task，每个Task 运行相同的应用程序，Task 数量就是应用的副本数。 |                                      Borg的服务发现通过BNS（Borg Name Service）来实现                                       |
|            non-prod:离线任务，也称为批处理任务(Batch)，比如一些分布式计算服务等。            | 通过定义 Cell 可以让Borg 对服务器资源进行统一抽象，作为用户就无需知道自己的应用跑 在 哪台机器上，也不用关心资源分配、程序安装、依佢甲憐轺圆鎧骗管理、健康检查故障恢复等。 |                                  每个 Job 可以定义属性、元信息和优先级，涉及到抢占式调度过程。                                  | 50.jfoo.ubar.cc.borggoogle.com 可表示在一个名为 cc 的 Cell中由用户 uBar 部署名为 jFoo 的 Job下的第50个 Task。 |
## 什么是kubernetes
Kubernetes 是谷歌开源的容器集群管理系统，是 Google 多年大规模容器管理技术 Borg 的开源版本，主要功能包括:
- 基于容器的应用部署、维护和滚动升级;
- 负载均衡和服务发现;
- 跨机器和跨地区的集群调度;
- 自动伸缩;
- 无状态服务和有状态服务;
- 插件机制保证扩展性。

![[kubernetes架构.excalidraw|1000*900]]
![[Kubernetes架构具体.png|750*]]

![[Kubernetes 集群.png]]
## Kubernetes:声明式系统
Kubernetes 的所有管理能力构建在对象抽象的基础上，核心对象包括:
- Node:计算节点的抽象，用来描述计算节点的资源抽象、健康状态等
- [[Namespace]]:资源隔离的基本单位，可以简单理解为文件系统中的目录结构
- Pod:用来描述应用实例，包括镜像地址、资源需求等。Kubernetes 中最核心的对象，也是打通应用和基础架构的秘密武器
- Service:服务如何将应用发布成服务，本质上是负载均衡和域名服务的声明
## [[etcd]]
etcd 是 CoreOs 基于 Raft 开发的分布式 key-value 存储，可用于服务发现、共享配置以及一致性保障(如数据库选主、分布式锁等)。
- 基本的 key-value 存储;
- watch监听机制;
- key 的过期及续约机制，用于监控和服务发现;
- 原子 CAS 和 CAD，用于分布式锁和 leader 选举,
## APIServer
Kube-APIServer是Kubernetes 最重要的核心组件之一，主要提供以下功能：
- 提供集群管理的 REST API接口，包括:
	- 认证 Authentication;
	- 授权 Authorization;
	- 准入Admission(Mutating & Valiating)。
-  提供其他模块之间的数据交互和通信的枢纽(其他模块通过 APIServer 查询或修改数据，只有 APIServer 才直接操作 etcd)。
- APIServer 提供 etcd 数据缓存以减少集群对 etcd 的访问。
![[APIServer展开.png]]
## Controller Manager
- Controller Manager 是集群的大脑，是确保整个集群动起来的关键
- 作用是确保 Kubernetes 遵循声明式系统规范，确保系统的真实状态(Actual State)与用户定义的期望状态(Desired State)一致;
- Controller Manager 是多个控制器的组合，每个Controler 事实上都是一个 controlloop，负责侦听其管控的对象，当对象发生变更时完成配置;
- Controller 配置失败通常会触发自动重试，整个集群会在控制器不断重试的机制下确保最终一致性( Eventual Consistency)。
![[控制器流程.png]]
## Informer的内部机制
![[Informer的内部机制.png]]
 ## 控制器协同工作
![[协同器工作原理.png]]
查看文件的hash值和原来比对不一样，控制器就认为改变了
### Scheduler
特殊的 Controller，工作原理与其他控制器无差别。
Scheduler 的特殊职责在于监控当前集群所有未调度的 Pod，并且获取当前集群所有节点的健康状况和资源使用情况，为待调度 Pod 选择最佳计算节点，完成调度。
调度阶段分为:
- Predict:过滤不能满足业务需求的节点，如资源不足、端口冲突等。
- Priority:按既定要素将满足调度需求的节点评分，选择最佳节点。
- Bind:将计算节点与 Pod 绑定，完成调度。
## Kubelet
Kubernetes 的初始化系统(init system)
- 从不同源获取 Pod 清单，并按需求启停 Pod 的核心组件:
	- Pod 清单可从本地文件目录，给定的 HTTPServer 或 Kube-APIServer 等源头获取
	- Kubelet 将运行时，网络和存储抽象成了CRI，CNI，CSI
- 负责汇报当前节点的资源信息和健康状态
- 负责 Pod 的健康检查和状态汇报
## Kube-Proxy
- 监控集群中用户发布的服务，并完成负载均衡配置。
- 每个节点的 Kube-Proxy 都会配置相同的负载均衡策略，使得整个集群的服务发现建立在分布式负载均衡器之上，服务调用无需经过额外的网络跳转(Network Hop)。
- 负载均衡配置基于不同插件实现:
	- userspace
	- 操作系统网络协议栈不同的 Hooks 点和插件
		- iptables ;
		- ipvs.
## 推荐的 Add-ons
- kube-dns:负责为整个集群提供 DNS 服务;
- Ingress Controller:为服务提供外网入口;
- MetricsServer:提供资源监控；
- Dashboard:提供 GUI;
- Fluentd-Elasticsearch:提供集群日志采集、存储与査询。
## Kubectl 命令和 kubeconfig
- kubectl 是一个 Kubernetes 的命令行工具，它允许Kubernetes 用户以命令行的方式与 Kubernetes 交互，其默认读取配置文件 ~/.kube/config。
- kubectl 会将接收到的用户请求转化为rest 调用以rest client 的形式与 apiserver 通讯。
- apiserver 的地址，用户信息等配置在 kubeconfig。
### get
- -oyaml
- -w 
- -owide
### descibe
### exec
- -it <name> bash 
### logs
- -f 
