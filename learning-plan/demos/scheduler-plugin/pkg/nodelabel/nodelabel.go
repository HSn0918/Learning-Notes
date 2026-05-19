// Package nodelabel 实现一个 Score 插件：NodeLabelScore。
//
// 规则：节点带 label  learning-plan/preferred=true  → 原始分 100；否则 0。
// 跑完 Score 后由 NormalizeScore 把本插件所有节点的原始分映射到 [0, MaxNodeScore]。
//
// 这是 framework.ScorePlugin 接口的教科书式实现，可作为编写自定义打分插件的脚手架。
package nodelabel

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	// fwk 是 k8s 1.34 抽出的稳定接口包：Status/Code/CycleState/NodeInfo 等都在这里。
	fwk "k8s.io/kube-scheduler/framework"
	// framework 仍保留具体类型：ScorePlugin/Handle/NodeScoreList/MaxNodeScore。
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Name 是插件名，需要与 KubeSchedulerConfiguration 里 plugins.score.enabled[].name 完全一致。
const Name = "NodeLabelScore"

// PreferredLabelKey/Value 写死在这里，方便演示。
// 生产场景应该把它做成 PluginConfig 的 Args（实现 KubeSchedulerProfileArgs 并在 New 里解析）。
const (
	PreferredLabelKey   = "learning-plan/preferred"
	PreferredLabelValue = "true"
)

// NodeLabelScore 实现 framework.ScorePlugin。
type NodeLabelScore struct {
	handle framework.Handle
}

// 编译期断言：确保我们正确实现了这两个接口。
var (
	_ framework.ScorePlugin     = &NodeLabelScore{}
	_ framework.ScoreExtensions = &NodeLabelScore{}
)

// New 是 PluginFactory，被 app.WithPlugin 注册到 out-of-tree registry。
// 签名见 runtime.PluginFactory： func(ctx, configuration runtime.Object, f framework.Handle)
//
//	(framework.Plugin, error)
//
// args 当前未用——如果以后想从 PluginConfig 接收参数，在这里 decode 即可。
func New(_ context.Context, _ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &NodeLabelScore{handle: h}, nil
}

// Name 实现 framework.Plugin。
func (pl *NodeLabelScore) Name() string { return Name }

// Score 对单个节点打分。Score 阶段会被 framework 跨节点并行调用，所以这里不要写共享状态。
//
// k8s 1.34 起 Score 直接收 fwk.NodeInfo（与 Filter 阶段同一份 snapshot），
// 不再需要自己从 SnapshotSharedLister 里 Get。
//
// 返回值：(原始分, *fwk.Status)。
//   - 原始分范围由插件自定（可大于 100），最终在 NormalizeScore 里再归一化。
//   - *Status 非 nil 表示打分失败，整个调度周期会被终止。
func (pl *NodeLabelScore) Score(ctx context.Context, state fwk.CycleState,
	pod *v1.Pod, nodeInfo fwk.NodeInfo) (int64, *fwk.Status) {

	node := nodeInfo.Node()
	if node == nil {
		return 0, fwk.NewStatus(fwk.Error, "node is nil in snapshot")
	}

	if node.Labels[PreferredLabelKey] == PreferredLabelValue {
		klog.V(4).InfoS("NodeLabelScore matched", "pod", klog.KObj(pod), "node", node.Name)
		return 100, nil
	}
	return 0, nil
}

// ScoreExtensions 返回 NormalizeScore 扩展。
// 如果你的插件原始分已经在 [0, framework.MaxNodeScore] 区间，可以直接 return nil 跳过归一化。
func (pl *NodeLabelScore) ScoreExtensions() framework.ScoreExtensions {
	return pl
}

// NormalizeScore 把本插件在所有节点上的原始分映射到 [0, framework.MaxNodeScore]。
//
// 当前实现：取本周期最大原始分作为分母，按比例放大。
// 这样：若至少有一个节点带 preferred label，它会被映射到 MaxNodeScore；
//
//	若没有任何节点匹配，所有节点都 0，保持不变。
func (pl *NodeLabelScore) NormalizeScore(ctx context.Context, state fwk.CycleState,
	pod *v1.Pod, scores framework.NodeScoreList) *fwk.Status {

	var highest int64
	for _, s := range scores {
		if s.Score > highest {
			highest = s.Score
		}
	}
	if highest == 0 {
		return nil // 全 0，无需归一化
	}
	for i := range scores {
		scores[i].Score = scores[i].Score * framework.MaxNodeScore / highest
	}
	return nil
}
