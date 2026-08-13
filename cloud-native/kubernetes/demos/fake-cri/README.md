# fake-cri

一个最小可运行的 **fake CRI runtime server**，对应学习笔记 `cloud-native/kubernetes/internals/cri-source.md` 第九节。

## 这是什么

在 `/tmp/fake-cri.sock` 上监听 gRPC，实现 CRI `RuntimeService` 的最小 RPC 子集：

| RPC | 行为 |
|-----|------|
| `Version` | 返回固定字符串 `fake-cri v0.0.1` |
| `Status` | 返回 `RuntimeReady=true`、`NetworkReady=true` |
| `RunPodSandbox` | 在内存 map 里登记 sandbox，返回伪 id |
| `StopPodSandbox` | 标记 NOTREADY（必须 idempotent） |
| `RemovePodSandbox` | 从 map 删除（必须 idempotent） |
| `PodSandboxStatus` | 单个 sandbox 详情，含假 IP `10.244.0.42` |
| `ListPodSandbox` | 全量返回 |
| 其它 RPC | 走 `UnimplementedRuntimeServiceServer`，返回 Unimplemented |

足够让 `crictl pods` / `crictl version` / `crictl runp` 跑通。**不能跑真业务容器**——没实现 CreateContainer / StartContainer / Image service。

## 用途

- 学习 CRI gRPC server 的最小可工作集合（"三个 RPC 让 kubelet 进入 NodeReady"）
- 给 kubelet 单元测试或本地实验做 mock runtime
- 验证 crictl / cri-tools 的命令行行为
- 复现 CRI 协议相关的 bug

## 构建

```bash
cd cloud-native/kubernetes/demos/fake-cri
go mod tidy
go build -o fake-cri .
```

## 运行

```bash
./fake-cri
# 默认 socket: /tmp/fake-cri.sock
# 可以用 -socket=/path/to/sock 覆盖
```

输出：

```
2026/05/15 10:00:00 fake-cri listening on unix:///tmp/fake-cri.sock
```

## crictl 探测

### 安装 crictl

macOS:
```bash
brew install crictl
```

Linux:
```bash
VERSION="v1.31.1"
wget https://github.com/kubernetes-sigs/cri-tools/releases/download/$VERSION/crictl-$VERSION-linux-amd64.tar.gz
sudo tar zxvf crictl-$VERSION-linux-amd64.tar.gz -C /usr/local/bin
```

### 版本探测

```bash
crictl --runtime-endpoint unix:///tmp/fake-cri.sock version
```

预期输出：
```
Version:  0.1.0
RuntimeName:  fake-cri
RuntimeVersion:  v0.0.1
RuntimeApiVersion:  v1
```

### 列 sandbox（初始空列表）

```bash
crictl --runtime-endpoint unix:///tmp/fake-cri.sock pods
```

预期输出：
```
POD ID   CREATED   STATE   NAME   NAMESPACE   ATTEMPT   RUNTIME
```

### 创建一个 sandbox

准备 `pod.json`：

```json
{
  "metadata": {
    "name": "demo-pod",
    "namespace": "default",
    "uid": "00000000-0000-0000-0000-000000000001",
    "attempt": 0
  },
  "log_directory": "/tmp",
  "linux": {}
}
```

执行：

```bash
crictl --runtime-endpoint unix:///tmp/fake-cri.sock runp pod.json
```

预期输出（sandbox id）：
```
sandbox-1747280400-1
```

再列：

```bash
crictl --runtime-endpoint unix:///tmp/fake-cri.sock pods
```

```
POD ID                 CREATED          STATE   NAME      NAMESPACE   ATTEMPT   RUNTIME
sandbox-1747280400-1   3 seconds ago    Ready   demo-pod  default     0
```

`crictl inspectp sandbox-1747280400-1` 能看到伪 IP `10.244.0.42`。

### 停止 + 删除

```bash
crictl --runtime-endpoint unix:///tmp/fake-cri.sock stopp sandbox-1747280400-1
crictl --runtime-endpoint unix:///tmp/fake-cri.sock rmp sandbox-1747280400-1
```

## 实现要点（对照 cri-source.md）

| 章节 | 代码体现 |
|------|----------|
| § 二 gRPC 契约 | `RegisterRuntimeServiceServer` 在 `main()` 把 fakeRuntime 注册到 gRPC server |
| § 三 sandbox 模型 | `RunPodSandbox` 只创建 in-memory 状态——真实 runtime 这里要启 pause + 调 CNI |
| § 四 cri-client 连接 | server 用 `grpc.NewServer()` + unix socket，与客户端 `grpc.DialContext(unix://...)` 对接 |
| § 八 RPC idempotency | `StopPodSandbox` / `RemovePodSandbox` 找不到 sandbox 也返回成功（idempotent） |
| § 十一 面试 Q9 | 没实现的 RPC 走 Unimplemented，对应"CRI 是同步 unary"——客户端会立刻拿到 codes.Unimplemented |

## 与真实运行时的差距

| 维度 | 这个 fake | containerd CRI plugin |
|------|-----------|------------------------|
| sandbox 持久化 | 内存 map，重启丢 | bolt-db + label 反查 |
| 真容器进程 | 无 | 通过 shim → runc 起 |
| 网络 | 假 IP 字符串 | CNI ADD/DEL |
| 镜像 | 无 ImageService | content store + snapshotter |
| Exec | 无 | streaming HTTP server |
| 日志 | 无 | 落盘到 log_directory |

要扩展到能跑真容器：实现 `PullImage`（可以本地解压一个 tar）、`CreateContainer`（生成 OCI spec.json）、`StartContainer`（fork+exec runc create+start），那就成了一个最简化的"基于 runc 的 CRI 运行时"——这条路径上的开源项目可以参考 `iximiuz/labs` 或 `containers/podman` 早期版本。

## 相关笔记

- `cloud-native/kubernetes/internals/cri-source.md` —— 主笔记 CRI 源码导读
- `cloud-native/kubernetes/internals/kubelet-cri-source.md` —— kubelet 端的 syncLoop / PLEG / syncPod
- `cloud-native/kubernetes/internals/cni-source.md` —— sandbox 创建后 runtime 如何调 CNI
- `cloud-native/kubernetes/infrastructure/oci-runtime.md` —— runc / OCI runtime spec
