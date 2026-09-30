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

type authCountingStore struct {
	store.Repository
	authByName int
	byID       int
}

func (c *authCountingStore) GetUserForAuth(username string) (*platform.User, bool) {
	c.authByName++
	return c.Repository.GetUserForAuth(username)
}

func (c *authCountingStore) GetUser(id string) (*platform.User, bool) {
	c.byID++
	return c.Repository.GetUser(id)
}

func TestAuthenticate_ResolvesByUsername(t *testing.T) {
	mem := store.NewMemory()
	_ = mem.SeedIAM()
	st := &authCountingStore{Repository: mem}
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	ident := identity.New(st)

	u := &platform.User{
		ID: store.NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: store.SystemRoleIDTenantAdmin, TenantID: store.NewID(), State: "active",
	}
	st.SaveUser(u)
	token, _, err := authSvc.IssueToken(u)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	var seen *auth.Actor
	h := Authenticate(authSvc, st, ident)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetActor(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vm-templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if seen == nil || seen.UserID != u.ID || seen.Username != u.Username {
		t.Fatalf("actor=%#v", seen)
	}
	if st.authByName != 1 || st.byID != 0 {
		t.Fatalf("want username resolve only, authByName=%d byID=%d", st.authByName, st.byID)
	}
}

func TestAuthenticate_UIDMismatchFallsBackSafely(t *testing.T) {
	mem := store.NewMemory()
	_ = mem.SeedIAM()
	st := &authCountingStore{Repository: mem}
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	ident := identity.New(st)

	alice := &platform.User{
		ID: store.NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: store.SystemRoleIDTenantAdmin, TenantID: store.NewID(), State: "active",
	}
	bob := &platform.User{
		ID: store.NewID(), Username: "bob", Role: platform.RoleUser,
		RoleID: store.SystemRoleIDTenantAdmin, TenantID: store.NewID(), State: "active",
	}
	st.SaveUser(alice)
	st.SaveUser(bob)

	// Craft claims: victim username + caller's UID (would escalate without UID check).
	spoof := &platform.User{ID: bob.ID, Username: alice.Username, Role: bob.Role, TenantID: bob.TenantID}
	token, _, err := authSvc.IssueToken(spoof)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	var seen *auth.Actor
	h := Authenticate(authSvc, st, ident)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = GetActor(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/snapshots", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if seen == nil || seen.UserID != bob.ID || seen.Username != bob.Username {
		t.Fatalf("want bob actor after fallback, got %#v", seen)
	}
	if st.authByName != 1 || st.byID != 1 {
		t.Fatalf("want username then GetUser, authByName=%d byID=%d", st.authByName, st.byID)
	}
}
