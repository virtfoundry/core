package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/identity"
)

type consoleFixture struct {
	chain   http.Handler
	store   store.Repository
	authSvc *auth.Service
	tickets *auth.ConsoleTicketStore
	reached *bool
	actor   **auth.Actor
}

// newConsoleFixture mounts /ws/console exactly as cmd/server does: ticket auth
// with header auth as fallback, then RequirePermission(vms:console).
func newConsoleFixture(t *testing.T) *consoleFixture {
	t.Helper()

	st := store.NewMemory()
	authSvc := auth.NewService("test-secret", 3600)
	identitySvc := identity.New(st)
	tickets := auth.NewConsoleTicketStore(auth.DefaultConsoleTicketTTL)

	reached := false
	var seen *auth.Actor
	terminal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		seen = GetActor(r.Context())
		w.WriteHeader(http.StatusSwitchingProtocols)
	})

	chain := ConsoleTicketAuth(tickets, Authenticate(authSvc, st, identitySvc))(
		RequirePermission(auth.PermVMsConsole)(terminal))

	return &consoleFixture{
		chain: chain, store: st, authSvc: authSvc, tickets: tickets,
		reached: &reached, actor: &seen,
	}
}

func (f *consoleFixture) tokenFor(t *testing.T, role platform.Role, tenantID string) string {
	t.Helper()

	u := &platform.User{
		ID: store.NewID(), Username: "user-" + string(role) + "-" + tenantID,
		Role: role, TenantID: tenantID, State: "active",
	}
	f.store.SaveUser(u)
	token, _, err := f.authSvc.IssueToken(u)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return token
}

func (f *consoleFixture) do(target string, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", target, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	f.chain.ServeHTTP(rec, req)
	return rec
}

func TestConsoleRejectsJWTInQueryString(t *testing.T) {
	f := newConsoleFixture(t)
	token := f.tokenFor(t, platform.RoleTenantAdmin, store.NewID())

	rec := f.do("/ws/console?name=vm-a&token="+token, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (a JWT in the URL must not authenticate)", rec.Code)
	}
	if *f.reached {
		t.Fatal("handler reached with a query-string JWT")
	}
}

func TestConsoleRejectsViewerWithHeaderToken(t *testing.T) {
	f := newConsoleFixture(t)
	token := f.tokenFor(t, platform.RoleUser, store.NewID())

	rec := f.do("/ws/console?name=vm-a", http.Header{"Authorization": {"Bearer " + token}})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for a viewer without vms:console", rec.Code)
	}
	if *f.reached {
		t.Fatal("viewer reached the console handler")
	}
}

func TestConsoleAllowsTenantAdminWithHeaderToken(t *testing.T) {
	f := newConsoleFixture(t)
	token := f.tokenFor(t, platform.RoleTenantAdmin, store.NewID())

	rec := f.do("/ws/console?name=vm-a", http.Header{"Authorization": {"Bearer " + token}})
	if rec.Code != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101 for a tenant admin", rec.Code)
	}
}

func TestConsoleTicketAuthorizesAndIsSingleUse(t *testing.T) {
	f := newConsoleFixture(t)
	tenantID := store.NewID()
	ticket, _, err := f.tickets.Issue(auth.ConsoleTicket{
		UserID: "u-1", Username: "operator", Role: platform.RoleUser,
		TenantID: tenantID, VMName: "vm-a",
	})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	rec := f.do("/ws/console?ticket="+ticket, nil)
	if rec.Code != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101 for a valid ticket", rec.Code)
	}
	if got := *f.actor; got == nil || len(got.Permissions) != 1 || got.Permissions[0] != auth.PermVMsConsole {
		t.Fatalf("ticket actor permissions = %+v, want only %s", got, auth.PermVMsConsole)
	}

	*f.reached = false
	if rec := f.do("/ws/console?ticket="+ticket, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed ticket status = %d, want 401", rec.Code)
	}
	if *f.reached {
		t.Fatal("replayed ticket reached the console handler")
	}
}

func TestConsoleRejectsUnknownTicket(t *testing.T) {
	f := newConsoleFixture(t)

	if rec := f.do("/ws/console?ticket=not-a-ticket", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if *f.reached {
		t.Fatal("handler reached with an unknown ticket")
	}
}

func TestConsoleRejectsMissingCredentials(t *testing.T) {
	f := newConsoleFixture(t)

	if rec := f.do("/ws/console?name=vm-a", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAuthenticateWSStillAcceptsQueryToken(t *testing.T) {
	st := store.NewMemory()
	authSvc := auth.NewService("test-secret", 3600)
	u := &platform.User{ID: store.NewID(), Username: "eventuser", Role: platform.RoleTenantAdmin, TenantID: store.NewID(), State: "active"}
	st.SaveUser(u)
	token, _, err := authSvc.IssueToken(u)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	reached := false
	h := AuthenticateWS(authSvc, st, identity.New(st))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/ws/events?token="+token, nil))
	if !reached {
		t.Fatalf("browser WebSocket auth broke for /ws/events: status %d", rec.Code)
	}
}
