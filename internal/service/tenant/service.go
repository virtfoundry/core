package tenant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/virtfoundry/core/internal/auth"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

// Service manages tenants and their K8s namespaces.
type Service struct {
	store             store.Repository
	k8s               *platformk8s.Manager
	operatorReconcile bool
	namespaceTimeout  time.Duration
}

func New(st store.Repository, k8s *platformk8s.Manager) *Service {
	return &Service{store: st, k8s: k8s, namespaceTimeout: time.Minute}
}

// SetOperatorReconcile makes the operator the only writer of tenant namespaces,
// quotas and policies: the API writes the Tenant CR and waits for the namespace.
func (s *Service) SetOperatorReconcile(enabled bool) { s.operatorReconcile = enabled }

// provision saves the tenant record and returns once its namespace exists.
func (s *Service) provision(ctx context.Context, name, slug string) (*platform.Tenant, error) {
	t := &platform.Tenant{ID: store.NewID(), Name: name, Slug: slug, State: "active", CreatedAt: store.Now()}
	if !s.operatorReconcile {
		res, err := s.k8s.EnsureTenantNamespace(ctx, t.ID, slug, platformk8s.DefaultTenantQuota())
		if err != nil {
			return nil, err
		}
		t.Namespace = res.Namespace
		s.store.SaveTenant(t)
		return t, nil
	}
	t.Namespace = platformk8s.TenantNamespace(slug)
	s.store.SaveTenant(t) // the store rewrites t with the CR identity
	// Re-read: if another replica created the CR first, t still holds a local ID.
	saved, ok := s.store.GetTenantBySlug(slug)
	if !ok {
		return nil, fmt.Errorf("save tenant %s: record was not persisted", slug)
	}
	if err := s.k8s.WaitNamespace(ctx, t.Namespace, s.namespaceTimeout); err != nil {
		return nil, fmt.Errorf("tenant %s: %w (is virtfoundry-operator running?)", slug, err)
	}
	return saved, nil
}

func (s *Service) CreateTenant(ctx context.Context, name, slug, adminPassword string) (*platform.Tenant, *platform.User, error) {
	slug = shared.SanitizeSlug(slug)
	if slug == "" {
		return nil, nil, fmt.Errorf("invalid tenant slug")
	}
	if strings.TrimSpace(adminPassword) == "" {
		return nil, nil, iaerrors.NewBadRequestError("admin password is required")
	}
	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return nil, nil, fmt.Errorf("hash admin password: %w", err)
	}

	tenant, err := s.provision(ctx, name, slug)
	if err != nil {
		return nil, nil, err
	}

	user := &platform.User{
		ID: store.NewID(), Username: slug + "-admin",
		Role: platform.RoleTenantAdmin, RoleID: store.SystemRoleIDTenantAdmin, TenantID: tenant.ID,
		PasswordHash: hash, State: "active", CreatedAt: store.Now(),
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
	existing, ok := s.store.GetTenantBySlug(slug)
	if !ok {
		return s.provision(ctx, name, slug)
	}
	if s.operatorReconcile {
		if err := s.k8s.WaitNamespace(ctx, existing.Namespace, s.namespaceTimeout); err != nil {
			return nil, fmt.Errorf("tenant %s: %w (is virtfoundry-operator running?)", slug, err)
		}
		return existing, nil
	}
	// Upsert namespace resources (quota + CDI importer egress NP) for
	// tenants created before this policy existed.
	if _, err := s.k8s.EnsureTenantNamespace(ctx, existing.ID, slug, platformk8s.DefaultTenantQuota()); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *Service) ListTenants() []*platform.Tenant {
	return s.store.ListTenants()
}

func (s *Service) GetTenant(id string) (*platform.Tenant, bool) {
	return s.store.GetTenant(id)
}

func (s *Service) DeleteTenant(ctx context.Context, id string) error {
	t, ok := s.store.GetTenant(id)
	if !ok {
		return iaerrors.NewNotFoundError("tenant", id)
	}
	if t.Slug == branding.DefaultTenantSlug {
		return iaerrors.NewForbiddenError("cannot delete the default tenant")
	}

	// With the operator, the Tenant finalizer removes the namespace.
	if !s.operatorReconcile {
		if err := s.k8s.DeleteTenantNamespace(ctx, t.Namespace); err != nil {
			return err
		}
	}
	s.store.PurgeTenantData(id)
	s.store.DeleteTenant(id)
	return nil
}

func (s *Service) Namespace(tenantID string) (string, error) {
	return shared.TenantNamespace(s.store, tenantID)
}
