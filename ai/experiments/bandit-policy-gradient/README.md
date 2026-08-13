# Two-armed bandit policy gradient

相关笔记：[[llm-rl-fundamentals]] | [[llm-post-training]]

## 目标

在两个 Bernoulli 奖励臂上实现最小 REINFORCE：policy 根据 logits 采样 action，环境给 reward，更新后的 policy 逐渐偏向期望 reward 更高的 arm。

## 环境

- Python 3.9+；仅 Python 标准库。
- Mac 或 Linux CPU 均可运行；不联网、不需要 GPU。

## 命令

```bash
python3 ai/experiments/bandit-policy-gradient/bandit_policy_gradient.py
```

## 预期现象

- 初始 policy 近似均匀。
- 训练后，reward 概率为 0.80 的 arm 概率明显高于 reward 概率为 0.20 的 arm。
- 输出训练前后概率与最近窗口平均 reward；末尾出现 `SELF-CHECK: PASS`。

## 解释

这里的 action 是选 arm，reward 是模拟的 0/1 回报。更新使用 `reward - baseline` 作为 advantage，令取得高于基线奖励的 action 概率上升。它对应 RL for LLM 的最小骨架，但并不包含语言模型、rollout 长序列、KL 约束、价值网络、PPO/GRPO、多样本 group advantage 或真实任务验证器。

## 自检

- [ ] 我能将 `arm → action`、Bernoulli 结果 → reward、logits → policy 对应到 LLM RL 术语。
- [ ] 我能说明 baseline 主要是降低方差，不是环境的真实奖励。
- [ ] 我不会把此收敛曲线当作 PPO/GRPO 或真实模型训练结论。

## 验证状态

2026-08-13：已在本仓库 Python CPU 环境运行。随机环境使用固定 seed；不同超参数和随机 seed 的收敛路径会不同。
