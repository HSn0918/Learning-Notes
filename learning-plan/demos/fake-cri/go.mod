module github.com/learning-notes/fake-cri

go 1.22.0

// --- 教学骨架说明 ---
//
// 一个最小可运行的 fake CRI runtime，对应 learning-plan/cri-source.md
// 第九节"手写简化复现"。
//
// 实现：
//   - RuntimeService.Version
//   - RuntimeService.Status
//   - RuntimeService.RunPodSandbox / StopPodSandbox / RemovePodSandbox
//   - RuntimeService.PodSandboxStatus / ListPodSandbox
//
// 没实现的 RPC（CreateContainer / Exec / PullImage / ...）走嵌入的
// UnimplementedRuntimeServiceServer，自动返回 Unimplemented 错误，
// 对 crictl pods + crictl version 这两个最常用命令完全够用。

require (
	google.golang.org/grpc v1.65.0
	k8s.io/cri-api v0.31.0
)

require (
	github.com/gogo/protobuf v1.3.2 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
