# scheduler-plugin demo: NodeLabelScore

> **状态**：⚠️ 需手填 go.mod `replace` 列表（K8s 主仓 staging 占位版本问题），见下方「编译说明」 · 详见 [demos 验证总表](../README.md)

一个最小可运行的 **out-of-tree kube-scheduler 自定义插件** 示例。它做的事很简单：
在原生 kube-scheduler 之上多注册一个 `Score` 扩展点插件 `NodeLabelScore`，
带 label `learning-plan/preferred=true` 的节点拿满分（`framework.MaxNodeScore`），
其他节点 0 分，再交给 framework 加权汇总参与最终选节点。

相关笔记：
- [[scheduler-framework-source]] —— framework 与扩展点源码导读
- [[scheduler-deep-dive]] —— 调度器整体设计
- [[k8s-development-roadmap]] —— K8s 开发学习路线

## 目录结构

```
scheduler-plugin/
├── main.go                  入口，使用 app.NewSchedulerCommand + app.WithPlugin 注册插件
├── pkg/
│   └── nodelabel/
│       └── nodelabel.go     ScorePlugin 实现（Name / Score / ScoreExtensions / NormalizeScore）
├── config.yaml              KubeSchedulerConfiguration v1 示例（profile: learning-plan-scheduler）
├── go.mod                   依赖 k8s.io/kubernetes + 一组 staging replace
└── README.md                本文档
```

## 代码要点

### 1. main.go — 用 `app.WithPlugin` 把插件挂进 registry

```go
command := app.NewSchedulerCommand(
    app.WithPlugin(nodelabel.Name, nodelabel.New),
)
```

`app.NewSchedulerCommand` 返回一个完整的 `*cobra.Command`，命令行参数、leader election、
配置文件解析、metrics endpoint 等全都和原版 kube-scheduler 一致——我们只是多塞了一个插件工厂。

### 2. nodelabel.go — 实现 `framework.ScorePlugin`

| 方法 | 作用 |
|------|------|
| `Name() string` | 返回 `"NodeLabelScore"`，必须与 KubeSchedulerConfiguration 里的 `plugins.score.enabled[].name` 一致 |
| `Score(ctx, state, pod, nodeName) (int64, *Status)` | 单节点打分。被 framework 跨节点并行调用 |
| `ScoreExtensions() ScoreExtensions` | 返回 NormalizeScore 扩展；不需要归一化可以 `return nil` |
| `NormalizeScore(...)` | 把本插件在所有节点上的原始分映射到 `[0, MaxNodeScore]` |

`New` 是 `framework.PluginFactory`，被 `app.WithPlugin` 注册到 out-of-tree registry，
其签名 `func(ctx, args runtime.Object, handle framework.Handle) (framework.Plugin, error)`。
通过 `handle.SnapshotSharedLister().NodeInfos().Get(nodeName)` 拿到当前周期的 snapshot 中的
NodeInfo —— **不要直接读 Informer cache**，否则会与 Filter 阶段看到的状态不一致。

### 3. config.yaml — 把插件注入到一个 profile

```yaml
profiles:
  - schedulerName: learning-plan-scheduler
    plugins:
      score:
        enabled:
          - name: NodeLabelScore
            weight: 5
```

业务 Pod 想用这套打分，就在 spec 里指定调度器名：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hello
spec:
  schedulerName: learning-plan-scheduler   # 不写则走 default-scheduler
  containers:
    - name: app
      image: nginx
