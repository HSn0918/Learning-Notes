# Learning Notes

技术学习笔记，基于 Obsidian 构建的个人知识库。

---

## Learning Plan

- [[k8s-development-roadmap]] - Kubernetes 开发学习路线图（12 主题 + 阶段计划）
- [[hami-learning-path]] - HAMi（GPU 虚拟化中间件）由浅入深学习路径
- [[learn-k8s-via-hami]] - 以 HAMi 为线索一步步学透 K8s 的 12 个核心机制（Mac 友好）
- [[progress]] - 6 周高阶学习打卡表（Scheduler → Device Plugin → CRI → CSI → etcd → CNI）
- **源码导读**（基于本地 `~/github/kubernetes` 源码，含真实行号引用 + 手写简化复现）：[[client-go-source]] | [[controller-runtime-source]] | [[scheduler-framework-source]] | [[kubelet-cri-source]] | [[etcd-source]] | [[csi-source]] | [[cni-source]] | [[gpu-scheduling-source]] | [[hami-source]]
- **可运行 demo**：[[demo-sample-controller]] | [[demo-kubebuilder-operator]] | [[demo-scheduler-plugin]] | [[demo-device-plugin]] | [[demo-raftexample-walkthrough]] | [[demo-csi-hostpath]] | [[demo-cni-bridge]] | [[demo-fake-gpu]] | [[demo-hami-mac]]（Mac 无 GPU 跑 HAMi 关键链路 + 50 行 C 的 libvgpu 简化 hook）

---

## Cloud Native

- **Docker**
  - [[docker-basics]] - Docker 容器技术基础
  - [[dockerfile]] | [[docker-commands]] | [[buildah-large-image]]
  - 底层技术：[[cgroup]] | [[namespace]] | [[union-fs]]
  - 网络：[[network-bridge]] | [[network-underlay]] | [[network-null]]

- **Kubernetes**
  - Infrastructure：[[kubernetes-basics]] | [[etcd]] | [[google-borg]] | [[oci-runtime]]
  - Control Plane：[[api-resource]] | [[informer]] | [[restful-api-design]] | [[rbac]] | [[scheduler-assume]] | [[scheduler-deep-dive]] | [[gpu-scheduling]]
  - Networking：[[cni]] | [[calico]] | [[cilium]] | [[flannel]] | [[weave]] | [[multus]] | [[service]] | [[headless-service]] | [[network-model]]
  - Storage：[[csi]] | [[ceph-csi]] | [[longhorn]] | [[openebs]] | [[nfs-csi]] | [[cloud-provider-csi]]
  - Extension：[[kubebuilder]] | [[velero]] | [[operator-pattern]]
  - Interview：[[k8s-interview]]

## Database

- **MySQL**：[[mysql-index]] | [[mysql-engine]]
- **Redis**：[[redis-data-types]]
- **Elasticsearch**：[[elasticsearch-basics]] | [[es-field-types]] | [[es-kibana-docker]]

## Middleware

- **Kafka**
  - [[kafka-basics]] - 架构设计与核心概念
  - [[kafka-interview]] | [[producer-partition]] | [[producer-compression]]
  - [[zero-message-loss]] | [[cluster-config]]
- **NATS**：[[nats-basics]] | [[nats-jetstream]]
- **Canal**：[[canal]] - MySQL Binlog 增量同步
- **Distributed Transaction**：[[distributed-transaction]] - 2PC、DTM 框架

## Go

- [[gmp-model]] - GMP 调度模型
- [[gc]] - 垃圾回收与三色标记法
- [[context]] - Context 上下文管理
- [[map-internals]] - Map 哈希表底层实现
- [[p-runnext]] - P.runnext 机制

## Algorithm

- **Search**：[[bfs]] | [[dfs]] | [[backtracking]] | [[binary-search]] | [[lca]]
- **Sorting**：[[sorting-overview]] | [[heap-sort]] | [[quick-sort]] | [[radix-sort]] | [[merge-sort]]
- **Techniques**：[[prefix-sum]] | [[diff-array]] | [[sliding-window]] | [[monotonic-stack]] | [[xor]] | [[bitwise-and]]
- **DP**：[[house-robber]] | [[unbounded-knapsack]]
- **Data Structures**：[[lru]] | [[lfu]] | [[linked-list]]

## AI

- [[llm-inference-pipeline]] - 大模型推理全流程
- [[prefill-cache-miss]] - Prefill 缓存未命中分析

## Misc

- [[zsh]] - 终端快捷键
