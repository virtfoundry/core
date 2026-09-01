package store

import (
	"context"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (k *Kubernetes) SaveTenant(t *platform.Tenant) {
	ctx := context.Background()
	name := mapping.TenantCRName(t.Slug)
	obj := mapping.TenantToUnstructured(t)

	existing, err := k.dyn.Resource(mapping.TenantGVR).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		created, createErr := k.dyn.Resource(mapping.TenantGVR).Create(ctx, obj, metav1.CreateOptions{})
		if createErr == nil {
			*t = *mapping.TenantFromUnstructured(created)
		}
		k.invalidateTenantCache()
		return
	}
	if err != nil {
		return
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	updated, updateErr := k.dyn.Resource(mapping.TenantGVR).Update(ctx, obj, metav1.UpdateOptions{})
	if updateErr == nil {
		*t = *mapping.TenantFromUnstructured(updated)
	}
	k.invalidateTenantCache()
}

func (k *Kubernetes) GetTenant(id string) (*platform.Tenant, bool) {
	t, ok := k.tenantSnapshot().byID[id]
	return t, ok
}

func (k *Kubernetes) GetTenantBySlug(slug string) (*platform.Tenant, bool) {
	ctx := context.Background()
	obj, err := k.dyn.Resource(mapping.TenantGVR).Get(ctx, mapping.TenantCRName(slug), metav1.GetOptions{})
	if err == nil {
		return mapping.TenantFromUnstructured(obj), true
	}
	if !apierrors.IsNotFound(err) {
		return nil, false
	}

	list, listErr := k.dyn.Resource(mapping.TenantGVR).List(ctx, metav1.ListOptions{
		LabelSelector: mapping.LabelSlug + "=" + slug,
	})
	if listErr != nil {
		return nil, false
	}
	for i := range list.Items {
		specSlug, _, _ := unstructured.NestedString(list.Items[i].Object, "spec", "slug")
		if specSlug == slug {
			return mapping.TenantFromUnstructured(&list.Items[i]), true
		}
	}
	return nil, false
}

func (k *Kubernetes) ListTenants() []*platform.Tenant {
	snap := k.tenantSnapshot()
	out := make([]*platform.Tenant, len(snap.tenants))
	copy(out, snap.tenants)
	return out
}

func (k *Kubernetes) DeleteTenant(id string) {
	t, ok := k.GetTenant(id)
	if !ok {
		return
	}
	_ = k.dyn.Resource(mapping.TenantGVR).Delete(context.Background(), mapping.TenantCRName(t.Slug), metav1.DeleteOptions{})
	k.invalidateTenantCache()
}
