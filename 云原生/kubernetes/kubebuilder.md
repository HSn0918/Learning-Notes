## 背景

在 Kubernetes 中，部署一个 Deployment 时，通常还需要创建一个 Service 以便外部可以访问这个 Deployment。为了简化这个过程，可以创建一个自定义的 Controller，使其在部署 Deployment 的同时自动创建相应的 Service。

# controller

## 自定义controller运行原理

只定义资源，不实现资源控制器是没有任何意义的。自定义控制器能够完成业务逻辑，最主要是依赖 client-go 库的各个组件的交互。下图展示它们之间的关系：

![[自定义controller运行原理.png]]
通过图示，可以看到几个核心组件的交互流程，蓝色表示 client-go，黄色是自定义 controller，各组件作用介绍如下：

### client-go 组件

- **Reflector**：reflector 用来 watch 特定的 k8s API 资源。具体的实现是通过 ListAndWatch 的方法，watch 可以是 k8s 内建的资源或者是自定义的资源。当 reflector 通过 watch API 接收到有关新资源实例存在的通知时，它使用相应的列表 API 获取新创建的对象，并将其放入 watchHandler 函数内的 Delta FIFO 队列中。
- **Informer**：informer 从 Delta FIFO 队列中弹出对象。执行此操作的功能是 processLoop。base controller 的作用是保存对象以供以后检索，并调用我们的控制器将对象传递给它。
- **Indexer**：索引器提供对象的索引功能。典型的索引用例是基于对象标签创建索引。 Indexer 可以根据多个索引函数维护索引。Indexer 使用线程安全的数据存储来存储对象及其键。 在 Store 中定义了一个名为 MetaNamespaceKeyFunc 的默认函数，该函数生成对象的键作为该对象的 namespace/name 组合。

### 自定义 controller 组件

- **Informer reference**：指的是 Informer 实例的引用，定义如何使用自定义资源对象。自定义控制器代码需要创建对应的 Informer。
- **Indexer reference**：自定义控制器对 Indexer 实例的引用。自定义控制器需要创建对应的 Indexer。
- **Resource Event Handlers**：资源事件回调函数，当它想要将对象传递给控制器时，它将被调用。编写这些函数的典型模式是获取调度对象的 key，并将该 key 排入工作队列以进行进一步处理。
- **Work queue**：任务队列。编写资源事件处理程序函数以提取传递的对象的 key 并将其添加到任务队列。
- **Process Item**：处理任务队列中对象的函数，这些函数通常使用 Indexer 引用或 Listing 包装器来重试与该 key 对应的对象。

## 自定义控制器运行原理
![](自定义控制器运行原理.png)

```
Reflector：通过检测Kubernetes API来跟踪资源变化，将变化存入DeltaFIFO队列。
DeltaFIFO：存储和管理Reflector发现的资源变化。
Informer：从DeltaFIFO队列中弹出对象，将其存储在Indexer中以便检索，并触发事件回调。
Indexer：提供对象的索引功能，支持根据多个索引函数进行检索。
ResourceEventHandler：事件回调函数，将对象的Key放入工作队列。
WorkQueue：存储需要处理的对象Key。
Controller：从工作队列中获取对象Key，从Indexer中获取对象，处理业务逻辑。
```

简单的说，整个处理流程大概为：Reflector 通过检测 Kubernetes API 来跟踪该扩展资源类型的变化，一旦发现有变化，就将该 Object 存储队列中，Informer 循环取出该 Object 并将其存入 Indexer 进行检索，同时触发 Callback 回调函数，并将变更的 Object Key 信息放入到工作队列中，此时自定义 Controller 里面的 Process Item 就会获取工作队列里面的 Key，并从 Indexer 中获取 Key 对应的 Object，从而进行相关的业务处理。

## 环境准备

- go1.21+
- kubebuilder
- kubectl

```
brew install golang
brew install kubectl
brew install kubebuilder
```

## 创建项目

```
mkdir controller-demo
cd controller-demo
kubebuilder init --domain appservice.com --owner appservice
```

## 创建api

### kubebuilder注释

#### 概述

