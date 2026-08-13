#ai #llm #inference #学习计划 #打卡

相关笔记：[[llm-inference-learning-path]] | [[llm-inference-pipeline]] | [[prefill-cache-miss]] | [[progress]]

# LLM 推理学习进度

## 用法

- 每完成一项把 `[ ]` 改成 `[x]`
- 每周末在「周末复盘」处写一两句话：跑了什么 benchmark、卡在哪、读了多少行源码
- 每周必须留下三个可检查产出：一张架构/时序图、一次 demo 或 benchmark 记录、一次 5 分钟口述答案
- 卡住的题目记在最底部「待解决问题」，每周回顾一次

## 关于「6-8 周」——先读这段，别被节奏绑架

下面的「Week 1 ~ Week 8」是**逻辑顺序**，不是**日历周**。LLM 推理这条线的源码总量比 K8s 大、而且要看论文，节奏比 [[progress]] 更松一些：

| 你的情况 | 实际预期 | 每个「Week」实际花 |
| :--- | :--- | :--- |
| 全职脱产 4-6h/天 | 6-8 周 | 约 1 周 |
| 在职 1.5-2h/天 + 周末 | 12-16 周 | 约 2 周 |
| 在职碎片时间 | 5-7 个月 | 视情况 |

**三个偏紧的点要特别注意**：
- **Week 2-3（vLLM 内核）**：PagedAttention + Scheduler 是 vLLM 最核心也最复杂的部分，新手吃透要 2 周。「1 周读完 vLLM 源码」是不现实的——能讲清楚 BlockManager 三状态机和 prefill/decode 怎么混在 batch 里就算达标。
- **Week 5-6（分布式 KV + PD 分离）**：要读 RDMA 相关代码 + 三篇论文，新手 2-3 周不算多。
- **Week 7（平台层）**：方案多（GIE / Production Stack / llm-d / AIBrix / Dynamo），别贪多，**选其中 2 个深挖**就够。

某周没按时完成 = 投入估得太乐观，往后顺延，别因此否定自己。

起点日期：____________  目标完成日期（按上表换算后填）：____________

前置假设：✅ 有 GPU 资源（自有/云租，至少能跑 7B 模型） ✅ 已读 [[llm-inference-pipeline]] 和 [[prefill-cache-miss]]

---

## Week 1：跑通 vLLM + SGLang，看到 Prefix Cache 效果

> 目标：建立体感——同样的请求第二次明显快；能解释为什么。
> 预计：全职 ~3 天 / 在职 ~1 周

### 阅读

- [ ] [[llm-inference-learning-path]] 阶段 0-1
- [ ] vLLM 官方 Quickstart：https://docs.vllm.ai/en/latest/getting_started/
- [ ] SGLang 官方 Quickstart
- [ ] vLLM 启动日志里 `# GPU blocks` / `# CPU blocks` / `Maximum concurrency` 三行是什么意思（先记下，下一周读源码）

### 动手

- [ ] 用 vLLM 离线推理跑通一个 7B 模型
- [ ] 起 vLLM OpenAI 兼容 server，curl 一次非流式 + 一次流式请求
- [ ] **对照试验**：同样的 prompt，第一次和第二次记录 TTFT（开 `--enable-prefix-caching`），第二次应该明显短
- [ ] curl `/metrics` 看 `vllm:prefix_cache_hit_rate` / `vllm:gpu_cache_usage_perc`
- [ ] 起 SGLang server，同样的 prompt 测一遍 TTFT/TPOT，做对比表

### 周末复盘（默写）

- [ ] 用一句话解释：为什么开 prefix cache 后第二次请求 TTFT 短
- [ ] 用一句话解释：TTFT vs TPOT vs 吞吐 vs 并发，分别衡量什么
- [ ] 列出 vLLM 启动日志里 4 个关键指标的含义

### 完成标准

- [ ] benchmark 记录：vLLM/SGLang 对同一组 prompt 的 TTFT、TPOT、吞吐对比表
- [ ] 口述答案：5 分钟讲清「Prefill 是 compute-bound、Decode 是 memory-bound 在指标上怎么体现」

