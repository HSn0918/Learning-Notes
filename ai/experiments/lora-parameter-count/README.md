# LoRA 参数量对比

相关笔记：[[lora-and-qlora]]

## 目标

只用矩阵维度计算，比较全量训练一个 `d_out × d_in` 投影与 LoRA 的低秩增量 `B × A` 需要训练的参数量；再扩展到多层、多投影的简化 Transformer 估算。

## 环境

- Python 3.9+；仅 Python 标准库。
- Mac 或 Linux CPU 均可运行；不联网、不下载权重、不需要 GPU。

## 命令

```bash
python3 ai/experiments/lora-parameter-count/lora_parameter_count.py
```

## 预期现象

- 对 `1024 × 1024` 投影，rank 16 的 LoRA 参数量为 `2 × 1024 × 16`，远小于全量矩阵。
- rank 上升时 LoRA 参数量线性上升。只有满足 `r(d_in + d_out) < d_in × d_out` 时，LoRA 的可训练参数才少于全量矩阵；对于 `d × d` 方阵，条件是 `r < d / 2`。
- 末尾出现 `SELF-CHECK: PASS`。

## 解释

LoRA 用 `W + BA` 表示更新：冻结原始 `W`，训练形状为 `d_out × r` 和 `r × d_in` 的两个矩阵。LoRA 参数量为 `r(d_in + d_out)`，全量矩阵参数量为 `d_in × d_out`；“低秩”描述矩阵分解结构，不自动保证参数更少。这个实验只计算**投影权重**的可训练参数，不含 embedding、LayerNorm、bias、optimizer state、activation、量化存储或通信开销；因此不能用作显存或训练时间的精确预测。

## 自检

- [ ] 我能写出全量投影与 LoRA adapter 的参数量公式。
- [ ] 我能解释为什么 rank 是效果、可训练参数和成本之间的旋钮。
- [ ] 我知道 QLoRA 的 4-bit 权重存储不等于所有训练状态都是 4-bit。

## 验证状态

2026-08-13：已在本仓库 Python CPU 环境运行。这是数学规模估算，未加载或训练真实模型。
