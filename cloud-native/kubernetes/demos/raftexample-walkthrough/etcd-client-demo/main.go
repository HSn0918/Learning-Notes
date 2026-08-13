// etcd-client-demo: 用 clientv3 演示 Put / Get / Watch / Lease 与 revision 概念
//
// 配套笔记：cloud-native/kubernetes/internals/etcd-source.md（MVCC / Watch 章节）
// 启动 etcd：docker run -d -p 2379:2379 --name etcd-demo \
//   quay.io/coreos/etcd:v3.5.18 \
//   etcd --advertise-client-urls=http://0.0.0.0:2379 --listen-client-urls=http://0.0.0.0:2379
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const demoKey = "/learning-notes/etcd-demo/foo"

func main() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalf("connect etcd: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// --- 1) 先起 Watch：从「当前 revision + 1」开始监听，再做写入观察事件流 ---
	// 等价于 K8s apiserver 里的 clientv3.WithRev(initialRev+1)
	statusResp, err := cli.Status(ctx, "127.0.0.1:2379")
	if err != nil {
		log.Fatalf("status: %v", err)
	}
	startRev := statusResp.Header.Revision + 1
	fmt.Printf("[start] current etcd revision = %d, watch from rev=%d\n",
		statusResp.Header.Revision, startRev)

	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()
	wch := cli.Watch(watchCtx, demoKey, clientv3.WithRev(startRev), clientv3.WithPrevKV())
	go func() {
		for wresp := range wch {
			for _, ev := range wresp.Events {
				fmt.Printf("[watch] rev=%d type=%s key=%s value=%s\n",
					ev.Kv.ModRevision, ev.Type, ev.Kv.Key, ev.Kv.Value)
			}
		}
	}()

	// --- 2) Put 三次：观察每次 ModRevision / CreateRevision 的变化 ---
	for i, v := range []string{"v1", "v2", "v3"} {
		putResp, err := cli.Put(ctx, demoKey, v)
		if err != nil {
			log.Fatalf("put #%d: %v", i, err)
		}
		// putResp.Header.Revision 就是这次写入分配的全局 revision
		// 在 K8s 里，apiserver 把它装进 ObjectMeta.ResourceVersion 返回给客户端
		fmt.Printf("[put]   #%d value=%s -> revision=%d\n", i, v, putResp.Header.Revision)
	}

	// --- 3) Get：拿到当前值 + CreateRevision/ModRevision/Version，对应 mvccpb.KeyValue ---
	getResp, err := cli.Get(ctx, demoKey)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	for _, kv := range getResp.Kvs {
		fmt.Printf("[get]   key=%s value=%s create=%d mod=%d version=%d (lease=%d)\n",
			kv.Key, kv.Value, kv.CreateRevision, kv.ModRevision, kv.Version, kv.Lease)
	}

	// --- 4) Get 历史 rev：MVCC 让我们能读「过去某个 revision 的值」 ---
	histResp, err := cli.Get(ctx, demoKey, clientv3.WithRev(startRev)) // startRev 是 v1 落地那次
	if err != nil {
		log.Fatalf("get historical: %v", err)
	}
	for _, kv := range histResp.Kvs {
		fmt.Printf("[get@rev=%d] value=%s\n", startRev, kv.Value)
	}

	// --- 5) Lease：5 秒 TTL 的租约，挂上一个 key，过期后 key 自动被删 ---
	leaseResp, err := cli.Grant(ctx, 5)
	if err != nil {
		log.Fatalf("grant lease: %v", err)
	}
	leaseKey := "/learning-notes/etcd-demo/lease-bound"
	if _, err = cli.Put(ctx, leaseKey, "ephemeral", clientv3.WithLease(leaseResp.ID)); err != nil {
		log.Fatalf("put with lease: %v", err)
	}
	fmt.Printf("[lease] id=%x ttl=5s key=%s (5s 后被自动 revoke)\n", leaseResp.ID, leaseKey)

	// 等一会让 watch 把所有事件打印出来，再等 lease 过期事件
	time.Sleep(6 * time.Second)
	getAfter, _ := cli.Get(context.Background(), leaseKey)
	fmt.Printf("[lease] after TTL: %s 还存在? %v\n", leaseKey, len(getAfter.Kvs) > 0)
}
