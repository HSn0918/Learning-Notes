#kubernetes #kubebuilder

相关笔记：[[informer]] | [[kubernetes-basics]] | [[api-resource]]

## 背景

在 Kubernetes 中，部署一个 Deployment 时，通常还需要创建一个 Service 以便外部可以访问这个 Deployment。为了简化这个过程，可以创建一个自定义的 Controller，使其在部署 Deployment 的同时自动创建相应的 Service。

## 自定义 Controller 运行原理

自定义控制器能够完成业务逻辑，最主要是依赖 client-go 库的各个组件的交互。

![[自定义controller运行原理.png]]

通过图示，可以看到几个核心组件的交互流程（蓝色表示 client-go，黄色是自定义 controller）。

```mermaid
flowchart LR
    subgraph client-go
        R[Reflector<br/>ListAndWatch] -->|资源变更| DF[DeltaFIFO]
        DF -->|Pop| INF[Informer]
        INF -->|存储| IDX[Indexer<br/>线程安全存储]
    end

    subgraph Custom Controller
        INF -->|触发回调| EH[Event Handler]
        EH -->|key 入队| WQ[WorkQueue]
        WQ -->|取出 key| PI[Process Item]
        PI -->|获取对象| IDX
        PI -->|业务逻辑| BL[Reconcile]
    end

    API[API Server] -->|Watch| R
```

### client-go 组件

- **Reflector**: 用来 watch 特定的 K8s API 资源。通过 ListAndWatch 方法，当 Reflector 通过 watch API 接收到新资源实例存在的通知时，使用相应的 list API 获取新创建的对象，并将其放入 DeltaFIFO 队列中
- **Informer**: 从 DeltaFIFO 队列中弹出对象。base controller 的作用是保存对象以供以后检索，并调用我们的控制器将对象传递给它
- **Indexer**: 提供对象的索引功能。典型的索引用例是基于对象标签创建索引。Indexer 使用线程安全的数据存储，默认函数 MetaNamespaceKeyFunc 生成对象的键为 namespace/name 组合

### 自定义 Controller 组件

- **Informer reference**: Informer 实例的引用，定义如何使用自定义资源对象
- **Indexer reference**: Indexer 实例的引用
- **Resource Event Handlers**: 资源事件回调函数，获取调度对象的 key 并将该 key 排入工作队列
- **Work queue**: 任务队列，存储需要处理的对象 Key
- **Process Item**: 处理任务队列中对象的函数，通常使用 Indexer 引用来重试与该 key 对应的对象

### 完整处理流程

![](自定义控制器运行原理.png)

简单来说：Reflector 通过检测 Kubernetes API 来跟踪资源变化，一旦发现有变化，就将该 Object 存储到队列中。Informer 循环取出该 Object 并将其存入 Indexer 进行检索，同时触发 Callback 回调函数，并将变更的 Object Key 信息放入到工作队列中。此时自定义 Controller 里面的 Process Item 就会获取工作队列里面的 Key，并从 Indexer 中获取 Key 对应的 Object，从而进行相关的业务处理。

## 环境准备

```bash
# 依赖工具
brew install golang      # go1.21+
brew install kubectl
brew install kubebuilder
```

## 创建项目

```bash
mkdir controller-demo
cd controller-demo
kubebuilder init --domain appservice.com --owner appservice
```

## 创建 API

### Kubebuilder 注释

在 Kubebuilder 中，使用注释定义自定义资源（CRD）和控制器的元数据。这些注释指导代码生成工具生成正确的 Kubernetes CRD 和控制器逻辑。

| 注释 | 作用 |
| --- | --- |
| `// +kubebuilder:object:root=true` | 声明该类型是自定义资源的根对象 |
| `// +kubebuilder:subresource:status` | 启用 status 子资源 |
| `// +kubebuilder:resource:scope=Cluster` | 设置资源范围为集群级 |
| `// +kubebuilder:resource:path=<p>,shortName=<s>` | 设置资源路径和短名称 |
| `// +kubebuilder:printcolumn:name=<n>,type=<t>,JSONPath=<j>` | 定义 kubectl get 输出的额外列 |
| `// +kubebuilder:validation:Minimum=<v>` | 字段最小值验证 |
| `// +kubebuilder:validation:Maximum=<v>` | 字段最大值验证 |
| `// +kubebuilder:validation:Enum=<v1>;<v2>` | 字段枚举值验证 |
| `// +kubebuilder:validation:Pattern=<regex>` | 字段正则表达式验证 |
| `// +kubebuilder:default=<value>` | 设置字段默认值 |

### 生成 API

```bash
kubebuilder create api --group batch --version v1 --kind AppService
```

### 定义 Spec 和 Status

```go
// AppServiceSpec defines the desired state of AppService
type AppServiceSpec struct {
    Replicas  *int32                      `json:"replicas"`
    Image     string                      `json:"image"`
    Resources corev1.ResourceRequirements `json:"resources,omitempty"`
    Envs      []corev1.EnvVar             `json:"envs,omitempty"`
    Ports     []corev1.ServicePort        `json:"ports,omitempty"`
}

// AppServiceStatus defines the observed state of AppService
type AppServiceStatus struct {
    appsv1.DeploymentStatus `json:",inline"`
}
```

```bash
# 确保 Kubernetes 资源定义文件是最新的
make manifests
```

## 设计自定义 Controller

### Reconcile 流程

Controller 的 Reconcile 逻辑如下：

```mermaid
flowchart TD
    A[Reconcile 开始] --> B{AppService 实例<br/>是否存在?}
    B -->|不存在| C[根据 Spec 创建<br/>Deployment + Service]
    B -->|存在| D{DeletionTimestamp<br/>是否为空?}
    D -->|非空| DEL[执行删除流程]
    D -->|空| E{Annotations 中的 Spec<br/>与当前 Spec 是否相同?}
    E -->|不同| F[更新 Deployment + Service]
    E -->|相同| G[无需操作]

    F --> H{是否有 Finalizer?}
    G --> H
    C --> H
    H -->|没有| I[增加 Finalizer]
    H -->|有| J[用 Annotations 记录当前 Spec]
```

![[AppService控制器流程.png]]

### 创建资源

- 如果 AppService 实例不存在，则根据 AppServiceSpec 创建:
  - 创建 Deployment 资源
  - 创建 Service 资源
- 如果 AppService 实例存在，则将 Annotations 中记录的 Spec 值与当前的 Spec 比较:
  - 如果前后的 Spec 不同: 更新 Deployment 和 Service 资源
  - 如果前后的 Spec 相同: 无需额外操作
- 判断是否有 Finalizer:
  - 没有: 增加 Finalizer
  - 有: 使用 Annotations 记录当前 Spec 的值

### 删除资源

```mermaid
flowchart TD
    A[开始删除处理] --> B{DeletionTimestamp<br/>是否为空?}
    B -->|是| END1[流程结束]
    B -->|否| C{是否包含<br/>Finalizer?}
    C -->|否| END2[流程结束]
    C -->|是| D[deleteAssociatedResources<br/>删除关联资源]
    D -->|成功| E[移除 Finalizer]
    D -->|失败| F[返回错误]
    E --> G[Patch 更新实例]
```

![[删除关联资源.png]]

- 检查 DeletionTimestamp 是否为空: 判断资源是否被标记为删除
- 检查是否包含 Finalizer: 如果包含，调用 `deleteAssociatedResources` 方法删除关联资源
  - 成功: 移除 Finalizer，通过 Patch 更新实例
  - 失败: 返回错误信息
