package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
)

func newTestConsoleHandler(st store.Repository) *ConsoleHandler {
	return NewConsoleHandler(nil, st, service.NewPlatformService(st, nil, nil, nil),
		auth.NewConsoleTicketStore(auth.DefaultConsoleTicketTTL), nil)
}

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

	h := newTestConsoleHandler(st)
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

	h := newTestConsoleHandler(st)
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

	h := newTestConsoleHandler(st)
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

	h := newTestConsoleHandler(st)
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

	h := newTestConsoleHandler(st)
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

	h := newTestConsoleHandler(st)
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
	h := newTestConsoleHandler(st)
	req := httptest.NewRequest("GET", "/ws/console?name=vm-target", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role: platform.RoleRoot,
	}))

	if _, _, err := h.resolveVMAccess(req); err == nil || err.Error() != "tenant_id required for root" {
		t.Fatalf("resolveVMAccess error = %v, want tenant_id required for root", err)
	}
}

func tenantWithVM(st store.Repository, slug, vmName string) *platform.Tenant {
	tenant := &platform.Tenant{
		ID:        store.NewID(),
		Name:      slug,
		Slug:      slug,
		Namespace: "virtfoundry-" + slug,
		State:     "active",
	}
	st.SaveTenant(tenant)
	st.SaveVM(&platform.PlatformVM{
		ID:        store.NewID(),
		TenantID:  tenant.ID,
		Name:      vmName,
		Namespace: tenant.Namespace,
		State:     "Running",
	})
	return tenant
}

func withActor(r *http.Request, actor *auth.Actor) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ContextClaims, &auth.Claims{
		UserID: actor.UserID, Username: actor.Username,
		Role: actor.Role, TenantID: actor.TenantID,
	})
	ctx = context.WithValue(ctx, middleware.ContextActor, actor)
	return r.WithContext(ctx)
}

func viewerActor(tenantID string) *auth.Actor {
	return &auth.Actor{
		UserID: store.NewID(), Username: "viewer", Role: platform.RoleUser,
		TenantID: tenantID, Permissions: auth.TenantViewerPermissions,
	}
}

func operatorActor(tenantID string) *auth.Actor {
	return &auth.Actor{
		UserID: store.NewID(), Username: "operator", Role: platform.RoleUser,
		TenantID: tenantID, Permissions: auth.TenantOperatorPermissions,
	}
}

func TestConsoleResolveVMAccessDeniesViewer(t *testing.T) {
	st := store.NewMemory()
	tenant := tenantWithVM(st, "tenant-a", "vm-a")

	h := newTestConsoleHandler(st)
	req := withActor(httptest.NewRequest("GET", "/ws/console?name=vm-a", nil), viewerActor(tenant.ID))

	_, _, err := h.resolveVMAccess(req)
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("resolveVMAccess error = %v, want %v", err, auth.ErrForbidden)
	}
}

func TestConsoleResolveVMAccessAllowsOperator(t *testing.T) {
	st := store.NewMemory()
	tenant := tenantWithVM(st, "tenant-a", "vm-a")

	h := newTestConsoleHandler(st)
	req := withActor(httptest.NewRequest("GET", "/ws/console?name=vm-a", nil), operatorActor(tenant.ID))

	name, namespace, err := h.resolveVMAccess(req)
	if err != nil {
		t.Fatalf("resolveVMAccess: %v", err)
	}
	if name != "vm-a" || namespace != tenant.Namespace {
		t.Fatalf("got (%q, %q), want (vm-a, %q)", name, namespace, tenant.Namespace)
	}
}

func TestConsoleResolveVMAccessRequiresAuthentication(t *testing.T) {
	st := store.NewMemory()
	tenantWithVM(st, "tenant-a", "vm-a")

	h := newTestConsoleHandler(st)
	req := httptest.NewRequest("GET", "/ws/console?name=vm-a", nil)

	if _, _, err := h.resolveVMAccess(req); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("resolveVMAccess error = %v, want %v", err, auth.ErrUnauthorized)
	}
}

