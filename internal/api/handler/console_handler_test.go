package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
)

func TestConsoleResolveVMAccessUsesTenantNamespace(t *testing.T) {
	st := store.NewMemory()
	tenant := &platform.Tenant{
		ID:        store.NewID(),
		Name:      "Tenant A",
		Slug:      "tenant-a",
		Namespace: "virtfoundry-tenant-a",
		State:     "active",
	}
	st.SaveTenant(tenant)
	st.SaveVM(&platform.PlatformVM{
		ID:        store.NewID(),
		TenantID:  tenant.ID,
		Name:      "vm-a",
		Namespace: tenant.Namespace,
		State:     "Running",
	})

	h := NewConsoleHandler(nil, st, service.NewPlatformService(st, nil, nil, nil))
	req := httptest.NewRequest("GET", "/ws/console?name=vm-a&namespace=other-tenant-ns", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleTenantAdmin,
		TenantID: tenant.ID,
	}))

	name, namespace, err := h.resolveVMAccess(req)
	if err != nil {
		t.Fatalf("resolveVMAccess: %v", err)
	}
	if name != "vm-a" {
		t.Fatalf("name = %q, want vm-a", name)
	}
	if namespace != tenant.Namespace {
		t.Fatalf("namespace = %q, want %q", namespace, tenant.Namespace)
	}
}

func TestConsoleResolveVMAccessRejectsCrossTenantVM(t *testing.T) {
	st := store.NewMemory()
	tenantA := &platform.Tenant{ID: store.NewID(), Name: "Tenant A", Slug: "tenant-a", Namespace: "virtfoundry-tenant-a", State: "active"}
	tenantB := &platform.Tenant{ID: store.NewID(), Name: "Tenant B", Slug: "tenant-b", Namespace: "virtfoundry-tenant-b", State: "active"}
	st.SaveTenant(tenantA)
	st.SaveTenant(tenantB)
	st.SaveVM(&platform.PlatformVM{
		ID:        store.NewID(),
		TenantID:  tenantB.ID,
		Name:      "vm-b",
		Namespace: tenantB.Namespace,
		State:     "Running",
	})

	h := NewConsoleHandler(nil, st, service.NewPlatformService(st, nil, nil, nil))
	req := httptest.NewRequest("GET", "/ws/console?name=vm-b", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleTenantAdmin,
		TenantID: tenantA.ID,
	}))

	if _, _, err := h.resolveVMAccess(req); err == nil || err.Error() != "vm not found" {
		t.Fatalf("resolveVMAccess error = %v, want vm not found", err)
	}
}

func TestConsoleResolveVMAccessAllowsRootTenantSelection(t *testing.T) {
	st := store.NewMemory()
	defaultTenant := &platform.Tenant{ID: store.NewID(), Name: "Default", Slug: "default", Namespace: "virtfoundry-default", State: "active"}
	targetTenant := &platform.Tenant{ID: store.NewID(), Name: "Target", Slug: "target", Namespace: "virtfoundry-target", State: "active"}
	st.SaveTenant(defaultTenant)
	st.SaveTenant(targetTenant)
	st.SaveVM(&platform.PlatformVM{
		ID:        store.NewID(),
		TenantID:  targetTenant.ID,
		Name:      "vm-target",
		Namespace: targetTenant.Namespace,
		State:     "Running",
	})

	h := NewConsoleHandler(nil, st, service.NewPlatformService(st, nil, nil, nil))
	req := httptest.NewRequest("GET", "/ws/console?name=vm-target&tenant_id="+targetTenant.ID, nil)
	ctx := context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleRoot,
		TenantID: defaultTenant.ID,
	})
	ctx = context.WithValue(ctx, middleware.ContextTenant, targetTenant.ID)
	req = req.WithContext(ctx)

	_, namespace, err := h.resolveVMAccess(req)
	if err != nil {
		t.Fatalf("resolveVMAccess: %v", err)
	}
	if namespace != targetTenant.Namespace {
		t.Fatalf("namespace = %q, want %q", namespace, targetTenant.Namespace)
	}
}

