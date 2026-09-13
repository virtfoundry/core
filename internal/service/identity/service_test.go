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

func TestTenantAdminCannotAssignPlatformRoot(t *testing.T) {
	st := store.NewMemory()
	if err := store.SeedIAM(st); err != nil {
		t.Fatal(err)
	}
	svc := New(st)
	tenantID := store.NewID()
	admin := &auth.Actor{Role: platform.RoleTenantAdmin, Permissions: auth.TenantAdminPermissions}

	if _, err := svc.CreateUser(tenantID, CreateUserInput{
		Username: "escalate-create", Password: "password", RoleID: store.SystemRoleIDRoot,
	}, admin); err == nil {
		t.Fatal("tenant admin created a platform root user")
	}

	target := &platform.User{ID: store.NewID(), Username: "target", TenantID: tenantID, Role: platform.RoleUser}
	st.SaveUser(target)
	if _, err := svc.UpdateUser(tenantID, target.ID, "", store.SystemRoleIDRoot, "", admin); err == nil {
		t.Fatal("tenant admin updated a user to platform root")
	}
}

func TestTenantRolePermissionsAreBoundedByActor(t *testing.T) {
	st := store.NewMemory()
	svc := New(st)
	tenantID := store.NewID()
	admin := &auth.Actor{Role: platform.RoleTenantAdmin, Permissions: auth.TenantAdminPermissions}

	for _, permissions := range [][]string{{auth.PermAll}, {auth.PermTenantsWrite}} {
		if _, err := svc.CreateRole(tenantID, CreateRoleInput{Name: store.NewID(), Permissions: permissions}, admin); err == nil {
			t.Fatalf("created tenant role with forbidden permissions %v", permissions)
		}
	}
	if _, err := svc.CreateRole(tenantID, CreateRoleInput{Name: "out-of-scope", Permissions: []string{auth.PermTenantsRead}}, admin); err == nil {
		t.Fatal("tenant admin granted a permission it does not hold")
	}

	role, err := svc.CreateRole(tenantID, CreateRoleInput{Name: "vm-reader", Permissions: []string{auth.PermVMsRead}}, admin)
	if err != nil {
		t.Fatalf("create legitimate tenant role: %v", err)
	}
	if _, err := svc.UpdateRole(tenantID, role.ID, "", []string{auth.PermTenantsWrite}, admin); err == nil {
		t.Fatal("updated tenant role with tenants:write")
	}
}

func TestTenantAdminCanAssignTenantRoleAndRootCanManageSystemRole(t *testing.T) {
	st := store.NewMemory()
	if err := store.SeedIAM(st); err != nil {
		t.Fatal(err)
	}
	svc := New(st)
	tenantID := store.NewID()
	admin := &auth.Actor{Role: platform.RoleTenantAdmin, Permissions: auth.TenantAdminPermissions}
	role, err := svc.CreateRole(tenantID, CreateRoleInput{Name: "vm-reader", Permissions: []string{auth.PermVMsRead}}, admin)
	if err != nil {
		t.Fatalf("create tenant role: %v", err)
	}
	user, err := svc.CreateUser(tenantID, CreateUserInput{Username: "reader", Password: "password", RoleID: role.ID}, admin)
	if err != nil {
		t.Fatalf("assign tenant role: %v", err)
	}
	if user.RoleID != role.ID {
		t.Fatalf("role_id = %q, want %q", user.RoleID, role.ID)
	}

	root := &auth.Actor{Role: platform.RoleRoot, Permissions: []string{auth.PermAll}}
	rootUser, err := svc.CreateUser(tenantID, CreateUserInput{
		Username: "platform-root", Password: "password", RoleID: store.SystemRoleIDRoot,
	}, root)
	if err != nil {
		t.Fatalf("root assigned platform root role: %v", err)
	}
	if rootUser.Role != platform.RoleRoot {
		t.Fatalf("role = %q, want %q", rootUser.Role, platform.RoleRoot)
	}
	if _, err := svc.UpdateRole(tenantID, store.SystemRoleIDTenantViewer, "root managed", []string{auth.PermVMsRead}, root); err != nil {
		t.Fatalf("root updated system role: %v", err)
	}
}
