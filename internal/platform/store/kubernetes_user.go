package store

import (
	"context"
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (k *Kubernetes) SaveUser(u *platform.User) {
	if u.RoleID == "" {
		u.RoleID = RoleIDForLegacy(u.Role)
	}
	if u.State == "" {
		u.State = "active"
	}

	ctx := context.Background()
	roleCRName := mapping.RoleCRNameForID(u.RoleID)
	if roleCRName == "" {
		if r, ok := k.GetRole(u.RoleID); ok {
			roleCRName = mapping.RoleCRName(r.Name)
		}
	}

	tenantCRName := ""
	if u.TenantID != "" {
		if t, ok := k.GetTenant(u.TenantID); ok {
			tenantCRName = mapping.TenantCRName(t.Slug)
		} else if t, ok := k.GetTenantBySlug(strings.TrimSuffix(u.Username, "-admin")); ok && t.ID == u.TenantID {
			tenantCRName = mapping.TenantCRName(t.Slug)
		}
	}

	crName := mapping.UserCRName(u.Username)
	obj := mapping.UserToUnstructured(u, roleCRName, tenantCRName)

	secret := mapping.UserSecret(crName, u.PasswordHash)
	_, secretErr := k.clientset.CoreV1().Secrets(k.systemNS()).Create(ctx, secret, metav1.CreateOptions{})
	if secretErr != nil && !apierrors.IsAlreadyExists(secretErr) {
		return
	}
	if apierrors.IsAlreadyExists(secretErr) {
		existing, getErr := k.clientset.CoreV1().Secrets(k.systemNS()).Get(ctx, secret.Name, metav1.GetOptions{})
		if getErr != nil {
			return
		}
		if existing.Data == nil {
			existing.Data = map[string][]byte{}
		}
		existing.Data[mapping.SecretKeyPasswordHash] = []byte(u.PasswordHash)
		_, _ = k.clientset.CoreV1().Secrets(k.systemNS()).Update(ctx, existing, metav1.UpdateOptions{})
	}

	existing, err := k.dyn.Resource(mapping.UserGVR).Get(ctx, crName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		created, createErr := k.dyn.Resource(mapping.UserGVR).Create(ctx, obj, metav1.CreateOptions{})
		if createErr == nil {
			*u = *k.userFromCR(ctx, created)
		}
		return
	}
	if err != nil {
		return
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	updated, updateErr := k.dyn.Resource(mapping.UserGVR).Update(ctx, obj, metav1.UpdateOptions{})
	if updateErr == nil {
		*u = *k.userFromCR(ctx, updated)
	}
}

func (k *Kubernetes) GetUserByUsername(username string) (*platform.User, bool) {
	obj, err := k.dyn.Resource(mapping.UserGVR).Get(context.Background(), mapping.UserCRName(username), metav1.GetOptions{})
	if err != nil {
		return nil, false
	}
	u := k.userFromCR(context.Background(), obj)
	return u, true
}

func (k *Kubernetes) HasRootUser() bool {
	list, err := k.dyn.Resource(mapping.UserGVR).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false
	}
	for i := range list.Items {
		roleRef, _, _ := unstructured.NestedString(list.Items[i].Object, "spec", "roleRef", "name")
		if roleRef == mapping.RoleCRName(platform.SystemRoleRoot) {
			return true
		}
	}
	return false
}

func (k *Kubernetes) GetUser(id string) (*platform.User, bool) {
	list, err := k.dyn.Resource(mapping.UserGVR).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, false
	}
	for i := range list.Items {
		if string(list.Items[i].GetUID()) == id {
			return k.userFromCR(context.Background(), &list.Items[i]), true
		}
	}
	return nil, false
}

func (k *Kubernetes) ListUsers() []*platform.User {
	list, err := k.dyn.Resource(mapping.UserGVR).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.User, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, k.userFromCR(context.Background(), &list.Items[i]))
	}
	return out
}

func (k *Kubernetes) ListUsersByTenant(tenantID string) []*platform.User {
	t, ok := k.GetTenant(tenantID)
	if !ok {
		return nil
	}
	tenantCR := mapping.TenantCRName(t.Slug)
	list, err := k.dyn.Resource(mapping.UserGVR).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	var out []*platform.User
	for i := range list.Items {
		ref, _, _ := unstructured.NestedString(list.Items[i].Object, "spec", "tenantRef", "name")
		username, _, _ := unstructured.NestedString(list.Items[i].Object, "spec", "username")
		if ref == tenantCR || username == tenantCR+"-admin" {
			out = append(out, k.userFromCR(context.Background(), &list.Items[i]))
		}
	}
	return out
}

func (k *Kubernetes) DeleteUser(id string) {
	u, ok := k.GetUser(id)
	if !ok {
		return
	}
	crName := mapping.UserCRName(u.Username)
	ctx := context.Background()
	_ = k.dyn.Resource(mapping.UserGVR).Delete(ctx, crName, metav1.DeleteOptions{})
	_ = k.clientset.CoreV1().Secrets(k.systemNS()).Delete(ctx, mapping.UserSecretName(crName), metav1.DeleteOptions{})
}

func (k *Kubernetes) userFromCR(ctx context.Context, obj *unstructured.Unstructured) *platform.User {
	crName := obj.GetName()
	secretRef, _, _ := unstructured.NestedString(obj.Object, "spec", "secretRef", "name")
	key, _, _ := unstructured.NestedString(obj.Object, "spec", "secretRef", "key")
	if key == "" {
		key = mapping.SecretKeyPasswordHash
	}
	if secretRef == "" {
		secretRef = mapping.UserSecretName(crName)
	}

	hash := ""
	if sec, err := k.clientset.CoreV1().Secrets(k.systemNS()).Get(ctx, secretRef, metav1.GetOptions{}); err == nil {
		hash = string(sec.Data[key])
	}

	u := mapping.UserFromUnstructured(obj, hash)

	tenantRef, _, _ := unstructured.NestedString(obj.Object, "spec", "tenantRef", "name")
	if tenantRef != "" {
		if tObj, err := k.dyn.Resource(mapping.TenantGVR).Get(ctx, tenantRef, metav1.GetOptions{}); err == nil {
			u.TenantID = string(tObj.GetUID())
		}
	}
	return u
}
