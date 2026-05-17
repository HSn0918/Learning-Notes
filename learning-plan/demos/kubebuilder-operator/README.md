# Kubebuilder Operator Demo (无需 CRD 版)

> **状态**：✅ Mac 验证可运行（controller-runtime v0.19.3，先 `go mod tidy`） · 详见 [demos 验证总表](../README.md)

这是一个最小可运行的 controller-runtime Operator 骨架，用来对照
`controller-runtime-source.md` 的源码走读。**它不依赖任何 CRD**，
只监听内置的 `ConfigMap` 资源，所以可以跑在任何 K8s 集群（Kind、
Minikube、远端集群都行），无需 `kubebuilder init` 或 `make install`。

## 业务逻辑

监听一个 namespace（默认 `default`）下所有 `ConfigMap`，对每个带
`label demo.learning-notes/source=true` 的源 CM：

1. 如果它有 annotation `demo.learning-notes/payload`，
   就创建（或更新）一个名为 `<source-name>-mirror` 的 ConfigMap，
   把该 annotation 复制过去。
2. 源 CM 被删时，级联删除 mirror。
3. mirror CM 上写有 `ownerReferences` 指回源 CM，演示 `Owns(...)`
   的反向链路：若有人手动改 mirror 的 annotation，Controller 会被
   触发并把它改回来。

## 文件结构

```
kubebuilder-operator/
  go.mod                       module + 依赖
  main.go                      Manager 装配 + 信号处理
  configmap_controller.go      Reconciler + SetupWithManager
  README.md                    本文件
  demo-kubebuilder-operator.md 配套笔记
```

对照 `controller-runtime-source.md` 中的源码片段：

| 本 demo | 对应真实库代码 |
|---------|--------------|
| `ctrl.NewManager(...)` | `pkg/manager/manager.go: New` |
| `(&ConfigMapReconciler{}).SetupWithManager` | `pkg/builder/controller.go: Build` |
| `For(&ConfigMap{})` | `EnqueueRequestForObject` 注册到 source.Kind |
| `Owns(&ConfigMap{})` | `EnqueueRequestForOwner` 注册到 source.Kind |
| `r.Get / r.Update / r.Create` | `DelegatingClient`（读 Cache、写直连） |
| `ctrl.SetControllerReference` | `pkg/controller/controllerutil` |
| `ctrl.SetupSignalHandler` | `pkg/manager/signals` |

## 本地运行

```bash
cd learning-plan/demos/kubebuilder-operator

# 1) 拉依赖（首次）
go mod tidy

# 2) 本地跑（用 ~/.kube/config 连集群，跑在你电脑上）
go run . --kubeconfig ~/.kube/config --namespace default

# 另开一个终端验证：
kubectl create configmap demo-src -n default --from-literal=x=1
kubectl label  configmap demo-src demo.learning-notes/source=true -n default
kubectl annotate configmap demo-src demo.learning-notes/payload="hello" -n default

# 观察 mirror 被创建
kubectl get configmap demo-src-mirror -n default -o yaml
# 应该看到 annotation demo.learning-notes/payload: hello
# 以及 metadata.ownerReferences 指向 demo-src

# 改 annotation：
kubectl annotate configmap demo-src demo.learning-notes/payload="world" --overwrite -n default
kubectl get configmap demo-src-mirror -n default -o jsonpath='{.metadata.annotations}'
# 应该变成 {"demo.learning-notes/payload":"world"}

# 演示 Owns 反向链路 —— 改 mirror 的 annotation，会被 reconcile 改回去：
kubectl annotate configmap demo-src-mirror demo.learning-notes/payload="tampered" --overwrite -n default
# 等一秒再看，Controller 会把它改回 "world"

# 清理：
kubectl delete configmap demo-src -n default
# mirror 也会被级联删除
```

## 调参

| flag | 默认 | 说明 |
|------|------|------|
| `--namespace` | `default` | 监听哪个 namespace |
| `--metrics-bind-address` | `:8080` | Prometheus metrics |
| `--health-probe-bind-address` | `:8081` | `/healthz`、`/readyz` |
| `--leader-elect` | `false` | 多副本时启用 Leader Election（基于 Lease） |
| `--zap-log-level` | `info` | 日志级别（zap.Options） |

## 注意

- 生产里监听 ConfigMap 这种全集群高频对象一定要加 namespace / label 限制，否则 Cache 内存会爆掉。这里只为演示。
- `MaxConcurrentReconciles: 2` 仅示意；真实工作负载按"每秒事件数 × Reconcile 耗时"估算。
- 没有 RBAC YAML —— 本地直接用 `kubectl` 上下文的权限跑；如果要打成镜像跑在集群里，需要 ClusterRole 给 `configmaps` 资源 `get/list/watch/create/update/patch/delete`。
