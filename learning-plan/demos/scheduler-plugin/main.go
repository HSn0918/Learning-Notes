// scheduler-plugin demo —— 一个最小可运行的 out-of-tree kube-scheduler，
// 在原生 default plugin 之上额外注册一个自定义 Score 插件 NodeLabelScore。
//
// 用法（编译完成后）：
//
//	./scheduler-plugin \
//	    --config=/etc/kubernetes/scheduler-plugin/config.yaml \
//	    --leader-elect=true \
//	    -v=4
//
// 参考：kubernetes-sigs/scheduler-plugins 的 cmd/scheduler/main.go。
package main

import (
	"os"

	"k8s.io/component-base/cli"
	_ "k8s.io/component-base/logs/json/register" // 注册 JSON 日志格式
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	"github.com/learning-notes/scheduler-plugin/pkg/nodelabel"
)

func main() {
	// 关键 API：app.NewSchedulerCommand 是 kube-scheduler 主入口；
	// app.WithPlugin 把自定义插件工厂加入 out-of-tree registry，让 KubeSchedulerConfiguration
	// 里能通过 plugins.<extPoint>.enabled[].name = "NodeLabelScore" 启用它。
	command := app.NewSchedulerCommand(
		app.WithPlugin(nodelabel.Name, nodelabel.New),
		// 可以同时注册多个：
		// app.WithPlugin(otherplugin.Name, otherplugin.New),
	)

	code := cli.Run(command)
	os.Exit(code)
}
