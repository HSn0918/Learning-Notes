#ai #llm #inference #kv-cache #pd-disaggregation #router #学习计划

相关笔记：[[llm-inference-progress]] | [[llm-inference-pipeline]] | [[prefill-cache-miss]] | [[k8s-development-roadmap]] | [[hami-learning-path]] | [[scheduler-framework-source]] | [[gpu-scheduling-source]]

# LLM 推理系统学习路线

## 概述

LLM 推理系统的三件大事——**Inference Router（请求路由层）**、**KV Cache（显存最大的金山）**、**PD 分离（Prefill/Decode 解耦）**——是 2024 年以后大模型基础设施的核心战场。本笔记给一条「自下而上」的学习路径：先把单机推理引擎（vLLM / SGLang）的内部机制吃透，再上升到分布式 KV Cache 存储，最后落到 K8s 上的路由层与平台（llm-d / AIBrix / KServe / NVIDIA Dynamo / vLLM Production Stack）。

> 版本提示：推理引擎目录、配置项和默认值变化很快。本路线固定的是问题拆解与阅读顺序；执行源码练习时，以所选 release 的官方文档和代码为准，并在笔记中记录 commit 或版本号。

> 适用前提：你已经过 [[llm-inference-pipeline]]（Prefill/Decode/Prefix Cache 三件事），并且会读 K8s 源码（参考 [[k8s-development-roadmap]] 的 Phase 1-3）。如果只想做平台层、不深入引擎，可以从阶段 5 直接切入；想吃透引擎内部，按阶段 0-7 顺着走。

## 三个主题与你已有笔记的对应关系

```mermaid
flowchart TB
    subgraph "上层平台 / Router（K8s）"
        ROUTER[Inference Router<br/>路由 / 调度 / 多副本]
        GW[Gateway / Envoy AI Gateway]
        AUTOSCALER[Autoscaler<br/>KV-aware HPA]
    end
    subgraph "服务层（分布式）"
        PD_P[Prefill Pool<br/>compute-bound]
        PD_D[Decode Pool<br/>memory-bound]
        KVSTORE[Distributed KV Store<br/>Mooncake / LMCache]
    end
    subgraph "引擎层（单机）"
        VLLM[vLLM / SGLang]
        PA[PagedAttention<br/>Block Manager]
        RADIX[RadixAttention<br/>Prefix Tree]
        CB[Continuous Batching]
    end
    subgraph "硬件层"
        GPU[GPU HBM]
        CPU[CPU DRAM]
        SSD[NVMe SSD]
        RDMA[RDMA / NVLink]
    end

    ROUTER --> GW
    GW --> PD_P
    GW --> PD_D
    PD_P -->|KV transfer<br/>RDMA| PD_D
    PD_P --> KVSTORE
    PD_D --> KVSTORE
    KVSTORE --> CPU
    KVSTORE --> SSD
    VLLM --> PA
    VLLM --> CB
    VLLM --> RADIX
    PA --> GPU
    KVSTORE --> RDMA
```

| 主题 | 关键问题 | 已有笔记 | 难度 |
| --- | --- | --- | --- |
| Inference Router | 路由层怎么知道哪台机器有 KV cache？怎么 PD 分流？怎么做 KV-aware 负载均衡？ | [[scheduler-framework-source]] [[gpu-scheduling-source]]（K8s 调度类比） | ★★★ |
| KV Cache | PagedAttention 的 block 怎么分配？Prefix Cache 怎么命中？怎么做 GPU↔CPU↔SSD 的多级缓存？ | [[llm-inference-pipeline]] [[prefill-cache-miss]] | ★★★★ |
| PD 分离 | 为什么要分？分了之后 KV 怎么传？谁负责调度？ | 无（新知识） | ★★★★ |
| 引擎内部 | Continuous Batching 怎么并发不同长度的请求？speculative decoding 怎么用？ | [[llm-inference-pipeline]] | ★★★ |

三大主题里，Router 层最像 K8s 调度（[[scheduler-framework-source]] 的思路完全适用），KV Cache 和 PD 分离是全新知识，靠论文 + 真实仓库读出来。

## 你该走哪条路？（决策图）

```mermaid
flowchart TD
    A[开始] --> B{推理引擎跑过没?}
    B -->|没跑过 vLLM| C[阶段 1: 先跑 vLLM 离线推理<br/>+ OpenAI 兼容 server]
    B -->|跑过| D{重点方向?}
    C --> D
    D -->|引擎内核<br/>PagedAttention/CB| E[路线①: vLLM 内核深挖<br/>阶段 2-3]
    D -->|分布式 KV / PD| F[路线②: 系统层<br/>阶段 4-5]
    D -->|K8s 平台 / Router| G[路线③: 平台层<br/>阶段 6]
    D -->|全栈| H[完整路径<br/>阶段 1-7, 6-8 周]

    style E fill:#f39c12,color:#000
    style F fill:#3498db,color:#fff
    style G fill:#2ecc71,color:#000
    style H fill:#9b59b6,color:#fff
```

- **路线①：引擎内核**——只关心 vLLM/SGLang 内部，做引擎贡献或自研引擎。走阶段 1-3 + 阶段 7 的源码改造。
- **路线②：系统层**——关心 PD 分离、分布式 KV、跨节点传输。走阶段 1（跑通） → 阶段 4-5。需要 RDMA 基础。
- **路线③：平台层**——做 K8s 上的推理平台、Inference Gateway、自动扩缩容。走阶段 1（跑通） → 阶段 6，引擎层只需理解输入输出契约。
- **完整路径**：全职 6-8 周 / 在职 12-16 周，所有阶段串起来——这是 AI Infra 岗位最完整的路径。

## 阶段清单

```mermaid
gantt
    title LLM 推理学习路径（建议 6-8 周）
    dateFormat  YYYY-MM-DD
    section 引擎层
    跑通 vLLM 推理            :a1, 2026-06-01, 3d
    PagedAttention 内核        :a2, after a1, 7d
    Continuous Batching       :a3, after a2, 5d
    section KV Cache
    Prefix / RadixAttention   :b1, after a3, 5d
    分布式 KV Store           :b2, after b1, 7d
    section PD 分离
    PD 论文 + 实现            :c1, after b2, 7d
    section 平台层
    Inference Router / K8s    :d1, after c1, 10d
    section 综合
    源码改造 + 自检           :e1, after d1, 7d
```

