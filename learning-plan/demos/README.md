# demos 验证状态

> **2026-05 最近一次本地验证**：Mac (Darwin 25.2.0) + Go 1.26
> 维护提示：动 demo 后跑 `cd <demo> && go build ./...` 验证，更新本表

| demo | 编译 | Mac 运行 | 前置依赖 | 备注 |
| :--- | :--- | :--- | :--- | :--- |
| [sample-controller](sample-controller/) | ✅ | ✅ | `go mod tidy` + kubeconfig | 5 分钟反馈，最快入门 |
| [device-plugin](device-plugin/) | ✅ | ⚠️ | `go mod tidy`、需 kind/真集群 | DaemonSet 部署，Mac kind 可跑 |
| [fake-gpu](fake-gpu/) | ✅ | ⚠️ | `go mod tidy`、kind | 同上 |
| [hami-mac](hami-mac/) | ✅ | ⚠️ | docker build + kind load | 看 README，含 walkthrough |
| [kubebuilder-operator](kubebuilder-operator/) | ✅ | ✅ | `go mod tidy`、kubeconfig | 升 controller-runtime v0.19.3 已修复 |
| [csi-hostpath](csi-hostpath/) | ⚠️ | ❌ | `GOOS=linux go build`，需 Linux 节点 | 用 Linux-only `unix.MS_BIND`，Mac 编不过；kind 节点里跑 |
| [cni-bridge](cni-bridge/) | N/A | ✅ | docker | bash 实现，`./run-in-docker.sh` 一键跑 |
| [scheduler-plugin](scheduler-plugin/) | ⚠️ | ⚠️ | 手填 `replace` 列表 | 详见 README「编译说明」；推荐 fork scheduler-plugins 仓用其 go.mod |
| [raftexample-walkthrough/etcd-client-demo](raftexample-walkthrough/etcd-client-demo/) | ✅ | ✅ | `go mod tidy`、需 etcd | 跑前 `etcd` 起本地实例 |

## 通用前置

```bash
# 首次构建任一 Go demo 都先：
cd learning-plan/demos/<name>
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
