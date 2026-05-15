// identity.go —— 实现 csi.IdentityServer。
//
// Identity service 必须由 Controller Pod 和 Node Pod 都实现：sidecar 在调
// Controller / Node service 之前会先 Probe，确认插件已经就绪。
package main

import (
	"context"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// identityServer embed Unimplemented，未实现的 RPC 自动返回 Unimplemented。
// 这是 gRPC 推荐的 forward-compatible 模式：CSI spec 后续加新方法时，
// 我们不需要立即去实现就能保持编译通过。
type identityServer struct {
	csi.UnimplementedIdentityServer
}

func newIdentityServer() *identityServer { return &identityServer{} }

// GetPluginInfo 返回驱动名 + 版本号。
// driverName 是 K8s 侧的"全局唯一身份"——StorageClass.provisioner、
// CSIDriver 对象名、VolumeAttachment.spec.attacher 都用这个名字。
func (s *identityServer) GetPluginInfo(_ context.Context, _ *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{
		Name:          driverName,
		VendorVersion: driverVersion,
	}, nil
}

// GetPluginCapabilities 告诉 K8s 本插件支持哪些上层 service。
// 教学骨架声明两条：
//   - CONTROLLER_SERVICE：实现了 Controller service（CreateVolume / DeleteVolume）
//   - VOLUME_ACCESSIBILITY_CONSTRAINTS：未声明（hostPath 不需要拓扑感知）
func (s *identityServer) GetPluginCapabilities(_ context.Context, _ *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

// Probe 健康检查。返回 ready=true 表示插件可以接受请求；
// livenessprobe sidecar 会周期性调它，失败则让 kubelet 重启 Pod。
func (s *identityServer) Probe(_ context.Context, _ *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{
		Ready: &wrapperspb.BoolValue{Value: true},
	}, nil
}