```

## 编译说明 —— 为什么 `go build` 不能直接跑通

`k8s.io/kubernetes` 主仓 go.mod 里把所有 staging 子模块（`k8s.io/api`、
`k8s.io/component-base`、`k8s.io/kube-scheduler` 等）都标成 `v0.0.0` 占位版本，
社区约定 out-of-tree 用户必须在自己的 go.mod 里通过 `replace` 把它们指向真实版本号
（或者直接指向源码目录）。

最干净的两种做法：

### 做法 A：fork `scheduler-plugins` 仓

```
git clone https://github.com/kubernetes-sigs/scheduler-plugins.git
cd scheduler-plugins
# 把本目录的 pkg/nodelabel 拷进去
# 在 cmd/scheduler/main.go 里加一行 app.WithPlugin(nodelabel.Name, nodelabel.New)
make local-image
```

`scheduler-plugins` 仓的 go.mod 已经准备好了完整的 staging replace 列表，
跟着它的版本走是最省心的方案，社区里 Volcano、Coscheduling、CapacityScheduling
等大家熟知的插件也都是这么打的包。

### 做法 B：手写 replace 列表

在本目录的 `go.mod` 里把注释掉的那一段 `replace (...)` 启用，并把版本号对齐到
你要跟随的 K8s 版本（例如要构建在 v1.34 之上，全部用 `v0.34.0`）。

```
go mod tidy
go build -o scheduler-plugin .
```

## 部署：作为第二调度器跑起来

把上面编译出的 `scheduler-plugin` 二进制塞进镜像（基础镜像用 `gcr.io/distroless/static:nonroot`
或 `registry.k8s.io/kube-scheduler` 同款 distroless 即可），然后用一份清单部署到 `kube-system`。

下面这份是**实测可用的完整清单**（在 kind 多节点集群上跑通），按 ServiceAccount → RBAC →
ConfigMap → Deployment 的顺序 apply：

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: learning-plan-scheduler
  namespace: kube-system
---
# 调度核心权限：直接复用 K8s 内置 ClusterRole system:kube-scheduler
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: learning-plan-scheduler
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:kube-scheduler
subjects:
  - kind: ServiceAccount
    name: learning-plan-scheduler
    namespace: kube-system
---
# 卷调度权限（VolumeBinding 插件需要）
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: learning-plan-scheduler-volume
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:volume-scheduler
subjects:
  - kind: ServiceAccount
    name: learning-plan-scheduler
    namespace: kube-system
---
# ★ 关键易漏点：system:kube-scheduler 只授权了名为 "kube-scheduler" 的那个 lease。
# 我们用了独立 resourceName=learning-plan-scheduler，必须单独补一个 Role 授权这个新 lease，
# 否则 Pod 起来后 leader election 会一直 forbidden、选不出 leader。
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: learning-plan-scheduler-leaderelection
  namespace: kube-system
rules:
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["create"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    resourceNames: ["learning-plan-scheduler"]
    verbs: ["get", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: learning-plan-scheduler-leaderelection
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: learning-plan-scheduler-leaderelection
subjects:
  - kind: ServiceAccount
    name: learning-plan-scheduler
    namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: learning-plan-scheduler-config
  namespace: kube-system
data:
  config.yaml: |
    apiVersion: kubescheduler.config.k8s.io/v1
    kind: KubeSchedulerConfiguration
    leaderElection:
      leaderElect: true
      resourceName: learning-plan-scheduler
      resourceNamespace: kube-system
    profiles:
      - schedulerName: learning-plan-scheduler
        plugins:
          score:
            enabled:
              - name: NodeLabelScore
                weight: 5
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: learning-plan-scheduler
  namespace: kube-system
  labels: { app: learning-plan-scheduler }
spec:
  replicas: 1
  selector:
    matchLabels: { app: learning-plan-scheduler }
  template:
    metadata:
      labels: { app: learning-plan-scheduler }
    spec:
      serviceAccountName: learning-plan-scheduler
      containers:
        - name: scheduler
          image: learning-plan-scheduler:dev
          imagePullPolicy: IfNotPresent   # kind load 进来的本地镜像，别让它去 pull
          command:
            - /scheduler-plugin
            - --config=/etc/kubernetes/scheduler-plugin/config.yaml
            - --leader-elect=true
            - -v=4
          volumeMounts:
            - name: config
              mountPath: /etc/kubernetes/scheduler-plugin
              readOnly: true
      volumes:
        - name: config
          configMap:
            name: learning-plan-scheduler-config
```

