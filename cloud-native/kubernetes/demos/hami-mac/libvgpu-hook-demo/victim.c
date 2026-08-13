// victim.c — 故意不停申请内存的进程, 用来验证 malloc-limit hook 是否生效.
// 没 hook 时会一路 malloc 直到系统 OOM, 加了 hook 会在 MEM_LIMIT_MB 处停下.
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

int main(void) {
    size_t total = 0;
    while (1) {
        void *p = malloc(1024 * 1024); // 每次申请 1MB
        if (!p) {
            fprintf(stderr, "victim: malloc returned NULL at total=%zu MB\n",
                    total / (1024 * 1024));
            return 1;
        }
        total += 1024 * 1024;
        printf("victim: allocated %zu MB\n", total / (1024 * 1024));
        usleep(50 * 1000);
    }
}
