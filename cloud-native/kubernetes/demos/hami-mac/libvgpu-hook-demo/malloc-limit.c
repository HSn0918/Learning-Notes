// malloc-limit.c — HAMi libvgpu 简化原型: 用 LD_PRELOAD 拦截 malloc, 按配额拒绝.
//
// 对应 HAMi 在 libvgpu.so 里 hook cuMemAlloc 的核心思想:
//   超过 CUDA_DEVICE_MEMORY_LIMIT_X 时返回 CUDA_ERROR_OUT_OF_MEMORY.
//
// 平台说明 (重要):
//   - Linux: LD_PRELOAD 拦截 malloc 完美工作 (libc malloc 走全局符号).
//     编译: gcc -shared -fPIC malloc-limit.c -o malloc-limit.so -ldl
//     使用: MEM_LIMIT_MB=10 LD_PRELOAD=./malloc-limit.so ./victim
//   - macOS: DYLD_INSERT_LIBRARIES 拦截 malloc 通常 *失败*. 因为 macOS libSystem
//     的 malloc 走 default_zone->malloc 函数指针, 不经全局 "malloc" 符号 ——
//     这是 macOS 拦 malloc 出了名的坑. 想验证可以 hook getenv 或 puts.
//     真实 HAMi 跑在 Linux 容器里, 所以不受此限制.
//     正确的 Mac 学习方式: 用 docker 跑 Linux 镜像, 在里面演示 hook,
//     见同目录 run-in-docker.sh.

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <dlfcn.h>
#include <stdatomic.h>

static atomic_size_t g_used = 0;
static size_t g_limit = 0;
static void *(*real_malloc)(size_t) = NULL;

__attribute__((constructor))
static void init_hook(void) {
    real_malloc = dlsym(RTLD_NEXT, "malloc");
    const char *env = getenv("MEM_LIMIT_MB");
    g_limit = env ? (size_t)atoll(env) * 1024 * 1024 : 0;
    fprintf(stderr, "[hook] malloc limit = %zu bytes (%s MB)\n",
            g_limit, env ? env : "unlimited");
}

void *malloc(size_t size) {
    if (!real_malloc) return NULL;
    size_t now = atomic_load(&g_used);
    if (g_limit && now + size > g_limit) {
        fprintf(stderr, "[hook] DENY malloc(%zu): used=%zu limit=%zu (simulated cuMemAlloc OOM)\n",
                size, now, g_limit);
        errno = ENOMEM;
        return NULL;
    }
    void *p = real_malloc(size);
    if (p) atomic_fetch_add(&g_used, size);
    return p;
}
