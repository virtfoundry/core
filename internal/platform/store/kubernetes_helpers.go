package store

import (
	"context"

	"github.com/virtfoundry/core/internal/platform/store/mapping"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (k *Kubernetes) ctx() context.Context { return context.Background() }

func (k *Kubernetes) searchNamespaces() []string {
	seen := map[string]struct{}{mapping.SystemNamespace: {}}
	out := []string{mapping.SystemNamespace}
	for _, t := range k.ListTenants() {
		if t.Namespace == "" {
			continue
		}
		if _, ok := seen[t.Namespace]; ok {
			continue
		}
		seen[t.Namespace] = struct{}{}
		out = append(out, t.Namespace)
	}
	return out
}

func (k *Kubernetes) upsertCluster(gvr schema.GroupVersionResource, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	ctx := k.ctx()
	name := obj.GetName()
	existing, err := k.dyn.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return k.dyn.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	}
	if err != nil {
		return nil, err
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	return k.dyn.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
}

func (k *Kubernetes) upsertNamespaced(gvr schema.GroupVersionResource, ns string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	ctx := k.ctx()
	name := obj.GetName()
	existing, err := k.dyn.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return k.dyn.Resource(gvr).Namespace(ns).Create(ctx, obj, metav1.CreateOptions{})
	}
	if err != nil {
		return nil, err
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	return k.dyn.Resource(gvr).Namespace(ns).Update(ctx, obj, metav1.UpdateOptions{})
}

func (k *Kubernetes) findClusterByID(gvr schema.GroupVersionResource, id string) (*unstructured.Unstructured, bool) {
	list, err := k.dyn.Resource(gvr).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil, false
	}
	for i := range list.Items {
		if mapping.MatchesID(&list.Items[i], id) {
			return &list.Items[i], true
		}
	}
	return nil, false
}

func (k *Kubernetes) findNamespacedByID(gvr schema.GroupVersionResource, id string) (*unstructured.Unstructured, string, bool) {
	for _, ns := range k.searchNamespaces() {
		list, err := k.dyn.Resource(gvr).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
		if err != nil {
			continue
		}
		for i := range list.Items {
			if mapping.MatchesID(&list.Items[i], id) {
				return &list.Items[i], ns, true
			}
		}
	}
	return nil, "", false
}

func (k *Kubernetes) deleteCluster(gvr schema.GroupVersionResource, name string) {
	_ = k.dyn.Resource(gvr).Delete(k.ctx(), name, metav1.DeleteOptions{})
}

func (k *Kubernetes) deleteNamespaced(gvr schema.GroupVersionResource, ns, name string) {
	_ = k.dyn.Resource(gvr).Namespace(ns).Delete(k.ctx(), name, metav1.DeleteOptions{})
}

func (k *Kubernetes) listNamespacedAll(gvr schema.GroupVersionResource) []unstructured.Unstructured {
	var out []unstructured.Unstructured
	for _, ns := range k.searchNamespaces() {
		list, err := k.dyn.Resource(gvr).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
		if err != nil {
			continue
		}
		out = append(out, list.Items...)
	}
	return out
}

func (k *Kubernetes) tenantSlug(tenantID string) string {
	t, ok := k.GetTenant(tenantID)
	if !ok {
		return ""
	}
	return t.Slug
}

func (k *Kubernetes) tenantNamespace(tenantID string) (string, bool) {
	t, ok := k.GetTenant(tenantID)
	if !ok || t.Namespace == "" {
		return "", false
	}
	return t.Namespace, true
}

func (k *Kubernetes) saveClusterMapped(gvr schema.GroupVersionResource, build func() *unstructured.Unstructured, apply func(*unstructured.Unstructured)) {
	obj := build()
	if saved, err := k.upsertCluster(gvr, obj); err == nil && saved != nil {
		apply(saved)
	}
}

func (k *Kubernetes) saveNamespacedMapped(gvr schema.GroupVersionResource, ns string, build func() *unstructured.Unstructured, apply func(*unstructured.Unstructured)) {
	obj := build()
	if saved, err := k.upsertNamespaced(gvr, ns, obj); err == nil && saved != nil {
		apply(saved)
	}
}
