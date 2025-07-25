# ✅ Kubernetes CSI（容器存储接口）讲解 + 面试速记

------

## 🧩 一、什么是 CSI？

> CSI，全称 **Container Storage Interface**，是 CNCF 制定的一套 **标准存储插件接口规范**，用于让 Kubernetes 和各种存储系统解耦。

### 📌 理解要点：

- 就像 CNI 是管理网络的插件接口，**CSI 是用于挂载和卸载存储卷的插件接口**。
- 支持动态创建、挂载、扩容、快照、回收等功能。
- 允许第三方厂商（如 Ceph、阿里云、腾讯云）编写 CSI 插件，兼容 K8s。

------

### 🤔 为什么有 CSI？

早期 Kubernetes 的存储插件是 **内置的（in-tree）**，耦合性高，维护困难。
 CSI 是 **out-of-tree 插件机制**，插件运行在用户空间，K8s 只调用统一接口，不再关心存储实现细节。

------

## 🧱 二、CSI 的核心组成

一个完整的 CSI 插件包括：

| 组件                     | 作用                                 |
| ------------------------ | ------------------------------------ |
| **external-provisioner** | 负责处理动态创建 PVC 的请求          |
| **external-attacher**    | 将卷附加到节点（attach）             |
| **external-snapshotter** | 管理卷快照（可选）                   |
| **external-resizer**     | 处理 PVC 扩容（可选）                |
| **node plugin**          | 在 Node 上运行，挂载卷到本地目录     |
| **controller plugin**    | 管理控制面（创建卷、删除卷、快照等） |

------

## 📦 三、CSI 在 Kubernetes 中是怎么工作的？

> 你可以用一句话记住 CSI 的职责： **当 Pod 用了 PVC 时，CSI 插件就负责给它找一个卷挂上去。**

------

### 🧰 工作流程图（简化版）：

```text
用户申请 PVC
  ↓
StorageClass 绑定了 CSI 插件
  ↓
external-provisioner 调用 controller plugin 创建卷
  ↓
external-attacher 绑定卷到 Node
  ↓
node plugin 挂载存储卷到 Pod 的目录
```

------

## 💾 四、常见的 CSI 插件（记几个常考的就够）

| 插件名称                   | 类型             | 适用场景              |
| -------------------------- | ---------------- | --------------------- |
| **hostPath**               | 本地             | 演示/测试环境，非生产 |
| **nfs-csi**                | 网络文件系统     | 多 Pod 共享读写       |
| **ceph-csi (RBD, CephFS)** | 分布式存储       | 云原生环境、高可用    |
| **alicloud/disk-csi**      | 云盘（块存储）   | 阿里云场景            |
| **ebs-csi-driver**         | AWS EBS          | 云上块存储            |
| **cinder-csi**             | OpenStack Cinder | 私有云场景            |
| **longhorn**               | 轻量级分布式存储 | Rancher 等环境        |

------

### 📌 面试建议掌握这几个：

1. **hostPath**：最简单，调试测试用
2. **nfs-csi**：多 Pod 共享场景，熟悉安装配置很有用
3. **ceph-csi**：中大型集群常用，支持快照、扩容、RWO/RWX
4. **云厂商 CSI 插件（EBS/Disk/Cinder）**：面试公司用哪个云你就答哪个！

------

## 🛠 五、你要能答出这些命令和配置：

```yaml
# StorageClass 示例（动态供给 + ceph-csi）
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ceph-block
provisioner: rbd.csi.ceph.com    # 对应的 CSI 插件名
parameters:
  pool: rbd-pool
  imageFormat: "2"
reclaimPolicy: Delete
volumeBindingMode: Immediate
# 创建 PVC，指定 StorageClass
kubectl apply -f pvc.yaml

# 查看已经加载的 CSI 驱动
kubectl get csidrivers

# 查看卷绑定情况
kubectl get pvc,pv
```

------

## 🧠 面试常问 Q&A（建议熟练背下来）

------

### Q1: **什么是 CSI？它解决了什么问题？**

**答：**
 CSI 是一套容器存储接口规范，Kubernetes 使用它来动态管理存储卷挂载。它解决了内置插件耦合度高、更新困难的问题，使得第三方可以独立开发和更新存储插件。

------

### Q2: **Kubernetes 中使用 CSI 的流程是怎样的？**

**答：**
 用户通过 PVC 请求存储 → StorageClass 绑定了 CSI 插件 → CSI 控制器创建卷 → 节点插件挂载卷 → Pod 使用挂载的目录作为持久化存储。

------

### Q3: **你用过哪些 CSI 插件？在什么场景？**

**答（示例）：**
 我使用过 ceph-csi 和 nfs-csi 插件。在需要共享读写时使用 NFS；在高性能、高可用场景下使用 Ceph RBD；也了解过 hostPath，用于本地测试。

------

### Q4: **StorageClass 和 CSI 是什么关系？**

**答：**
 StorageClass 是 PVC 和 CSI 插件之间的桥梁，它定义了要用哪个 CSI 插件（provisioner 字段）和存储参数（如 pool、fsType）。PVC 绑定到 StorageClass 后，Kubernetes 会调用对应 CSI 插件来处理存储卷。

------

### Q5: **你如何调试 PVC 挂载失败的问题？**

| 检查点        | 命令                                            |
| ------------- | ----------------------------------------------- |
| PVC 状态      | `kubectl describe pvc`                          |
| PV 状态       | `kubectl get pv`                                |
| CSI 插件日志  | `kubectl logs -n kube-system -l app=ceph-csi-*` |
| Node 插件状态 | `kubectl get pod -n kube-system -o wide`        |
| 事件          | `kubectl describe pod <pod>` 查看挂载失败错误   |

------

## ✅ 一句话总结：

> CSI 是 Kubernetes 的标准存储插件机制，允许将第三方存储系统无缝接入 K8s，实现持久化、共享、动态供给、扩容和快照等能力，是容器存储的基础组件。