在 Kubebuilder 中，使用注释定义自定义资源（CRD）和控制器的元数据。这些注释指导代码生成工具生成正确的 Kubernetes CRD 和控制器逻辑

#### 注释列表

`**// +kubebuilder:object:root=true**`

- 声明该类型是自定义资源的根对象。

`**// +kubebuilder:subresource:status**`

- 启用 `status` 子资源，用于报告对象的当前状态。

`**// +kubebuilder:resource:scope=Cluster**`

- 设置资源的范围为集群级（Cluster），而非命名空间级（Namespaced）。

`**// +kubebuilder:resource:path=<path>,shortName=<shortName>**`

- 设置资源的路径和短名称。
- 示例：`// +kubebuilder:resource:path=guestbooks,shortName=gb`

`**// +kubebuilder:printcolumn:name=<name>,type=<type>,description=<description>,JSONPath=<jsonpath>**`

- 定义 `kubectl get` 命令输出的额外列。
- 示例：`// +kubebuilder:printcolumn:name="Foo",type="string",description="Foo Field",JSONPath=".spec.foo"`

`**// +kubebuilder:validation:Minimum=<value>**`

- 定义字段的最小值验证。
- 示例：`// +kubebuilder:validation:Minimum=0`

`**// +kubebuilder:validation:Maximum=<value>**`

- 定义字段的最大值验证。
- 示例：`// +kubebuilder:validation:Maximum=100`

`**// +kubebuilder:validation:Enum=<value1>;...;<valueN>**`

- 定义字段的枚举值验证。
- 示例：`// +kubebuilder:validation:Enum="Option1";"Option2";"Option3"`

`**// +kubebuilder:validation:Pattern=<regex>**`

- 定义字段的正则表达式验证。
- 示例：`// +kubebuilder:validation:Pattern="^[a-z0-9]+$"`

`**// +kubebuilder:default=<value>**`

- 设置字段的默认值。
- 示例：`// +kubebuilder:default="default_value"`

### 生成

```
kubebuilder create api --group batch --version v1 --kind AppService  
```

```
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

```
// 命令可以确保你的项目中的Kubernetes资源定义文件是最新的
make manifests
```

## 设计自定义controller

Controller 的 Reconcile 如下：

### 创建资源

- 如果 AppService 实例不存在，则根据 AppServiceSpec 创建

- 创建 Deployment 资源
- 创建 Service 资源

- 如果 AppService 实例存在，则将 Annotations 中记录的 Spec 值与当前的 Spec 比较

- 如果前后的 Spec 不同

- 更新 Deployment 资源
- 更新 Service 资源

- 如果前后的 Spec 相同，则无需额外的操作
- 判断是否有Finalizer

- 没包含相应Finalizer

- 增加

- 包含相应Finalizer

- 使用 Annotations 记录当前 Spec 的值

![[AppService控制器流程.png]]
### 删除资源

- **开始删除处理**：表示流程的起始点。
- **检查 DeletionTimestamp 是否为空**：判断资源是否被标记为删除。

- **是**：如果 `DeletionTimestamp` 为空，表示资源未被标记为删除，流程结束。
- **否**：如果 `DeletionTimestamp` 非空，继续检查是否有 `Finalizer`。

- **检查是否包含 Finalizer**：判断资源是否包含特定的 `Finalizer`。

- **是**：如果包含，调用 `deleteAssociatedResources` 方法删除关联资源。

- **成功**：如果删除关联资源成功，接下来移除 `Finalizer`。
- **失败**：如果删除关联资源时发生错误，返回错误信息。

- **否**：如果不包含 `Finalizer`，没有需要执行的删除操作，流程结束。

- **移除 Finalizer**：将 `Finalizer` 从资源中移除。
- **使用 Patch 更新实例**：通过 Patch 方法更新实例，主要是移除了 `Finalizer`。

![[删除关联资源.png]]

  

```
package controller

import (
	"context"
	"reflect"
	"slices"

	"k8s.io/apimachinery/pkg/util/sets"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"

	batchv1 "appservice.com/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	finalizerName = "appservice.batch.appservice.com"
	spec          = "spec"
)

