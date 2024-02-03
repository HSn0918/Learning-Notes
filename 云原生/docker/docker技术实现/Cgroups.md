#Cgroups
# Cgroups
- cgroups(Control Groups)是 Linux 下用于对一个或一组进程进行资源控制和监控的机制;
- 可以对诸如 CPU 使用时间、内存、磁盘/等进程所需的资源进行限制;
- 不同资源的具体管理工作由相应的 Cgroup 子系统( Subsystem )来实现 ;
- 针对不同类型的资源限制，只要将限制策略在不同的的子系统上进行关联即可;
- Cgroups 在不同的系统资源管理子系统中以层级树( Hierarchy )的方式来组织管理:每个group 都可以包含其他的子 Cgroup，因此子 Cgroup 能使用的资源除了受本 Cgroup 配置的资源参数限制，还受到父Cgroup 设置的资源限制。
## 可配额/可度量 - Control Groups (cgroups)
![[可配额可度量 - Control Groups (cgroups).png]]
- blkio: 这个子系统设置限制每个块设备的输入输出控制。例如:磁盘，光盘以及 USB等
- cpu: 这个子系统使用调度程序为cgroup 任务提供CPU的访问
- cpuacct: 产生cgroup 任务的CPU 资源报告
- cpuset: 如果是多核心的 CPU，这个子系统会为 cgroup 任务分配单独的CPU和内存
- devices: 允许或拒绝cgroup 任务对设备的访问
- freezer: 暂停和恢复 cgroup任务
- memory: 设置每个cgroup的内存限制以及产生内存资源报告
- net_cls: 标记每个网络包以供cgroup 方便使用
- ns: 名称空间子系统
- pid: 进程标识子系统
## CPU子系统
- cpu.shares: 可出让的能获得CPU 使用时间的相对值
- cpu.cfs_period_us: cfs_period_us 用来配置时间周期长度，单位为 us(微秒)
- cpu.cfs_quota us: cfs quota us 用来配置当前 Cgroup 在 cfs period us 时间内最多能使用的 CPU 时间数，单位为 us(微秒)。
- cpu.stat: Cgroup 内的进程使用的 CPU 时间统计
- nr_periods: 经过cpu.cfs_period_us 的时间周期数量
- nr throttled: 在经过的周期内，有多少次因为进程在指定的时间周期内用光了配额时间而受到限制。
- throttled time: Cgroup中的进程被限制使用 CPU 的总用时，单位是ns(纳秒)。
## Linux调度器
>内核默认提供了5个调度器，Linux内核使用struct sched class来对调度器进行抽象:
-  Stop调度器，stop_sched_class:优先级最高的调度类，可以抢占其他所有进程，不能被其他进程抢占
- Deadline调度器，dl_sched_lass:使用红黑树，把进程按照绝对截止期限进行排序，选择最小进程进行调度运行;
- RT调度器，rt_sched_class:实时调度器，为每个优先级维护一个队列;
- ==CFS调度器==，cfs_sched_class: 完全公平调度器，采用完全公平调度算法，引入虚拟运行时间概念
- IDLE-Task调度器，idle_sched_class:空闲调度器，每个CPU都会有一个idle线程，当没有其他进程可以调度时，调度运行idle线程;
### CFS调度器
- CFS是Completely Fair Scheduler简称，即完全公平调度器
- CFS 实现的主要思想是维护为任务提供处理器时间方面的平衡，这意味着应给进程分配相当数量的处理器
- 分给某个任务的时间失去平衡时，应给失去平衡的任务分配时间，让其执行
- CFS通过虚拟运行时间(vruntime)来实现平衡，维护提供给某个任务的时间量
	- vruntime=实际运行时间*1024/进程权重
- 进程按照各自不同的速率在物理时钟节拍内前进，优先级高则权重大，其虚拟时钟比真实时钟跑得慢但获得比较多的运行时间
#### vruntime红黑树
CFS调度器没有将进程维护在运行队列中，而是维护了一个以虚拟运行时间为顺序的红黑树。红黑树的主要特点有:
1. 自平衡，树上没有一条路径会比其他路径长出俩倍
2. 0(log n)时间复杂度，能够在树上进行快速高效地插入或删除进程
## cpuacct 子系统
>用于统计 Cgroup 及其子 Cgroup 下进程的 CPU 的使用情况
- cpuacct.usage
>包含该 Cgroup 及其子 Cgroup 下进程使用 CPU 的时间，单位是 ns(纳秒)
- cpuacct.stat
>包含该 Cgroup及其子 Cgroup 下进程使用的 CPU 时间，以及用户态和内核态的时间.
## Memory 子系统
- memory.usage_in_bytes
	cgroup 下进程使用的内存，包含 cgroup 及其子cgroup 下的进程使用的内存
- memory.max_usage_in_bytes
	cgroup 下进程使用内存的最大值，包含子cgroups 的内存使用量
- memory.limit_in_bytes
	设置Cgroup 下进程最多能使用的内存。如果设置为 -1，表示对该 cgroup 的内存使用不做限制。
- memory.soft_limit in_bytes
	这个限制并不会阻止进程使用超过限额的内存，只是在系统内存足够时，会优先回收超过限额的内存，使之向限定值靠拢。
- memory.oom_control
	==设置是否在 Cgroup 中使用 OOM ( Out of Memory ) Killer，默认为使用。当属于该 group 的进程使用的内存超过最大的限定值时，会立刻被 OOM Killer 处理。==
## Cgroup driver
systemd:
- 当操作系统使用systemd作为init system时，初始化进程生成一个根group目录结构并作为cgroup管理器。
-  systemd与cgroup紧密结合，并且为每个systemd unit分配cgroup。
cgroupfs:
- docker默认用cgroupfs作为cgroup驱动
存在问题:
- 因此，在systemd作为init system的系统中，默认并存着两套groupdriver。
- 这会使的系统中docker和kubelet管理的 进程被cgroupfs驱动管，而systemd拉起的服务由systemd驱动管，让cgroup管理混乱且容易在资源紧张时引发问题。
==因此kubelet会默认--cgroup-driver=systemd，若运行时cgroup不一致时，kubelet会报错==

