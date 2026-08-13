#!/usr/bin/env bash
# 在 docker 容器里演示 learning-bridge CNI 插件全流程:
#   1) 模拟 2 个 Pod (两个 netns) 调 ADD
#   2) 互 ping 验证连通
#   3) 调 DEL 清理
#
# 为什么用 docker:
#   - Mac 没有 netns / ip netns / iptables, 本来就跑不了 CNI.
#   - 用 ubuntu 容器开 --privileged 借宿主机 kernel 跑全套 net 工具.

set -euo pipefail
cd "$(dirname "$0")"

docker run --rm -it --privileged \
  -v "$PWD":/work -w /work \
  ubuntu:22.04 bash -c '
    set -e
    apt-get update -qq && apt-get install -qq -y iproute2 iptables jq iputils-ping >/dev/null

    chmod +x learning-bridge

    NETCONF="\''{\"cniVersion\":\"0.4.0\",\"name\":\"learn\",\"type\":\"learning-bridge\",\"subnet\":\"10.244.0.0/24\",\"gateway\":\"10.244.0.1\",\"bridge\":\"cni0\"}\''"

    add_pod() {
        local id=$1
        local ns="pod-$id"
        ip netns add "$ns"
        echo
        echo "=== ADD $ns ==="
        # 通过 env 模拟 kubelet 调 CNI 的方式
        CNI_COMMAND=ADD \
        CNI_CONTAINERID=$id \
        CNI_NETNS=/run/netns/$ns \
        CNI_IFNAME=eth0 \
        CNI_PATH=$PWD \
        ./learning-bridge <<< "$(eval echo $NETCONF)"
    }

    del_pod() {
        local id=$1
        local ns="pod-$id"
        echo
        echo "=== DEL $ns ==="
        CNI_COMMAND=DEL \
        CNI_CONTAINERID=$id \
        CNI_NETNS=/run/netns/$ns \
        CNI_IFNAME=eth0 \
        ./learning-bridge <<< "$(eval echo $NETCONF)"
        ip netns del "$ns" || true
    }

    add_pod aaaaaaaa1111
    add_pod bbbbbbbb2222

    echo
    echo "=== ip a in pod-aaaaaaaa1111 ==="
    ip netns exec pod-aaaaaaaa1111 ip -4 a show eth0
    echo "=== ip a in pod-bbbbbbbb2222 ==="
    ip netns exec pod-bbbbbbbb2222 ip -4 a show eth0

    echo
    echo "=== 互 ping (pod-a -> pod-b) ==="
    PEER_IP=$(ip netns exec pod-bbbbbbbb2222 ip -4 -o a show eth0 | awk "{print \$4}" | cut -d/ -f1)
    ip netns exec pod-aaaaaaaa1111 ping -c 3 -W 1 "$PEER_IP"

    echo
    echo "=== 出网 (pod-a -> 1.1.1.1) ==="
    ip netns exec pod-aaaaaaaa1111 ping -c 2 -W 1 1.1.1.1 || echo "(可能 docker 网络出不去, 不影响演示)"

    echo
    echo "=== 调试日志 (插件 stderr) ==="
    cat /tmp/learning-bridge.log

    del_pod aaaaaaaa1111
    del_pod bbbbbbbb2222
    echo
    echo "=== 清理后宿主机网桥状态 ==="
    ip a show cni0 || true
  '
