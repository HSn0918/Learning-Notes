# Learning Notes - Claude Code 项目指令

## 项目概述
这是一个面向 Kubernetes/云原生岗位求职的技术学习笔记仓库，涵盖云原生、Go、数据库、中间件、算法等领域，目标是系统梳理面试知识点。

## 笔记规范

### 文件命名
- 使用英文小写 + 连字符作为文件名（如 `mysql-index.md`, `gmp-model.md`）
- 目录名使用英文小写 + 连字符（如 `cloud-native/`, `data-structures/`）
- 每个主题目录下的图片统一放在 `图片/` 子目录中

### 语言风格
- 中英混合：技术术语、组件名、命令用英文，解释说明用中文
- 代码注释可以中文

### 标签约定
- 每篇笔记开头使用 `#标签` 标记所属领域
- 常用标签：`#kubernetes` `#docker` `#go` `#算法` `#mysql` `#redis` `#kafka` `#elasticsearch` `#postgresql` `#cni` `#csi` `#gpu` `#ai-infra`

### 双向链接
- 新建笔记时，必须在开头添加 `相关笔记：[[...]]` 链接到相关主题
- 更新现有笔记时，检查是否有新的关联笔记需要链接
- 使用 `[[文件名]]` 格式，不需要写完整路径

### 内容结构
- 每篇笔记应有清晰的标题层级（## 主标题，### 子标题）
- 核心机制必须有 mermaid 图（架构图、流程图、时序图）
- 代码示例使用 ` ```language ` 格式包裹
- 图片使用 `![[图片名.png]]` 格式引用
- 每篇笔记末尾添加 `## 面试要点` 章节，列出 5-10 个高频问答

## 编辑约定
- 不删除有实质内容的笔记（超过 5 行有意义内容即视为有实质内容）
- 修改笔记后检查 wikilink 是否仍然有效
- 合并笔记时保留两篇笔记中不重复的内容
- README.md 作为根 MOC（知识地图 + 求职学习路线）；每个领域目录下另有 `README.md` 学习索引（推荐顺序 + 难度 + 一句话简介）
- 新增笔记后需同步更新：① 所在领域的 `领域/README.md` 索引，② 根 README.md（涉及新领域或重点笔记时）
- CLAUDE.md 目录结构部分需与实际目录保持同步

## 目录结构
```
learning-plan/      - Kubernetes 开发学习路线图与源码导读（k8s-development-roadmap 为入口）
  source/           - 13 篇源码导读：client-go / controller-runtime / scheduler-framework / scheduler-podgroup / volcano / kubelet-cri / cri / etcd / csi / cni / gpu-scheduling / hami / agent-development（+ hami-learning-path）
  topics/           - 组件拆解 + 研发深挖专题：components/ 下按组件拆分 kube-apiserver / etcd / scheduler / controller-manager / kubelet / kube-proxy / CNI / CSI / addon；networking/storage/agent 下保留排障与专题深挖
  interviews/       - 面试纪要模块化沉淀：K8s Operator、GPU 调度、Device Plugin、系统设计追问
  demos/            - 10 个可运行 demo：sample-controller / kubebuilder-operator / scheduler-plugin / device-plugin / fake-cri / csi-hostpath / cni-bridge / fake-gpu / hami-mac / raftexample-walkthrough
  llm-inference-learning-path.md / llm-inference-progress.md - LLM 推理（Router/KV Cache/PD 分离）8 阶段路径与打卡表
  agent-development-learning-path.md - Agent 开发学习路径：RAG / ReAct / Planning / Graph Workflow / OpenTelemetry / 工具链选型

cloud-native/
  docker/           - 容器基础、Dockerfile、网络模式、底层技术（Cgroup/Namespace/UnionFS）
  kubernetes/
    control-plane/  - APIServer、Informer、调度器、RBAC、GPU 调度
    networking/     - CNI 概览、Calico、Cilium、Flannel、Weave、Multus、Service、网络模型
    storage/        - CSI 概览、Ceph、Longhorn、OpenEBS、NFS、云厂商 CSI
    infrastructure/ - K8s 基础、etcd、Google Borg、OCI Runtime
    extension/      - Kubebuilder、Velero、Operator 模式
    interview/      - K8s 面试题汇总

database/
  mysql/            - 索引、存储引擎、事务/MVCC、锁机制
  redis/            - 数据类型、持久化、集群/哨兵、缓存问题
  elasticsearch/    - 基础、字段类型、Docker 部署
  postgresql/       - 基础架构/MVCC、高级特性/锁/复制/连接池

middleware/
  kafka/            - 架构、面试题、生产者、零消息丢失、集群配置
  nats/             - NATS 消息系统、JetStream 持久化
  canal/            - MySQL Binlog 同步
  distributed-transaction/ - 2PC、TCC、Saga、DTM

go/                 - GMP、GC、Context、Map、Channel、Interface、Slice、版本特性

algorithm/
  search/           - BFS、DFS、回溯、二分、LCA、图论、Dijkstra、拓扑排序、并查集
  sorting/          - 排序总结、堆排序、快排、基数、归并
  techniques/       - 前缀和、差分、滑动窗口、单调栈、位运算
  dp/               - 打家劫舍、完全背包
  data-structures/  - LRU、LFU、链表题
  string/           - KMP、Trie

ai/                 - LLM 推理全流程、Prefill 缓存分析

misc/               - zsh 快捷键
```
