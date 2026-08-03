package store

import (
	"time"

	"github.com/virtfoundry/core/internal/platform"
)

func (m *Memory) ListUsersByTenant(tenantID string) []*platform.User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.User
	for _, u := range m.users {
		if u.TenantID == tenantID {
			out = append(out, u)
		}
	}
	return out
}

func (m *Memory) DeleteUser(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return
	}
	delete(m.users, id)
	delete(m.usersByName, u.Username)
}

func (m *Memory) SaveRole(r *platform.RoleRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.roles == nil {
		m.roles = make(map[string]*platform.RoleRecord)
	}
	m.roles[r.ID] = r
}

func (m *Memory) GetRole(id string) (*platform.RoleRecord, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.roles[id]
	return r, ok
}

func (m *Memory) GetRoleByName(tenantID, name string) (*platform.RoleRecord, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.roles {
		if r.Name == name && r.TenantID == tenantID {
			return r, true
		}
	}
	return nil, false
}

func (m *Memory) ListRoles(tenantID string) []*platform.RoleRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.RoleRecord
	for _, r := range m.roles {
		if r.IsSystem {
			out = append(out, r)
		} else if tenantID != "" && r.TenantID == tenantID {
			out = append(out, r)
		}
	}
	return out
}

func (m *Memory) DeleteRole(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.roles, id)
	delete(m.rolePerms, id)
}

func (m *Memory) GetRolePermissions(roleID string) ([]string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.rolePerms[roleID]
	if !ok {
		return nil, false
	}
	out := append([]string(nil), p...)
	return out, true
}

func (m *Memory) SetRolePermissions(roleID string, perms []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rolePerms == nil {
		m.rolePerms = make(map[string][]string)
	}
	m.rolePerms[roleID] = append([]string(nil), perms...)
}

func (m *Memory) SaveAPIKey(k *platform.APIKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.apiKeys == nil {
		m.apiKeys = make(map[string]*platform.APIKey)
	}
	m.apiKeys[k.ID] = k
}

func (m *Memory) GetAPIKey(id string) (*platform.APIKey, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.apiKeys[id]
	return k, ok
}

func (m *Memory) GetAPIKeyByPrefix(prefix string) (*platform.APIKey, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, k := range m.apiKeys {
		if k.Prefix == prefix && k.RevokedAt == nil {
			return k, true
		}
	}
	return nil, false
}

func (m *Memory) ListAPIKeys(userID string) []*platform.APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.APIKey
	for _, k := range m.apiKeys {
		if k.UserID == userID {
			out = append(out, k)
		}
	}
	return out
}

func (m *Memory) ListAPIKeysByTenant(tenantID string) []*platform.APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.APIKey
	for _, k := range m.apiKeys {
		if k.TenantID == tenantID {
			out = append(out, k)
		}
	}
	return out
}

func (m *Memory) DeleteAPIKey(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if k, ok := m.apiKeys[id]; ok {
		now := time.Now().UTC()
		k.RevokedAt = &now
	}
}

func (m *Memory) TouchAPIKeyLastUsed(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if k, ok := m.apiKeys[id]; ok {
		now := time.Now().UTC()
		k.LastUsedAt = &now
	}
}

func (m *Memory) SeedIAM() error {
	if err := SeedIAM(m); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.RoleID == "" {
			u.RoleID = RoleIDForLegacy(u.Role)
		}
		if u.State == "" {
			u.State = "active"
		}
	}
	return nil
}
