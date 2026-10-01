package identity

import (
	"testing"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

type countingPermStore struct {
	store.Repository
	getRolePerms int
	getRole      int
}

func (c *countingPermStore) GetRolePermissions(roleID string) ([]string, bool) {
	c.getRolePerms++
	return c.Repository.GetRolePermissions(roleID)
}

func (c *countingPermStore) GetRole(id string) (*platform.RoleRecord, bool) {
	c.getRole++
	return c.Repository.GetRole(id)
}

func TestForUser_SystemRoleIDSkipsStore(t *testing.T) {
	mem := store.NewMemory()
	if err := store.SeedIAM(mem); err != nil {
		t.Fatal(err)
	}
	c := &countingPermStore{Repository: mem}
	r := NewPermissionResolver(c)

	u := &platform.User{
		Username: "root", Role: platform.RoleRoot, RoleID: store.SystemRoleIDRoot,
	}
	perms := r.ForUser(u)
	if len(perms) != 1 || perms[0] != auth.PermAll {
		t.Fatalf("perms = %#v, want [*]", perms)
	}
	if c.getRolePerms != 0 || c.getRole != 0 {
		t.Fatalf("system RoleID must skip store; GetRolePermissions=%d GetRole=%d", c.getRolePerms, c.getRole)
	}
}

func TestBuiltinPermissionsForRoleID(t *testing.T) {
	if got := BuiltinPermissionsForRoleID(store.SystemRoleIDTenantAdmin); len(got) == 0 {
		t.Fatal("expected tenant-admin builtins")
	}
	if got := BuiltinPermissionsForRoleID("custom-role-uuid"); got != nil {
		t.Fatalf("custom id should be nil, got %#v", got)
	}
}
