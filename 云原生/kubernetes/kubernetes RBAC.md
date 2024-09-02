## 背景

基于角色（Role）的访问控制（RBAC）是一种基于组织中用户的角色来调节控制对计算机或网络资源的访问的方法。

RBAC 鉴权机制使用 `rbac.authorization.k8s.io` [API 组](https://kubernetes.io/zh-cn/docs/concepts/overview/kubernetes-api/#api-groups-and-versioning)来驱动鉴权决定， 允许你通过 Kubernetes API 动态配置策略。

## k8s中的api对象

RBAC API 声明了四种 Kubernetes 对象：**Role**、**ClusterRole**、**RoleBinding** 和 **ClusterRoleBinding**。你可以像使用其他 Kubernetes 对象一样， 通过类似 `kubectl` 这类工具描述或修补 RBAC 对象。

### Role和ClusterRole

其中 **Role** 针对的是 **namespace** ，而 **ClusterRole** 针对的是**集群**范围的角色

### RoleBinding和ClusterRoleBinding

RoleBinding 在指定的名字空间中执行授权，而 ClusterRoleBinding 在集群范围执行授权。

一个 RoleBinding 可以引用同一的名字空间中的任何 Role。 或者，一个 RoleBinding 可以引用某 ClusterRole 并将该 ClusterRole 绑定到 RoleBinding 所在的名字空间。 如果你希望将某 ClusterRole 绑定到集群中所有名字空间，你要使用 ClusterRoleBinding。

### 总结

![](https://cdn.nlark.com/yuque/0/2024/png/46821905/1721878083743-1b364649-77b5-44ca-8124-a6231204d75b.png)

## 测试

```
---
apiVersion: v1
kind: Namespace
metadata:
  name: staging
---
apiVersion: v1
kind: Namespace
metadata:
  name: prod
```

```
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: qa-sa
  namespace: staging
```

```
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: qa-role
  namespace: staging
rules:
  - apiGroups:
      - ""
    resources:
      - services
      - pods
      - pods/log
    verbs:
      - get
      - list
```

```
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: qa-role-binding
  namespace: staging
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: qa-role
subjects:
  - kind: ServiceAccount
    name: qa-sa
    namespace: staging
```

```
---
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  namespace: staging
spec:
  containers:
    - name: myapp
      image: aputra/myapp-192:v2
      ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  namespace: prod
spec:
  containers:
    - name: myapp
      image: aputra/myapp-192:v2
      ports:
        - containerPort: 8080
```

```
kubectl auth can-i get pods -n staging --as=system:serviceaccount:staging:qa-sa
kubectl auth can-i get pods -n prod --as=system:serviceaccount:staging:qa-sa
```