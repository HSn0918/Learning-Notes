module github.com/learning-notes/csi-hostpath

go 1.26

// --- 重要：构建说明 ---
//
// 这是一个**教学骨架**。真正能在集群里跑生产负载请看
// https://github.com/kubernetes-csi/csi-driver-host-path —— 它的代码组织
// 是社区参考实现，本 demo 是它的极简化版本。
//
// 本 demo 实现：
//   - Identity service：GetPluginInfo / Probe / GetPluginCapabilities
//   - Node service：NodePublishVolume / NodeUnpublishVolume / NodeGetInfo / NodeGetCapabilities
//   - Controller service：CreateVolume / DeleteVolume / ControllerGetCapabilities（stub）
//
// 在 Linux 节点上能真实做 bind mount；macOS 上 unix.Mount 会返回错误属正常，
// 代码本身能编译通过（用了 golang.org/x/sys/unix 跨平台抽象）。

require (
	github.com/container-storage-interface/spec v1.9.0
	golang.org/x/sys v0.20.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.1
	k8s.io/klog/v2 v2.130.1
)

// 间接依赖（go mod tidy 会自动补齐）：
//   - github.com/golang/protobuf
//   - golang.org/x/net
//   - golang.org/x/text
//   - google.golang.org/genproto/googleapis/rpc
