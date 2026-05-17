# sample-controller demo

> **状态**：✅ Mac 验证可运行（Go 1.26，先 `go mod tidy`） · 详见 [demos 验证总表](../README.md)

一个最小化、可直接运行的 client-go 控制器示例：监听集群中所有 `ConfigMap`，把变更入队后异步 reconcile（这里只是打印一条 `reconciled <ns>/<name>` 日志）。

它演示了标准 sample-controller 模式的完整骨架：

```
Reflector → DeltaFIFO → Indexer → EventHandler.enqueue
                                          ↓
                                       Workqueue
                                          ↓
                                runWorker → syncHandler
```

## 运行

```bash
cd learning-plan/demos/sample-controller
go mod tidy
go run . --kubeconfig ~/.kube/config
```

不传 `--kubeconfig` 时默认读取 `$HOME/.kube/config`。

## 预期行为

启动时：

```
Starting sample-controller (watching ConfigMaps)
Caches synced, starting workers
reconciled kube-system/kube-root-ca.crt (resourceVersion=..., keys=1)
reconciled kube-public/cluster-info (resourceVersion=..., keys=2)
...
```

之后在任意 namespace 创建/修改/删除 ConfigMap 都会触发对应日志：

```bash
kubectl create configmap demo --from-literal=foo=bar
kubectl patch configmap demo -p '{"data":{"foo":"baz"}}'
kubectl delete configmap demo
```

输出：

```
reconciled default/demo (resourceVersion=..., keys=1)
reconciled default/demo (resourceVersion=..., keys=1)
reconciled default/demo (deleted)
```

Ctrl+C 触发 `SIGINT`，控制器优雅退出（关闭 workqueue → worker 自然结束）。

## 与笔记对应

| 行为 | 笔记 `client-go-source.md` 的对应章节 |
| --- | --- |
| `informers.NewSharedInformerFactory` + `factory.Start` | SharedInformerFactory 共享机制 |
| `cmInformer.Informer().AddEventHandler` | processorListener：事件分发 |
| `cache.WaitForCacheSync` | Resync 与 HasSynced |
| `workqueue.NewTypedRateLimitingQueue` + `DefaultTypedControllerRateLimiter` | Workqueue：去重、限速、重试 |
| `c.enqueue` 把对象转 key 后 `queue.Add` | 基础队列的去重与并发安全 |
| `processNextItem` 中的 `Get`/`Done`/`Forget`/`AddRateLimited` | RateLimiter 与指数退避 |
| `cmLister.ConfigMaps(ns).Get(name)` 读本地缓存 | Indexer 与 ThreadSafeStore |
| `syncHandler` 必须幂等且处理 NotFound | 自定义控制器骨架 |

## 文件结构

```
sample-controller/
├── go.mod                           # 模块定义，依赖 k8s.io/client-go v0.33.0
├── main.go                          # 全部代码，约 180 行
├── README.md                        # 本文
└── demo-sample-controller.md        # 笔记式简介（含标签、相关笔记链接）
```
