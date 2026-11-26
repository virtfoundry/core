package tenant

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform/store"
)

func TestOperatorReconcileWritesNoClusterObjects(t *testing.T) {
	client := fake.NewClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "virtfoundry-tenant-acme"},
	})
	st := store.NewMemory()
	svc := New(st, &platformk8s.Manager{Clientset: client})
	svc.SetOperatorReconcile(true)

	tenant, user, err := svc.CreateTenant(context.Background(), "Acme", "acme", "a-long-password")
	if err != nil {
		t.Fatal(err)
	}
	if user.TenantID != tenant.ID {
		t.Fatalf("user tenant = %q, want %q", user.TenantID, tenant.ID)
	}
	if err := svc.DeleteTenant(context.Background(), tenant.ID); err != nil {
		t.Fatal(err)
	}
	for _, a := range client.Actions() {
		if a.GetVerb() != "get" {
			t.Fatalf("unexpected write: %s %s", a.GetVerb(), a.GetResource().Resource)
		}
	}
}

func TestOperatorReconcileFailsWithoutNamespace(t *testing.T) {
	svc := New(store.NewMemory(), &platformk8s.Manager{Clientset: fake.NewClientset()})
	svc.SetOperatorReconcile(true)
	svc.namespaceTimeout = 10 * time.Millisecond

	if _, err := svc.EnsureTenant(context.Background(), "Acme", "acme"); err == nil {
		t.Fatal("expected a timeout error")
	}
}
