package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/virtfoundry/core/internal/platform"
)

func (m *MySQL) applyMigration005() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS roles (
			id CHAR(36) PRIMARY KEY,
			tenant_id CHAR(36) NULL,
			name VARCHAR(128) NOT NULL,
			description TEXT NULL,
			is_system TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME(3) NOT NULL,
			UNIQUE KEY uk_role_tenant_name (tenant_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS role_permissions (
			role_id CHAR(36) NOT NULL,
			permission VARCHAR(64) NOT NULL,
			PRIMARY KEY (role_id, permission)
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id CHAR(36) PRIMARY KEY,
			user_id CHAR(36) NOT NULL,
			tenant_id CHAR(36) NULL,
			name VARCHAR(255) NOT NULL,
			prefix VARCHAR(16) NOT NULL,
			secret_hash VARCHAR(255) NOT NULL,
			scopes_json JSON NULL,
			expires_at DATETIME(3) NULL,
			last_used_at DATETIME(3) NULL,
			revoked_at DATETIME(3) NULL,
			created_at DATETIME(3) NOT NULL,
			UNIQUE KEY uk_api_key_prefix (prefix),
			INDEX idx_api_key_user (user_id)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := m.db.Exec(stmt); err != nil {
			return err
		}
	}
	_, _ = m.db.Exec(`ALTER TABLE users ADD COLUMN role_id CHAR(36) NULL`)
	_, _ = m.db.Exec(`ALTER TABLE users ADD COLUMN state VARCHAR(32) NOT NULL DEFAULT 'active'`)
	_, _ = m.db.Exec(`INSERT IGNORE INTO schema_migrations (version) VALUES ('005_iam')`)
	return m.SeedIAM()
}

func (m *MySQL) SeedIAM() error {
	if err := SeedIAM(m); err != nil {
		return err
	}
	_, _ = m.db.Exec(`UPDATE users SET role_id=? WHERE role=? AND (role_id IS NULL OR role_id='')`, SystemRoleIDRoot, platform.RoleRoot)
	_, _ = m.db.Exec(`UPDATE users SET role_id=? WHERE role=? AND (role_id IS NULL OR role_id='')`, SystemRoleIDTenantAdmin, platform.RoleTenantAdmin)
	_, _ = m.db.Exec(`UPDATE users SET role_id=? WHERE role=? AND (role_id IS NULL OR role_id='')`, SystemRoleIDTenantViewer, platform.RoleUser)
	return nil
}

func (m *MySQL) ListUsersByTenant(tenantID string) []*platform.User {
	rows, err := m.db.Query(`SELECT id, username, password_hash, role, role_id, tenant_id, email, state, created_at FROM users WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (m *MySQL) DeleteUser(id string) {
	_, _ = m.db.Exec(`DELETE FROM users WHERE id=?`, id)
}

func (m *MySQL) SaveRole(r *platform.RoleRecord) {
	_, _ = m.db.Exec(`INSERT INTO roles (id, tenant_id, name, description, is_system, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), description=VALUES(description), is_system=VALUES(is_system)`,
		r.ID, nullStr(r.TenantID), r.Name, nullStr(r.Description), r.IsSystem, r.CreatedAt)
}

func (m *MySQL) GetRole(id string) (*platform.RoleRecord, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, name, description, is_system, created_at FROM roles WHERE id=?`, id)
	return scanRole(row)
}

func (m *MySQL) GetRoleByName(tenantID, name string) (*platform.RoleRecord, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, name, description, is_system, created_at FROM roles WHERE name=? AND (tenant_id=? OR (tenant_id IS NULL AND ?=''))`,
		name, nullStr(tenantID), tenantID)
	return scanRole(row)
}

func (m *MySQL) ListRoles(tenantID string) []*platform.RoleRecord {
	rows, err := m.db.Query(`SELECT id, tenant_id, name, description, is_system, created_at FROM roles
		WHERE is_system=1 OR tenant_id=? ORDER BY is_system DESC, name`, nullStr(tenantID))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.RoleRecord
	for rows.Next() {
		var r platform.RoleRecord
		var tid sql.NullString
		var desc sql.NullString
		var isSystem int
		if err := rows.Scan(&r.ID, &tid, &r.Name, &desc, &isSystem, &r.CreatedAt); err != nil {
			continue
		}
		r.TenantID = tid.String
		r.Description = desc.String
		r.IsSystem = isSystem == 1
		perms, _ := m.GetRolePermissions(r.ID)
		r.Permissions = perms
		out = append(out, &r)
	}
	return out
}

func (m *MySQL) DeleteRole(id string) {
	_, _ = m.db.Exec(`DELETE FROM role_permissions WHERE role_id=?`, id)
	_, _ = m.db.Exec(`DELETE FROM roles WHERE id=? AND is_system=0`, id)
}

func (m *MySQL) GetRolePermissions(roleID string) ([]string, bool) {
	rows, err := m.db.Query(`SELECT permission FROM role_permissions WHERE role_id=?`, roleID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil {
			out = append(out, p)
		}
	}
	return out, len(out) > 0
}

func (m *MySQL) SetRolePermissions(roleID string, perms []string) {
	_, _ = m.db.Exec(`DELETE FROM role_permissions WHERE role_id=?`, roleID)
	for _, p := range perms {
		_, _ = m.db.Exec(`INSERT INTO role_permissions (role_id, permission) VALUES (?, ?)`, roleID, p)
	}
}

func (m *MySQL) SaveAPIKey(k *platform.APIKey) {
	scopes, _ := json.Marshal(k.Scopes)
	_, _ = m.db.Exec(`INSERT INTO api_keys (id, user_id, tenant_id, name, prefix, secret_hash, scopes_json, expires_at, last_used_at, revoked_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), scopes_json=VALUES(scopes_json), revoked_at=VALUES(revoked_at), last_used_at=VALUES(last_used_at)`,
		k.ID, k.UserID, nullStr(k.TenantID), k.Name, k.Prefix, k.SecretHash, string(scopes),
		nullTimePtr(k.ExpiresAt), nullTimePtr(k.LastUsedAt), nullTimePtr(k.RevokedAt), k.CreatedAt)
}

func (m *MySQL) GetAPIKey(id string) (*platform.APIKey, bool) {
	row := m.db.QueryRow(`SELECT id, user_id, tenant_id, name, prefix, secret_hash, scopes_json, expires_at, last_used_at, revoked_at, created_at FROM api_keys WHERE id=?`, id)
	return scanAPIKey(row)
}

func (m *MySQL) GetAPIKeyByPrefix(prefix string) (*platform.APIKey, bool) {
	row := m.db.QueryRow(`SELECT id, user_id, tenant_id, name, prefix, secret_hash, scopes_json, expires_at, last_used_at, revoked_at, created_at FROM api_keys WHERE prefix=? AND revoked_at IS NULL`, prefix)
	return scanAPIKey(row)
}

func (m *MySQL) ListAPIKeys(userID string) []*platform.APIKey {
	rows, err := m.db.Query(`SELECT id, user_id, tenant_id, name, prefix, secret_hash, scopes_json, expires_at, last_used_at, revoked_at, created_at FROM api_keys WHERE user_id=?`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanAPIKeys(rows)
}

func (m *MySQL) ListAPIKeysByTenant(tenantID string) []*platform.APIKey {
	rows, err := m.db.Query(`SELECT id, user_id, tenant_id, name, prefix, secret_hash, scopes_json, expires_at, last_used_at, revoked_at, created_at FROM api_keys WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanAPIKeys(rows)
}

func (m *MySQL) DeleteAPIKey(id string) {
	now := time.Now().UTC()
	_, _ = m.db.Exec(`UPDATE api_keys SET revoked_at=? WHERE id=?`, now, id)
}

func (m *MySQL) TouchAPIKeyLastUsed(id string) {
	now := time.Now().UTC()
	_, _ = m.db.Exec(`UPDATE api_keys SET last_used_at=? WHERE id=?`, now, id)
}

func scanRole(row *sql.Row) (*platform.RoleRecord, bool) {
	var r platform.RoleRecord
	var tid, desc sql.NullString
	var isSystem int
	if err := row.Scan(&r.ID, &tid, &r.Name, &desc, &isSystem, &r.CreatedAt); err != nil {
		return nil, false
	}
	r.TenantID = tid.String
	r.Description = desc.String
	r.IsSystem = isSystem == 1
	return &r, true
}

func scanAPIKey(row *sql.Row) (*platform.APIKey, bool) {
	var k platform.APIKey
	var tid, scopes sql.NullString
	var expires, lastUsed, revoked sql.NullTime
	if err := row.Scan(&k.ID, &k.UserID, &tid, &k.Name, &k.Prefix, &k.SecretHash, &scopes, &expires, &lastUsed, &revoked, &k.CreatedAt); err != nil {
		return nil, false
	}
	k.TenantID = tid.String
	if scopes.Valid && scopes.String != "" {
		_ = json.Unmarshal([]byte(scopes.String), &k.Scopes)
	}
	if expires.Valid {
		t := expires.Time
		k.ExpiresAt = &t
	}
	if lastUsed.Valid {
		t := lastUsed.Time
		k.LastUsedAt = &t
	}
	if revoked.Valid {
		t := revoked.Time
		k.RevokedAt = &t
	}
	return &k, true
}

func scanAPIKeys(rows *sql.Rows) []*platform.APIKey {
	var out []*platform.APIKey
	for rows.Next() {
		var k platform.APIKey
		var tid, scopes sql.NullString
		var expires, lastUsed, revoked sql.NullTime
		if err := rows.Scan(&k.ID, &k.UserID, &tid, &k.Name, &k.Prefix, &k.SecretHash, &scopes, &expires, &lastUsed, &revoked, &k.CreatedAt); err != nil {
			continue
		}
		k.TenantID = tid.String
		if scopes.Valid && scopes.String != "" {
			_ = json.Unmarshal([]byte(scopes.String), &k.Scopes)
		}
		if expires.Valid {
			t := expires.Time
			k.ExpiresAt = &t
		}
		if lastUsed.Valid {
			t := lastUsed.Time
			k.LastUsedAt = &t
		}
		if revoked.Valid {
			t := revoked.Time
			k.RevokedAt = &t
		}
		out = append(out, &k)
	}
	return out
}

func scanUsers(rows *sql.Rows) []*platform.User {
	var out []*platform.User
	for rows.Next() {
		var u platform.User
		var tenantID, email, roleID, state sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &roleID, &tenantID, &email, &state, &u.CreatedAt); err != nil {
			continue
		}
		u.TenantID = tenantID.String
		u.Email = email.String
		u.RoleID = roleID.String
		u.State = state.String
		out = append(out, &u)
	}
	return out
}

func nullTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}
