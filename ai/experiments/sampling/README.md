# Sampling: temperature、top-p 与 seed

相关笔记：[[llm-fundamentals]] | [[llm-inference-pipeline]]

## 目标

用固定的一组 logits 手写最小采样器，观察 `temperature` 如何改变分布尖锐程度、`top_p` 如何裁剪候选集合，以及固定 seed 如何令随机选择可复现。

## 环境

- Python 3.9+；仅 Python 标准库。
- Mac 或 Linux CPU 均可运行；不联网、不下载模型、不需要 GPU。

## 命令

从仓库根目录运行：

```bash
python3 ai/experiments/sampling/sampling_lab.py
```

## 预期现象

- `temperature=0.35` 的最大概率高于 `temperature=1.0`，分布更集中。
- `top_p=0.70` 的候选集合少于或等于完整分布，且累计概率达到阈值后不再保留低概率 token。
- 同一 `seed` 两次采样得到完全相同的 token 序列。
- 末尾出现 `SELF-CHECK: PASS`。

## 解释

实验中的 logits 是人为构造，不是模型输出。softmax 先将 logits 变为概率；温度越低，较大的 logit 被放大的相对优势越明显。nucleus (`top_p`) sampling 先按概率排序，仅在累计概率覆盖阈值的最小前缀内重新归一化后抽样。seed 只控制随机数流，不会让不同 logits 或不同采样参数得到相同结果。

## 自检

- [ ] 我能说明 `temperature` 不会修改模型权重。
- [ ] 我能解释 `top_p` 与 `top_k` 都是采样阶段的候选裁剪，但裁剪规则不同。
- [ ] 我知道固定 seed 保障的是同一实现、同一输入和同一参数下的可复现性。

## 验证状态

2026-08-13：已在本仓库 Python CPU 环境运行。该实验验证算法不变量，不是对任何模型服务随机数实现的兼容性测试。
