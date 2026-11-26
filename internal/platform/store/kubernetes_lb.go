package store

import (
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
)

// LoadBalancer and TargetGroup are CRs reconciled by the operator into a
// Service and EndpointSlices. Listeners live in LoadBalancer.spec.listeners and
// targets in TargetGroup.spec.targets, so every replica sees the same state.

func (k *Kubernetes) SaveLoadBalancer(lb *platform.LoadBalancer) {
	ns, ok := k.tenantNamespace(lb.TenantID)
	if !ok {
		return
	}
	slug := k.tenantSlug(lb.TenantID)
	k.saveNamespacedMapped(mapping.LoadBalancerGVR, ns, func() *unstructured.Unstructured {
		return mapping.LBToUnstructured(lb, slug)
	}, func(saved *unstructured.Unstructured) {
		*lb = *mapping.LBFromUnstructured(saved, lb.TenantID)
	})
}

func (k *Kubernetes) GetLoadBalancer(id string) (*platform.LoadBalancer, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.LoadBalancerGVR, id)
	if !ok {
		return nil, false
	}
	return mapping.LBFromUnstructured(obj, k.tenantIDForNamespace(ns)), true
}

func (k *Kubernetes) ListLoadBalancers(tenantID string) []*platform.LoadBalancer {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.LoadBalancerGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.LoadBalancer, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapping.LBFromUnstructured(&list.Items[i], tenantID))
	}
	return out
}

func (k *Kubernetes) DeleteLoadBalancer(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.LoadBalancerGVR, id); ok {
		k.deleteNamespaced(mapping.LoadBalancerGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) SaveTargetGroup(tg *platform.TargetGroup) {
	ns, ok := k.tenantNamespace(tg.TenantID)
	if !ok {
		return
	}
	slug := k.tenantSlug(tg.TenantID)
	k.saveNamespacedMapped(mapping.TargetGroupGVR, ns, func() *unstructured.Unstructured {
		return mapping.TGToUnstructured(tg, slug)
	}, func(saved *unstructured.Unstructured) {
		*tg = *mapping.TGFromUnstructured(saved, tg.TenantID)
	})
}

func (k *Kubernetes) GetTargetGroup(id string) (*platform.TargetGroup, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.TargetGroupGVR, id)
	if !ok {
		return nil, false
	}
	return mapping.TGFromUnstructured(obj, k.tenantIDForNamespace(ns)), true
}

func (k *Kubernetes) ListTargetGroups(tenantID string) []*platform.TargetGroup {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.TargetGroupGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.TargetGroup, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapping.TGFromUnstructured(&list.Items[i], tenantID))
	}
	return out
}

