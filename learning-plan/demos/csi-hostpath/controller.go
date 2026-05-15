// controller.go —— 实现 csi.ControllerServer 的最小 stub。
//
// 本 demo 的 Controller service 只为了让 external-provisioner sidecar 能"看到"
// 这是个支持 dynamic provisioning 的驱动；实际"创建卷"的动作很轻——
// 就是在本节点的 dataRoot 下 mkdir 一个目录，模拟 hostPath 卷的"卷源"。
// 在真实远端存储里，CreateVolume 会调云厂商 API / Ceph monitor 创建块设备。
package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

type controllerServer struct {
	csi.UnimplementedControllerServer

	dataRoot string
}

func newControllerServer(dataRoot string) *controllerServer {
	return &controllerServer{dataRoot: dataRoot}
}

// CreateVolume 必须**幂等**：相同 req.Name 多次调用必须返回同一个 volume_id。
// external-provisioner 用 PVC UID 算出稳定的 req.Name，重试时会带同样的 name 过来。
// 这里我们直接把 name 哈希成 volumeId，并 MkdirAll 创建源目录——天然幂等。
func (s *controllerServer) CreateVolume(_ context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "Name missing")
	}
	if len(req.GetVolumeCapabilities()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "VolumeCapabilities missing")
	}

	volumeID := hashVolumeID(req.GetName())
	source := filepath.Join(s.dataRoot, volumeID)

	if err := os.MkdirAll(source, 0750); err != nil {
		return nil, status.Errorf(codes.Internal, "create source dir %s: %v", source, err)
	}

	capacity := req.GetCapacityRange().GetRequiredBytes()
	if capacity == 0 {
		capacity = 1 << 30 // 默认 1Gi
	}

	klog.Infof("CreateVolume: name=%s volumeId=%s capacity=%d", req.GetName(), volumeID, capacity)
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: capacity,
			// VolumeContext 会被 K8s 透传给后续的 NodePublish/NodeStage，
			// 这里塞一个 source 字段方便排查。
			VolumeContext: map[string]string{
				"source": source,
			},
		},
	}, nil
}

// DeleteVolume：删除源目录。幂等：目录不存在视为成功。
func (s *controllerServer) DeleteVolume(_ context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "VolumeId missing")
	}

	source := filepath.Join(s.dataRoot, req.GetVolumeId())
	if err := os.RemoveAll(source); err != nil {
		return nil, status.Errorf(codes.Internal, "remove source dir %s: %v", source, err)
	}

	klog.Infof("DeleteVolume: volumeId=%s ok", req.GetVolumeId())
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerGetCapabilities 声明 Controller service 支持哪些能力。
// CREATE_DELETE_VOLUME 是 dynamic provisioning 必须；不声明 PUBLISH_UNPUBLISH_VOLUME
// 意味着不需要走 attach 阶段（hostPath 卷本来就在节点上，无 attach 概念）。
func (s *controllerServer) ControllerGetCapabilities(_ context.Context, _ *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	rpcCap := func(t csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
		return &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{Type: t},
			},
		}
	}
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			rpcCap(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME),
		},
	}, nil
}

func hashVolumeID(name string) string {
	h := sha1.Sum([]byte(name))
	return "vol-" + hex.EncodeToString(h[:8])
}
