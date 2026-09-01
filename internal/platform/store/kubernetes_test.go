package store

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesStore_TenantRoundTrip(t *testing.T) {
	dyn := newTestDynamicClient()
	cs := kubefake.NewSimpleClientset()
	repo := &Kubernetes{dyn: dyn, clientset: cs}

	tenant := &platform.Tenant{
		Name: "Acme", Slug: "acme", Namespace: "virtfoundry-tenant-acme", State: "active", CreatedAt: Now(),
	}
	repo.SaveTenant(tenant)

	got, ok := repo.GetTenantBySlug("acme")
	if !ok || got.Slug != "acme" || got.Name != "Acme" {
		t.Fatalf("expected tenant by slug acme, got %#v ok=%v", got, ok)
	}
	if tenant.Slug != "acme" || tenant.Name != "Acme" {
		t.Fatalf("SaveTenant should refresh tenant fields, got %#v", tenant)
	}

	if got.ID != "" {
		byID, ok := repo.GetTenant(got.ID)
		if !ok || byID.Name != "Acme" {
			t.Fatalf("expected tenant by id %q, got %#v ok=%v", got.ID, byID, ok)
		}
	}

	list := repo.ListTenants()
	if len(list) != 1 {
		t.Fatalf("expected 1 tenant, got %d", len(list))
	}
}

func TestKubernetesStore_UserSecretRoundTrip(t *testing.T) {
	dyn := newTestDynamicClient()
	cs := kubefake.NewSimpleClientset()
	repo := &Kubernetes{dyn: dyn, clientset: cs}
	_ = repo.SeedIAM()

	user := &platform.User{
		Username:     "root",
		PasswordHash: "$2a$10$hash",
		Role:         platform.RoleRoot,
		RoleID:       SystemRoleIDRoot,
		State:        "active",
		CreatedAt:    Now(),
	}
	repo.SaveUser(user)

	got, ok := repo.GetUserByUsername("root")
	if !ok {
		t.Fatal("expected root user")
	}
	if got.PasswordHash != user.PasswordHash {
		t.Fatalf("password hash: got %q want %q", got.PasswordHash, user.PasswordHash)
	}
	if got.RoleID != SystemRoleIDRoot {
		t.Fatalf("role id: got %q want %q", got.RoleID, SystemRoleIDRoot)
	}
	if !repo.HasRootUser() {
		t.Fatal("expected HasRootUser true")
	}
}

func newTestDynamicClient() *fake.FakeDynamicClient {
	listKinds := map[schema.GroupVersionResource]string{
		mapping.TenantGVR: "TenantList",
		mapping.UserGVR:   "UserList",
		mapping.RoleGVR:   "RoleList",
	}
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return fake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds)
}

func TestTenantMapping(t *testing.T) {
	obj := mapping.TenantToUnstructured(&platform.Tenant{Name: "Acme", Slug: "acme"})
	if obj.GetName() != "acme" {
		t.Fatalf("cr name: got %q", obj.GetName())
	}
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	if name != "Acme" {
		t.Fatalf("spec name: got %q", name)
	}

	obj.SetUID("uid-123")
	obj.SetCreationTimestamp(metav1.Now())
	_ = unstructured.SetNestedField(obj.Object, "virtfoundry-tenant-acme", "status", "namespace")
	got := mapping.TenantFromUnstructured(obj)
	if got.ID != "uid-123" || got.Namespace != "virtfoundry-tenant-acme" {
		t.Fatalf("unexpected mapped tenant: %#v", got)
	}
}
