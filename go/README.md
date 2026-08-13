# Go 语言学习索引

本目录从语言语义与常用机制逐步进入 Runtime 源码。基础笔记回答“怎么用、为什么”，`internals/` 负责当前版本的结构、热路径、版本边界和排障。

> ⬆ 返回 [知识库首页](../README.md)

## 推荐学习顺序

**入门**：Slice 底层与扩容 → Map 哈希表实现 → Interface 与反射 → Go 版本新特性

**进阶**：Context 控制与取消传播 → Channel 底层实现 → GMP 调度模型

**深入**：P.runnext 调度优化 → 并发三色标记 GC

## 笔记清单

### 数据结构

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Slice 底层实现与扩容](slice.md) | 讲 slice 三字段结构（指针/len/cap）、共享底层数组与 append 扩容策略带来的陷阱 | 入门 |
| [Map 哈希表底层实现](map-internals.md) | 讲 hmap/bmap 结构、拉链法解决冲突与渐进式扩容，避免一次性搬迁的性能抖动 | 进阶 |
| [Interface 与 reflect](interface.md) | 讲 iface/eface 与 itab 结构、动态分发与类型断言，以及反射的底层原理 | 进阶 |

### 语言特性

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Go 版本新特性演进](go-versions.md) | 梳理 Go 1.18 泛型到 1.23 迭代器的关键特性，覆盖 GOMEMLIMIT、PGO、loop 变量修复等 | 入门 |

### 并发

| 笔记 | 简介 | 难度 |
|------|------|------|
| [Context 取消与传播](context.md) | 讲 Context 树形结构、取消信号与超时控制，用于管理 goroutine 生命周期与请求作用域数据 | 进阶 |
| [Channel 底层实现](channel.md) | 讲 hchan 结构体、环形缓冲区与 recvq/sendq 等待队列，剖析收发阻塞与唤醒机制 | 进阶 |

### 运行时调度

| 笔记 | 简介 | 难度 |
|------|------|------|
| [GMP 调度模型](gmp-model.md) | 讲 G/M/P 三实体的 M:N 调度方案、本地/全局队列与 work-stealing，是 Go 并发的核心 | 深入 |
| [P.runnext 调度优化](p-runnext.md) | 讲 P.runnext 优先级高于本地/全局队列的机制，及其对 goroutine 执行顺序的影响 | 深入 |

### 内存与GC

| 笔记 | 简介 | 难度 |
|------|------|------|
| [并发三色标记 GC](gc.md) | 讲三色标记清除算法、写屏障保证标记正确性，及如何减少 STW 停顿时间 | 深入 |

## 源码深入

完成上述基础后进入 [Go Runtime 源码学习索引](internals/README.md)，按 GMP → Channel/Context → Slice/Map → GC → 新旧实现对比推进。
