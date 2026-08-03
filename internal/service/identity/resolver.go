package identity

import (
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

// PermissionResolver loads effective permissions for users and roles.
type PermissionResolver struct {
	store store.Repository
}

func NewPermissionResolver(st store.Repository) *PermissionResolver {
	return &PermissionResolver{store: st}
}

// ForUser resolves permissions from role_id or legacy role enum.
func (r *PermissionResolver) ForUser(u *platform.User) []string {
	if u == nil {
		return nil
	}
	if u.RoleID != "" {
		if perms, ok := r.store.GetRolePermissions(u.RoleID); ok && len(perms) > 0 {
			return perms
		}
		if role, ok := r.store.GetRole(u.RoleID); ok && role.IsSystem {
			return BuiltinPermissions(role.Name)
		}
	}
	return auth.LegacyRolePermissions(u.Role)
}

// ForAPIKey intersects key scopes with user permissions.
func (r *PermissionResolver) ForAPIKey(u *platform.User, key *platform.APIKey) []string {
	userPerms := r.ForUser(u)
	if key == nil || len(key.Scopes) == 0 {
		return userPerms
	}
	return auth.FilterScopes(key.Scopes, userPerms)
}

// BuiltinPermissions returns permissions for system role names.
func BuiltinPermissions(roleName string) []string {
	switch roleName {
	case platform.SystemRoleRoot:
		return []string{auth.PermAll}
	case platform.SystemRoleTenantAdmin:
		return append([]string{}, auth.TenantAdminPermissions...)
	case platform.SystemRoleTenantOperator:
		return append([]string{}, auth.TenantOperatorPermissions...)
	case platform.SystemRoleTenantViewer:
		return append([]string{}, auth.TenantViewerPermissions...)
	default:
		return nil
	}
}