func TestConsoleResolveVMAccessIgnoresTenantQueryForNonRoot(t *testing.T) {
	st := store.NewMemory()
	tenantA := &platform.Tenant{ID: store.NewID(), Name: "Tenant A", Slug: "tenant-a", Namespace: "virtfoundry-tenant-a", State: "active"}
	tenantB := &platform.Tenant{ID: store.NewID(), Name: "Tenant B", Slug: "tenant-b", Namespace: "virtfoundry-tenant-b", State: "active"}
	st.SaveTenant(tenantA)
	st.SaveTenant(tenantB)
	st.SaveVM(&platform.PlatformVM{
		ID:        store.NewID(),
		TenantID:  tenantB.ID,
		Name:      "payments-prod",
		Namespace: tenantB.Namespace,
		State:     "Running",
	})

	h := NewConsoleHandler(nil, st, service.NewPlatformService(st, nil, nil, nil))
	req := httptest.NewRequest("GET", "/ws/console?name=payments-prod&tenant_id="+tenantB.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleTenantAdmin,
		TenantID: tenantA.ID,
	}))

	if _, _, err := h.resolveVMAccess(req); err == nil || err.Error() != "vm not found" {
		t.Fatalf("resolveVMAccess error = %v, want vm not found", err)
	}
}

func TestConsoleResolveVMAccessPrefersContextTenantOverRootQuery(t *testing.T) {
	st := store.NewMemory()
	defaultTenant := &platform.Tenant{ID: store.NewID(), Name: "Default", Slug: "default", Namespace: "virtfoundry-default", State: "active"}
	targetTenant := &platform.Tenant{ID: store.NewID(), Name: "Target", Slug: "target", Namespace: "virtfoundry-target", State: "active"}
	decoyTenant := &platform.Tenant{ID: store.NewID(), Name: "Decoy", Slug: "decoy", Namespace: "virtfoundry-decoy", State: "active"}
	st.SaveTenant(defaultTenant)
	st.SaveTenant(targetTenant)
	st.SaveTenant(decoyTenant)
	st.SaveVM(&platform.PlatformVM{
		ID:        store.NewID(),
		TenantID:  targetTenant.ID,
		Name:      "vm-target",
		Namespace: targetTenant.Namespace,
		State:     "Running",
	})

	h := NewConsoleHandler(nil, st, service.NewPlatformService(st, nil, nil, nil))
	req := httptest.NewRequest("GET", "/ws/console?name=vm-target&tenant_id="+decoyTenant.ID, nil)
	ctx := context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleRoot,
		TenantID: defaultTenant.ID,
	})
	ctx = context.WithValue(ctx, middleware.ContextTenant, targetTenant.ID)
	req = req.WithContext(ctx)

	_, namespace, err := h.resolveVMAccess(req)
	if err != nil {
		t.Fatalf("resolveVMAccess: %v", err)
	}
	if namespace != targetTenant.Namespace {
		t.Fatalf("namespace = %q, want %q", namespace, targetTenant.Namespace)
	}
}

func TestConsoleResolveVMAccessRequiresVMName(t *testing.T) {
	st := store.NewMemory()
	tenant := &platform.Tenant{ID: store.NewID(), Name: "Tenant A", Slug: "tenant-a", Namespace: "virtfoundry-tenant-a", State: "active"}
	st.SaveTenant(tenant)

	h := NewConsoleHandler(nil, st, service.NewPlatformService(st, nil, nil, nil))
	req := httptest.NewRequest("GET", "/ws/console", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleTenantAdmin,
		TenantID: tenant.ID,
	}))

	if _, _, err := h.resolveVMAccess(req); err == nil || err.Error() != "name required" {
		t.Fatalf("resolveVMAccess error = %v, want name required", err)
	}
}

func TestConsoleResolveVMAccessRejectsRootWithoutTenantSelection(t *testing.T) {
	st := store.NewMemory()
	h := NewConsoleHandler(nil, st, service.NewPlatformService(st, nil, nil, nil))
	req := httptest.NewRequest("GET", "/ws/console?name=vm-target", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role: platform.RoleRoot,
	}))

	if _, _, err := h.resolveVMAccess(req); err == nil || err.Error() != "tenant_id required for root" {
		t.Fatalf("resolveVMAccess error = %v, want tenant_id required for root", err)
	}
}
