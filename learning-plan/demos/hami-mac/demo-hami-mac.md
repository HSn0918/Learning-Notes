#kubernetes #gpu #hami #device-plugin #demo

相关笔记：[[hami-learning-path]] | [[hami-source]] | [[demo-fake-gpu]] | [[gpu-scheduling-source]] | [[gpu-scheduling]] | [[controller-runtime-source]]

## 概述

本 demo 在 [[demo-fake-gpu]] 的基础上升级，专门用来在 Mac（无 NVIDIA GPU）+ kind 上复现 **HAMi device-plugin 的两件关键事**：

1. **1 张物理卡 → N 份 vGPU 切片** 的 `ListAndWatch` 上报
2. **`Allocate` 注入 HAMi 风格的配额契约 env**：`LD_PRELOAD` + `CUDA_DEVICE_MEMORY_LIMIT_X` + `CUDA_DEVICE_SM_LIMIT_X`

为什么这么定位：HAMi 由 webhook + scheduler-extender + device-plugin + libvgpu.so 四件套组成（详见 [[hami-learning-path]]），其中**前三件都在 K8s 生态内可复现**，第四件 libvgpu.so 必须在真实 NVIDIA driver 上才能 hook CUDA API。所以一个 Mac 友好的 HAMi 学习 demo，最高 ROI 的做法就是把 device-plugin 这块的"对外契约"做出来，让你在 `kubectl logs` 里直接看到 HAMi 之所以是 HAMi 的两个关键 env：`LD_PRELOAD` 和 `CUDA_DEVICE_*_LIMIT_X`。

跑测步骤与详细 walkthrough 见 [README](./README.md)。

## 设计要点

1. **资源名沿用 `nvidia.com/gpu`**：方便后面接真实 HAMi 控制面（webhook/scheduler 都按 nvidia.com 这套 GroupName 处理）。
2. **vGPU 切片 ID 编码物理卡信息**：`GPU-{phys-uuid}-vgpu-{i}`，HAMi-scheduler 才能按 prefix 把同卡的切片聚合到一起做 binpack/spread。
3. **`NVIDIA_VISIBLE_DEVICES` 只放物理 UUID 集合**：从切片 ID 反解出 phys-UUID 去重后写入，这是 nvidia-container-runtime 真正能用的格式。
4. **CUDA 配额按容器内 device index 编号**：`CUDA_DEVICE_MEMORY_LIMIT_0` / `_1`，与 HAMi libvgpu 的读取约定一致。多卡 Pod 时 X 从 0 数到 N-1。
5. **故意不模拟 webhook**：给一个 `HAMI_MAC_DEFAULT_MEM` 进程级默认值就够了，省得引入 cert + ValidatingWebhookConfiguration 等繁琐 yaml；如果真需要，[[controller-runtime-source]] 第 8 节有完整路径。

## 与 [[demo-fake-gpu]] 的差异表

| 维度 | demo-fake-gpu | demo-hami-mac |
| :--- | :--- | :--- |
| 资源名 | `fake-gpu.k8s.io/gpu` | `nvidia.com/gpu` |
| Node capacity | = 物理卡数（4） | = 物理卡数 × 切片数（4×10=40） |
| Allocate 注入 env | NVIDIA_VISIBLE_DEVICES | + LD_PRELOAD + CUDA_DEVICE_MEMORY_LIMIT_X + CUDA_DEVICE_SM_LIMIT_X |
| 学习对象 | NVIDIA Device Plugin 基本 SHAPE | HAMi 之所以是 HAMi 的两件事 |

## 在 [[hami-learning-path]] 中的位置

| HAMi 学习阶段 | 本 demo 提供什么 |
| :--- | :--- |
| 阶段 0–1（先决条件 / 跑通最小集群） | Mac 上的最小可跑替代品，免买 GPU |
| 阶段 3（webhook 源码） | 反例：演示"如果没有 webhook 兜底，default mem 怎么落下来" |
| 阶段 4（scheduler extender 源码） | 可作为下游接口：让真 HAMi-scheduler 跑在 kind，后端接这个 plugin |
| 阶段 5（device-plugin 源码） | **本阶段的产出物**，对照 HAMi 源码读 vGPU 切片 + env 注入 |
| 阶段 6（libvgpu hook） | 不能替代；阶段 6 必须上云租 GPU 才能做 |

## 局限与陷阱

1. **`LD_PRELOAD` 指向的文件在容器里不存在**：busybox 里 `/usr/local/vgpu/libvgpu.so` 没有；某些情况下会让 `sh` 启动报 "error while loading shared libraries"。本 demo 用 `sh -c` 包了一层，不影响 echo env。如果你要复用做严肃测试，要么把 LD_PRELOAD 值改空，要么镜像里塞一个空 .so 占位。
2. **不演示「同卡共享 + 显存隔离」的实际效果**：那需要真 libvgpu.so。本 demo 只能让你看到「同一物理卡的两个不同 vGPU 切片分给两个 Pod」这一步，配额是不是真生效要看 hook。
3. **kubelet DeviceManager 随便挑切片，没有 binpack/spread**：那是 HAMi-scheduler 的活，本 demo 不带 scheduler。如果要看 binpack 选卡策略，需要部署真 HAMi-scheduler 或自己写 framework plugin（见 [[demo-scheduler-plugin]]）。