## 阶段 0：先决条件检查（半天）

下面这些不达标，后面看源码会一头雾水：

- ✅ 已读 [[llm-inference-pipeline]]：Prefill 计算密集 / Decode 内存带宽密集 / Prefix Cache 复用前缀 KV
- ✅ 已读 [[prefill-cache-miss]]：理解为什么 1 个 token 差异就 miss 全部缓存
- ✅ 知道 Transformer Attention 的 Q/K/V 计算公式（不需要会推导，知道 KV 是从输入投影出来的就行）
- ✅ 用过 OpenAI 兼容的 chat API（`/v1/chat/completions`）
- ✅ 知道一些基础概念：TTFT（首 token 延迟）、TPOT（每 token 延迟）、ITL（inter-token latency）、吞吐 vs 延迟的权衡
- ✅ 有一台 GPU 机器（消费卡 RTX 3090/4090 / 24GB 显存够跑 7B 模型，14B 以上模型需要 A100/H100）

**产出**：能用一句话解释清楚：「为什么 LLM 推理需要 KV Cache？没有 KV Cache 会怎样？」

---

## 阶段 1：跑通最小推理引擎（2-3 天）

**目标**：先让 vLLM 跑起来，体感「Prefill 慢、Decode 流式快」的实际差别。

### 1.1 vLLM 离线推理（30 分钟）

```python
from vllm import LLM, SamplingParams

llm = LLM(model="Qwen/Qwen2.5-7B-Instruct", gpu_memory_utilization=0.85)
out = llm.generate(
    ["介绍下 Kubernetes 的调度器"],
    SamplingParams(temperature=0.7, max_tokens=512),
)
print(out[0].outputs[0].text)
```

跑通之后，注意观察：
- 启动时打印的 `# GPU blocks: xxxxx`、`# CPU blocks: xxxxx` —— 这就是 **PagedAttention 的 block 池**，是 KV Cache 的物理单元
- 启动时打印的 `Maximum concurrency for X tokens per request: Y` —— 这是 block 池能支撑的最大并发

### 1.2 起 OpenAI 兼容 server

```bash
python -m vllm.entrypoints.openai.api_server \
    --model Qwen/Qwen2.5-7B-Instruct \
    --gpu-memory-utilization 0.85 \
    --max-model-len 8192 \
    --enable-prefix-caching
```

```bash
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen/Qwen2.5-7B-Instruct",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
  }'
```

**第一次请求和第二次请求（同样的 prompt）的 TTFT 对比** —— 第二次因为 prefix cache 命中，TTFT 应该明显短。这就是 Prefix Cache 的实际效果，亲眼看到比读 10 篇论文记得牢。

### 1.3 看 metrics

```bash
curl http://localhost:8000/metrics | grep vllm
```

重点关注：
- `vllm:gpu_cache_usage_perc` —— GPU 上 KV cache 占用率
- `vllm:num_requests_running` / `vllm:num_requests_waiting` —— 正在处理 / 排队
- `vllm:prefix_cache_hit_rate` —— 前缀缓存命中率
- `vllm:time_to_first_token_seconds` —— TTFT 分布
- `vllm:time_per_output_token_seconds` —— TPOT 分布

这些指标后面 Router 层做调度决策时要用到，先眼熟。

### 1.4 同时跑 SGLang 对照

```bash
python -m sglang.launch_server \
    --model-path Qwen/Qwen2.5-7B-Instruct \
    --port 30000
```

vLLM 和 SGLang 都是主流开源推理引擎。PagedAttention 首先解决 KV Cache 的分页式物理内存管理，RadixAttention 强调共享前缀的索引与复用；两者关注层次不同，现代引擎也会相互吸收能力，不能把它们简单理解为互斥方案。早跑早对比，后面源码读起来才有抓手。

**产出**：
- 一份 benchmark 记录：vLLM/SGLang 对同一组 prompt 的 TTFT、TPOT、吞吐
- 解释为什么开了 `--enable-prefix-caching` 后第二次请求快

---

## 阶段 2：源码 1 — PagedAttention 与 Block Manager（5-7 天）

**目标**：读懂 vLLM 最核心的设计——把 KV Cache 切成定长 block 像虚拟内存一样管理。

### 2.1 论文：vLLM / PagedAttention (SOSP'23)

