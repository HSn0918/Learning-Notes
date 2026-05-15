# csi-hostpath demo

一个最小可运行的 **hostPath 风格 CSI driver** 教学骨架。它把 dataRoot 下的子目录 bind mount 到 kubelet 给的 targetPath，模拟一个真实 CSI driver 的 publish 行为。

相关笔记：
- [[csi-source]] —— CSI 源码导读（spec + K8s 客户端）
- [[csi]] —— CSI 入门
- [[k8s-development-roadmap]] —— K8s 开发学习路线

> **这是教学骨架，不是生产级实现**。想在真实集群里跑业务，请直接看
> [kubernetes-csi/csi-driver-host-path](https://github.com/kubernetes-csi/csi-driver-host-path)
> —— 它是社区参考实现，本 demo 是它去掉大部分容错与边界处理后的精简版本。

## 目录结构

```
csi-hostpath/
├── main.go             gRPC server bootstrap、信号处理、unix socket 解析
├── identity.go         Identity service：GetPluginInfo / Probe / GetPluginCapabilities
├── node.go             Node service：NodePublishVolume / NodeUnpublishVolume / NodeGetInfo / NodeGetCapabilities
├── controller.go       Controller service：CreateVolume / DeleteVolume / ControllerGetCapabilities
├── daemonset.yaml      Controller StatefulSet + Node DaemonSet + RBAC
├── csidriver.yaml      CSIDriver 对象
├── storageclass.yaml   StorageClass(provisioner=learning-plan.csi.k8s.io)
├── pvc.yaml            测试 PVC + Pod
├── go.mod              依赖（CSI spec / grpc / klog）
├── demo-csi-hostpath.md 走读笔记
└── README.md           本文档
```

## 代码要点

### 1. main.go —— gRPC server 起在 unix socket

```go
lis, _ := net.Listen("unix", "/csi/csi.sock")
server := grpc.NewServer()
csi.RegisterIdentityServer(server, newIdentityServer())
csi.RegisterNodeServer(server, newNodeServer(*nodeID, *dataRoot))
csi.RegisterControllerServer(server, newControllerServer(*dataRoot))
server.Serve(lis)
```

启动前 `os.Remove(sockPath)` 清理残留 socket，否则 driver Pod 重启时 bind 会 `address already in use`。`SIGTERM` 时调 `srv.GracefulStop()` 给进行中的 RPC 留时间。

### 2. identity.go —— 用 `Unimplemented*Server` embed 拿到 forward-compatible

```go
type identityServer struct {
    csi.UnimplementedIdentityServer   // 未实现的 RPC 自动返回 Unimplemented
}
```

CSI spec 后续新增 RPC 时旧驱动不需要改代码就能编译通过。这是 gRPC 推荐的兼容性模式。

### 3. node.go —— bind mount 的最小实现

```go
flags := uintptr(unix.MS_BIND)
if req.GetReadonly() { flags |= unix.MS_RDONLY }
unix.Mount(source, target, "", flags, "")
```

等价于 `mount --bind source target`。**幂等性**靠两点保证：
- `os.MkdirAll` 多次调用安全（已存在视为成功）
- `unix.Mount` 返回 `EBUSY` 时认作"已经挂着同一卷"，直接返回成功

生产驱动会用 `k8s.io/mount-utils` 的 `IsLikelyNotMountPoint` 在 mount 之前判断，逻辑更严谨。

### 4. controller.go —— CreateVolume 必须幂等

```go
volumeID := hashVolumeID(req.GetName())   // 同一个 name 永远算出同一个 id
os.MkdirAll(filepath.Join(dataRoot, volumeID), 0750)
```

external-provisioner 用 PVC UID 算 `req.Name`，重试时会带同一个 name 过来——driver 必须返回同一个 volume_id，否则会泄漏孤儿卷。

## 部署到 kind 集群

### 1. build 镜像

```bash
# 在仓库根目录运行
cd learning-plan/demos/csi-hostpath

# 构建一个 Linux 二进制（注意：在 macOS 上交叉编译）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o csi-hostpath .

# 用 distroless 基础镜像打包
cat > Dockerfile <<'EOF'
FROM gcr.io/distroless/static-debian12
COPY csi-hostpath /csi-hostpath
ENTRYPOINT ["/csi-hostpath"]
EOF
docker build -t ghcr.io/learning-notes/csi-hostpath:dev .

# 推进 kind 节点（也可以推到镜像仓库）
kind load docker-image ghcr.io/learning-notes/csi-hostpath:dev
```

### 2. 部署 driver

```bash
kubectl apply -f csidriver.yaml
kubectl apply -f daemonset.yaml      # 含 ServiceAccount + RBAC + Controller + Node
kubectl apply -f storageclass.yaml
```

观察 Pod / CSIDriver / CSINode：

```bash
kubectl get pods -n kube-system -l app=csi-hostpath-node -w
kubectl get csidriver
kubectl describe csinode <node-name>   # 看 spec.drivers[] 里是不是有 learning-plan.csi.k8s.io
```

### 3. 跑测试 PVC

```bash
kubectl apply -f pvc.yaml

kubectl get pvc csi-hostpath-test -w
# 期望：STATUS 从 Pending -> Bound（external-provisioner 调 CreateVolume 创建出 PV）

kubectl exec csi-hostpath-test -- cat /data/marker
# 看到 "csi-hostpath demo works at ..." 说明 bind mount + 读写都正常
```

### 4. 清理

```bash
kubectl delete -f pvc.yaml
kubectl delete -f storageclass.yaml
kubectl delete -f daemonset.yaml
kubectl delete -f csidriver.yaml
```

## 已知限制（教学目的有意省略）

- 没实现 `NodeStageVolume` / `NodeUnstageVolume`，所以 RWX 多 Pod 共享场景下没有共享 stage 层（每个 Pod 各自 publish）。
- 没实现 `ControllerExpandVolume` —— SC 里 `allowVolumeExpansion: false`。
- 没实现 `CreateSnapshot` —— 不接入 external-snapshotter。
- 没有真正的容量管理 —— `CreateVolume` 不检查 dataRoot 剩余空间。
- 错误处理简化 —— 生产驱动会区分 gRPC `Aborted` / `FailedPrecondition` / `Internal` 等更精细的状态码。

每一项都是生产 csi-driver-host-path 里好几百行代码的话题；想深入研究就从那个仓库开始。

## 在 macOS 上 build / 调试

```bash
# 只验证编译，不真跑：
GOOS=linux go build -o /dev/null .

# 在 mac 上能起 gRPC server 但 unix.Mount 会失败——
# 适合用 grpcurl 验证 RPC 是否能调通：
./csi-hostpath --endpoint=unix:///tmp/csi.sock --node-id=mac &
grpcurl -plaintext -unix /tmp/csi.sock list
grpcurl -plaintext -unix /tmp/csi.sock csi.v1.Identity/GetPluginInfo
```
