# Learning Notes 项目说明

本仓库按知识领域组织个人技术学习笔记，兼容 Obsidian 与 GitHub。

## 稳定目录职责

```text
ai/
  foundations/      LLM 基础与零基础路线
  inference/        推理链路、vLLM、SGLang 与进阶路线
  post-training/    LoRA、SFT、DPO、RLHF、PPO、GRPO
  agents/           Agent 路线、源码与生产化

cloud-native/
  docker/           容器、镜像、隔离与网络
  kubernetes/
    foundations/    Kubernetes 全局基础
    infrastructure/ etcd 等基础设施
    control-plane/  API、RBAC、Informer
    node/           Probe、OCI、节点运行时
    scheduling/     Scheduler 与 GPU 调度
    networking/     CNI、Service、网络实现与排障
    storage/        CSI、Volume、驱动与排障
    extension/      Kubebuilder、Operator、Velero
    components/     组件职责与故障边界
    internals/      源码调用链
    roadmaps/       学习路线与进度
    demos/          可运行代码
    interview/      复习题库

go/
  internals/        Runtime 与标准库源码导读
```

`algorithm/`、`database/`、`middleware/` 和 `misc/` 继续按各自领域 MOC 管理。

## 维护规则

- 详细写作、链接、MOC 和验证规则以 `AGENTS.md` 为准。
- 架构决策与文档类型边界见 `docs/architecture.md`。
- 不在本文件维护精确篇数或逐文件清单。
- 结构验证命令：`python3 scripts/validate_notes.py`。