读论文：[Efficient Memory Management for Large Language Model Serving with PagedAttention](https://arxiv.org/abs/2309.06180)

核心思想：
- **传统问题**：KV cache 按 max_seq_len 预留连续显存，碎片严重（内部碎片 + 外部碎片），实际利用率经常 < 40%
- **PagedAttention**：KV 按固定大小 block（默认 16 token）切片存储，类比 OS 的 page，逻辑连续物理可分散
- **block table**：每个 sequence 维护一个 block table，记录逻辑 token 位置 → 物理 block 编号
- **CoW（Copy-on-Write）**：beam search / parallel sampling 时共享 block，写时复制

### 2.2 vLLM 源码切入

```bash
git clone https://github.com/vllm-project/vllm
cd vllm
```

**关键目录**：
- `vllm/core/block_manager.py` —— BlockManager（最核心）
- `vllm/core/block/` —— BlockPool、BlockAllocator
- `vllm/attention/backends/` —— FlashAttention / xformers / FlashInfer backend
- `vllm/worker/` —— Worker / ModelRunner
- `vllm/engine/llm_engine.py` —— 顶层调度
- `csrc/attention/paged_attention_v2.cu` —— CUDA kernel 实现

**阅读顺序**：
1. `LLMEngine.step()` —— 看一轮迭代干什么（Schedule → Execute → Process Output）
2. `Scheduler.schedule()` —— 选哪些 sequence 进这一轮、给它们分配 block
3. `BlockManager.allocate()` / `BlockManager.append_slot()` —— block 怎么分配 / 追加
4. `BlockManager.swap_in()` / `swap_out()` —— GPU 满了怎么换到 CPU

### 2.3 关键问题

读完应该能答：
- **block size 为什么常见为 16？改成 1 / 128 各有什么权衡？**

  > [!question]- 参考答案（点击展开）
  > 固定 block 的大小是在元数据、调度/Kernel 间接寻址开销和 KV 内部碎片之间折中。更小的 block 降低尾部碎片、提高细粒度复用，但 block table 和分配次数更多；更大的 block 相反。`16` 不是跨版本或跨模型的定律，应以目标引擎版本、上下文长度与压测结果选择。

- **waiting / running / swapped 三个队列各自的角色？**

  > [!question]- 参考答案（点击展开）
  > `waiting` 是尚未获得初始 KV block、等待 admission 的请求；`running` 是本轮可继续 prefill/decode 的请求；`swapped` 是 KV 被迁到 CPU 后等待换回 GPU 的请求。后两者是否存在及其转换细节随 vLLM 版本和是否启用 offload 而变，应以对应版本的 scheduler 状态机为准。

- **preemption（抢占）什么时候发生？recompute 和 swap 怎么选？**

  > [!question]- 参考答案（点击展开）
  > 当运行请求无法再追加 KV slot、且系统需要释放 GPU block 给更应优先推进的请求时会触发抢占。`recompute` 丢弃可重建 KV，省去 CPU 内存和传输，适合短上下文或重算便宜的请求；`swap` 保留 KV 到主存，适合重算昂贵但会增加 PCIe 传输与恢复延迟。选择取决于 KV 大小、可用主存和预期恢复时间。

- **block table 在 attention kernel 里怎么用？**

  > [!question]- 参考答案（点击展开）
  > 它把逻辑 token block 映射到不连续的物理 KV block。Kernel 先由 token position 算出逻辑 block 与 block 内 offset，再查表定位物理地址，因此 sequence 扩容、回收或复用不要求 KV 连续。具体表布局和 kernel 名称是引擎版本实现细节。

### 2.4 联系到你已学的

- BlockManager 的 free list + lookup table 设计 = OS page table 的简化版
- Scheduler 三队列状态机 = K8s scheduler activeQ/backoffQ/unschedulableQ 的同构思想（[[scheduler-framework-source]]）
- preemption 重算 vs 换出 = 缺页中断的 swap-in/swap-out

**产出**：
- 自己画一张 PagedAttention 的 block table 示意图（不抄论文）
- 解释清楚：一个 1024 token 的 sequence 进入引擎时，block manager 干了什么

---

## 阶段 3：源码 2 — Continuous Batching 与 Scheduler（3-5 天）

**目标**：读懂 vLLM 怎么把不同进度的请求拼成一个 batch。

### 3.1 论文：Orca (OSDI'22)

读论文：[Orca: A Distributed Serving System for Transformer-Based Generative Models](https://www.usenix.org/conference/osdi22/presentation/yu)

核心思想：
- **传统 static batching**：等齐 batch 里所有请求生成完才放下一批，慢的拖累快的
- **iteration-level scheduling（即 continuous batching）**：每个 iteration 决定哪些 seq 参与，刚生成完的立刻让位
- **selective batching**：Attention 算子在 sequence 维度做（不同 seq 长度不同，无法直接 batch），其他算子在 token 维度做

### 3.2 vLLM 调度循环

`vllm/core/scheduler.py` 的 `_schedule()`：

```python
# 伪代码（实际更复杂）
def schedule():
    # 1. 先把已 running 的请求拿出来（continuous batching 主场）
    running = self.running_queue.pop_all()
    # 2. 给 running 请求追加 block（每个请求需要 1 个新 token 的 slot）
    for seq in running:
        if not block_manager.can_append_slot(seq):
            # GPU 满了，preempt 一个低优先级的
            preempt(seq)
    # 3. 把 waiting 队列里能塞进来的新请求加入
    while waiting and block_manager.can_allocate(waiting[0]):
        running.append(waiting.pop(0))
    return running
```

**关键设计点**：
- prefill 和 decode 在同一个 batch 里跑吗？—— 早期 vLLM 是 chunked prefill（混跑），后来引入 PD 分离思路
- prefill chunk size 怎么定？—— `max_num_batched_tokens` 控制
- 长 prompt（比如 32k）怎么处理？—— chunked prefill，切成多个 chunk

### 3.3 关键问题

- **chunked prefill 解决了什么问题？**

  > [!question]- 参考答案（点击展开）
  > 它把长 prompt 的 prefill 切成受 token budget 限制的小段，与 decode 轮次交错，避免一次长 prefill 长时间占满 batch 而抬高已有流式请求的 ITL/P99。代价是长请求完成 prefill 的轮次更多；chunk 大小应由 TTFT、ITL 和吞吐的压测共同决定。

- **`max_num_seqs` / `max_num_batched_tokens` 两个旋钮分别控制什么？**

  > [!question]- 参考答案（点击展开）
  > 前者约束一轮内可并发服务的 sequence 数，主要影响并发和每请求调度机会；后者约束一轮可处理的 token 总量，主要限制 prefill/decode 的计算量和显存压力。两者不是独立的性能开关，应在固定模型、上下文分布和硬件上联合调优。

- **vLLM 的 throughput-first 与 latency-first 模式有什么参数差异？**

  > [!question]- 参考答案（点击展开）
  > throughput-first 通常提高 batch token/并发上限并容忍排队，以换取更高 GPU 利用率；latency-first 则限制批大小、优先让 decode 获得及时调度，以控制 TTFT/ITL。没有一组跨版本通用 flag，必须记录 vLLM 版本、scheduler 配置和压测负载。

**产出**：
- 画一张时序图：3 个不同长度的请求（短/中/长 prompt）在 continuous batching 下如何并行
- 解释为什么 chunked prefill 能改善 P99 ITL

---

## 阶段 4：KV Cache 进阶 — Prefix Cache 与 RadixAttention（3-5 天）

**目标**：理解多请求之间怎么复用 KV，不同引擎的不同思路。

### 4.1 vLLM 的 Prefix Caching

`vllm/core/block/prefix_caching_block.py`：
- 每个 block 计算一个 hash（基于 token id + 父 block hash）
- 哈希命中 → 直接复用物理 block，引用计数 +1
- evict 用 LRU，当显存满了驱逐引用计数为 0 的

读源码重点：
- `PrefixCachingBlockAllocator.allocate_immutable_block()` —— hash 命中怎么判定
- `compute_block_hash()` —— hash 怎么算（影响命中率，细微差别参考 [[prefill-cache-miss]]）

### 4.2 SGLang 的 RadixAttention

论文：[SGLang: Efficient Execution of Structured Language Model Programs](https://arxiv.org/abs/2312.07104)

**核心创新**：用 **Radix Tree（基数树）** 而不是哈希表组织 KV cache。
- 每个节点是一段 token 序列（不是单个 token）
- 共享前缀自动合并到同一路径
- match 操作是树上的最长前缀匹配，比 vLLM 的 block hash 更细粒度

```mermaid
flowchart TD
    R[root]
    R --> S1["System: 你是助手"]
    S1 --> A1["User: 写代码"]
    S1 --> A2["User: 翻译"]
    A1 --> A1A["Asst: def..."]
    A1 --> A1B["Asst: function..."]
```

读源码：`python/sglang/srt/mem_cache/radix_cache.py`

### 4.3 两种思路的本质差异

| 维度 | vLLM PagedAttention | SGLang RadixAttention |
| :--- | :--- | :--- |
| 物理单元 | 固定 size block（16 token） | 变长 segment |
| 共享判定 | block hash 精确匹配 | 树上最长前缀匹配 |
| 适合场景 | 通用，并发数高 | 多轮对话、prompt 模板分支多 |
| 实现难度 | 中 | 高（树维护 + 锁） |

### 4.4 关键问题

- **哪些场景 RadixAttention 命中率明显高于 Block Hash？**

  > [!question]- 参考答案（点击展开）
  > 当请求共享很长前缀、但在任意 token 边界分叉或有多轮对话的频繁分支时，radix tree 的最长前缀匹配通常比只能复用完整固定 block 的方案更灵活。实际命中率仍取决于 prompt/tokenizer 一致性、并发驻留时间和淘汰策略，不能只按数据结构名称断言。

- **Prefix Cache 在多 LoRA / 多模型版本场景怎么处理？**

  > [!question]- 参考答案（点击展开）
  > 缓存 key 必须包含决定前向计算的模型版本、LoRA adapter/权重版本、tokenizer 与位置编码相关配置；只有这些兼容时才能共享 KV。adapter 热更新、卸载或基础模型升级时应按 namespace/版本失效，不能把“token 相同”误当作语义兼容。

- **如果两个请求 token ID 完全一样但 sampling 参数不同，能共享 KV 吗？**

  > [!question]- 参考答案（点击展开）
  > 可以共享已给定 prompt 的 KV，因为 sampling 发生在 logits 之后，不改变该 prompt 的前向计算；后续生成 token 一旦不同，cache 链就分叉。前提仍是模型、adapter、tokenizer、position/rope 等前向条件相同。

**产出**：
- 跑一个对比实验：同样的 system prompt + 10 个不同 user query，记 vLLM/SGLang 各自的命中率与 TTFT
- 解释为什么 RadixAttention 适合 agent / 多轮场景

---

## 阶段 5：分布式 KV Cache Store（5-7 天）

**目标**：理解 KV cache 怎么扩展到 GPU↔CPU↔SSD↔对象存储多级，跨节点共享。

### 5.1 为什么要分布式 KV

单机 GPU 显存 80GB（H100），KV cache 占大头。两个核心问题：
1. **跨副本复用**：副本 A 算过的前缀，副本 B 收到相同 prompt 时也想复用，但 KV 在 A 的显存里
2. **超长上下文**：128k context 的 KV cache 一个 sequence 就 GB 级，要分层存储

### 5.2 LMCache

仓库：https://github.com/LMCache/LMCache

设计要点（按重要性排序）：
- **CPU offload**：GPU 显存满了的 block 下沉到 CPU DRAM
- **NVMe offload**：CPU 也满了下沉到 NVMe SSD
- **跨实例共享**：通过 redis/etcd 注册 KV 位置，副本间 CPU↔CPU 互拉
- **non-prefix sharing**：能在中间 chunk 复用 KV（论文 CacheBlend / CacheGen），不局限于前缀

读源码：
- `lmcache/lmcache_engine.py` —— 入口
- `lmcache/storage_backend/` —— 各级存储后端
- vLLM 集成：vLLM 0.7+ 的 `vllm/distributed/kv_transfer/`

### 5.3 Mooncake

论文：[Mooncake: A KVCache-centric Disaggregated Architecture for LLM Serving](https://arxiv.org/abs/2407.00079)

仓库：https://github.com/kvcache-ai/Mooncake

Mooncake 是 Kimi（月之暗面）的生产系统，**KVCache-centric** 是它的核心设计哲学：
- **Mooncake Store**：分布式 KV cache 池，用所有节点的空闲 DRAM + SSD
- **Transfer Engine**：RDMA / NVLink 高速传输 KV 块
- **Conductor**：调度器，KV-aware 选实例

读源码：
- `mooncake-transfer-engine/` —— 传输引擎（C++，RDMA-heavy）
- `mooncake-store/` —— 分布式 KV 池
- `mooncake-integration/` —— 和 vLLM/SGLang 的集成层

### 5.4 关键问题

- **CPU offload 的瓶颈在哪？**

  > [!question]- 参考答案（点击展开）
  > 关键瓶颈是 GPU↔host 的传输带宽、同步延迟和主存容量/NUMA 位置；被换出的 KV 在恢复时会直接拉长请求延迟，并与其他 PCIe/NVLink 流量竞争。标称带宽不是可直接套用的常数，应实测实际链路、并发 DMA 和 copy/compute 重叠效果。

- **RDMA 比走 TCP 快多少？为什么 Mooncake 必须用 RDMA？**

  > [!question]- 参考答案（点击展开）
  > RDMA 通常能减少 CPU 参与和拷贝，降低端到端延迟并提高大块 KV 传输吞吐；具体倍数由网卡、拓扑、消息大小和拥塞决定。Mooncake 的设计从高速互连中获益很大，但“必须 RDMA”不是协议正确性前提：TCP 可以工作，只是在高频远端 KV 传输时往往难以满足其性能目标。

- **跨实例拉 KV 和直接重算 prefill 怎么决策？**

  > [!question]- 参考答案（点击展开）
  > 比较 `查找 + 传输 + 排队` 与本地 `prefill 计算 + 排队` 的预期延迟，并确认 cache 命中、版本兼容和目的端容量。短前缀、低速网络或远端拥塞时重算可能更快；命中长前缀且高速互连可用时拉取更有利。失败时必须能退回本地 prefill。

- **模型权重更新后旧 KV cache 怎么办？**

  > [!question]- 参考答案（点击展开）
  > KV 与生成它的 base model、adapter、tokenizer 和关键运行配置绑定。发布新版本时应使用版本化 cache namespace，并在路由和传输层拒绝旧版本命中；可让旧 namespace 自然淘汰或显式回收，不能跨权重版本复用。

**产出**：
- 自己画一张「冷热分层」图：GPU HBM / CPU DRAM / NVMe / 远端节点 各自存什么
- 算一笔账：100 个副本，每个副本 80GB GPU，加上 1TB CPU + 4TB NVMe，总 KV 池能存多少 token 的 KV

---

## 阶段 6：PD 分离（Prefill/Decode Disaggregation）（5-7 天）

**目标**：理解为什么要把 Prefill 和 Decode 拆到不同实例，以及它们之间怎么传 KV。

### 6.1 三篇必读论文

1. **DistServe（OSDI'24）**：[DistServe: Disaggregating Prefill and Decoding for Goodput-optimized Large Language Model Serving](https://arxiv.org/abs/2401.09670)
   - 核心论点：Prefill 是 compute-bound、Decode 是 memory-bound，混跑互相干扰
   - 拆开后各自的 batching 策略、并行策略（TP/PP/DP）独立优化
2. **Splitwise（ISCA'24）**：[Splitwise: Efficient Generative LLM Inference Using Phase Splitting](https://arxiv.org/abs/2311.18677)
   - 微软方案，重点是 KV 传输怎么不阻塞 Decode
3. **Mooncake**（前面读过）：生产级 PD 分离，KV-centric

### 6.2 为什么要分

```mermaid
flowchart LR
    subgraph "Co-located（不分）"
        REQ1[长 prompt 请求] --> CO[同一 GPU]
        REQ2[正在 decode 的请求] --> CO
        CO -->|prefill 占满 GPU<br/>decode 卡顿| OUT1[ITL 抖动严重]
    end
    subgraph "Disaggregated（PD 分离）"
        REQ1B[长 prompt 请求] --> P_POOL[Prefill Pool<br/>大 batch / 算力优先]
        P_POOL -->|KV transfer<br/>RDMA| D_POOL[Decode Pool<br/>大并发 / 显存优先]
        REQ2B[正在 decode] --> D_POOL
        D_POOL --> OUT2[TTFT/ITL 稳定]
    end
```

### 6.3 KV 传输的工程难点

- **Prefill 实例的 layer-by-layer KV 怎么序列化？**

  > [!question]- 参考答案（点击展开）
  > 传输协议至少要固定模型/权重版本、layer、KV dtype、tensor shape、并行分片、token range 和内存布局，再用版本化 metadata 指向连续或分块 buffer。接收端必须校验兼容性；不能只传一段裸 bytes 后假设两端布局相同。

- **传输为什么通常选择 RDMA / NVLink，而不是普通 TCP？**

  > [!question]- 参考答案（点击展开）
  > RDMA/NVLink 的价值是降低主机拷贝和 CPU 开销，并提供更高、更稳定的带宽；TCP 在功能上可用，但可能让 KV 传输超过直接重算 Prefill 的成本。是否必须使用取决于 KV 大小、拓扑、并发和 SLO，应以端到端压测而不是协议名称判断。

- **传输能不能和 Decode 的下一层计算重叠？**

  > [!question]- 参考答案（点击展开）
  > 可以在 layer 粒度建立就绪信号：Decode 计算已到达的当前层时，用独立 stream/DMA 传输后续层，并用双缓冲减少等待。重叠程度受 layer 依赖、链路竞争、buffer 数量和抖动限制；未到达时仍必须等待或回退。

- **Decode 实例挂了，正在传的 KV 怎么办？**

  > [!question]- 参考答案（点击展开）
  > 把传输视为可失败、可重试的有版本对象：目录中只发布完整就绪的 KV，部分传输不能被消费。Decode 失败后 Router 停止分流，请求按策略重选实例并重传，或在新实例重算 Prefill；是否保留远端 KV 取决于复制、生命周期和 SLO 成本。

### 6.4 真实实现的源码

读至少其中一个：

**a) vLLM PD 分离**：vLLM 0.7+ 有 `vllm/distributed/kv_transfer/`
- KVConnector 抽象：拉 / 推 KV
- LMCacheConnector / MooncakeConnector / NixlConnector 等实现
- 起 vLLM 时 `--kv-transfer-config` 配置

**b) NVIDIA Dynamo**：https://github.com/ai-dynamo/dynamo
- 开源的高性能 PD 分离推理框架，2025 年发布
- 用 NIXL（NVIDIA Inference eXchange Library）做 KV 传输
- 内置 router / planner / autoscaler

**c) llm-d**：https://github.com/llm-d/llm-d
- IBM/Red Hat 主导的 K8s 上的 PD 分离方案
- 用 vLLM 当引擎，K8s 当编排，Gateway API Inference Extension 当路由

### 6.5 关键问题

- **PD 分离一定比 co-located 好吗？什么时候不该分？**

  > [!question]- 参考答案（点击展开）
  > 不一定。它在 prompt/输出长度差异大、请求量足够高且 KV 传输可控时能分别优化两阶段；低流量、短上下文、网络慢或运维复杂度无法接受时，co-located 往往延迟更低、故障面更小。应以端到端 goodput、P99 和成本验证，而非只看单阶段吞吐。

- **Prefill 和 Decode 的并行策略可以不同吗？**

  > [!question]- 参考答案（点击展开）
  > 可以：Prefill 的矩阵计算更易受吞吐导向的并行策略影响，Decode 则更受每 token 延迟、KV 访问和长上下文通信约束影响。TP/SP 等具体搭配依模型架构、序列长度和互连而定，不能把某一经验组合写成普适规则。

- **KV 传输延迟与重算延迟怎么选？**

  > [!question]- 参考答案（点击展开）
  > 选择预计更短且更稳定的一侧：KV 已命中、链路带宽/排队可控时传输；KV 较小、链路拥塞或兼容性不成立时重算。调度器应把该比较做成实时/近实时成本模型，并保留传输失败后的重算 fallback。

- **1P:N D 还是 1P:1D？比例怎么定？**

  > [!question]- 参考答案（点击展开）
  > 没有固定比例。根据到达请求的输入/输出 token 分布、各池的服务率、TTFT/TPOT SLO、KV 传输容量和 GPU 利用率估算，并随负载调整；输出较长时 decode 容量常更紧，长输入突发时 prefill 容量可能成为瓶颈。

**产出**：
- 部署一个 PD 分离的最小集群（1 prefill + 1 decode，本地两块 GPU 或单卡用 MPS 模拟）
- 测量 KV 传输延迟（layer 数 × layer size / 带宽）和实际 RTT 的差距
- 解释清楚 Mooncake 的「KV-centric」和 DistServe 的「phase-disaggregated」核心差异

---

## 阶段 7：Inference Router 与 K8s 平台层（7-10 天）

**目标**：理解 K8s 上的 LLM 推理平台怎么组装，重点是路由层（router）的设计。

### 7.1 Router 干什么

LLM 路由跟传统 HTTP 负载均衡有四个本质不同：

| 维度 | 传统 LB | LLM Router |
| :--- | :--- | :--- |
| 请求大小 | 均匀 | 极不均匀（128 token vs 32k token） |
| 处理时长 | ms 级 | s ~ min 级 |
| 实例状态 | 无状态 | 有状态（KV cache、LoRA、模型版本） |
| 路由维度 | URL/header | model + LoRA + prefix hash + KV 位置 |

**关键能力**：
1. **Model-aware routing**：按模型/LoRA 路由
2. **Prefix-aware routing**：把相同前缀的请求路到同一实例（命中 Prefix Cache）
3. **Load-aware routing**：看实例的 KV cache 占用率、排队长度
4. **PD-aware routing**：先路到 Prefill Pool，再到 Decode Pool
5. **Disaggregated metrics**：把 TTFT、TPOT、KV usage 暴露给 autoscaler

### 7.2 主流方案对照

| 方案 | 路由层 | 编排 | 引擎 | 主导方 |
| :--- | :--- | :--- | :--- | :--- |
| **KServe** | KServe Predictor + Knative | K8s | 任意（vLLM/TGI/Triton） | CNCF |
| **vLLM Production Stack** | vLLM Router（HTTP, prefix-aware） | K8s | vLLM | UChicago + vLLM 社区 |
| **llm-d** | Gateway API Inference Extension (GIE) + Envoy AI Gateway | K8s | vLLM | IBM + RedHat + Google |
| **AIBrix** | AIBrix Gateway / Router | K8s | vLLM | ByteDance |
| **NVIDIA Dynamo** | Dynamo Router | 自带 / K8s | TRT-LLM + vLLM | NVIDIA |

### 7.3 必读：Gateway API Inference Extension（GIE）

K8s 官方推进的 LLM 推理 Gateway 标准，仓库：https://github.com/kubernetes-sigs/gateway-api-inference-extension

核心 CRD：
- `InferenceModel`：声明一个模型 / LoRA 服务
- `InferencePool`：声明实例池
- 配合 Gateway API 的 `HTTPRoute` 做路由

读源码：
- `pkg/epp/`（endpoint picker plugin）—— 路由决策点
- `pkg/scheduling/` —— filter / scorer 链（**这是 K8s scheduler framework 思路的复用，对 [[scheduler-framework-source]] 熟的人零成本切入**）

### 7.4 vLLM Production Stack

仓库：https://github.com/vllm-project/production-stack

特色：
- Router 是单独的 Go 程序，做 prefix-aware + load-aware 路由
- 自带 LMCache 集成
- 提供 helm chart 一键部署

读源码：
- `src/router/` —— Go 实现的路由器
- `helm/` —— 部署清单

### 7.5 AIBrix

仓库：https://github.com/vllm-project/aibrix

特色：
- ByteDance 生产系统开源版
- Gateway 层做 LoRA 动态加载、KV cache 多级管理
- Autoscaler 用 KV-aware 指标（不是只看 CPU/QPS）

读源码：
- `pkg/plugins/gateway/` —— Envoy gateway 插件
- `pkg/controller/` —— K8s controller（用 controller-runtime，对 [[controller-runtime-source]] 熟的人直接看懂）

### 7.6 NVIDIA Dynamo

仓库：https://github.com/ai-dynamo/dynamo

特色：
- 用 Rust 写，性能为先
- 内置 KV-aware Router、Disaggregated Planner（PD 比例自动调整）
- 走 NIXL 做 KV 传输（不必走 LMCache/Mooncake）

读源码（即使不熟 Rust 也要扫一眼）：
- `lib/llm/src/kv_router/` —— KV-aware 路由
- `components/planner/` —— PD 比例规划器
- `lib/runtime/src/transports/` —— NIXL 集成

### 7.7 关键问题

- **Prefix-aware routing 在 router 端怎么知道哪个实例有哪个 prefix？**

  > [!question]- 参考答案（点击展开）
  > 实例需要上报可复用 prefix 的摘要或目录，router 再通过周期同步、事件推送或两者结合维护近似视图。该视图允许短暂陈旧，因此路由决定应同时考虑负载并接受 cache miss；精确目录、Bloom filter 或集中 catalog 是不同实现取舍，而非唯一方案。

- **一个请求路错了（命中率不如预期）怎么 fallback？**

  > [!question]- 参考答案（点击展开）
  > 目标实例 cache miss 时可在本实例冷 prefill，或按容量转发到兼容的备选实例；不能为了追求命中无限重试。应记录预测命中与实际命中、限制重路由次数，并在模型/adapter 不兼容、传输失败时走确定的本地 prefill 路径。

- **KV-aware autoscaler 怎么定义「过载」？**

  > [!question]- 参考答案（点击展开）
  > 应组合 queue wait、TTFT/TPOT 分位数、decode token rate、KV 使用率/可回收空间和实例可用性，而非只看 QPS。扩容要区分需要新模型副本、decode 容量还是缓存容量；缩容需先迁移/驱逐可安全失效的 KV，并设置滞后避免抖动。

- **LoRA 动态加载场景：router 怎么决定要不要在某实例上 warm up 一个 LoRA？**

  > [!question]- 参考答案（点击展开）
  > 基于近期请求频率、预计驻留收益、加载成本、GPU 显存余量和副本数做预算；热门 adapter 可预热，低频 adapter 应按需加载或路由到已有副本。决策还要区分 adapter 版本，并避免在用户请求关键路径上无界排队加载。

### 7.8 联系到你已学的

- GIE 的 epp scheduling chain = K8s scheduler framework 的 filter/score plugin 思路（[[scheduler-framework-source]]）
- AIBrix Controller = controller-runtime 的标准用法（[[controller-runtime-source]]）
- KV-aware HPA = 自定义 metrics adapter 那一套
- LoRA 动态分配 = Device Plugin 的另一种形态（[[gpu-scheduling-source]]）

**产出**：
- 部署 vLLM Production Stack 或 llm-d 最小集群（2-3 个 vLLM 实例 + Router）
- 用 wrk / k6 压一组带前缀重叠的请求，看 prefix cache 命中率有没有起来
- 画一张完整端到端图：用户 → Gateway → Router → Prefill Pool → KV Transfer → Decode Pool → 流式返回

---

## 阶段 8：源码改造 + 综合实战（5-7 天）

**目标**：选一个小切入点，做一次真实改造或贡献。

### 8.1 候选题目（任选其一）

1. **给 vLLM Router 加一个新策略**：在 vLLM Production Stack 的 router 里加一个 "lora-aware" 路由策略
2. **给 GIE 写一个 scheduler plugin**：参考 K8s scheduler framework，写一个 PD-aware scorer
3. **给 LMCache 加一个存储后端**：比如 Redis / S3
4. **复现 DistServe 论文的 PD 分配实验**：在 vLLM 上模拟 1P:2D vs 2P:1D 的吞吐对比
5. **写一个 KV cache 监控 dashboard**：抓 `/metrics`，做 prefix cache 命中率、KV usage 的实时可视化

### 8.2 综合实战：用 K8s 拼一个推理平台

参考 hami-learning-path 的「完成后能做什么」，给一个综合题：

> 拼一个最小可用的 LLM 推理平台：
> - 1 个 K8s 集群（kind 多节点，或真集群）
> - 部署 vLLM Production Stack 或 llm-d
> - 接入 Prometheus + Grafana，画出 TTFT / TPOT / KV usage / prefix cache hit rate
> - 写一个简单的 CRD（用 kubebuilder）声明 "InferenceWorkload"，控制器把它翻译成 vLLM Deployment + InferenceModel
> - 故意触发一次 KV cache 满，看 preemption / swap 行为

**产出**：仓库 + README + 一张端到端架构图 + 一篇 200 字的「最让我意外的设计是什么」。

---

## 阶段对照检查表

| 阶段 | 关键问题 | 通过标准 |
| --- | --- | --- |
| 1 | 跑通 vLLM 没？看到 prefix cache 实际效果没？ | 第二次请求 TTFT 明显短，能解释为什么 |
| 2 | block manager 怎么分块？ | 默写一张 block table 图，解释 swap/recompute |
| 3 | continuous batching 调度？ | 画三个不同长度请求的 iteration-level 时序 |
| 4 | vLLM vs SGLang 怎么不同？ | 列出 block hash vs radix tree 的本质差异 |
| 5 | 分布式 KV 分层？ | 画 GPU/CPU/NVMe/远端的冷热图，算容量 |
| 6 | PD 分离的传输与权衡？ | 解释 1P:N D 比例、KV transfer 重叠 decode |
| 7 | Router 的四种 awareness？ | model / prefix / load / PD 各举一个实现案例 |
| 8 | 真改造一次？ | 提一个 PR 或交付一个可跑的小项目 |

## 学习资源

### 论文（按重要性排）

1. **PagedAttention / vLLM**（SOSP'23）—— 必读
2. **Orca**（OSDI'22）—— continuous batching 鼻祖
3. **DistServe**（OSDI'24）—— PD 分离必读
4. **Splitwise**（ISCA'24）—— PD 分离另一视角
5. **Mooncake**（FAST'25）—— 生产级分布式 KV
6. **SGLang / RadixAttention**（NeurIPS'24）
7. **SARATHI**（OSDI'24）—— chunked prefill 系统化
8. **CacheBlend / CacheGen**—— 非前缀 KV 复用

### 仓库

| 类型 | 仓库 | 看点 |
| :--- | :--- | :--- |
| 引擎 | https://github.com/vllm-project/vllm | PagedAttention / 调度 / 多硬件后端 |
| 引擎 | https://github.com/sgl-project/sglang | RadixAttention / 高吞吐 |
| KV Store | https://github.com/LMCache/LMCache | 多级 KV cache |
| KV Store | https://github.com/kvcache-ai/Mooncake | 分布式 KV、RDMA |
| 平台 | https://github.com/vllm-project/production-stack | vLLM 官方 K8s 栈 |
| 平台 | https://github.com/llm-d/llm-d | K8s + GIE 路由 |
| 平台 | https://github.com/vllm-project/aibrix | ByteDance 平台 |
| 平台 | https://github.com/ai-dynamo/dynamo | NVIDIA PD 分离 |
| 标准 | https://github.com/kubernetes-sigs/gateway-api-inference-extension | GIE 官方 |

### 其他

- vLLM 官方文档：https://docs.vllm.ai/
- SGLang 文档：https://sgl-project.github.io/
- NVIDIA Inference Microservice (NIM)：商业方案，但架构可学
- 论文跟踪：[Awesome-LLM-Inference](https://github.com/DefTruth/Awesome-LLM-Inference)

## 完成全部 8 阶段后能做什么

- 面试时能 30 分钟讲清楚「一次 LLM 请求从用户到 GPU 再回到用户的完整链路」
- 能在 K8s 上从零搭一个生产级的 LLM 推理平台
- 能 review 一个 vLLM PR 或给 GIE / llm-d / AIBrix 提 issue
- 能根据 workload 特性（高 QPS 短 prompt / 低 QPS 长 prompt / agent 多轮 / RAG）给出合理的部署方案
- 能解释什么时候 PD 该分、什么时候不该分

## 面试要点

### Q：为什么 LLM 推理要 KV Cache？

> [!question]- 参考答案（点击展开）
>
> Decode 每生成 1 个 token 都要对所有历史 token 做 attention，没 cache 就要重算 O(N²)，有了 cache 是 O(N)。代价是显存——KV 大小 = 2 × layer × head × dim × seq_len × dtype，128k context 一个 sequence 就 GB 级。

### Q：PagedAttention 解决了什么？

> [!question]- 参考答案（点击展开）
>
> 传统 KV 按 max_seq_len 预留连续显存，浪费严重（30-60% 碎片）。PagedAttention 把 KV 切 block（默认 16 token），像 OS page 一样按需分配，碎片率压到个位数，并发数翻几倍。

### Q：Continuous Batching 与传统 batching 的本质差异？

> [!question]- 参考答案（点击展开）
>
> 传统按 batch 等齐：所有 seq 必须同时开始同时结束，慢的拖累快的。Continuous batching 是 iteration-level scheduling：每个 iteration 决定哪些 seq 参与，刚结束的立刻让位给新请求，GPU 利用率从 30% 拉到 70%+。

### Q：vLLM 的 prefix cache 怎么命中？

> [!question]- 参考答案（点击展开）
>
> 按 block 算 hash（基于 block 内 token + 父 block hash），新请求 prefill 时按 block 查 hash 表，命中的 block 直接复用引用计数 +1，跳过这部分的 prefill 计算。block size 是 hash 粒度的下限，所以差 1 个 token 也可能 miss 整个 block。

### Q：SGLang RadixAttention 比 vLLM 强在哪？

> [!question]- 参考答案（点击展开）
>
> Radix tree 上做最长前缀匹配，比固定 block hash 粒度更细，多分支共享（多轮对话、agent）命中率明显高。代价是树维护 + 并发锁更复杂。

### Q：PD 分离解决什么？

> [!question]- 参考答案（点击展开）
>
> Prefill 是 compute-bound、Decode 是 memory-bound，混跑时长 prompt 占满 GPU 导致 decode ITL 抖动。拆开后各自的 batch 策略、并行策略、扩缩容独立优化，TTFT 和 ITL 同时改善。代价是要传 KV，靠 RDMA / NVLink。

### Q：PD 比例怎么定？

> [!question]- 参考答案（点击展开）
>
> 取决于 workload：长 prompt + 短输出 → Prefill 多；短 prompt + 长输出（agent / 写作）→ Decode 多。Dynamo 的 planner、Mooncake 的 conductor 都能动态调整。生产经验常见 1P:2D ~ 1P:4D。

### Q：跨实例怎么共享 KV？

> [!question]- 参考答案（点击展开）
>
> 两条路：① 走分布式 KV Store（Mooncake / LMCache），用 redis/etcd 注册位置，RDMA 拉；② PD 分离的 KV transfer，Prefill 算完直接 push 到 Decode。瓶颈在 PCIe / RDMA 带宽和 layer-wise 重叠程度。

### Q：LLM Router 为什么不能复用普通 HTTP LB？

> [!question]- 参考答案（点击展开）
>
> 4 个本质不同：① 请求大小差几个量级（128 vs 32k token）；② 处理时长秒级 vs 毫秒级；③ 有状态（KV/LoRA/模型版本）；④ 路由维度多了 prefix hash、KV 位置、Pool 类型。所以要 prefix-aware + load-aware + KV-aware。

### Q：KV-aware autoscaler 跟普通 HPA 区别？

> [!question]- 参考答案（点击展开）
>
> HPA 看 CPU/QPS，LLM 要看 KV cache 占用率、queue 长度、TTFT P99。AIBrix / Dynamo 都自定义了 metrics，关键挑战是 KV 占用是 stateful 的——缩容前要把 KV 迁出去或重算，不能直接 kill pod。

### Q：GIE（Gateway API Inference Extension）是什么？

> [!question]- 参考答案（点击展开）
>
> K8s 官方推的 LLM 推理 Gateway 标准，定义 InferenceModel / InferencePool CRD，配合 Gateway API HTTPRoute。endpoint picker plugin（epp）做路由决策，scheduling chain 复用 K8s scheduler framework 的 filter/scorer 思路。llm-d 等基于它。

### Q：大模型推理面试常被问的最难问题？

> [!question]- 参考答案（点击展开）
>
> "你怎么诊断一个推理服务 P99 TTFT 突然变高？" —— 检查路径：① 看 prefix cache 命中率掉了没；② 看 KV cache 占用率是否接近满（导致 preempt）；③ 看 prefill chunk size 是否被某些长 prompt 拖住；④ 看 router 路由分布是否倾斜；⑤ 看 GPU 利用率是不是因为 batch 抖动忽高忽低；⑥ 看 KV transfer 链路（PD 分离场景）。
