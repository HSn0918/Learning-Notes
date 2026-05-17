# libvgpu-hook-demo

50 行 C 代码版的 HAMi libvgpu 简化原型 —— 用 `LD_PRELOAD`（Linux）/ `DYLD_INSERT_LIBRARIES`（Mac）拦截 `malloc`，把 HAMi 在 `libvgpu.so` 里 hook `cuMemAlloc` 的核心思想"按配额拒绝分配"做出来。

配套阅读：
- [[../../../hami-learning-path]] 「阶段 6」
- [[../demo-hami-mac]]（外层 device-plugin demo，本子目录是它的"内层"故事）

## 它能让你看到什么

```
[hook] malloc limit = 10485760 bytes (10 MB)
victim: allocated 1 MB
victim: allocated 2 MB
...
victim: allocated 10 MB
[hook] DENY malloc(1048576): used=10485760 limit=10485760 (simulated cuMemAlloc OOM)
victim: malloc returned NULL at total=10 MB
```

把 `malloc` 换成 `cuMemAlloc`，把 `MEM_LIMIT_MB` 换成 `CUDA_DEVICE_MEMORY_LIMIT_X`，**就是 HAMi libvgpu 的精神**。区别只在于：

- `cuMemAlloc` 是 CUDA Driver API，hook 它要 dlopen `libcuda.so` 拿 real symbol
- HAMi 还要 hook `nvmlDeviceGetMemoryInfo` 让 `nvidia-smi` 看到假数据
- HAMi 还要 hook `cuLaunchKernel` 实现"算力配额"（sleep 一段）

但拦截机制完全一致。

## 在 Mac 上跑（推荐：用 docker 起 Linux 容器）

Mac 的 libSystem malloc 不走全局符号，`DYLD_INSERT_LIBRARIES` 拦不住。**所以最佳学习方式是用 docker 拉个 Linux 镜像在里面跑**——反正 HAMi 也只在 Linux 容器里有用。

```bash
cd learning-plan/demos/hami-mac/libvgpu-hook-demo
./run-in-docker.sh
# 或: make docker
```

期望输出：

```
[hook] malloc limit = 5242880 bytes (5 MB)
victim: allocated 1 MB
victim: allocated 2 MB
victim: allocated 3 MB
victim: allocated 4 MB
[hook] DENY malloc(1048576): used=5242880 limit=5242880 (simulated cuMemAlloc OOM)
victim: malloc returned NULL at total=5 MB
```

## 在 Linux 上直接跑

```bash
gcc -shared -fPIC malloc-limit.c -o malloc-limit.so -ldl
gcc victim.c -o victim
MEM_LIMIT_MB=5 LD_PRELOAD=./malloc-limit.so ./victim
```

## 关键点

1. **`dlsym(RTLD_NEXT, "malloc")` 拿到 libc 真 malloc**：HAMi 同样的套路从 `libcuda.so` 拿 `cuMemAlloc`。
2. **构造函数 `__attribute__((constructor))`**：进程加载本 .so 时立即跑一次，提前解析 real symbol 并读 env。HAMi libvgpu 在 constructor 里读 `CUDA_DEVICE_MEMORY_LIMIT_*` 和 `CUDA_DEVICE_SM_LIMIT_*` 这些 env，这就是 [[../demo-hami-mac]] 那个 Allocate 注入 env 的接收端。
3. **`atomic_size_t` 累计 used**：单进程多线程下避免 race。真实 HAMi 多容器共享一卡，统计放在 `/dev/shm/` 共享内存里跨进程协商。
4. **不 hook free**：本 demo 故意省掉，让你看到"配额单调增长"的纯净效果。真实场景必须 hook `cuMemFree` 才能正确减账。

## 与上层 device-plugin demo 的关系

```
demos/hami-mac/                       <- 外层: K8s 这一层做的事
├── plugin.go: Allocate 注入 LD_PRELOAD=/usr/local/vgpu/libvgpu.so
└── plugin.go: Allocate 注入 CUDA_DEVICE_MEMORY_LIMIT_0=3000m
                                        ↓
                                       env 进入容器
                                        ↓
                  ld.so / dyld 加载 LD_PRELOAD 指向的 .so
                                        ↓
demos/hami-mac/libvgpu-hook-demo/     <- 内层: .so 这一层做的事
├── malloc-limit.c: constructor 读 MEM_LIMIT_MB env
└── malloc-limit.c: hook malloc, 超额返回 NULL
```

把这两层接起来 = 简化版 HAMi。真版唯一多的是把 `malloc` 换成 `cuMemAlloc`，需要真 NVIDIA driver。
