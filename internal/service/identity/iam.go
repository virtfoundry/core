package identity

import (
	"fmt"
	"strings"
	"time"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

type CreateUserInput struct {
	Username string
	Password string
	Email    string
	RoleID   string
	RoleName string
}

type CreateRoleInput struct {
	Name        string
	Description string
	Permissions []string
}

type CreateAPIKeyInput struct {
	Name          string
	ExpiresInDays int
	Scopes        []string
}

type CreateAPIKeyResult struct {
	Key    *platform.APIKey
	Secret string
}

func (s *Service) CreateUser(tenantID string, in CreateUserInput, actor *auth.Actor) (*platform.User, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id required")
	}
	username := strings.TrimSpace(in.Username)
	if username == "" || strings.TrimSpace(in.Password) == "" {
		return nil, fmt.Errorf("username and password required")
	}
	if _, ok := s.store.GetUserByUsername(username); ok {
		return nil, fmt.Errorf("username already exists")
	}
	roleID, role, err := s.resolveRoleForTenant(tenantID, in.RoleID, in.RoleName, actor)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	u := &platform.User{
		ID: store.NewID(), Username: username, PasswordHash: hash,
		Role: role, RoleID: roleID, TenantID: tenantID,
		Email: in.Email, State: "active", CreatedAt: store.Now(),
	}
	s.store.SaveUser(u)
	return u, nil
}

func (s *Service) ListUsers(tenantID string) []*platform.User {
	if tenantID == "" {
		return s.store.ListUsers()
	}
	return s.store.ListUsersByTenant(tenantID)
}

func (s *Service) UpdateUser(tenantID, userID, email, roleID, state string) (*platform.User, error) {
	u, ok := s.store.GetUser(userID)
	if !ok || u.TenantID != tenantID {
		return nil, fmt.Errorf("user not found")
	}
	if u.Role == platform.RoleRoot {
		return nil, fmt.Errorf("cannot modify root user")
	}
	if email != "" {
		u.Email = email
	}
	if state != "" {
		u.State = state
	}
	if roleID != "" {
		role, err := s.roleForTenant(tenantID, roleID)
		if err != nil {
			return nil, err
		}
		u.RoleID = role.ID
		u.Role = legacyRoleFromSystemName(role.Name)
	}
	s.store.SaveUser(u)
	return u, nil
}

func (s *Service) DeleteUser(tenantID, userID string) error {
	u, ok := s.store.GetUser(userID)
	if !ok || u.TenantID != tenantID {
		return fmt.Errorf("user not found")
	}
	if u.Role == platform.RoleRoot {
		return fmt.Errorf("cannot delete root user")
	}
	s.store.DeleteUser(userID)
	return nil
}

func (s *Service) CreateRole(tenantID string, in CreateRoleInput) (*platform.RoleRecord, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("role name required")
	}
	if _, ok := s.store.GetRoleByName(tenantID, name); ok {
		return nil, fmt.Errorf("role name already exists")
	}
	r := &platform.RoleRecord{
		ID: store.NewID(), TenantID: tenantID, Name: name,
		Description: in.Description, CreatedAt: store.Now(),
	}
	s.store.SaveRole(r)
	s.store.SetRolePermissions(r.ID, in.Permissions)
	r.Permissions = in.Permissions
	return r, nil
}

func (s *Service) ListRoles(tenantID string) []*platform.RoleRecord {
	return s.store.ListRoles(tenantID)
}

func (s *Service) UpdateRole(tenantID, roleID string, desc string, perms []string) (*platform.RoleRecord, error) {
	r, ok := s.store.GetRole(roleID)
	if !ok || r.IsSystem || (r.TenantID != "" && r.TenantID != tenantID) {
		return nil, fmt.Errorf("role not found")
	}
	if desc != "" {
		r.Description = desc
	}
	s.store.SaveRole(r)
	if perms != nil {
		s.store.SetRolePermissions(roleID, perms)
		r.Permissions = perms
	} else {
		r.Permissions, _ = s.store.GetRolePermissions(roleID)
	}
	return r, nil
}

func (s *Service) DeleteRole(tenantID, roleID string) error {
	r, ok := s.store.GetRole(roleID)
	if !ok || r.IsSystem || r.TenantID != tenantID {
		return fmt.Errorf("role not found")
	}
	s.store.DeleteRole(roleID)
	return nil
}

