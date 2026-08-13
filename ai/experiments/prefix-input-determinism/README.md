# Prefix 输入确定性：canonical JSON、字段顺序与 hash

相关笔记：[[prefill-cache-miss]] | [[llm-inference-pipeline]]

## 目标

演示同一组业务字段因为 JSON 字段顺序不同而产生不同 bytes 与 SHA-256；再用 canonical JSON 让等价对象稳定编码。这里刻意只测 bytes/hash，**不是 tokenizer 或真实 Prefix Cache 命中率实验**。

## 环境

- Python 3.9+；仅 Python 标准库。
- Mac 或 Linux CPU 均可运行；不联网、不需要模型或 GPU。

## 命令

```bash
python3 ai/experiments/prefix-input-determinism/prefix_determinism.py
```

## 预期现象

- 两个键值相同、插入顺序不同的普通 JSON 字节串不同，hash 也不同。
- `sort_keys=True` 后 canonical JSON 字节串和 hash 相同。
- 末尾出现 `SELF-CHECK: PASS`。

## 解释

真实推理引擎按 tokenizer 产生的 Token IDs 与引擎 cache key 决定是否复用。本实验的 bytes/hash 仅是更低成本的前置代理：byte 不同必然不能称为“相同 Prompt”；byte 相同也不能单独证明某个服务会命中，因为 chat template、tokenizer、模型 revision、缓存设置和路由仍可不同。

## 自检

- [ ] 我能区分“对象语义相同”“JSON bytes 相同”和“Token IDs 相同”。
- [ ] 我能说明为什么把动态时间戳、随机 ID 放进公共前缀会破坏复用。
- [ ] 我不会把本实验的 hash 结果报告为 vLLM/SGLang 的 cache-hit 证据。

## 验证状态

2026-08-13：已在本仓库 Python CPU 环境运行。tokenizer 与 GPU 推理服务集成未在本实验中执行。
