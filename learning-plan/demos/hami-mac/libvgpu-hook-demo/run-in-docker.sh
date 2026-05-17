#!/usr/bin/env bash
# 在 Linux 容器里编 + 跑 malloc hook demo, Mac 用户用这个验证.
# Mac 上 libSystem 的 malloc 不走全局符号, DYLD_INSERT_LIBRARIES 拦不住,
# 所以最佳学习方式是用 docker 拉个 ubuntu 镜像在 Linux 环境里跑.
set -euo pipefail

cd "$(dirname "$0")"

docker run --rm -it \
  -v "$PWD":/work -w /work \
  gcc:12 bash -c '
    set -e
    echo "--- 编译 ---"
    gcc -shared -fPIC malloc-limit.c -o malloc-limit.so -ldl
    gcc victim.c -o victim
    echo
    echo "--- 跑带 hook (上限 5MB), 应该在 ~5MB 处 deny + NULL ---"
    MEM_LIMIT_MB=5 LD_PRELOAD=./malloc-limit.so ./victim || true
    echo
    echo "--- 跑不带 hook (sanity check, 跑 2 秒 Ctrl+C 模拟) ---"
    timeout 2 ./victim || true
  '
