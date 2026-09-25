package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/api/ws"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
	"github.com/virtfoundry/core/internal/service/identity"
)

type eventsFixture struct {
	hub     *ws.Hub
	srv     *httptest.Server
	store   store.Repository
	authSvc *auth.Service
	tenantA *platform.Tenant
	tenantB *platform.Tenant
}

// newEventsFixture wires /ws/events exactly as cmd/server does — behind
// middleware.AuthenticateWS — so the tests exercise the real auth path.
func newEventsFixture(t *testing.T) *eventsFixture {
	t.Helper()

	st := store.NewMemory()
	tenantA := &platform.Tenant{ID: store.NewID(), Name: "Tenant A", Slug: "tenant-a", Namespace: "virtfoundry-tenant-a", State: "active"}
	tenantB := &platform.Tenant{ID: store.NewID(), Name: "Tenant B", Slug: "tenant-b", Namespace: "virtfoundry-tenant-b", State: "active"}
	st.SaveTenant(tenantA)
	st.SaveTenant(tenantB)

	authSvc := auth.NewService("test-secret", 3600)
	identitySvc := identity.New(st)
	platformSvc := service.NewPlatformService(st, nil, nil, nil)
	hub := ws.NewHub()
	h := NewEventsHandler(hub, platformSvc, nil)

	router := http.NewServeMux()
	router.Handle("/ws/events", middleware.AuthenticateWS(authSvc, st, identitySvc)(http.HandlerFunc(h.Events)))
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &eventsFixture{hub: hub, srv: srv, store: st, authSvc: authSvc, tenantA: tenantA, tenantB: tenantB}
}

func (f *eventsFixture) token(t *testing.T, role platform.Role, tenantID string) string {
	t.Helper()

	u := &platform.User{
		ID:       store.NewID(),
		Username: "user-" + tenantID,
		Role:     role,
		TenantID: tenantID,
		State:    "active",
	}
	f.store.SaveUser(u)
	token, _, err := f.authSvc.IssueToken(u)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return token
}

func (f *eventsFixture) dial(t *testing.T, query string) (*websocket.Conn, *http.Response, error) {
	t.Helper()

	url := "ws" + strings.TrimPrefix(f.srv.URL, "http") + "/ws/events"
	if query != "" {
		url += "?" + query
	}
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if conn != nil {
		t.Cleanup(func() { conn.Close() })
	}
	return conn, resp, err
}

func readEvent(t *testing.T, conn *websocket.Conn) (ws.Event, error) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		// gorilla/websocket forbids another Read after any error (including
		// deadline). Callers must not retry ReadMessage on this conn.
		return ws.Event{}, err
	}
	var ev ws.Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	return ev, nil
}

// awaitSubscription blocks until the tenant's client is registered, so a later
// negative assertion cannot pass just because the broadcast raced the upgrade.
//
// Probes are broadcast from a goroutine while we do a single continuous read
// with one deadline. Looping ReadMessage after a timeout panics with
// "repeated read on failed websocket connection" (gorilla marks the conn dead).
func (f *eventsFixture) awaitSubscription(t *testing.T, conn *websocket.Conn, tenantID string) {
	t.Helper()

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		f.hub.BroadcastTenant(tenantID, "subscription.probe", nil)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				f.hub.BroadcastTenant(tenantID, "subscription.probe", nil)
			}
		}
	}()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("client for tenant %s never received its own events: %v", tenantID, err)
		}
		var ev ws.Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		if ev.Type == "subscription.probe" {
			return
		}
	}
}

func TestEventsRejectsUnauthenticatedUpgrade(t *testing.T) {
	f := newEventsFixture(t)

	conn, resp, err := f.dial(t, "")
	if err == nil {
		conn.Close()
		t.Fatal("unauthenticated WebSocket upgrade succeeded")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want 401", resp)
	}
}

func TestEventsRejectsInvalidToken(t *testing.T) {
	f := newEventsFixture(t)

	conn, resp, err := f.dial(t, "token=not-a-jwt")
	if err == nil {
		conn.Close()
		t.Fatal("upgrade with invalid token succeeded")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %v, want 401", resp)
	}
}

