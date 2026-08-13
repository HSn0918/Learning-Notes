# demos 验证状态

> **2026-05 最近一次本地验证**：Mac (Darwin 25.2.0) + Go 1.26
> 维护提示：动 demo 后跑「最后验证命令」，再更新本表。

| demo | 编译 | Mac 运行 | 前置依赖 | 最后验证命令 | 备注 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| [sample-controller](sample-controller/) | ✅ | ✅ | `go mod tidy` + kubeconfig | `cd sample-controller && go build ./...` | 5 分钟反馈，最快入门 |
| [device-plugin](device-plugin/) | ✅ | ⚠️ | `go mod tidy`、需 kind/真集群 | `cd device-plugin && go build ./...` | DaemonSet 部署，Mac kind 可跑 |
| [fake-cri](fake-cri/) | ✅ | ✅ | `go mod tidy`、`crictl` | `cd fake-cri && go build ./...` | 最小 fake CRI server，`crictl` 直接探测 |
| [fake-gpu](fake-gpu/) | ✅ | ⚠️ | `go mod tidy`、kind | `cd fake-gpu && go build ./...` | 同上 |
| [hami-mac](hami-mac/) | ✅ | ⚠️ | docker build + kind load | `cd hami-mac && go build ./...` | 看 README，含 walkthrough |
| [kubebuilder-operator](kubebuilder-operator/) | ✅ | ✅ | `go mod tidy`、kubeconfig | `cd kubebuilder-operator && go build ./...` | 升 controller-runtime v0.19.3 已修复 |
| [csi-hostpath](csi-hostpath/) | ⚠️ | ❌ | `GOOS=linux go build`，需 Linux 节点 | `cd csi-hostpath && GOOS=linux go build ./...` | 用 Linux-only `unix.MS_BIND`，Mac 编不过；kind 节点里跑 |
| [cni-bridge](cni-bridge/) | N/A | ✅ | docker | `cd cni-bridge && ./run-in-docker.sh` | bash 实现，Mac 通过 Docker 跑 |
| [scheduler-plugin](scheduler-plugin/) | ⚠️ | ⚠️ | 手填 `replace` 列表 | `cd scheduler-plugin && go build ./...` | 详见 README「编译说明」；推荐 fork scheduler-plugins 仓用其 go.mod |
| [raftexample-walkthrough/etcd-client-demo](raftexample-walkthrough/etcd-client-demo/) | ✅ | ✅ | `go mod tidy`、需 etcd | `cd raftexample-walkthrough/etcd-client-demo && go build ./...` | 跑前 `etcd` 起本地实例 |

## 通用前置

```bash
# 首次构建任一 Go demo 都先：
cd cloud-native/kubernetes/demos/<name>
go mod tidy
go build ./...
```

不跑 `go mod tidy` 会报 `missing go.sum entry` —— 这是 K8s 子模块依赖太多、初始 go.sum 没法手列全的副作用，不是 demo 损坏。

## 图例

- ✅ = 在 Mac 直接通过
- ⚠️ = 需要额外步骤（已注明）
- ❌ = Mac 上不可行（需 Linux 环境）
- N/A = 非 Go demo

## 排错快查

| 报错 | 原因 | 解决 |
| :--- | :--- | :--- |
| `missing go.sum entry` | go.sum 不全 | `go mod tidy` |
| `undefined: unix.MS_BIND`（csi-hostpath） | Mac 没有 Linux syscall 常量 | `GOOS=linux go build`，或在 kind 节点容器里编 |
| `not enough arguments in call to fn`（kubebuilder-operator 旧版） | controller-runtime v0.18 与 k8s.io v0.31 API 不兼容 | 已升 v0.19.3，重新 `go mod tidy` |
| `package k8s.io/kubernetes/pkg/scheduler/framework: ... v0.0.0`（scheduler-plugin） | K8s 主仓 staging 占位版本 | 手填 go.mod 的 replace 列表，见该 demo README |