func (k *Kubernetes) DeleteTargetGroup(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.TargetGroupGVR, id); ok {
		k.deleteNamespaced(mapping.TargetGroupGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) SaveLBListener(l *platform.LBListener) {
	tgName, ok := k.crNameByID(mapping.TargetGroupGVR, l.TargetGroupID)
	if !ok {
		return
	}
	k.mutate(mapping.LoadBalancerGVR, l.LoadBalancerID, func(obj *unstructured.Unstructured) {
		entries := replaceEntry(mapping.LBListeners(obj), mapping.LBListenerEntry(l, tgName))
		mapping.SetLBListeners(obj, entries)
	})
}

func (k *Kubernetes) GetLBListener(id string) (*platform.LBListener, bool) {
	for _, obj := range k.listNamespacedAll(mapping.LoadBalancerGVR) {
		for _, l := range k.listenersOf(&obj) {
			if l.ID == id {
				return l, true
			}
		}
	}
	return nil, false
}

func (k *Kubernetes) ListLBListeners(loadBalancerID string) []*platform.LBListener {
	obj, _, ok := k.findNamespacedByID(mapping.LoadBalancerGVR, loadBalancerID)
	if !ok {
		return nil
	}
	return k.listenersOf(obj)
}

func (k *Kubernetes) DeleteLBListener(id string) {
	for _, obj := range k.listNamespacedAll(mapping.LoadBalancerGVR) {
		for _, l := range k.listenersOf(&obj) {
			if l.ID == id {
				k.mutate(mapping.LoadBalancerGVR, l.LoadBalancerID, func(o *unstructured.Unstructured) {
					mapping.SetLBListeners(o, removeEntry(mapping.LBListeners(o), id))
				})
				return
			}
		}
	}
}

func (k *Kubernetes) SaveLBTarget(t *platform.LBTarget) {
	k.mutate(mapping.TargetGroupGVR, t.TargetGroupID, func(obj *unstructured.Unstructured) {
		mapping.SetTGTargets(obj, replaceEntry(mapping.TGTargets(obj), mapping.TGTargetEntry(t)))
	})
}

func (k *Kubernetes) GetLBTarget(id string) (*platform.LBTarget, bool) {
	for _, obj := range k.listNamespacedAll(mapping.TargetGroupGVR) {
		for _, t := range k.targetsOf(&obj) {
			if t.ID == id {
				return t, true
			}
		}
	}
	return nil, false
}

func (k *Kubernetes) ListLBTargets(targetGroupID string) []*platform.LBTarget {
	obj, _, ok := k.findNamespacedByID(mapping.TargetGroupGVR, targetGroupID)
	if !ok {
		return nil
	}
	return k.targetsOf(obj)
}

func (k *Kubernetes) DeleteLBTarget(id string) {
	for _, obj := range k.listNamespacedAll(mapping.TargetGroupGVR) {
		for _, t := range k.targetsOf(&obj) {
			if t.ID == id {
				k.mutate(mapping.TargetGroupGVR, t.TargetGroupID, func(o *unstructured.Unstructured) {
					mapping.SetTGTargets(o, removeEntry(mapping.TGTargets(o), id))
				})
				return
			}
		}
	}
}

func (k *Kubernetes) DeleteLBTargetsByTargetGroup(targetGroupID string) {
	k.mutate(mapping.TargetGroupGVR, targetGroupID, func(obj *unstructured.Unstructured) {
		mapping.SetTGTargets(obj, nil)
	})
}

func (k *Kubernetes) listenersOf(obj *unstructured.Unstructured) []*platform.LBListener {
	tenantID := k.tenantIDForNamespace(obj.GetNamespace())
	lbID := mapping.ResourceID(obj)
	var out []*platform.LBListener
	for _, e := range mapping.LBListeners(obj) {
		tgID := ""
		if tg, ok := k.getByName(mapping.TargetGroupGVR, obj.GetNamespace(), mapping.TGRefName(e)); ok {
			tgID = mapping.ResourceID(tg)
		}
		l := mapping.LBListenerFromEntry(e, lbID, tenantID, tgID)
		l.CreatedAt = obj.GetCreationTimestamp().Time
		out = append(out, l)
	}
	return out
}

func (k *Kubernetes) targetsOf(obj *unstructured.Unstructured) []*platform.LBTarget {
	tg := mapping.TGFromUnstructured(obj, k.tenantIDForNamespace(obj.GetNamespace()))
	var out []*platform.LBTarget
	for _, e := range mapping.TGTargets(obj) {
		out = append(out, mapping.TGTargetFromEntry(e, tg))
	}
	return out
}

func (k *Kubernetes) getByName(gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, bool) {
	obj, err := k.dyn.Resource(gvr).Namespace(ns).Get(k.ctx(), name, metav1.GetOptions{})
	return obj, err == nil
}

func (k *Kubernetes) crNameByID(gvr schema.GroupVersionResource, id string) (string, bool) {
	obj, _, ok := k.findNamespacedByID(gvr, id)
	if !ok {
		return "", false
	}
	return obj.GetName(), true
}

// mutate re-reads the CR and applies fn on every conflict, so two API replicas
// editing listeners or targets of the same parent do not lose each other's write.
func (k *Kubernetes) mutate(gvr schema.GroupVersionResource, id string, fn func(*unstructured.Unstructured)) {
	_ = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		obj, ns, ok := k.findNamespacedByID(gvr, id)
		if !ok {
			return nil
		}
		fn(obj)
		_, err := k.dyn.Resource(gvr).Namespace(ns).Update(k.ctx(), obj, metav1.UpdateOptions{})
		return err
	})
}

func replaceEntry(list []map[string]interface{}, e map[string]interface{}) []map[string]interface{} {
	for i := range list {
		if list[i]["id"] == e["id"] {
			list[i] = e
			return list
		}
	}
	return append(list, e)
}

func removeEntry(list []map[string]interface{}, id string) []map[string]interface{} {
	out := list[:0]
	for _, e := range list {
		if e["id"] != id {
			out = append(out, e)
		}
	}
	return out
}
