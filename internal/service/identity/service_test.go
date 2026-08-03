package identity

import (
	"testing"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

func TestResolveTenantID_RootUsesDefaultTenant(t *testing.T) {
	st := store.NewMemory()
	svc := New(st)

	defaultTenantID := store.NewID()
	st.SaveUser(&platform.User{
		ID:       store.NewID(),
		Username: "root",
		Role:     platform.RoleRoot,
		TenantID: defaultTenantID,
	})

	claims := &auth.Claims{Role: platform.RoleRoot, TenantID: defaultTenantID}
	got, err := svc.ResolveTenantID(claims, "")
	if err != nil {
		t.Fatalf("ResolveTenantID: %v", err)
	}
	if got != defaultTenantID {
		t.Fatalf("got tenant %q, want %q", got, defaultTenantID)
	}
}

func TestResolveTenantID_RootImpersonatesOtherTenant(t *testing.T) {
	st := store.NewMemory()
	svc := New(st)

	defaultTenantID := store.NewID()
	otherTenantID := store.NewID()

	claims := &auth.Claims{Role: platform.RoleRoot, TenantID: defaultTenantID}
	got, err := svc.ResolveTenantID(claims, otherTenantID)
	if err != nil {
		t.Fatalf("ResolveTenantID: %v", err)
	}
	if got != otherTenantID {
		t.Fatalf("got tenant %q, want %q", got, otherTenantID)
	}
}

func TestLinkRootToTenant(t *testing.T) {
	st := store.NewMemory()
	svc := New(st)

	rootID := store.NewID()
	st.SaveUser(&platform.User{ID: rootID, Username: "root", Role: platform.RoleRoot})

	tenantID := store.NewID()
	svc.LinkRootToTenant(tenantID)

	root, ok := st.GetUserByUsername("root")
	if !ok {
		t.Fatal("root user not found")
	}
	if root.TenantID != tenantID {
		t.Fatalf("root tenant_id = %q, want %q", root.TenantID, tenantID)
	}

	svc.LinkRootToTenant(store.NewID())
	root, _ = st.GetUserByUsername("root")
	if root.TenantID != tenantID {
		t.Fatalf("LinkRootToTenant overwrote existing tenant_id")
	}
}
