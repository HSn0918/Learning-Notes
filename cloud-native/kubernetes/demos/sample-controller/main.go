// sample-controller demo:
// 监听集群中所有 ConfigMap 资源，把变更入队后异步 reconcile（这里只是打印日志）。
//
// 对应笔记：cloud-native/kubernetes/internals/client-go-source.md
// 数据流：Reflector -> DeltaFIFO -> Indexer -> EventHandler -> Workqueue -> Worker -> syncHandler
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// Controller 是标准 sample-controller 风格的控制器：
//   - cmLister 从本地缓存读 ConfigMap
//   - cmSynced 用来等首批 List 完成
//   - queue 解耦事件与处理，提供去重 + 重试
type Controller struct {
	kubeclient kubernetes.Interface
	cmLister   corelisters.ConfigMapLister
	cmSynced   cache.InformerSynced
	queue      workqueue.TypedRateLimitingInterface[string]
}

func NewController(kubeclient kubernetes.Interface, cmInformer coreinformers.ConfigMapInformer) *Controller {
	c := &Controller{
		kubeclient: kubeclient,
		cmLister:   cmInformer.Lister(),
		cmSynced:   cmInformer.Informer().HasSynced,
		queue: workqueue.NewTypedRateLimitingQueue[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
		),
	}

	// EventHandler 只做 enqueue —— 重活留给 syncHandler
	_, err := cmInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCM := oldObj.(*corev1.ConfigMap)
			newCM := newObj.(*corev1.ConfigMap)
			// ResourceVersion 相同说明是 Resync，可按需过滤掉
			if oldCM.ResourceVersion == newCM.ResourceVersion {
				return
			}
			c.enqueue(newObj)
		},
		DeleteFunc: c.enqueue,
	})
	if err != nil {
		log.Fatalf("AddEventHandler failed: %v", err)
	}
	return c
}

// enqueue 把对象转成 namespace/name key 后入队。
func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("compute key: %w", err))
		return
	}
	c.queue.Add(key)
}

// Run 是控制器入口：等缓存同步 -> 启动 N 个 worker -> 等 stop 信号。
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	log.Println("Starting sample-controller (watching ConfigMaps)")

	// 等首批 List 全部进入 Indexer 后再开始处理，避免基于残缺数据 reconcile
	if !cache.WaitForCacheSync(ctx.Done(), c.cmSynced) {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	log.Println("Caches synced, starting workers")

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	<-ctx.Done()
	log.Println("Shutting down")
	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

// processNextItem 阻塞取一个 key，调用 syncHandler，按结果决定 Forget/AddRateLimited。
func (c *Controller) processNextItem(ctx context.Context) bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key) // 无论成败都要 Done

	err := c.syncHandler(ctx, key)
	switch {
	case err == nil:
		c.queue.Forget(key) // 成功：清零该 key 的失败计数
	case c.queue.NumRequeues(key) < 5:
		log.Printf("retry %s (attempt %d): %v", key, c.queue.NumRequeues(key), err)
		c.queue.AddRateLimited(key) // 失败但还有配额：按指数退避重入队
	default:
		c.queue.Forget(key) // 重试超限：放弃
		utilruntime.HandleError(fmt.Errorf("dropping %s after 5 retries: %w", key, err))
	}
	return true
}

// syncHandler 是真正的 reconcile 逻辑（这里只打印一条日志）。
// 注意必须幂等：同一 key 可能因 Resync/重试/多次变更被处理多次。
func (c *Controller) syncHandler(_ context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("split key %q: %w", key, err)
	}

	cm, err := c.cmLister.ConfigMaps(namespace).Get(name)
	if errors.IsNotFound(err) {
		// 对象已删除：执行清理逻辑，视为成功
		log.Printf("reconciled %s/%s (deleted)", namespace, name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("lister get %s/%s: %w", namespace, name, err)
	}

	log.Printf("reconciled %s/%s (resourceVersion=%s, keys=%d)",
		cm.Namespace, cm.Name, cm.ResourceVersion, len(cm.Data))
	return nil
}

func main() {
	var kubeconfig string
	if home := os.Getenv("HOME"); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "absolute path to the kubeconfig file")
	workers := flag.Int("workers", 2, "number of reconcile workers")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("build kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("build clientset: %v", err)
	}

	// 30s Resync 给一次「兜底重新触发 UpdateFunc」的机会，不会访问 API Server
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
	cmInformer := factory.Core().V1().ConfigMaps()

	controller := NewController(clientset, cmInformer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Start 之后 factory 内的所有已创建 Informer 会启动各自的 Reflector
	factory.Start(ctx.Done())

	if err := controller.Run(ctx, *workers); err != nil {
		log.Fatalf("controller exited: %v", err)
	}
}
