# OpenAI-compatible inference benchmark

相关笔记：[[llm-inference-pipeline]] | [[vllm-and-sglang]] | [[llm-inference-learning-path]]

## 目标

对一个 OpenAI-compatible `/v1/chat/completions` 服务发送可控请求，记录流式首个 SSE 数据块延迟、端到端延迟、服务返回的 completion token 数，以及由它们推导的近似 TPOT 和吞吐。脚本可分别指向 vLLM 或 SGLang；它不试图比较未经控制的两套配置。

## 环境

- 本脚本：Python 3.9+、仅标准库。
- 被测服务：Linux + NVIDIA GPU、已启动的 vLLM 或 SGLang OpenAI-compatible server，且模型已可用。
- 建议先用 0.5B/0.6B 级别模型验证流程；真实所需显存取决于模型、精度、`max_model_len`、KV cache、并发和服务版本。

GPU 服务集成**未在本仓库验证**，不要把本 README 中的命令或脚本自测当成 vLLM/SGLang 的性能结论。

## 命令

先在 GPU 主机上按各项目当前官方文档启动服务。以下仅为示例，模型名、安装与参数必须以服务实际版本为准：

```bash
# terminal A: vLLM example; verify current vLLM documentation before use
vllm serve Qwen/Qwen3-0.6B --port 8000 --enable-prefix-caching

# terminal B: benchmark repeated-prefix workload
python3 ai/experiments/openai-compatible-benchmark/openai_compatible_benchmark.py \
  --base-url http://127.0.0.1:8000 \
  --model Qwen/Qwen3-0.6B \
  --requests 20 \
  --concurrency 1 \
  --prompt-kind repeated-prefix \
  --max-tokens 64
```

在另一台相同硬件、相同模型 revision、相同精度和相同请求集的 SGLang 服务上，仅改变 `--base-url` 再运行一次。不要同时运行两个服务抢占同一张 GPU。

无需 GPU 的脚本自测：

```bash
python3 ai/experiments/openai-compatible-benchmark/openai_compatible_benchmark.py --self-test
python3 ai/experiments/openai-compatible-benchmark/openai_compatible_benchmark.py --help
```

## 预期现象

- `--self-test` 输出 `SELF-CHECK: PASS`；它只校验本地统计逻辑，不会发 HTTP 请求。
- 真实流式运行输出 JSON Lines 记录和汇总：`ttft_ms`、`e2e_ms`、`approx_tpot_ms`、`completion_tokens`、`output_tokens_per_second`。
- 在服务实际开启 prefix cache 且请求字节前缀完全一致时，重复前缀请求**可能**降低 TTFT；它不是必然现象，需以服务 metrics 和请求记录共同验证。

## 解释与测量边界

- `ttft_ms` 计到第一个非 `[DONE]` SSE `data:` 事件。服务可能先发送 role/header chunk，因此这更准确地说是 first-stream-event latency；请结合服务版本与其 TTFT metrics 解释。
- HTTP 200 本身不算成功：响应中必须至少出现一个可解析、并含 `choices` 或 `usage` 的 JSON SSE 事件；否则该请求记录为失败。
- `approx_tpot_ms = (E2E - first event) / (completion_tokens - 1)`，只有服务在流尾返回 `usage.completion_tokens > 1` 时才输出。它是响应级近似，不用 SSE chunk 数冒充 token 数。
- 吞吐按服务返回 completion token 数除以本轮 wall-clock 计算。缺失 usage 时输出 `null`，而不是伪造 token 数据。
- cache hit、GPU 利用率、队列长度和精确 token-level ITL 不属于 OpenAI HTTP 契约；应额外抓服务 `/metrics`、GPU 工具和启动日志。

## 自检

- [ ] 两次对比固定了模型 revision、tokenizer/chat template、精度、GPU、`max_model_len`、服务参数、prompt bytes、预热和并发。
- [ ] 结果同时保存原始 JSONL、服务 `/metrics` 摘要、GPU 型号/driver 与命令行。
- [ ] 我没有用 non-streaming 请求的端到端延迟冒充 TTFT。
- [ ] 我知道“第二次更快”只能构成观察，不能单独证明 prefix cache 命中。

## 验证状态

2026-08-13：`--help` 与 `--self-test` 已在本仓库 Python CPU 环境运行。真实 vLLM/SGLang、CUDA、模型下载、GPU metrics 和性能结果均未验证。
