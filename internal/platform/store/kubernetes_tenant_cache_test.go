package store

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

// Two API replicas share the cluster but not the 30s tenant cache.
func TestTenantCreatedByAnotherReplicaIsVisible(t *testing.T) {
	dyn := newTestDynamicClient()
	assignUIDsOnCreate(dyn)
	cs := kubefake.NewSimpleClientset()
	a := &Kubernetes{dyn: dyn, clientset: cs}
	b := &Kubernetes{dyn: dyn, clientset: cs}
	b.ListTenants() // b caches the empty list

	tenant := &platform.Tenant{Name: "Acme", Slug: "acme", State: "active", CreatedAt: Now()}
	a.SaveTenant(tenant)
	a.SaveUser(&platform.User{Username: "root", TenantID: tenant.ID, State: "active"})

	u, ok := b.GetUserByUsername("root")
	if !ok || u.TenantID != tenant.ID {
		t.Fatalf("user hydrated without tenant: ok=%v user=%+v", ok, u)
	}

	b.invalidateTenantCache()
	b.tenantCache = &cachedTenants{snap: &tenantSnapshot{}, expires: Now().Add(tenantCacheTTL)} // stale again
	b.SaveUser(u)
	if got, _ := a.GetUserByUsername("root"); got.TenantID != tenant.ID {
		t.Fatalf("save from a stale replica dropped tenantRef: %+v", got)
	}
	if _, ok := b.GetTenant(tenant.ID); !ok {
		t.Fatal("GetTenant missed a tenant created by another replica")
	}

	a.SaveUser(&platform.User{Username: "root", TenantID: "unknown-id", State: "active"})
	if got, _ := a.GetUserByUsername("root"); got.TenantID != tenant.ID {
		t.Fatalf("unresolved tenant ID erased tenantRef: %+v", got)
	}

	dyn.ClearActions()
	a.GetTenant("")
	if n := countResourceVerbs(dyn.Actions(), mapping.TenantGVR.Resource, "list"); n != 0 {
		t.Fatalf("empty tenant ID listed tenants %d times", n)
	}
}