func TestEventsDoesNotDeliverOtherTenantEvents(t *testing.T) {
	f := newEventsFixture(t)
	token := f.token(t, platform.RoleTenantAdmin, f.tenantA.ID)

	conn, _, err := f.dial(t, "token="+token)
	if err != nil {
		t.Fatalf("dial as tenant A: %v", err)
	}
	f.awaitSubscription(t, conn, f.tenantA.ID)

	f.hub.BroadcastTenant(f.tenantB.ID, "vm.created", map[string]string{"name": "payments-prod"})

	if ev, err := readEvent(t, conn); err == nil {
		t.Fatalf("tenant A received tenant B event %+v", ev)
	}
}

func TestEventsDeliversSameTenantEvents(t *testing.T) {
	f := newEventsFixture(t)
	token := f.token(t, platform.RoleTenantAdmin, f.tenantA.ID)

	conn, _, err := f.dial(t, "token="+token)
	if err != nil {
		t.Fatalf("dial as tenant A: %v", err)
	}
	f.awaitSubscription(t, conn, f.tenantA.ID)

	f.hub.BroadcastTenant(f.tenantA.ID, "vm.created", map[string]string{"name": "web-01"})

	ev, err := readEvent(t, conn)
	if err != nil {
		t.Fatalf("tenant A did not receive its own event: %v", err)
	}
	if ev.Type != "vm.created" {
		t.Fatalf("event type = %q, want vm.created", ev.Type)
	}
}

func TestEventsResolveScopeIgnoresTenantQueryForNonRoot(t *testing.T) {
	f := newEventsFixture(t)
	h := NewEventsHandler(f.hub, service.NewPlatformService(f.store, nil, nil, nil), nil)

	req := httptest.NewRequest("GET", "/ws/events?tenant_id="+f.tenantB.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleTenantAdmin,
		TenantID: f.tenantA.ID,
	}))

	scope, err := h.resolveScope(req)
	if err != nil {
		t.Fatalf("resolveScope: %v", err)
	}
	if scope.AllTenants {
		t.Fatal("tenant admin was granted the all-tenants scope")
	}
	if scope.TenantID != f.tenantA.ID {
		t.Fatalf("scope tenant = %q, want %q", scope.TenantID, f.tenantA.ID)
	}
}

func TestEventsResolveScopeIgnoresAllTenantsForNonRoot(t *testing.T) {
	f := newEventsFixture(t)
	h := NewEventsHandler(f.hub, service.NewPlatformService(f.store, nil, nil, nil), nil)

	req := httptest.NewRequest("GET", "/ws/events?all_tenants=true", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleTenantAdmin,
		TenantID: f.tenantA.ID,
	}))

	scope, err := h.resolveScope(req)
	if err != nil {
		t.Fatalf("resolveScope: %v", err)
	}
	if scope.AllTenants {
		t.Fatal("tenant admin escalated to the all-tenants scope")
	}
	if scope.TenantID != f.tenantA.ID {
		t.Fatalf("scope tenant = %q, want %q", scope.TenantID, f.tenantA.ID)
	}
}

func TestEventsResolveScopeAllowsRootAllTenants(t *testing.T) {
	f := newEventsFixture(t)
	h := NewEventsHandler(f.hub, service.NewPlatformService(f.store, nil, nil, nil), nil)

	req := httptest.NewRequest("GET", "/ws/events?all_tenants=true", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.ContextClaims, &auth.Claims{
		Role:     platform.RoleRoot,
		TenantID: f.tenantA.ID,
	}))

	scope, err := h.resolveScope(req)
	if err != nil {
		t.Fatalf("resolveScope: %v", err)
	}
	if !scope.AllTenants {
		t.Fatal("root was not granted the all-tenants scope")
	}
}

func TestEventsResolveScopeRejectsMissingClaims(t *testing.T) {
	f := newEventsFixture(t)
	h := NewEventsHandler(f.hub, service.NewPlatformService(f.store, nil, nil, nil), nil)

	req := httptest.NewRequest("GET", "/ws/events", nil)

	if _, err := h.resolveScope(req); err != auth.ErrUnauthorized {
		t.Fatalf("resolveScope error = %v, want %v", err, auth.ErrUnauthorized)
	}
}
