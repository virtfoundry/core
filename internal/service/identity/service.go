package identity

import (
	"fmt"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/platform/store"
)

// Service handles users and tenant resolution from JWT claims.
type Service struct {
	store store.Repository
}

func New(st store.Repository) *Service {
	return &Service{store: st}
}

func (s *Service) BootstrapRoot(username, password string) (*platform.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &platform.User{
		ID:           store.NewID(),
		Username:     username,
		Role:         platform.RoleRoot,
		RoleID:       store.SystemRoleIDRoot,
		PasswordHash: hash,
		Email:        username + "@" + branding.EmailDomain,
		State:        "active",
		CreatedAt:    store.Now(),
	}
	s.store.SaveUser(u)
	return u, nil
}

// EnsureRootPasswordHash restores the root credential hash when the root User
// record exists but its credential Secret is missing or unreadable. The
// password is the bootstrap ROOT_PASSWORD value; callers must not pass a
// generated password for an existing root user, since that would rotate the
// credential unexpectedly.
//
// It returns true only when it had to persist a new hash.
func (s *Service) EnsureRootPasswordHash(username, password string) (bool, error) {
	u, ok := s.store.GetUserByUsername(username)
	if !ok {
		return false, fmt.Errorf("root user %q not found while verifying credential hash", username)
	}
	if u.PasswordHash != "" {
		return false, nil
	}
	if password == "" {
		return false, fmt.Errorf("root user %q has no password hash; ROOT_PASSWORD is required to restore its credential Secret", username)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, fmt.Errorf("hash root password: %w", err)
	}
	u.PasswordHash = hash
	s.store.SaveUser(u)

	// SaveUser implementations intentionally have no error return. Read back
	// the hash so a failed Secret write is surfaced during startup instead of
	// becoming a later, opaque login failure.
	stored, ok := s.store.GetUserByUsername(username)
	if !ok || stored.PasswordHash == "" {
		return false, fmt.Errorf("root user %q credential hash was not persisted", username)
	}
	if !auth.CheckPassword(stored.PasswordHash, password) {
		return false, fmt.Errorf("root user %q persisted credential hash does not match ROOT_PASSWORD", username)
	}
	return true, nil
}

// LinkRootToTenant assigns the default tenant to root when not yet set.
func (s *Service) LinkRootToTenant(tenantID string) {
	root, ok := s.store.GetUserByUsername("root")
	if !ok || root.TenantID != "" {
		return
	}
	root.TenantID = tenantID
	s.store.SaveUser(root)
}

func (s *Service) ResolveTenantID(claims *auth.Claims, requestedTenant string) (string, error) {
	if claims.Role == platform.RoleRoot {
		if requestedTenant != "" {
			return requestedTenant, nil
		}
		if claims.TenantID != "" {
			return claims.TenantID, nil
		}
		return "", fmt.Errorf("tenant_id required for root")
	}
	if claims.TenantID == "" {
		return "", fmt.Errorf("no tenant assigned")
	}
	return claims.TenantID, nil
}