// AppServiceReconciler reconciles a AppService object
type AppServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=batch.appservice.com,resources=appservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.appservice.com,resources=appservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.appservice.com,resources=appservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *AppServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &batchv1.AppService{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if instance.DeletionTimestamp.IsZero() {
		if slices.Contains(instance.Finalizers, finalizerName) {
			if err := r.deleteAssociatedResources(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
			// 回收成功 删除finalize
			merge := client.MergeFrom(instance.DeepCopy())
			instance.Finalizers = sets.NewString(instance.Finalizers...).Delete(finalizerName).UnsortedList()
			if err := r.Client.Patch(ctx, instance, merge); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	deployment := &appsv1.Deployment{}
	// 1. 不存在，则创建
	if err := r.Client.Get(ctx, req.NamespacedName, deployment); err != nil {
		// 如果不是不存在报错 返回
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		// 1.不存在 则创建deployment
		deployment = NewDeployment(instance)
		if err := r.Client.Create(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
		// 2.创建 Service
		svc := NewService(instance)
		if err := r.Client.Create(ctx, svc); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// 2. 存在，则对比spec
		// 保留spec到oldSpec
		oldSpec := &batchv1.AppServiceSpec{}
		if err := json.Unmarshal([]byte(instance.Annotations[spec]), oldSpec); err != nil {
			return ctrl.Result{}, err
		}
		if instance.Annotations == nil{
			instance.Annotations = map[string]string{}
		}
		if !reflect.DeepEqual(instance.Spec, *oldSpec) {
			newDeployment := NewDeployment(instance)
			currDeployment := &appsv1.Deployment{}
			if err := r.Client.Get(ctx, req.NamespacedName, currDeployment); err != nil {
				return ctrl.Result{}, err
			}
			currDeployment.Spec = newDeployment.Spec
			if err := r.Client.Update(ctx, currDeployment); err != nil {
				return ctrl.Result{}, err
			}

			newService := NewService(instance)
			currService := &corev1.Service{}
			if err := r.Client.Get(ctx, req.NamespacedName, currService); err != nil {
				return ctrl.Result{}, err
			}

			currIP := currService.Spec.ClusterIP
			currService.Spec = newService.Spec
			currService.Spec.ClusterIP = currIP
			if err := r.Client.Update(ctx, currService); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	if !slices.Contains(instance.Finalizers, finalizerName) {
		merge := client.MergeFrom(instance.DeepCopy())
		instance.Finalizers = append(instance.Finalizers, finalizerName)
		if err := r.Client.Patch(ctx, instance, merge); err != nil {
			return ctrl.Result{}, err
		}
	}
	// 3. 关联 Annotations
	data, _ := json.Marshal(instance.Spec)
	if instance.Annotations != nil {
		instance.Annotations[spec] = string(data)
	} else {
		instance.Annotations = map[string]string{spec: string(data)}
	}
	if err := r.Client.Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.AppService{}).
		Complete(r)
}

// controllers/appservice_controller.go
func NewDeployment(app *batchv1.AppService) *appsv1.Deployment {
	labels := map[string]string{"app": app.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   batchv1.GroupVersion.Group,
					Version: batchv1.GroupVersion.Version,
					Kind:    app.Kind,
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Replicas,
			Selector: selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{Containers: newContainer(app)},
			},
		},
	}
}

// controllers/appservice_controller.go
func newContainer(app *batchv1.AppService) []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range app.Spec.Ports {
		cport := corev1.ContainerPort{}
		cport.ContainerPort = svcPort.TargetPort.IntVal
		containerPorts = append(containerPorts, cport)
	}
	return []corev1.Container{
		{
			Name:            app.Name,
			Image:           app.Spec.Image,
			Resources:       app.Spec.Resources,
			Ports:           containerPorts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env:             app.Spec.Envs,
		},
	}
}

// controllers/appservice_controller.go
func NewService(app *batchv1.AppService) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   batchv1.GroupVersion.Group,
					Version: batchv1.GroupVersion.Version,
					Kind:    app.Kind,
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeNodePort,
			Ports: app.Spec.Ports,
			Selector: map[string]string{
				"app": app.Name,
			},
		},
	}
}

