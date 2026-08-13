# AI Experiments

> 返回 [AI 学习索引](../README.md)

本目录把概念页中的关键机制落为小型、可检查的实验。除推理服务基准外，所有实验均只使用 Python 标准库，不需要网络、模型权重或 GPU。

| 实验 | 主题 | 环境 | 验证状态 |
| --- | --- | --- | --- |
| [Sampling](sampling/README.md) | `temperature`、`top_p` 与可复现采样 | Mac / Linux CPU | 已在本仓库 CPU 环境运行 |
| [Prefix input determinism](prefix-input-determinism/README.md) | canonical JSON、字段顺序与输入 hash | Mac / Linux CPU | 已在本仓库 CPU 环境运行 |
| [LoRA parameter count](lora-parameter-count/README.md) | 全量投影与 LoRA adapter 参数量 | Mac / Linux CPU | 已在本仓库 CPU 环境运行 |
| [Bandit policy gradient](bandit-policy-gradient/README.md) | REINFORCE 的 reward-driven policy update | Mac / Linux CPU | 已在本仓库 CPU 环境运行 |
| [OpenAI-compatible benchmark](openai-compatible-benchmark/README.md) | vLLM / SGLang TTFT、TPOT、吞吐测量 | Linux + NVIDIA GPU | 脚本自测通过；真实服务集成未验证 |

## 统一约定

- 每个目录的 `README.md` 说明目标、环境、命令、预期现象、解释、自检与验证状态。
- CPU 实验使用固定 seed，重点验证机制和不变量，不把输出中的某个浮点数当作跨 Python 版本的永久基准。
- 推理基准不预填性能数字。真实结果必须同时记录模型 revision、硬件、服务参数、prompt 集、预热、并发和时间戳。

## 一次运行全部 CPU 实验

```bash
python3 ai/experiments/sampling/sampling_lab.py
python3 ai/experiments/prefix-input-determinism/prefix_determinism.py
python3 ai/experiments/lora-parameter-count/lora_parameter_count.py
python3 ai/experiments/bandit-policy-gradient/bandit_policy_gradient.py
```

预期每个命令末尾均输出 `SELF-CHECK: PASS`。
