package store

import (
	"time"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
)

// Fixed IDs for system roles (stable across deployments).
const (
	SystemRoleIDRoot          = "00000000-0000-4000-8000-000000000001"
	SystemRoleIDTenantAdmin   = "00000000-0000-4000-8000-000000000002"
	SystemRoleIDTenantOperator = "00000000-0000-4000-8000-000000000003"
	SystemRoleIDTenantViewer  = "00000000-0000-4000-8000-000000000004"
)

type iamSeeder interface {
	SaveRole(r *platform.RoleRecord)
	SetRolePermissions(roleID string, perms []string)
	GetRoleByName(tenantID, name string) (*platform.RoleRecord, bool)
}

// SeedIAM creates system roles if missing.
func SeedIAM(r Repository) error {
	s, ok := r.(iamSeeder)
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	systemRoles := []struct {
		id, name, desc string
		perms          []string
	}{
		{SystemRoleIDRoot, platform.SystemRoleRoot, "Platform operator", []string{auth.PermAll}},
		{SystemRoleIDTenantAdmin, platform.SystemRoleTenantAdmin, "Full tenant access", auth.TenantAdminPermissions},
		{SystemRoleIDTenantOperator, platform.SystemRoleTenantOperator, "Compute and network operator", auth.TenantOperatorPermissions},
		{SystemRoleIDTenantViewer, platform.SystemRoleTenantViewer, "Read-only tenant access", auth.TenantViewerPermissions},
	}
	for _, sr := range systemRoles {
		if existing, ok := s.GetRoleByName("", sr.name); ok {
			// Refresh permissions so new IAM caps land on existing installs.
			s.SetRolePermissions(existing.ID, sr.perms)
			continue
		}
		s.SaveRole(&platform.RoleRecord{
			ID: sr.id, Name: sr.name, Description: sr.desc,
			IsSystem: true, CreatedAt: now,
		})
		s.SetRolePermissions(sr.id, sr.perms)
	}
	return nil
}

// RoleIDForLegacy maps legacy User.Role to system role_id.
func RoleIDForLegacy(role platform.Role) string {
	switch role {
	case platform.RoleRoot:
		return SystemRoleIDRoot
	case platform.RoleTenantAdmin:
		return SystemRoleIDTenantAdmin
	case platform.RoleUser:
		return SystemRoleIDTenantViewer
	default:
		return SystemRoleIDTenantViewer
	}
}