注意几点：
- 必须用**独立的** `leaderElection.resourceName`，否则与 default-scheduler 抢同一把锁。
- RBAC 一共要 3 块：`system:kube-scheduler`（调度主权限）、`system:volume-scheduler`
  （卷调度）、外加一个**自建 Role 授权新 lease**——最后这块最容易漏，README 旧版只说
  「复制一份 ClusterRoleBinding」是不够的。
- 业务侧只要在 Pod `spec.schedulerName` 写 `learning-plan-scheduler`，调度走向就切过来了；
  其他 Pod 仍走 default-scheduler，互不干扰。

### 启动后日志里会有一条 forbidden 告警——可忽略

Pod 起来后日志里会看到：

```
W ... Error looking up in-cluster authentication configuration: configmaps
  "extension-apiserver-authentication" is forbidden: User
  "system:serviceaccount:kube-system:learning-plan-scheduler" cannot get
  resource "configmaps" in API group "" in the namespace "kube-system"
```

原因：`system:kube-scheduler` 这个内置 ClusterRole **不含**读 `extension-apiserver-authentication`
configmap 的权限，而该 configmap 只有把调度器自身的 metrics/healthz 端点做成「带认证的
HTTPS 服务」时才需要。kube-scheduler 默认带 `--authentication-tolerate-lookup-failure=true`，
查不到就降级继续，**不影响调度功能**。想消除这条噪音，可以再 `RoleBinding` 一个
`extension-apiserver-authentication-reader`（namespace `kube-system`）给本 SA。

## 验证

```bash
# 1. 确认调度器拿到了独立的 leader lease（不和 default-scheduler 抢锁）
kubectl -n kube-system get lease learning-plan-scheduler
kubectl -n kube-system logs deploy/learning-plan-scheduler | grep "successfully acquired lease"

# 2. 给两个 worker 打相反的 label
kubectl label node <node-a> learning-plan/preferred=true  --overwrite
kubectl label node <node-b> learning-plan/preferred=false --overwrite

# 3. 跑一批测试 Pod（用 Deployment 拉 6 个副本，比单 Pod 更能看出倾向）
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nl-test
spec:
  replicas: 6
  selector:
    matchLabels: { app: nl-test }
  template:
    metadata:
      labels: { app: nl-test }
    spec:
      schedulerName: learning-plan-scheduler   # 不写则走 default-scheduler
      containers:
        - name: pause
          image: registry.k8s.io/pause:3.10
EOF

# 4. 看落点分布
kubectl get pods -l app=nl-test -o wide --no-headers | awk '{print $7}' | sort | uniq -c
# 实测结果：6 个副本全部落在 preferred=true 的节点。

# 5. 确认是自定义调度器调的，且打分插件命中
kubectl get events --field-selector reason=Scheduled | grep nl-test
kubectl -n kube-system logs deploy/learning-plan-scheduler | grep "NodeLabelScore matched"
```

`NodeLabelScore` 配了 `weight: 5`，会压过 `PodTopologySpread`（weight 2）的均摊倾向，
所以现象是「全堆到一个节点」而不是分散——这正说明打分确实生效了。

**反转对照**：把 label 在两个节点间对调，再跑一批新 Pod，应当看到它们整体跟着
`preferred=true` 迁移。这一步能排除「恰好都落同一节点」的巧合。

如果你想看每个节点的逐项打分，把 scheduler 的 `-v=6` 打开，日志里会有各节点经过
`NodeLabelScore` 后的 weighted score。

> 本 demo 已在 kind 多节点集群（1 control-plane + 2 worker，K8s v1.35 node 镜像、
> 调度器编在 v1.34 上）实测跑通：正向 6/6、反转对照 6/6。
