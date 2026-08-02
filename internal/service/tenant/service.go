package tenant

import (
	"context"
	"fmt"

	"github.com/virtforge-cloud/virtforge/internal/auth"
	platformk8s "github.com/virtforge-cloud/virtforge/internal/platform/k8s"
	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service/shared"
)

// Service manages tenants and their K8s namespaces.
type Service struct {
	store store.Repository
	k8s   *platformk8s.Manager
}

func New(st store.Repository, k8s *platformk8s.Manager) *Service {
	return &Service{store: st, k8s: k8s}
}

func (s *Service) CreateTenant(ctx context.Context, name, slug, adminPassword string) (*platform.Tenant, *platform.User, error) {
	slug = shared.SanitizeSlug(slug)
	if slug == "" {
		return nil, nil, fmt.Errorf("invalid tenant slug")
	}

	tenantID := store.NewID()
	res, err := s.k8s.EnsureTenantNamespace(ctx, tenantID, slug, platformk8s.DefaultTenantQuota())
	if err != nil {
		return nil, nil, err
	}

	tenant := &platform.Tenant{
		ID: tenantID, Name: name, Slug: slug,
		Namespace: res.Namespace, State: "active", CreatedAt: store.Now(),
	}
	s.store.SaveTenant(tenant)

	hash, _ := auth.HashPassword(adminPassword)
	user := &platform.User{
		ID: store.NewID(), Username: slug + "-admin",
		Role: platform.RoleTenantAdmin, TenantID: tenantID,
		PasswordHash: hash, CreatedAt: store.Now(),
	}
	s.store.SaveUser(user)
	return tenant, user, nil
}

// EnsureTenant creates a tenant namespace and record if the slug does not exist yet.
func (s *Service) EnsureTenant(ctx context.Context, name, slug string) (*platform.Tenant, error) {
	slug = shared.SanitizeSlug(slug)
	if slug == "" {
		return nil, fmt.Errorf("invalid tenant slug")
	}
	if existing, ok := s.store.GetTenantBySlug(slug); ok {
		return existing, nil
	}

	tenantID := store.NewID()
	res, err := s.k8s.EnsureTenantNamespace(ctx, tenantID, slug, platformk8s.DefaultTenantQuota())
	if err != nil {
		return nil, err
	}

	tenant := &platform.Tenant{
		ID: tenantID, Name: name, Slug: slug,
		Namespace: res.Namespace, State: "active", CreatedAt: store.Now(),
	}
	s.store.SaveTenant(tenant)
	return tenant, nil
}

func (s *Service) ListTenants() []*platform.Tenant {
	return s.store.ListTenants()
}

func (s *Service) GetTenant(id string) (*platform.Tenant, bool) {
	return s.store.GetTenant(id)
}

func (s *Service) Namespace(tenantID string) (string, error) {
	return shared.TenantNamespace(s.store, tenantID)
}
