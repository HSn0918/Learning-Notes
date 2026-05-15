module github.com/learning-notes/scheduler-plugin

go 1.26

// --- 重要：构建说明 ---
//
// 直接 `require k8s.io/kubernetes vX.Y.Z` 通常会失败，因为 kubernetes 主仓
// 的 go.mod 用了一组 staging 子模块（k8s.io/api、k8s.io/apimachinery、
// k8s.io/component-base、k8s.io/kube-scheduler 等），而这些子模块在 kubernetes
// 主仓里只发 `v0.0.0` 占位版本。要真正能 `go build`，需要把这些 staging 模块
// 用 replace 指回 kubernetes/staging/src/k8s.io/<module>。
//
// 完整可参考的真实例子：
//   https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/go.mod
// （它就是社区主流的 out-of-tree scheduler plugin 模板。）
//
// 本 demo 着重展示代码结构与插件接口的正确用法；要在本地真正构建出二进制，
// 建议直接 fork scheduler-plugins 仓库或在它的 go.mod 上加一行 replace 把
// 本目录的 pkg/nodelabel 引进去。

require (
	k8s.io/api v0.34.0
	k8s.io/apimachinery v0.34.0
	k8s.io/component-base v0.34.0
	k8s.io/klog/v2 v2.130.1
	k8s.io/kubernetes v1.34.0
)

// 下面这一段 replace 列表是真实使用时必须有的；这里只列出最常用的几个作为占位。
// 实际版本号需要与 require 的 k8s.io/kubernetes 对齐，例如 v1.34.0 时全部用 v0.34.0。
//
// replace (
// 	k8s.io/api                    => k8s.io/api v0.34.0
// 	k8s.io/apimachinery           => k8s.io/apimachinery v0.34.0
// 	k8s.io/apiserver              => k8s.io/apiserver v0.34.0
// 	k8s.io/client-go              => k8s.io/client-go v0.34.0
// 	k8s.io/cloud-provider         => k8s.io/cloud-provider v0.34.0
// 	k8s.io/cluster-bootstrap      => k8s.io/cluster-bootstrap v0.34.0
// 	k8s.io/code-generator         => k8s.io/code-generator v0.34.0
// 	k8s.io/component-base         => k8s.io/component-base v0.34.0
// 	k8s.io/component-helpers      => k8s.io/component-helpers v0.34.0
// 	k8s.io/controller-manager     => k8s.io/controller-manager v0.34.0
// 	k8s.io/cri-api                => k8s.io/cri-api v0.34.0
// 	k8s.io/csi-translation-lib    => k8s.io/csi-translation-lib v0.34.0
// 	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.34.0
// 	k8s.io/kms                    => k8s.io/kms v0.34.0
// 	k8s.io/kube-aggregator        => k8s.io/kube-aggregator v0.34.0
// 	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.34.0
// 	k8s.io/kube-proxy             => k8s.io/kube-proxy v0.34.0
// 	k8s.io/kube-scheduler         => k8s.io/kube-scheduler v0.34.0
// 	k8s.io/kubectl                => k8s.io/kubectl v0.34.0
// 	k8s.io/kubelet                => k8s.io/kubelet v0.34.0
// 	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.34.0
// 	k8s.io/metrics                => k8s.io/metrics v0.34.0
// 	k8s.io/mount-utils            => k8s.io/mount-utils v0.34.0
// 	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.34.0
// 	k8s.io/sample-apiserver       => k8s.io/sample-apiserver v0.34.0
// )
