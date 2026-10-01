package store

import (
	"time"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const tenantCacheTTL = 30 * time.Second
const rolePermCacheTTL = 30 * time.Second

type tenantSnapshot struct {
	tenants []*platform.Tenant
	byID    map[string]*platform.Tenant
	bySlug  map[string]*platform.Tenant
	nsToID  map[string]string
}

func (k *Kubernetes) invalidateTenantCache() {
	k.tenantCacheMu.Lock()
	k.tenantCache = nil
	k.tenantCacheMu.Unlock()
}

func (k *Kubernetes) invalidateRolePermCache() {
	k.rolePermCacheMu.Lock()
	k.rolePermCache = nil
	k.rolePermCacheMu.Unlock()
}

func (k *Kubernetes) cachedRolePermissions(roleID string) ([]string, bool) {
	k.rolePermCacheMu.RLock()
	c := k.rolePermCache
	k.rolePermCacheMu.RUnlock()
	if c == nil || time.Now().After(c.expires) {
		return nil, false
	}
	perms, ok := c.byID[roleID]
	if !ok {
		return nil, false
	}
	return append([]string(nil), perms...), true
}

func (k *Kubernetes) storeRolePermissionsCache(roleID string, perms []string) {
	k.rolePermCacheMu.Lock()
	defer k.rolePermCacheMu.Unlock()
	now := time.Now()
	if k.rolePermCache == nil || now.After(k.rolePermCache.expires) {
		k.rolePermCache = &cachedRolePerms{
			byID:    map[string][]string{},
			expires: now.Add(rolePermCacheTTL),
		}
	}
	k.rolePermCache.byID[roleID] = append([]string(nil), perms...)
}

type cachedRolePerms struct {
	byID    map[string][]string
	expires time.Time
}

func (k *Kubernetes) tenantSnapshot() *tenantSnapshot {
	k.tenantCacheMu.RLock()
	if k.tenantCache != nil && time.Now().Before(k.tenantCache.expires) {
		snap := k.tenantCache.snap
		k.tenantCacheMu.RUnlock()
		return snap
	}
	k.tenantCacheMu.RUnlock()

	k.tenantCacheMu.Lock()
	defer k.tenantCacheMu.Unlock()
	if k.tenantCache != nil && time.Now().Before(k.tenantCache.expires) {
		return k.tenantCache.snap
	}

	list, err := k.dyn.Resource(mapping.TenantGVR).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		if k.tenantCache != nil {
			return k.tenantCache.snap
		}
		return &tenantSnapshot{
			byID:   map[string]*platform.Tenant{},
			bySlug: map[string]*platform.Tenant{},
			nsToID: map[string]string{},
		}
	}

	snap := &tenantSnapshot{
		tenants: make([]*platform.Tenant, 0, len(list.Items)),
		byID:    make(map[string]*platform.Tenant, len(list.Items)),
		bySlug:  make(map[string]*platform.Tenant, len(list.Items)),
		nsToID:  make(map[string]string, len(list.Items)),
	}
	for i := range list.Items {
		t := mapping.TenantFromUnstructured(&list.Items[i])
		snap.tenants = append(snap.tenants, t)
		snap.byID[t.ID] = t
		snap.bySlug[t.Slug] = t
		if t.Namespace != "" {
			snap.nsToID[t.Namespace] = t.ID
		}
	}
	k.tenantCache = &cachedTenants{snap: snap, expires: time.Now().Add(tenantCacheTTL)}
	return snap
}

type cachedTenants struct {
	snap    *tenantSnapshot
	expires time.Time
}