func TestConsoleResolveVMAccessTicketOverridesQueryVM(t *testing.T) {
	st := store.NewMemory()
	tenant := tenantWithVM(st, "tenant-a", "vm-a")
	st.SaveVM(&platform.PlatformVM{
		ID: store.NewID(), TenantID: tenant.ID, Name: "vm-b",
		Namespace: tenant.Namespace, State: "Running",
	})

	h := newTestConsoleHandler(st)
	req := httptest.NewRequest("GET", "/ws/console?name=vm-b", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextConsoleTicket,
		&auth.ConsoleTicket{TenantID: tenant.ID, VMName: "vm-a"}))

	name, _, err := h.resolveVMAccess(req)
	if err != nil {
		t.Fatalf("resolveVMAccess: %v", err)
	}
	if name != "vm-a" {
		t.Fatalf("name = %q, want vm-a (ticket must win over query)", name)
	}
}

func TestConsoleResolveVMAccessTicketCannotCrossTenant(t *testing.T) {
	st := store.NewMemory()
	tenantA := tenantWithVM(st, "tenant-a", "vm-a")
	tenantWithVM(st, "tenant-b", "vm-b")

	h := newTestConsoleHandler(st)
	req := httptest.NewRequest("GET", "/ws/console?name=vm-b", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextConsoleTicket,
		&auth.ConsoleTicket{TenantID: tenantA.ID, VMName: "vm-b"}))

	if _, _, err := h.resolveVMAccess(req); err == nil || err.Error() != "vm not found" {
		t.Fatalf("resolveVMAccess error = %v, want vm not found", err)
	}
}

func issueTicketRequest(t *testing.T, h *ConsoleHandler, actor *auth.Actor, vmName string) *httptest.ResponseRecorder {
	t.Helper()
	req := withActor(httptest.NewRequest("POST", "/api/v1/vms/"+vmName+"/console-ticket", nil), actor)
	req = mux.SetURLVars(req, map[string]string{"name": vmName})
	rec := httptest.NewRecorder()
	h.IssueConsoleTicket(rec, req)
	return rec
}

func TestIssueConsoleTicketDeniesViewer(t *testing.T) {
	st := store.NewMemory()
	tenant := tenantWithVM(st, "tenant-a", "vm-a")

	rec := issueTicketRequest(t, newTestConsoleHandler(st), viewerActor(tenant.ID), "vm-a")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestIssueConsoleTicketDeniesCrossTenantVM(t *testing.T) {
	st := store.NewMemory()
	tenantA := tenantWithVM(st, "tenant-a", "vm-a")
	tenantWithVM(st, "tenant-b", "vm-b")

	rec := issueTicketRequest(t, newTestConsoleHandler(st), operatorActor(tenantA.ID), "vm-b")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestIssueConsoleTicketIsRedeemableOnceForItsVM(t *testing.T) {
	st := store.NewMemory()
	tenant := tenantWithVM(st, "tenant-a", "vm-a")
	h := newTestConsoleHandler(st)

	rec := issueTicketRequest(t, h, operatorActor(tenant.ID), "vm-a")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Ticket    string `json:"ticket"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Ticket == "" || body.ExpiresAt == "" {
		t.Fatalf("incomplete ticket response: %s", rec.Body.String())
	}

	ticket, err := h.tickets.Redeem(body.Ticket)
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	if ticket.VMName != "vm-a" || ticket.TenantID != tenant.ID {
		t.Fatalf("ticket = %+v, want vm-a in tenant %s", ticket, tenant.ID)
	}
	if _, err := h.tickets.Redeem(body.Ticket); !errors.Is(err, auth.ErrConsoleTicketInvalid) {
		t.Fatalf("second Redeem error = %v, want %v", err, auth.ErrConsoleTicketInvalid)
	}
}
