// fake-cri demo —— 一个最小可运行的 fake CRI runtime server。
//
// 在 /tmp/fake-cri.sock 上监听 gRPC，注册 RuntimeService（用 in-memory map
// 假装管理 sandbox），让 crictl 能拿到 version 和 pods 列表。
//
// 用途：
//   - 学习 CRI gRPC server 的最小实现集
//   - 给 kubelet 单元测试或本地实验做 mock
//   - 验证 cri-tools / crictl 的行为
//
// 启动：
//
//	go run . [-socket=/tmp/fake-cri.sock]
//
// 探测：
//
//	crictl --runtime-endpoint unix:///tmp/fake-cri.sock version
//	crictl --runtime-endpoint unix:///tmp/fake-cri.sock pods
//	crictl --runtime-endpoint unix:///tmp/fake-cri.sock runp pod.json
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// fakeRuntime 实现 runtimeapi.RuntimeServiceServer 的一个最小子集。
// 嵌入 UnimplementedRuntimeServiceServer 让未实现的 RPC 自动返回 Unimplemented。
type fakeRuntime struct {
	runtimeapi.UnimplementedRuntimeServiceServer

	mu        sync.Mutex
	sandboxes map[string]*runtimeapi.PodSandbox // id -> sandbox
	counter   int
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{
		sandboxes: map[string]*runtimeapi.PodSandbox{},
	}
}

// Version 返回固定的 runtime name / version。
// kubelet 启动时 validateServiceConnection 会先调一次 Version 探测；
// crictl version 也调它。
func (f *fakeRuntime) Version(_ context.Context, req *runtimeapi.VersionRequest) (*runtimeapi.VersionResponse, error) {
	log.Printf("Version(req.Version=%q)", req.Version)
	return &runtimeapi.VersionResponse{
		Version:           "0.1.0", // kubelet runtime API version
		RuntimeName:       "fake-cri",
		RuntimeVersion:    "v0.0.1",
		RuntimeApiVersion: "v1",
	}, nil
}

// Status 返回 Runtime/Network ready=true，让 kubelet 把节点视为 healthy。
func (f *fakeRuntime) Status(_ context.Context, _ *runtimeapi.StatusRequest) (*runtimeapi.StatusResponse, error) {
	return &runtimeapi.StatusResponse{
		Status: &runtimeapi.RuntimeStatus{
			Conditions: []*runtimeapi.RuntimeCondition{
				{Type: runtimeapi.RuntimeReady, Status: true, Reason: "FakeReady"},
				{Type: runtimeapi.NetworkReady, Status: true, Reason: "FakeNetReady"},
			},
		},
	}, nil
}

// RunPodSandbox 在内存里登记一个 sandbox 对象，返回伪造的 id。
// 真正的 runtime 会启动 pause 容器、创建 netns、调 CNI 配网 —— 这里全跳过。
func (f *fakeRuntime) RunPodSandbox(_ context.Context, req *runtimeapi.RunPodSandboxRequest) (*runtimeapi.RunPodSandboxResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.counter++
	id := fmt.Sprintf("sandbox-%d-%d", time.Now().Unix(), f.counter)

	var metaName string
	if req.Config != nil && req.Config.Metadata != nil {
		metaName = req.Config.Metadata.Name
	}

	sb := &runtimeapi.PodSandbox{
		Id:             id,
		Metadata:       req.Config.GetMetadata(),
		State:          runtimeapi.PodSandboxState_SANDBOX_READY,
		CreatedAt:      time.Now().UnixNano(),
		Labels:         req.Config.GetLabels(),
		Annotations:    req.Config.GetAnnotations(),
		RuntimeHandler: req.RuntimeHandler,
	}
	f.sandboxes[id] = sb

	log.Printf("RunPodSandbox: id=%s pod=%s handler=%s", id, metaName, req.RuntimeHandler)
	return &runtimeapi.RunPodSandboxResponse{PodSandboxId: id}, nil
}

