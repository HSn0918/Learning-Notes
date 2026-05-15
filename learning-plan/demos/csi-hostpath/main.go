// csi-hostpath demo —— 一个最小可运行的 hostPath 风格 CSI 驱动骨架。
//
// 启动后在 /csi/csi.sock 上监听 gRPC 请求，并注册 Identity / Node / Controller 三个 service。
// 在 Linux 节点上，NodePublishVolume 会把 hostPath 源目录 bind mount 到 kubelet 给的
// targetPath，模拟一个真实 CSI driver 的 "publish" 行为。
//
// 命令行用法：
//
//	./csi-hostpath \
//	    --endpoint=unix:///csi/csi.sock \
//	    --node-id=$(hostname) \
//	    --data-root=/var/lib/csi-hostpath
//
// 实际部署详见 daemonset.yaml。
package main

import (
	"context"
	"flag"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

const (
	driverName    = "learning-plan.csi.k8s.io"
	driverVersion = "0.1.0"
)

func main() {
	klog.InitFlags(nil)

	endpoint := flag.String("endpoint", "unix:///csi/csi.sock",
		"CSI endpoint URL (unix domain socket)")
	nodeID := flag.String("node-id", "",
		"node ID returned in NodeGetInfo, 一般取 $(hostname)")
	dataRoot := flag.String("data-root", "/var/lib/csi-hostpath",
		"本地存放 hostPath 卷源目录的根路径")
	flag.Parse()

	if *nodeID == "" {
		klog.Fatal("--node-id is required")
	}
	if err := os.MkdirAll(*dataRoot, 0750); err != nil {
		klog.Fatalf("failed to create data root %s: %v", *dataRoot, err)
	}

	// 解析 unix:///csi/csi.sock，监听对应 socket。
	network, addr, err := parseEndpoint(*endpoint)
	if err != nil {
		klog.Fatalf("parse endpoint: %v", err)
	}
	if network == "unix" {
		// 启动前清理残留 socket 文件，否则 bind 会报 address already in use。
		_ = os.Remove(addr)
		if err := os.MkdirAll(strings.TrimSuffix(addr, "/csi.sock"), 0755); err != nil {
			klog.Fatalf("ensure socket dir: %v", err)
		}
	}

	lis, err := net.Listen(network, addr)
	if err != nil {
		klog.Fatalf("listen %s: %v", *endpoint, err)
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(logInterceptor),
	)
	csi.RegisterIdentityServer(server, newIdentityServer())
	csi.RegisterNodeServer(server, newNodeServer(*nodeID, *dataRoot))
	csi.RegisterControllerServer(server, newControllerServer(*dataRoot))

	// 信号处理：SIGTERM 时优雅停止，给进行中的 RPC 留时间。
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		klog.Info("received shutdown signal, gracefully stopping gRPC server")
		server.GracefulStop()
	}()

	klog.Infof("CSI driver %q v%s listening on %s (node=%s)",
		driverName, driverVersion, *endpoint, *nodeID)
	if err := server.Serve(lis); err != nil {
		klog.Fatalf("grpc serve: %v", err)
	}
}

func parseEndpoint(ep string) (network, addr string, err error) {
	u, err := url.Parse(ep)
	if err != nil {
		return "", "", err
	}
	switch u.Scheme {
	case "unix":
		return "unix", u.Path, nil
	case "tcp":
		return "tcp", u.Host, nil
	default:
		// 兼容直接传裸路径：./csi-hostpath --endpoint=/tmp/csi.sock
		return "unix", ep, nil
	}
}

func logInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.V(4).Infof("GRPC call: %s req=%+v", info.FullMethod, req)
	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("GRPC %s failed: %v", info.FullMethod, err)
	} else {
		klog.V(4).Infof("GRPC %s ok resp=%+v", info.FullMethod, resp)
	}
	return resp, err
}
