package store

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
)

type countingRepo struct {
	Repository
	authByName int
	byID       int
}

func (c *countingRepo) GetUserForAuth(username string) (*platform.User, bool) {
	c.authByName++
	return c.Repository.GetUserForAuth(username)
}

func (c *countingRepo) GetUser(id string) (*platform.User, bool) {
	c.byID++
	return c.Repository.GetUser(id)
}

func TestResolveJWTUser_PrefersUsername(t *testing.T) {
	mem := NewMemory()
	u := &platform.User{
		ID: NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: SystemRoleIDTenantAdmin, State: "active",
	}
	mem.SaveUser(u)
	c := &countingRepo{Repository: mem}

	got, ok := ResolveJWTUser(c, u.ID, u.Username)
	if !ok || got.ID != u.ID {
		t.Fatalf("resolve: ok=%v got=%#v", ok, got)
	}
	if c.authByName != 1 || c.byID != 0 {
		t.Fatalf("want username path only, authByName=%d byID=%d", c.authByName, c.byID)
	}
}

func TestResolveJWTUser_UIDMismatchFallsBackToGetUser(t *testing.T) {
	mem := NewMemory()
	alice := &platform.User{
		ID: NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: SystemRoleIDTenantAdmin, State: "active",
	}
	bob := &platform.User{
		ID: NewID(), Username: "bob", Role: platform.RoleUser,
		RoleID: SystemRoleIDTenantAdmin, State: "active",
	}
	mem.SaveUser(alice)
	mem.SaveUser(bob)
	c := &countingRepo{Repository: mem}

	// JWT claims username=alice but UserID=bob → reject username hit, fall back by ID.
	got, ok := ResolveJWTUser(c, bob.ID, alice.Username)
	if !ok || got.ID != bob.ID {
		t.Fatalf("fallback: ok=%v got=%#v want bob", ok, got)
	}
	if c.authByName != 1 || c.byID != 1 {
		t.Fatalf("want username then GetUser, authByName=%d byID=%d", c.authByName, c.byID)
	}
}

func TestResolveJWTUser_EmptyUsernameUsesGetUser(t *testing.T) {
	mem := NewMemory()
	u := &platform.User{
		ID: NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: SystemRoleIDTenantAdmin, State: "active",
	}
	mem.SaveUser(u)
	c := &countingRepo{Repository: mem}

	got, ok := ResolveJWTUser(c, u.ID, "")
	if !ok || got.ID != u.ID {
		t.Fatalf("resolve: ok=%v got=%#v", ok, got)
	}
	if c.authByName != 0 || c.byID != 1 {
		t.Fatalf("want GetUser only, authByName=%d byID=%d", c.authByName, c.byID)
	}
}