// StopPodSandbox 把 sandbox 状态翻成 NOTREADY；必须 idempotent。
func (f *fakeRuntime) StopPodSandbox(_ context.Context, req *runtimeapi.StopPodSandboxRequest) (*runtimeapi.StopPodSandboxResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sb, ok := f.sandboxes[req.PodSandboxId]; ok {
		sb.State = runtimeapi.PodSandboxState_SANDBOX_NOTREADY
		log.Printf("StopPodSandbox: id=%s", req.PodSandboxId)
	}
	return &runtimeapi.StopPodSandboxResponse{}, nil
}

// RemovePodSandbox 从内存 map 中删除。必须 idempotent。
func (f *fakeRuntime) RemovePodSandbox(_ context.Context, req *runtimeapi.RemovePodSandboxRequest) (*runtimeapi.RemovePodSandboxResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sandboxes, req.PodSandboxId)
	log.Printf("RemovePodSandbox: id=%s", req.PodSandboxId)
	return &runtimeapi.RemovePodSandboxResponse{}, nil
}

// PodSandboxStatus 查询单个 sandbox。crictl inspectp 用它。
func (f *fakeRuntime) PodSandboxStatus(_ context.Context, req *runtimeapi.PodSandboxStatusRequest) (*runtimeapi.PodSandboxStatusResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sb, ok := f.sandboxes[req.PodSandboxId]
	if !ok {
		return nil, fmt.Errorf("sandbox %q not found", req.PodSandboxId)
	}
	return &runtimeapi.PodSandboxStatusResponse{
		Status: &runtimeapi.PodSandboxStatus{
			Id:        sb.Id,
			Metadata:  sb.Metadata,
			State:     sb.State,
			CreatedAt: sb.CreatedAt,
			Network: &runtimeapi.PodSandboxNetworkStatus{
				Ip: "10.244.0.42", // 假 IP
			},
			Labels:         sb.Labels,
			Annotations:    sb.Annotations,
			RuntimeHandler: sb.RuntimeHandler,
		},
	}, nil
}

// ListPodSandbox 返回所有 sandbox。crictl pods 用它。
// filter 字段这里不实现，全量返回。
func (f *fakeRuntime) ListPodSandbox(_ context.Context, _ *runtimeapi.ListPodSandboxRequest) (*runtimeapi.ListPodSandboxResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := make([]*runtimeapi.PodSandbox, 0, len(f.sandboxes))
	for _, s := range f.sandboxes {
		list = append(list, s)
	}
	return &runtimeapi.ListPodSandboxResponse{Items: list}, nil
}

// ListContainers / ContainerStatus / ListImages 等 RPC 走
// UnimplementedRuntimeServiceServer 的默认实现（返回 Unimplemented），
// 对 crictl pods + crictl version 这两个最常用命令足够。

func main() {
	socket := flag.String("socket", "/tmp/fake-cri.sock",
		"unix socket path to listen on")
	flag.Parse()

	// 清理上次残留的 socket file —— 否则 Listen 会 EADDRINUSE。
	if err := os.Remove(*socket); err != nil && !os.IsNotExist(err) {
		log.Printf("warn: remove old socket: %v", err)
	}

	lis, err := net.Listen("unix", *socket)
	if err != nil {
		log.Fatalf("listen %s: %v", *socket, err)
	}
	// 让所有用户都能拨这个 socket —— 真实 runtime 通常会限制到 root，
	// 这里 demo 放宽方便 crictl 直连。
	_ = os.Chmod(*socket, 0666)

	server := grpc.NewServer()
	rt := newFakeRuntime()
	runtimeapi.RegisterRuntimeServiceServer(server, rt)

	// 信号处理：SIGTERM / SIGINT 时优雅停止。
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM, syscall.SIGINT)
		<-sigc
		log.Println("shutting down...")
		server.GracefulStop()
		_ = os.Remove(*socket)
	}()

	log.Printf("fake-cri listening on unix://%s", *socket)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
