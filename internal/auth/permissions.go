package auth

import "github.com/virtfoundry/core/internal/platform"

// Permission constants for VirtFoundry IAM.
const (
	PermAll = "*"

	PermTenantsRead  = "tenants:read"
	PermTenantsWrite = "tenants:write"

	PermUsersRead  = "users:read"
	PermUsersWrite = "users:write"

	PermVPCsRead  = "vpcs:read"
	PermVPCsWrite = "vpcs:write"

	PermNetworksRead  = "networks:read"
	PermNetworksWrite = "networks:write"

	PermSecurityGroupsRead  = "security_groups:read"
	PermSecurityGroupsWrite = "security_groups:write"

	PermVolumesRead  = "volumes:read"
	PermVolumesWrite = "volumes:write"

	PermVMsRead      = "vms:read"
	PermVMsWrite     = "vms:write"
	PermVMsConsole   = "vms:console"

	PermSSHKeysRead  = "ssh_keys:read"
	PermSSHKeysWrite = "ssh_keys:write"

	PermLoadBalancersRead  = "load_balancers:read"
	PermLoadBalancersWrite = "load_balancers:write"
)

// TenantAdminPermissions is the default set for tenant.admin.
var TenantAdminPermissions = []string{
	PermUsersRead, PermUsersWrite,
	PermVPCsRead, PermVPCsWrite,
	PermNetworksRead, PermNetworksWrite,
	PermSecurityGroupsRead, PermSecurityGroupsWrite,
	PermVolumesRead, PermVolumesWrite,
	PermVMsRead, PermVMsWrite, PermVMsConsole,
	PermSSHKeysRead, PermSSHKeysWrite,
	PermLoadBalancersRead, PermLoadBalancersWrite,
}

// TenantOperatorPermissions for tenant.operator.
var TenantOperatorPermissions = []string{
	PermVPCsRead, PermVPCsWrite,
	PermNetworksRead, PermNetworksWrite,
	PermSecurityGroupsRead, PermSecurityGroupsWrite,
	PermVolumesRead, PermVolumesWrite,
	PermVMsRead, PermVMsWrite, PermVMsConsole,
	PermSSHKeysRead, PermSSHKeysWrite,
	PermLoadBalancersRead, PermLoadBalancersWrite,
}

// TenantViewerPermissions for tenant.viewer.
var TenantViewerPermissions = []string{
	PermVPCsRead, PermNetworksRead, PermSecurityGroupsRead,
	PermVolumesRead, PermVMsRead, PermSSHKeysRead,
	PermLoadBalancersRead,
}

// HasPermission checks actor permissions including wildcard and *:read for viewers.
func HasPermission(perms []string, required string) bool {
	for _, p := range perms {
		if p == PermAll || p == required {
			return true
		}
		if p == "*:read" && len(required) > 5 && required[len(required)-5:] == ":read" {
			return true
		}
	}
	return false
}

// LegacyRolePermissions maps pre-IAM role enum to permissions (backward compatible).
func LegacyRolePermissions(role platform.Role) []string {
	switch role {
	case platform.RoleRoot:
		return []string{PermAll}
	case platform.RoleTenantAdmin:
		return append([]string{}, TenantAdminPermissions...)
	case platform.RoleUser:
		return append([]string{}, TenantViewerPermissions...)
	default:
		return nil
	}
}

// FilterScopes returns scopes that are allowed for the user permissions.
func FilterScopes(requested, userPerms []string) []string {
	if len(requested) == 0 {
		return userPerms
	}
	var out []string
	for _, s := range requested {
		if HasPermission(userPerms, s) {
			out = append(out, s)
		}
	}
	return out
}
