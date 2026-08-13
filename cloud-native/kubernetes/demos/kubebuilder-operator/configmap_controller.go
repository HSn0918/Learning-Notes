package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// 只有打了这个 label 的 ConfigMap 才被视为 "source"，触发 Reconcile
	sourceLabelKey   = "demo.learning-notes/source"
	sourceLabelValue = "true"

	// 要同步的 annotation key
	syncAnnotation = "demo.learning-notes/payload"

	mirrorSuffix = "-mirror"
)

// ConfigMapReconciler 监听一个 namespace 下的 ConfigMap，
// 把带 source label 的 CM 的指定 annotation 复制到镜像 CM。
type ConfigMapReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	WatchNamespace string
}

// Reconcile 是核心逻辑。注意它只拿到 Request (namespace/name)，对象本身要自己 Get。
func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("configmap", req.NamespacedName)

	// 1) 取源对象（走 Cache）。NotFound 说明已被删除，无需重试
	var source corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &source); err != nil {
		if apierrors.IsNotFound(err) {
			// 同时尝试删除 mirror，保持级联
			mirrorName := req.Name + mirrorSuffix
			_ = r.Delete(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: req.Namespace, Name: mirrorName},
			})
			logger.Info("source deleted, mirror cleaned")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get source configmap: %w", err)
	}

	// 2) 期望状态：mirror 的 annotation 等于 source 的 annotation
	payload, ok := source.Annotations[syncAnnotation]
	if !ok {
		logger.V(1).Info("no payload annotation on source, skip")
		return ctrl.Result{}, nil
	}

	mirrorKey := types.NamespacedName{Namespace: req.Namespace, Name: req.Name + mirrorSuffix}

	// 3) 取 mirror（可能不存在）
	var mirror corev1.ConfigMap
	err := r.Get(ctx, mirrorKey, &mirror)
	switch {
	case apierrors.IsNotFound(err):
		// 不存在 → 创建
		mirror = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   mirrorKey.Namespace,
				Name:        mirrorKey.Name,
				Annotations: map[string]string{syncAnnotation: payload},
			},
			Data: map[string]string{"source": req.Name},
		}
		// SetControllerReference 写入 ownerRef，让 Owns(...) 的反向链路生效
		if err := ctrl.SetControllerReference(&source, &mirror, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("set owner ref: %w", err)
		}
		if err := r.Create(ctx, &mirror); err != nil {
			return ctrl.Result{}, fmt.Errorf("create mirror: %w", err)
		}
		logger.Info("mirror created", "payload", payload)
		return ctrl.Result{}, nil

	case err != nil:
		return ctrl.Result{}, fmt.Errorf("get mirror: %w", err)
	}

	// 4) 存在 → 比较并按需更新（幂等）
	current := mirror.Annotations[syncAnnotation]
	if current == payload {
		logger.V(1).Info("mirror up-to-date")
		return ctrl.Result{}, nil
	}
	if mirror.Annotations == nil {
		mirror.Annotations = map[string]string{}
	}
	mirror.Annotations[syncAnnotation] = payload
	if err := r.Update(ctx, &mirror); err != nil {
		return ctrl.Result{}, fmt.Errorf("update mirror: %w", err)
	}
	logger.Info("mirror updated", "payload", payload)
	return ctrl.Result{}, nil
}

// SetupWithManager 用 Builder 装配 Watch / Predicate / 并发度
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// 只关心带 source label 的 ConfigMap —— 用 Predicate 在入队前过滤掉无关对象，
	// 否则 cluster 里成千上万的 CM 都会进 worker
	sourcePred := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		if obj.GetNamespace() != r.WatchNamespace {
			return false
		}
		return obj.GetLabels()[sourceLabelKey] == sourceLabelValue
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("configmap-mirror").
		// For: 主资源 ConfigMap，使用 EnqueueRequestForObject
		For(&corev1.ConfigMap{}, builder.WithPredicates(sourcePred)).
		// Owns: 声明 mirror（也是 ConfigMap）由 source 拥有，
		// 子对象被外部改动时通过 ownerRef 反向触发 source 的 Reconcile
		Owns(&corev1.ConfigMap{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}

// 编译期断言：ConfigMapReconciler 满足 reconcile.Reconciler 接口
var _ reconcile.Reconciler = (*ConfigMapReconciler)(nil)
