package store

import (
	"context"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (k *Kubernetes) SaveRole(r *platform.RoleRecord) {
	ctx := context.Background()
	ns := k.systemNS()
	if !r.IsSystem && r.TenantID != "" {
		if t, ok := k.GetTenant(r.TenantID); ok {
			ns = t.Namespace
		}
	}

	obj := mapping.RoleToUnstructured(r, ns)
	existing, err := k.dyn.Resource(mapping.RoleGVR).Namespace(ns).Get(ctx, obj.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		created, createErr := k.dyn.Resource(mapping.RoleGVR).Namespace(ns).Create(ctx, obj, metav1.CreateOptions{})
		if createErr == nil {
			*r = *mapping.RoleFromUnstructured(created)
		}
		return
	}
	if err != nil {
		return
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	perms, _, _ := unstructured.NestedStringSlice(existing.Object, "spec", "permissions")
	if len(r.Permissions) == 0 && len(perms) > 0 {
		r.Permissions = perms
	}
	updated, updateErr := k.dyn.Resource(mapping.RoleGVR).Namespace(ns).Update(ctx, obj, metav1.UpdateOptions{})
	if updateErr == nil {
		*r = *mapping.RoleFromUnstructured(updated)
	}
}

func (k *Kubernetes) GetRole(id string) (*platform.RoleRecord, bool) {
	if crName := mapping.RoleCRNameForID(id); crName != "" {
		obj, err := k.dyn.Resource(mapping.RoleGVR).Namespace(k.systemNS()).Get(context.Background(), crName, metav1.GetOptions{})
		if err == nil {
			return mapping.RoleFromUnstructured(obj), true
		}
	}

	list, err := k.dyn.Resource(mapping.RoleGVR).Namespace(k.systemNS()).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, false
	}
	for i := range list.Items {
		if list.Items[i].GetAnnotations()[mapping.AnnRoleID] == id || string(list.Items[i].GetUID()) == id {
			return mapping.RoleFromUnstructured(&list.Items[i]), true
		}
	}
	return nil, false
}

func (k *Kubernetes) GetRoleByName(tenantID, name string) (*platform.RoleRecord, bool) {
	ns := k.systemNS()
	if tenantID != "" {
		if t, ok := k.GetTenant(tenantID); ok {
			ns = t.Namespace
		}
	}
	crName := mapping.RoleCRName(name)
	obj, err := k.dyn.Resource(mapping.RoleGVR).Namespace(ns).Get(context.Background(), crName, metav1.GetOptions{})
	if err != nil {
		return nil, false
	}
	r := mapping.RoleFromUnstructured(obj)
	if r.Name != name {
		return nil, false
	}
	return r, true
}

func (k *Kubernetes) ListRoles(tenantID string) []*platform.RoleRecord {
	var out []*platform.RoleRecord

	systemList, err := k.dyn.Resource(mapping.RoleGVR).Namespace(k.systemNS()).List(context.Background(), metav1.ListOptions{})
	if err == nil {
		for i := range systemList.Items {
			out = append(out, mapping.RoleFromUnstructured(&systemList.Items[i]))
		}
	}

	if tenantID != "" {
		if t, ok := k.GetTenant(tenantID); ok && t.Namespace != "" {
			tenantList, listErr := k.dyn.Resource(mapping.RoleGVR).Namespace(t.Namespace).List(context.Background(), metav1.ListOptions{})
			if listErr == nil {
				for i := range tenantList.Items {
					r := mapping.RoleFromUnstructured(&tenantList.Items[i])
					if !r.IsSystem {
						out = append(out, r)
					}
				}
			}
		}
	}
	return out
}

func (k *Kubernetes) DeleteRole(id string) {
	r, ok := k.GetRole(id)
	if !ok {
		return
	}
	ns := k.systemNS()
	if !r.IsSystem && r.TenantID != "" {
		if t, ok := k.GetTenant(r.TenantID); ok {
			ns = t.Namespace
		}
	}
	_ = k.dyn.Resource(mapping.RoleGVR).Namespace(ns).Delete(context.Background(), mapping.RoleCRName(r.Name), metav1.DeleteOptions{})
}

func (k *Kubernetes) GetRolePermissions(roleID string) ([]string, bool) {
	r, ok := k.GetRole(roleID)
	if !ok {
		return nil, false
	}
	if len(r.Permissions) > 0 {
		return append([]string(nil), r.Permissions...), true
	}
	return nil, false
}

func (k *Kubernetes) SetRolePermissions(roleID string, perms []string) {
	r, ok := k.GetRole(roleID)
	if !ok {
		return
	}
	r.Permissions = append([]string(nil), perms...)
	k.SaveRole(r)
}
