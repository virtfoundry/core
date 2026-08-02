package identity

import (
	"fmt"

	"github.com/virtforge-cloud/virtforge/internal/auth"
	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/branding"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
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
		PasswordHash: hash,
		Email:        username + "@" + branding.EmailDomain,
		CreatedAt:    store.Now(),
	}
	s.store.SaveUser(u)
	return u, nil
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