func (s *Service) CreateAPIKey(userID, tenantID string, in CreateAPIKeyInput, actor *auth.Actor) (*CreateAPIKeyResult, error) {
	u, ok := s.store.GetUser(userID)
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	if u.TenantID != tenantID && actor.Role != platform.RoleRoot {
		return nil, fmt.Errorf("forbidden")
	}
	full, prefix, hash, err := auth.GenerateAPIKey()
	if err != nil {
		return nil, err
	}
	resolver := NewPermissionResolver(s.store)
	userPerms := resolver.ForUser(u)
	scopes := auth.FilterScopes(in.Scopes, userPerms)
	var expires *time.Time
	if in.ExpiresInDays > 0 {
		t := time.Now().UTC().Add(time.Duration(in.ExpiresInDays) * 24 * time.Hour)
		expires = &t
	}
	k := &platform.APIKey{
		ID: store.NewID(), UserID: userID, TenantID: u.TenantID,
		Name: strings.TrimSpace(in.Name), Prefix: prefix, SecretHash: hash,
		Scopes: scopes, ExpiresAt: expires, CreatedAt: store.Now(),
	}
	if k.Name == "" {
		k.Name = "default"
	}
	s.store.SaveAPIKey(k)
	return &CreateAPIKeyResult{Key: k, Secret: full}, nil
}

func (s *Service) ListAPIKeys(userID, tenantID string, adminView bool) []*platform.APIKey {
	if adminView && tenantID != "" {
		return s.store.ListAPIKeysByTenant(tenantID)
	}
	return s.store.ListAPIKeys(userID)
}

func (s *Service) RevokeAPIKey(userID, keyID string, admin bool) error {
	k, ok := s.store.GetAPIKey(keyID)
	if !ok {
		return fmt.Errorf("api key not found")
	}
	if !admin && k.UserID != userID {
		return fmt.Errorf("forbidden")
	}
	s.store.DeleteAPIKey(keyID)
	return nil
}

func (s *Service) AuthenticateAPIKey(token string) (*auth.Actor, error) {
	prefix, ok := auth.ParseAPIKeyPrefix(token)
	if !ok {
		return nil, auth.ErrUnauthorized
	}
	k, ok := s.store.GetAPIKeyByPrefix(prefix)
	if !ok || k.RevokedAt != nil {
		return nil, auth.ErrUnauthorized
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return nil, auth.ErrUnauthorized
	}
	if !auth.VerifyAPIKey(token, k.SecretHash) {
		return nil, auth.ErrUnauthorized
	}
	u, ok := s.store.GetUser(k.UserID)
	if !ok || u.State == "disabled" {
		return nil, auth.ErrUnauthorized
	}
	resolver := NewPermissionResolver(s.store)
	perms := resolver.ForAPIKey(u, k)
	s.store.TouchAPIKeyLastUsed(k.ID)
	return &auth.Actor{
		UserID: u.ID, Username: u.Username, TenantID: u.TenantID,
		Role: u.Role, RoleID: u.RoleID, Permissions: perms,
		AuthMethod: "api_key", APIKeyID: k.ID,
	}, nil
}

func (s *Service) ActorFromUser(u *platform.User) *auth.Actor {
	resolver := NewPermissionResolver(s.store)
	return &auth.Actor{
		UserID: u.ID, Username: u.Username, TenantID: u.TenantID,
		Role: u.Role, RoleID: u.RoleID, Permissions: resolver.ForUser(u),
		AuthMethod: "jwt",
	}
}

func (s *Service) resolveRoleForTenant(tenantID, roleID, roleName string, actor *auth.Actor) (string, platform.Role, error) {
	if roleID != "" {
		r, err := s.roleForTenant(tenantID, roleID)
		if err != nil {
			return "", "", err
		}
		return r.ID, legacyRoleFromSystemName(r.Name), nil
	}
	name := strings.TrimSpace(roleName)
	if name == "" {
		name = platform.SystemRoleTenantOperator
	}
	if r, ok := s.store.GetRoleByName("", name); ok {
		return r.ID, legacyRoleFromSystemName(r.Name), nil
	}
	if r, ok := s.store.GetRoleByName(tenantID, name); ok {
		return r.ID, platform.RoleUser, nil
	}
	return "", "", fmt.Errorf("role not found")
}

func (s *Service) roleForTenant(tenantID, roleID string) (*platform.RoleRecord, error) {
	r, ok := s.store.GetRole(roleID)
	if !ok {
		return nil, fmt.Errorf("role not found")
	}
	if r.IsSystem {
		return r, nil
	}
	if r.TenantID != tenantID {
		return nil, fmt.Errorf("role not found")
	}
	return r, nil
}

func legacyRoleFromSystemName(name string) platform.Role {
	switch name {
	case platform.SystemRoleRoot:
		return platform.RoleRoot
	case platform.SystemRoleTenantAdmin:
		return platform.RoleTenantAdmin
	default:
		return platform.RoleUser
	}
}
