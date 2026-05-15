// Package main 是一个最小的 controller-runtime Operator 示例。
//
// 它演示了 Manager / Builder / Reconciler 的核心装配，而不依赖任何 CRD：
//   - 监听某个 namespace 下所有 ConfigMap 的变化
//   - 当源 ConfigMap (label demo.learning-notes/source=true) 有特定 annotation 时，
//     把该 annotation 同步到名为 <source-name>-mirror 的 ConfigMap 上。
//
// 跑法见 README.md。
package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// 把内置类型（ConfigMap、Pod、Deployment 等）注册到 scheme
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		watchNamespace       string
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "metrics endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "health probes binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "enable leader election for HA.")
	flag.StringVar(&watchNamespace, "namespace", "default", "namespace to watch ConfigMaps in.")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// 创建 Manager —— 它会读 kubeconfig（in-cluster 或 ~/.kube/config，由
	// ctrl.GetConfigOrDie 决定），构造 Cache / Client / EventRecorder。
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		Metrics:                 metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress:  probeAddr,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "kubebuilder-operator-demo.learning-notes.io",
		// 限定 Cache 监听的 namespace，避免拉全集群对象造成内存浪费
		// 真实库 Options.Cache.DefaultNamespaces 接受 map
		// （v0.16+ API；旧版本用 Namespace 字段）
		// 这里保持注释形式，运行时通过 --namespace 控制 Reconciler 自己 filter
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// 注册 ConfigMapReconciler
	if err := (&ConfigMapReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		WatchNamespace: watchNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup ConfigMapReconciler")
		os.Exit(1)
	}

	// 健康检查端点（kubectl probe / k8s liveness 用）
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager", "namespace", watchNamespace)
	// SetupSignalHandler 监听 SIGTERM/SIGINT —— 第一次取消 ctx，第二次直接 os.Exit
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
