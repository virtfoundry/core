package auth

import "github.com/virtfoundry/core/internal/platform"

// Actor is the authenticated principal for a request.
type Actor struct {
	UserID      string
	Username    string
	TenantID    string
	Role        platform.Role
	RoleID      string
	Permissions []string
	AuthMethod  string // jwt | api_key
	APIKeyID    string
}
