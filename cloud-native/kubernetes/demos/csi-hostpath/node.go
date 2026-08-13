// node.go —— 实现 csi.NodeServer。
//
// Node service 跑在每个节点的 DaemonSet Pod 里，负责把卷"落地"到 Pod 能用到的路径。
// 本 demo 用 bind mount 把 dataRoot/<volumeId> 挂载到 kubelet 给的 targetPath，
// 模拟一个最简化的 hostPath 风格 CSI driver。
package main

import (
	"context"
	"os"
	"path/filepath"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

type nodeServer struct {
	csi.UnimplementedNodeServer

	nodeID   string
	dataRoot string // hostPath 卷源目录的根，所有 volume 在它下面有一个子目录
}

func newNodeServer(nodeID, dataRoot string) *nodeServer {
	return &nodeServer{nodeID: nodeID, dataRoot: dataRoot}
}

// NodePublishVolume：把卷源目录 bind mount 到 Pod 的 targetPath。
//
// 关键步骤：
//   1. 校验参数
//   2. 在 dataRoot 下确保有该 volume 的源目录（hostPath 风格驱动一般在
//      CreateVolume 时就 MkdirAll；这里为了 demo 容错也做一次）
//   3. MkdirAll(targetPath) —— kubelet 会要求 targetPath 已存在
//   4. syscall mount("none", targetPath, "", MS_BIND, "") 做 bind mount
//
// 幂等性：如果 targetPath 已经挂着同一个源，重复调直接返回成功（这里简化为
// "目录已存在且 mount 成功"——生产代码会读 /proc/mounts 严格判断）。
func (s *nodeServer) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeId missing")
	}
	if req.GetTargetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "TargetPath missing")
	}
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "VolumeCapability missing")
	}

	source := filepath.Join(s.dataRoot, req.GetVolumeId())
	if err := os.MkdirAll(source, 0750); err != nil {
		return nil, status.Errorf(codes.Internal, "ensure source dir %s: %v", source, err)
	}

	target := req.GetTargetPath()
	if err := os.MkdirAll(target, 0750); err != nil {
		return nil, status.Errorf(codes.Internal, "ensure target dir %s: %v", target, err)
	}

	// 标志位：ReadOnly 时加 MS_RDONLY。生产实现还要根据 access_type
	// （Block vs Mount）区分；hostPath demo 只支持 Mount。
	flags := uintptr(unix.MS_BIND)
	if req.GetReadonly() {
		flags |= unix.MS_RDONLY
	}

	// bind mount：fstype 留空，source 就是块设备或目录路径。
	// 在 Linux 节点上等价于 `mount --bind source target`。
	if err := unix.Mount(source, target, "", flags, ""); err != nil {
		// 如果已经挂着（EBUSY/EINVAL with already-mounted）就当幂等成功。
		// 真实驱动会用 mount-utils 的 IsLikelyNotMountPoint / IsNotMountPoint 判断。
		if err == unix.EBUSY {
			klog.V(4).Infof("NodePublishVolume: %s already mounted, idempotent return", target)
			return &csi.NodePublishVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "bind mount %s -> %s: %v", source, target, err)
	}

	klog.Infof("NodePublishVolume: %s -> %s ok", source, target)
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume：unmount + 删除 targetPath。
// 幂等性：未挂载 / 目录不存在都视为成功。
func (s *nodeServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeId missing")
	}
	target := req.GetTargetPath()
	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "TargetPath missing")
	}

	if err := unix.Unmount(target, 0); err != nil && err != unix.EINVAL && err != unix.ENOENT {
		return nil, status.Errorf(codes.Internal, "unmount %s: %v", target, err)
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		// 卷被多 Pod 引用时父目录可能还在，忽略 ENOTEMPTY。
		klog.V(4).Infof("NodeUnpublishVolume: rm %s (non-fatal): %v", target, err)
	}

	klog.Infof("NodeUnpublishVolume: %s ok", target)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetInfo 在 driver 注册阶段被 kubelet 调一次。
// 返回的 NodeId 是"驱动眼中的节点标识"——本 demo 直接用 hostname；
// 真实驱动如 EBS 会返回 EC2 instance-id（"i-1234..."）。
// AccessibleTopology 用于拓扑感知调度，hostPath 不需要，传 nil。
func (s *nodeServer) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId:            s.nodeID,
		MaxVolumesPerNode: 0, // 0 表示无限制
	}, nil
}

// NodeGetCapabilities 声明本 Node service 支持哪些可选 RPC。
// 本 demo 声明 STAGE_UNSTAGE_VOLUME 表示驱动愿意被走 stage 流程；
// 但因为 Node{Stage,Unstage}Volume 在 demo 里没实际实现（embed Unimplemented 返回 Unimplemented）,
// kubelet 真的调过来会失败——生产驱动要么实现它们，要么这里不声明该 capability。
// 本骨架特意把它注释掉，保持 demo 的可运行性；想完整复现"stage + publish 两步走"
// 自行去掉注释并实现两个方法即可。
func (s *nodeServer) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	caps := []*csi.NodeServiceCapability{
		// {
		// 	Type: &csi.NodeServiceCapability_Rpc{
		// 		Rpc: &csi.NodeServiceCapability_RPC{
		// 			Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		// 		},
		// 	},
		// },
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: caps}, nil
}