**本周复盘笔记**：
```
（在这里写一两句话：跑了什么 benchmark、卡在哪）
```

---

## Week 2：PagedAttention 与 Block Manager 源码（上）

> 目标：读懂 vLLM 最核心的 KV cache 物理管理。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] 论文 [PagedAttention (SOSP'23)](https://arxiv.org/abs/2309.06180) 至少两遍
- [ ] [[llm-inference-learning-path]] 阶段 2 全文
- [ ] vLLM 源码：`vllm/core/block_manager.py`（核心）
- [ ] vLLM 源码：`vllm/core/block/block_table.py`、`block_pool.py`
- [ ] vLLM 源码：`vllm/engine/llm_engine.py` 的 `step()`
- [ ] 翻一遍 `csrc/attention/paged_attention_*.cu`，不求懂，眼熟 block table indirect addressing 怎么进 kernel

### 动手

- [ ] **改 block_size**：vLLM 启动参数加 `--block-size 8` 和 `--block-size 32`，对比 `# GPU blocks`、prefix cache 命中率
- [ ] **触发 OOM/swap**：用 `--gpu-memory-utilization 0.5` + 高并发请求，观察 `swap_in/swap_out` 日志
- [ ] 写一段脚本，并发发送 N 个相同前缀 + 不同后缀的请求，看 prefix cache 命中率

### 周末复盘

- [ ] 默写 BlockManager 三个队列（waiting/running/swapped）的转换条件
- [ ] 答出：block_size 越小越好吗（提示：管理开销 vs 碎片）
- [ ] 答出：preemption 时 recompute vs swap 各自适用场景

### 完成标准

- [ ] 白板图：BlockManager 三态状态机 + block table 物理-逻辑映射
- [ ] 源码笔记：把 `step()` → `Scheduler.schedule()` → `BlockManager.allocate()` 的调用链画出来
- [ ] 口述答案：5 分钟讲清 PagedAttention 解决了什么、代价是什么

**本周复盘笔记**：
```
```

---

## Week 3：Continuous Batching + Prefix Cache 源码

> 目标：理解 iteration-level scheduling 和 prefix cache 在源码里怎么实现。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] 论文 [Orca (OSDI'22)](https://www.usenix.org/conference/osdi22/presentation/yu)
- [ ] 论文 [SARATHI / chunked prefill](https://arxiv.org/abs/2308.16369)
- [ ] [[llm-inference-learning-path]] 阶段 3-4 全文
- [ ] vLLM 源码：`vllm/core/scheduler.py` 的 `_schedule()`、`_schedule_chunked_prefill()`
- [ ] vLLM 源码：`vllm/core/block/prefix_caching_block.py`
- [ ] 论文 [SGLang / RadixAttention](https://arxiv.org/abs/2312.07104)
- [ ] SGLang 源码：`python/sglang/srt/mem_cache/radix_cache.py`

### 动手

- [ ] **对照实验 1**：同样的 system prompt + 10 个不同 user query，分别在 vLLM/SGLang 测 prefix cache 命中率
- [ ] **对照实验 2**：chunked prefill 开/关，发一个 32k 长 prompt + 一组短 decode 请求，看 P99 ITL 变化
- [ ] 修改 `max_num_batched_tokens` 和 `max_num_seqs`，记录吞吐与延迟变化

### 周末复盘

- [ ] 答出：vLLM block hash 和 SGLang radix tree 的命中率差异在什么场景显现
- [ ] 答出：chunked prefill 解决了什么 P99 抖动
- [ ] 答出：为什么 max_num_seqs 和 max_num_batched_tokens 是两个独立旋钮

### 完成标准

- [ ] 白板图：3 个不同长度请求在 continuous batching 下的 iteration 时序
- [ ] 实验记录：chunked prefill 开关对 P99 ITL 的实际影响数据
- [ ] 口述答案：5 分钟讲清 prefix cache 在 vLLM/SGLang 的不同实现思路

**本周复盘笔记**：
```
```

---

## 阶段复盘 A：引擎层（Week 1-3）

- [ ] 用一张图串起 vLLM 的：Engine.step → Scheduler → BlockManager → Worker → CUDA kernel
- [ ] 整理 3 个最重要的不变量：block size 是 hash 粒度、prefill/decode 可以混 batch（chunked）、preemption 触发条件
- [ ] 从面试题里挑 1-5 题，闭卷答一遍

---

## Week 4：分布式 KV Store（LMCache / Mooncake）

> 目标：理解 KV 怎么扩展到 GPU↔CPU↔SSD↔远端节点。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] 论文 [Mooncake (FAST'25)](https://arxiv.org/abs/2407.00079)
- [ ] 论文 CacheBlend（非前缀 KV 复用，能找到的 OSDI/SOSP 版本）
- [ ] [[llm-inference-learning-path]] 阶段 5 全文
- [ ] LMCache 源码：`lmcache/lmcache_engine.py`、`lmcache/storage_backend/`
- [ ] Mooncake 源码：扫一遍 `mooncake-transfer-engine/`（C++，看接口即可）、`mooncake-store/`
- [ ] vLLM 源码：`vllm/distributed/kv_transfer/` 的 KVConnector 接口

### 动手

- [ ] 部署 vLLM + LMCache（按官方教程），跑一次跨实例 KV 复用
- [ ] 测一下 GPU↔CPU offload 的实际延迟（用 nsys / nvtop）
- [ ] **可选**：在两台机器之间用 LMCache 的远端后端，测 KV 跨节点拉取延迟
- [ ] **可选**：本地 RDMA 不方便就用 GDRCopy / NCCL P2P 模拟测带宽

### 周末复盘

- [ ] 默写 KV cache 四级分层（GPU HBM / CPU DRAM / NVMe / 远端节点）各自的带宽量级
- [ ] 答出：什么时候拉远端 KV 比本地重算 prefill 更快
- [ ] 答出：Mooncake 为什么坚持 KV-centric 而不是 model-centric

### 完成标准

- [ ] 白板图：四级 KV 分层图 + 容量算账
- [ ] 实验记录：CPU offload 延迟、跨节点拉取延迟（或合理估算）
- [ ] 口述答案：5 分钟讲清 LMCache vs Mooncake 的设计差异

**本周复盘笔记**：
```
```

---

## Week 5：PD 分离（Prefill/Decode Disaggregation）

> 目标：理解为什么分、怎么分、KV 怎么传，能跑一个最小 PD 集群。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] 论文 [DistServe (OSDI'24)](https://arxiv.org/abs/2401.09670) 必读
- [ ] 论文 [Splitwise (ISCA'24)](https://arxiv.org/abs/2311.18677)
- [ ] [[llm-inference-learning-path]] 阶段 6 全文
- [ ] 选一个实现读源码（按熟悉度选）：
  - vLLM `vllm/distributed/kv_transfer/`（NixlConnector / LMCacheConnector / MooncakeConnector）
  - NVIDIA Dynamo `lib/llm/src/kv_router/` + `components/planner/`
  - llm-d `docs/architectures/`

### 动手

- [ ] **最小 PD 集群**：本地 2 块 GPU（或单卡 MPS 模拟），1 个 prefill 实例 + 1 个 decode 实例
- [ ] vLLM 启动时配置 `--kv-transfer-config '{"kv_connector":"...","kv_role":"kv_producer/consumer"}'`
- [ ] 跑同一组请求，对比 co-located vs PD 分离的 TTFT、ITL、吞吐
- [ ] 测 KV 传输延迟：layer 数 × layer size / 带宽 vs 实际 RTT

### 周末复盘

- [ ] 答出：PD 分离不适合的场景（短 prompt 短输出、低 QPS、单卡资源）
- [ ] 答出：1P:N D 比例怎么定，依据什么 workload 特征
- [ ] 答出：KV 传输怎么和 decode 下一层计算重叠

### 完成标准

- [ ] 白板图：co-located vs PD 分离的两套时序对比
- [ ] benchmark 记录：自己测的 PD 分离前后 TTFT/ITL 数据
- [ ] 口述答案：5 分钟讲清 DistServe 和 Mooncake 各自的核心论点

**本周复盘笔记**：
```
```

---

## 阶段复盘 B：分布式与 PD（Week 4-5）

- [ ] 用一张图串起：单机 PagedAttention → 多级 KV 分层 → 跨实例 KV 复用 → PD 分离 KV transfer
- [ ] 整理 3 个最重要的不变量：KV 是金山、计算-传输 trade-off 必须实测、RDMA / NVLink 是分布式 KV 的命门
- [ ] 从面试题里挑 6-10 题，闭卷答一遍

---

## Week 6：Inference Router 与平台层（上）—— GIE + vLLM Production Stack

> 目标：理解 K8s 上的 LLM 平台架构，重点读路由层。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] [[llm-inference-learning-path]] 阶段 7 的 7.1-7.4
- [ ] [[scheduler-framework-source]] 第 9 节回顾（对照看 GIE 的 epp 复用 K8s scheduler framework 思路）
- [ ] Gateway API Inference Extension（GIE）官方文档：https://gateway-api-inference-extension.sigs.k8s.io/
- [ ] GIE 源码：`pkg/epp/`（endpoint picker）、`pkg/scheduling/`
- [ ] vLLM Production Stack 源码：`src/router/`、`helm/`

### 动手

- [ ] 部署 vLLM Production Stack（helm chart 一键），起 2-3 个 vLLM 副本
- [ ] 用 wrk / k6 压一组带前缀重叠的请求，看 prefix cache 命中率有没有起来
- [ ] 故意打偏路由（关掉 prefix-aware）做对照
- [ ] **可选**：装一个 GIE + Envoy AI Gateway 的 demo

### 周末复盘

- [ ] 列出 LLM Router vs 传统 LB 的 4 个本质不同
- [ ] 答出：Router 怎么知道哪个实例有哪个 prefix（事件推 vs 周期 sync）
- [ ] 答出：路由错了怎么 fallback

### 完成标准

- [ ] 白板图：Gateway → Router → vLLM 副本池 → KV Store 的端到端图
- [ ] 实验记录：prefix-aware on/off 的命中率与 TTFT 对比
- [ ] 口述答案：5 分钟讲清 GIE 的 InferenceModel / InferencePool / epp 三件套

**本周复盘笔记**：
```
```

---

## Week 7：平台层（下）—— AIBrix / Dynamo / llm-d 对照

> 目标：通过对比 3 个真实平台，搞清楚 KV-aware autoscaler、LoRA 动态加载、Disaggregated planner 等高阶能力。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读

- [ ] [[llm-inference-learning-path]] 阶段 7 的 7.5-7.8
- [ ] AIBrix 源码：`pkg/plugins/gateway/`、`pkg/controller/`
- [ ] Dynamo 源码：`lib/llm/src/kv_router/`、`components/planner/`
- [ ] llm-d 文档 + 架构图
- [ ] 三家关于 KV-aware autoscaler / LoRA 动态加载 的设计文档

### 动手

- [ ] 部署 AIBrix 或 llm-d（选一个，看你机器资源），跑通端到端
- [ ] 写一个简单 CRD（用 kubebuilder），把它翻译成 InferenceModel + Deployment
- [ ] 用 Prometheus 抓 vLLM/router metrics，Grafana 画 TTFT/TPOT/KV usage/命中率
- [ ] 故意触发一次 KV cache 满，观察 preempt / autoscale 行为

### 周末复盘

- [ ] 默写：传统 HPA 和 KV-aware autoscaler 的 metric 差异
- [ ] 答出：LoRA 动态加载，router 怎么决定 warm up 哪个 LoRA
- [ ] 答出：AIBrix vs Dynamo vs llm-d 的差异化定位

### 完成标准

- [ ] 白板图：自己挑一个平台画完整组件图
- [ ] Grafana 截图：TTFT/TPOT/KV usage/命中率 4 张图
- [ ] 口述答案：5 分钟讲清 KV-aware autoscaler 的关键挑战（KV 是 stateful、缩容不能直接 kill）

**本周复盘笔记**：
```
```

---

## Week 8：源码改造 + 综合实战 + 面试冲刺

> 目标：选一个小切入点真改一次代码；面试题过 2 遍。
> 预计：全职 ~1 周 / 在职 ~2 周

### 阅读 / 准备

- [ ] [[llm-inference-learning-path]] 阶段 8 全文
- [ ] 再过一遍各源码导读和面试要点
- [ ] 把这条线和 K8s 主线（[[k8s-development-roadmap]]、[[hami-learning-path]]）的接口想清楚（GPU 调度 → 推理引擎 → 推理平台 是连贯的）

### 动手（任选其一）

- [ ] 给 vLLM Production Stack 的 router 加一个新策略（lora-aware / latency-aware）
- [ ] 给 GIE 写一个 scheduler plugin（PD-aware scorer）
- [ ] 给 LMCache 加一个 Redis / S3 后端
- [ ] 复现 DistServe 的 1P:2D vs 2P:1D 吞吐对比
- [ ] 写一个 KV cache 监控 dashboard（Grafana JSON + 自己加几个 PromQL）
- [ ] **综合**：在 K8s 上拼一个最小生产可用的推理平台（vLLM + Router + Prometheus + 自定义 CRD），写 README

### 周末复盘 + 模拟面试

- [ ] 模拟 30 分钟技术分享：讲清一次 LLM 请求的端到端链路
- [ ] 再过一遍面试要点（[[llm-inference-learning-path]] 最后一节）
- [ ] 高阶自检题（下面那批）能在 5 分钟内答完

### 完成标准

- [ ] 改造产出：PR 链接 或 项目 README + 截图
- [ ] 端到端架构图：从用户请求到 GPU kernel 再回到用户
- [ ] 200 字总结：哪个机制最让你「原来如此」

**本周复盘笔记**：
```
```

---

## 高阶自检题（8 周后应该全部能答）

1. PagedAttention 为什么用 16 token 的 block？改小改大各有什么权衡？
2. BlockManager 三个队列的转换条件，preemption 时 recompute 和 swap 怎么选
3. continuous batching 的 iteration-level scheduling 和传统 batching 在 GPU 利用率上差几倍
4. chunked prefill 解决了什么 P99 ITL 抖动，max_num_batched_tokens 怎么定
5. vLLM block hash 命中和 SGLang radix tree 命中的本质差异，哪些场景后者明显赢
6. Prefix Cache 在多 LoRA / 多模型版本场景怎么处理
7. KV cache 大小怎么算，128k context 一个 sequence 多大显存
8. CPU offload 的 PCIe 瓶颈，GPU↔CPU 实际带宽
9. Mooncake 为什么必须用 RDMA，CPU 之间走 TCP 不行吗
10. DistServe 的 phase-disaggregated 和 Mooncake 的 KV-centric 的核心差异
11. PD 分离不适合哪些场景，1P:N D 比例怎么定
12. KV transfer 怎么和 decode 下一层计算重叠（layer-wise streaming）
13. LLM Router 跟 HTTP LB 的 4 个本质不同
14. Prefix-aware routing：router 怎么知道哪个实例有哪个 prefix
15. KV-aware autoscaler 跟 HPA 的差异，KV stateful 怎么缩容
16. GIE 的 InferenceModel / InferencePool 是什么，epp 怎么决策
17. AIBrix / Dynamo / llm-d 三家差异化定位
18. 一次推理服务 P99 TTFT 突然变高，你怎么诊断（6 步排查路径）
19. speculative decoding 怎么用，drafter 模型怎么选
20. 一张 H100 80GB 跑 Qwen 7B，理论最大并发是多少（block 池算账）

---

## 待解决问题

> 看不懂的、卡住的、半懂半不懂的，先记在这里。每周末统一回顾。

- [ ]
- [ ]
- [ ]

---

## 完成纪念

8 周收官时在这里贴：
- 自己画的端到端推理平台架构图
- 综合项目的截图 / 仓库链接
- 一段不超过 200 字的总结：哪个机制最让你「原来如此」