func (r *AppServiceReconciler) deleteAssociatedResources(ctx context.Context, app *batchv1.AppService) error {
	deployment := &appsv1.Deployment{}
	// 没有err，说明找到了deployment，需要删除
	if err := r.Client.Get(ctx, client.ObjectKey{Name: app.Name, Namespace: app.Namespace}, deployment); err == nil {
		if err := r.Client.Delete(ctx, deployment); err != nil {
			return err
		}
	}

	svc := &corev1.Service{}
	// 没有err，说明找到了service，需要删除
	if err := r.Client.Get(ctx, client.ObjectKey{Name: app.Name, Namespace: app.Namespace}, svc); err == nil {
		if err := r.Client.Delete(ctx, svc); err != nil {
			return err
		}
	}
	return nil
}
```

  

# 自定义webhook运行原理

Webhook 是 HTTP 回调，它接收许可请求，处理这些请求并返回许可响应。

Kubernetes 提供以下类型的许可 Webhook：

- Mutating Admission Webhook:

该 Webhook 可接受或拒绝对象请求，并且可能变更对象。

- ValidatingAdmission Webhook:

该 Webhook 可在不更改对象的情况下接受或拒绝对象请求。


![[APIServer展开 1.png]]
# webhook

## 创建Webhook

```
kubebuilder create webhook --group batch --version v1 --kind AppService --defaulting --programmatic-validation
```

```
// MutatingWebhook
func (r *AppService) Default() {
	appservicelog.Info("default", "name", r.Name)
}
// ValidatingWebhook
var _ webhook.Validator = &AppService{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AppService) ValidateCreate() (admission.Warnings, error) {
	appservicelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AppService) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	appservicelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AppService) ValidateDelete() (admission.Warnings, error) {
	appservicelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
```

  

### Mutatingwebhook

例如想在上面的自定义的controller添加新的功能,例如:检验`image`是否有版本号，如果不存在就给他添加`:leaest` 。

修改代码如下:

```
const (
	colon        = ":"
	leastVersion = ":least"
	versionReg   = `^\d+\.\d+\.\d+$`
)
func (r *AppService) Default() {
	appservicelog.Info("default", "name", r.Name)
	if r.DeletionTimestamp.IsZero() {
		return
	}
	// 检查 Image 字段的最后一部分是否包含冒号（版本号）
	if lastColon := strings.LastIndex(r.Spec.Image, colon); lastColon == -1 {
		// 如果没有冒号，说明没有版本号，添加 ":latest"
		r.Spec.Image += leastVersion
	} else {
		// 从最后一个冒号位置开始检查是否有合法的版本号
		versionPart := r.Spec.Image[lastColon+1:]
		if !isValidVersion(versionPart) {
			r.Spec.Image += leastVersion
		}
	}
}
func isValidVersion(version string) bool {
	// 检查是否符合语义版本格式，例如 "1.0.0"
	matched, err := regexp.MatchString(versionReg, version)
	if err != nil {
		return false
	}
	return matched
}
```

### Validtatingwebhook

```
var (
	defaultReplicas = int32(1)
)
func (r *AppService) ValidateCreate() (admission.Warnings, error) {
	appservicelog.Info("validate create", "name", r.Name)
	if !checkSpecReplicas(r.Spec.Replicas) {
		r.Spec.Replicas = &defaultReplicas
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AppService) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	appservicelog.Info("validate update", "name", r.Name)
	if !checkSpecReplicas(r.Spec.Replicas) {
		r.Spec.Replicas = &defaultReplicas
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AppService) ValidateDelete() (admission.Warnings, error) {
	appservicelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
func checkSpecReplicas(i *int32) bool {
	if i == nil {
		return false // 添加了对 nil 指针的检查
	}
	if *i < 0 {
		return false
	}
	return true
}
```

## 添加证书

首先需要在cmd/main.go中增加cert路径

```
	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
		CertDir: "./cert",
	})
```

生成证书脚本

```
echo "根证书私钥"
openssl genrsa -out ca/private.key 2048

echo "根证书"
openssl req -new -x509 -days 36500 -key ca/private.key -out ca/public.crt  -config ca/fd.cnf

echo "服务器私钥"
openssl genrsa -out server/private.key 2048

echo "创建服务器证书csr"
openssl req -new -config server/fd.cnf  -key server/private.key -out server/fd.csr

echo "创建服务器证书"
openssl x509 -req -days 36500  -CA ca/public.crt -CAkey ca/private.key -set_serial 01 -in server/fd.csr -out server/public.crt -extfile server/fd.cnf -extensions SAN

mv ca/public.crt ca.crt
mv server/public.crt tls.crt
mv server/private.key tls.key
```

修改下面ca和server配置,将webhook-service改成你的,具体参照公司中controller项目中的cert目录。

```
[req]
prompt = no
distinguished_name = dn
[dn]
emailAddress = adsad@sda.com
C = CN
ST = Zhejiang
L = HangZhou
O = harmonycloud
OU = skyview
CN = olympus-controller
[SAN]
extendedKeyUsage=serverAuth
subjectAltName=@alt_name
[alt_name]
DNS.0 = webhook-service.caas-system.svc
```

将tls.crt和tls.key放到cert目录下,然后将tls.crt base64编码后放入caBundle

```
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: controller-mutate-admission-demo
webhooks:
  - admissionReviewVersions:
      - v1
    clientConfig:
      caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURyakNDQXBZQ0NRQ0YvNDNxSGdjUTZ6QU5CZ2txaGtpRzl3MEJBUXNGQURDQmx6RWNNQm9HQ1NxR1NJYjMKRFFFSkFSWU5ZV1J6WVdSQWMyUmhMbU52YlRFTE1Ba0dBMVVFQmhNQ1EwNHhFVEFQQmdOVkJBZ01DRnBvWldwcApZVzVuTVJFd0R3WURWUVFIREFoSVlXNW5XbWh2ZFRFVk1CTUdBMVVFQ2d3TWFHRnliVzl1ZVdOc2IzVmtNUkF3CkRnWURWUVFMREFkemEzbDJhV1YzTVJzd0dRWURWUVFEREJKdmJIbHRjSFZ6TFdOdmJuUnliMnhzWlhJd0lCY04KTWpRd056STFNRGd6T1RRNFdoZ1BNakV5TkRBM01ERXdPRE01TkRoYU1JR1hNUnd3R2dZSktvWklodmNOQVFrQgpGZzFoWkhOaFpFQnpaR0V1WTI5dE1Rc3dDUVlEVlFRR0V3SkRUakVSTUE4R0ExVUVDQXdJV21obGFtbGhibWN4CkVUQVBCZ05WQkFjTUNFaGhibWRhYUc5MU1SVXdFd1lEVlFRS0RBeG9ZWEp0YjI1NVkyeHZkV1F4RURBT0JnTlYKQkFzTUIzTnJlWFpwWlhjeEd6QVpCZ05WQkFNTUVtOXNlVzF3ZFhNdFkyOXVkSEp2Ykd4bGNqQ0NBU0l3RFFZSgpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFOeGgyZE05aXZUL1dXOFBqU3BwQXNSMUdlcFNXS3ZoCm1oUjJlTmRNODVSQVlxV2ZIS1oyeTBiNWZhMEV3cmFQWWg3OFF6WnBXUDVJeUEweXIzTzZWTlJ0RUNMcGQvdkUKSEhqRWhrNStLTG12S0pRaEt5anZJV0FQeGp6QS9BOE5WQVMwdlVEZ3IyUkZLZzNRZXpSVGhhc0Q3Y1NXa0gxbQpPVFdaSVg4dzkxaWpQTW1kS1ZlYk5nNkR2aFdmK0tydVVvWTVmaENoRWxDWXRRdXpZaWVuT2tFTXMwbmVFaDduCmNvNFpabERpOUtCUW5kNXJnVEpHemFJSUZmSmFVTFoyOHZaSlZ1MXdBUTBsSW5idjBPOXVtR0llYmVGNnpDTW4KZ3hCcXpuakN3bXI0MVg5UHRJWlhsVjg4ai9sU2RSVFlWbEpvQ1J2SVpFRDEvR3JYZkllMEpta0NBd0VBQVRBTgpCZ2txaGtpRzl3MEJBUXNGQUFPQ0FRRUFzNGFZWDc3RG1YQ1F5SVQrRXU2NmI2TjJpcTlsZUFhMmN3QjVzZHliCkJtNUZNditseExNVUx1MFJ5RFcvQUk0OFNHZFJ3KzRGY0s1UTdSeFVKZUZtM3B0V0FTV0FKVFVqTjNDNEdYWlcKd1ZrTXpwUFgvTE5CbnpQVUcxLzl6eDdid0ZBZUUxQ2I5R2xyRUFaMmtWYW1FSVVLVFY1b0VNQTI4RG1WdDlEeApmNXF3UlhMSGRkRmF4bnFmSlNRSjVqK21heDRFUXhXOWZ5dXhWR1VNVGdQMGxKaW9LenluV2xFOG1oeFZEYzVtCm1vcFk5Ryt2ZHhqdUFDUm5ZZi8wcmRnWWk0dWpPL2FyTUk2eEZqcWlmaWM1STdXcU9YNTZGYzdHalBncFVlSHcKaEQ0dG1pZ0ZIVi80ZUtCWWJ5Q1NWRzRJUUhIaTFVa3JHRjA5MDRnY3VFMHk0QT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
      service:
        name: webhook-service
        namespace: caas-system
        path: /mutate-batch-appservice-com-v1-appservice
        port: 9443
    failurePolicy: Fail
    matchPolicy: Equivalent
    name: mutating.batch.appservice.com.appservices
    reinvocationPolicy: Never
    rules:
      - apiGroups:
          - "batch.appservice.com"
        apiVersions:
          - "*"
        operations:
          - CREATE
          - UPDATE
        resources:
          - appservices
        scope: '*'
    sideEffects: None
    timeoutSeconds: 30
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: controller-demo-validating
webhooks:
  - admissionReviewVersions:
      - v1
    clientConfig:
      caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURyakNDQXBZQ0NRQ0YvNDNxSGdjUTZ6QU5CZ2txaGtpRzl3MEJBUXNGQURDQmx6RWNNQm9HQ1NxR1NJYjMKRFFFSkFSWU5ZV1J6WVdSQWMyUmhMbU52YlRFTE1Ba0dBMVVFQmhNQ1EwNHhFVEFQQmdOVkJBZ01DRnBvWldwcApZVzVuTVJFd0R3WURWUVFIREFoSVlXNW5XbWh2ZFRFVk1CTUdBMVVFQ2d3TWFHRnliVzl1ZVdOc2IzVmtNUkF3CkRnWURWUVFMREFkemEzbDJhV1YzTVJzd0dRWURWUVFEREJKdmJIbHRjSFZ6TFdOdmJuUnliMnhzWlhJd0lCY04KTWpRd056STFNRGd6T1RRNFdoZ1BNakV5TkRBM01ERXdPRE01TkRoYU1JR1hNUnd3R2dZSktvWklodmNOQVFrQgpGZzFoWkhOaFpFQnpaR0V1WTI5dE1Rc3dDUVlEVlFRR0V3SkRUakVSTUE4R0ExVUVDQXdJV21obGFtbGhibWN4CkVUQVBCZ05WQkFjTUNFaGhibWRhYUc5MU1SVXdFd1lEVlFRS0RBeG9ZWEp0YjI1NVkyeHZkV1F4RURBT0JnTlYKQkFzTUIzTnJlWFpwWlhjeEd6QVpCZ05WQkFNTUVtOXNlVzF3ZFhNdFkyOXVkSEp2Ykd4bGNqQ0NBU0l3RFFZSgpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFOeGgyZE05aXZUL1dXOFBqU3BwQXNSMUdlcFNXS3ZoCm1oUjJlTmRNODVSQVlxV2ZIS1oyeTBiNWZhMEV3cmFQWWg3OFF6WnBXUDVJeUEweXIzTzZWTlJ0RUNMcGQvdkUKSEhqRWhrNStLTG12S0pRaEt5anZJV0FQeGp6QS9BOE5WQVMwdlVEZ3IyUkZLZzNRZXpSVGhhc0Q3Y1NXa0gxbQpPVFdaSVg4dzkxaWpQTW1kS1ZlYk5nNkR2aFdmK0tydVVvWTVmaENoRWxDWXRRdXpZaWVuT2tFTXMwbmVFaDduCmNvNFpabERpOUtCUW5kNXJnVEpHemFJSUZmSmFVTFoyOHZaSlZ1MXdBUTBsSW5idjBPOXVtR0llYmVGNnpDTW4KZ3hCcXpuakN3bXI0MVg5UHRJWlhsVjg4ai9sU2RSVFlWbEpvQ1J2SVpFRDEvR3JYZkllMEpta0NBd0VBQVRBTgpCZ2txaGtpRzl3MEJBUXNGQUFPQ0FRRUFzNGFZWDc3RG1YQ1F5SVQrRXU2NmI2TjJpcTlsZUFhMmN3QjVzZHliCkJtNUZNditseExNVUx1MFJ5RFcvQUk0OFNHZFJ3KzRGY0s1UTdSeFVKZUZtM3B0V0FTV0FKVFVqTjNDNEdYWlcKd1ZrTXpwUFgvTE5CbnpQVUcxLzl6eDdid0ZBZUUxQ2I5R2xyRUFaMmtWYW1FSVVLVFY1b0VNQTI4RG1WdDlEeApmNXF3UlhMSGRkRmF4bnFmSlNRSjVqK21heDRFUXhXOWZ5dXhWR1VNVGdQMGxKaW9LenluV2xFOG1oeFZEYzVtCm1vcFk5Ryt2ZHhqdUFDUm5ZZi8wcmRnWWk0dWpPL2FyTUk2eEZqcWlmaWM1STdXcU9YNTZGYzdHalBncFVlSHcKaEQ0dG1pZ0ZIVi80ZUtCWWJ5Q1NWRzRJUUhIaTFVa3JHRjA5MDRnY3VFMHk0QT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
      service:
        name: webhook-service
        namespace: caas-system
        path: /validate-batch-appservice-com-v1-appservice
        port: 9443
    failurePolicy: Fail
    matchPolicy: Equivalent
    name: validating.batch.appservice.com.appservices
    rules:
      - apiGroups:
          - "batch.appservice.com"
        apiVersions:
          - "*"
        operations:
          - CREATE
          - UPDATE
        resources:
          - appservices
        scope: '*'
    sideEffects: None
    timeoutSeconds: 30
```

# 测试

修改Dockerfile

```
# Build the manager binary
FROM 10.10.103.155/library/golang:1.22 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor vendor
# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/


# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -mod=vendor -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM 10.10.103.155/library/alpine:3.18
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
COPY cert cert

ENTRYPOINT ["/manager"]
```

deploy.yaml

```
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-demo-sa
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-demo
spec:
  selector:
    matchLabels:
      app: controller-demo
  template:
    metadata:
      labels:
        app: controller-demo
    spec:
      serviceAccountName: controller-demo-sa
      containers:
        - name: controller-demo
          image: 10.10.103.155/testdemo/controller-demo:latest
          imagePullPolicy: Always
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: controller-demo-role
rules:
  - apiGroups:
      - "batch.appservice.com"
    resources:
      - appservices
    verbs:
      - get
      - list
      - create
      - delete
      - update
      - patch
  - apiGroups:
      - "apps"
    resources:
      - deployments
    verbs:
      - get
      - list
      - create
      - delete
      - update
      - patch
  - apiGroups:
      - ""
    resources:
      - services
    verbs:
      - get
      - list
      - create
      - delete
      - update
      - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: controller-demo-rolebinding
subjects:
  - kind: ServiceAccount
    name: controller-demo-sa
    namespace: caas-system
roleRef:
  kind: ClusterRole
  name: controller-demo-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: v1
kind: Service
metadata:
  name: webhook-service
spec:
  selector:
    app: controller-demo
  ports:
    - port: 9443
      targetPort: 9443
      protocol: TCP
      name: controller-demo
```

```
kubectl apply -f deploy.yaml
kubectl apply -f webhook.yaml
make install
make run
kubectl apply -f nginx.yaml
```

# 总结

使用kubebuilder只需要关注todo:

在使用controller更新字段，建议使用patch来更新

在使用webhook的时候需要注意证书