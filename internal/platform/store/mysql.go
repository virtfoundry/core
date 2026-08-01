package store

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/virtforge-cloud/virtforge/internal/platform"
)

//go:embed migrations/schema.sql
var schemaFS embed.FS

var _ Repository = (*MySQL)(nil)

type MySQL struct {
	db *sql.DB
}

func NewMySQL(dsn string) (*MySQL, error) {
	if !strings.Contains(dsn, "parseTime") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "parseTime=true&loc=UTC&multiStatements=true"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	m := &MySQL{db: db}
	if err := m.Migrate(); err != nil {
		return nil, err
	}
	if err := SeedCatalog(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *MySQL) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *MySQL) Migrate() error {
	raw, err := schemaFS.ReadFile("migrations/schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if _, err := m.db.Exec(string(raw)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	if err := m.applyMigration002(); err != nil {
		return err
	}
	_, _ = m.db.Exec(`INSERT IGNORE INTO schema_migrations (version) VALUES ('001_initial')`)
	return nil
}

func (m *MySQL) applyMigration002() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS audit_events (
			id CHAR(36) PRIMARY KEY,
			actor_user_id CHAR(36) NOT NULL,
			actor_role VARCHAR(32) NOT NULL,
			target_tenant_id CHAR(36) NOT NULL,
			action VARCHAR(64) NOT NULL,
			method VARCHAR(16) NOT NULL,
			path VARCHAR(512) NOT NULL,
			resource_type VARCHAR(64) NULL,
			resource_id VARCHAR(128) NULL,
			created_at DATETIME(3) NOT NULL,
			INDEX idx_audit_tenant (target_tenant_id),
			INDEX idx_audit_actor (actor_user_id),
			INDEX idx_audit_created (created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS ip_addresses (
			id CHAR(36) PRIMARY KEY,
			network_id CHAR(36) NOT NULL,
			address VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'available',
			vm_nic_id CHAR(36) NULL,
			created_at DATETIME(3) NOT NULL,
			UNIQUE KEY uk_ip_network_addr (network_id, address),
			INDEX idx_ip_status (network_id, status)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := m.db.Exec(stmt); err != nil {
			return fmt.Errorf("migration 002: %w", err)
		}
	}
	// Best-effort column adds for existing deployments.
	_, _ = m.db.Exec(`ALTER TABLE networks ADD COLUMN network_type VARCHAR(32) NOT NULL DEFAULT 'isolated'`)
	_, _ = m.db.Exec(`ALTER TABLE networks MODIFY tenant_id CHAR(36) NULL`)
	_, _ = m.db.Exec(`ALTER TABLE networks MODIFY vpc_id CHAR(36) NULL`)
	_, _ = m.db.Exec(`INSERT IGNORE INTO schema_migrations (version) VALUES ('002_audit_networking')`)
	return nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scanTime(t sql.NullTime) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}

func (m *MySQL) SaveUser(u *platform.User) {
	_, _ = m.db.Exec(`INSERT INTO users (id, username, password_hash, role, tenant_id, email, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE username=VALUES(username), password_hash=VALUES(password_hash),
		role=VALUES(role), tenant_id=VALUES(tenant_id), email=VALUES(email)`,
		u.ID, u.Username, u.PasswordHash, u.Role, nullStr(u.TenantID), nullStr(u.Email), u.CreatedAt)
}

func (m *MySQL) GetUserByUsername(username string) (*platform.User, bool) {
	row := m.db.QueryRow(`SELECT id, username, password_hash, role, tenant_id, email, created_at FROM users WHERE username=?`, username)
	return scanUser(row)
}

func (m *MySQL) HasRootUser() bool {
	var n int
	_ = m.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role=?`, platform.RoleRoot).Scan(&n)
	return n > 0
}

func (m *MySQL) GetUser(id string) (*platform.User, bool) {
	row := m.db.QueryRow(`SELECT id, username, password_hash, role, tenant_id, email, created_at FROM users WHERE id=?`, id)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*platform.User, bool) {
	var u platform.User
	var tenantID, email sql.NullString
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &tenantID, &email, &u.CreatedAt); err != nil {
		return nil, false
	}
	u.TenantID = tenantID.String
	u.Email = email.String
	return &u, true
}

func (m *MySQL) ListUsers() []*platform.User {
	rows, err := m.db.Query(`SELECT id, username, password_hash, role, tenant_id, email, created_at FROM users`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.User
	for rows.Next() {
		var u platform.User
		var tenantID, email sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &tenantID, &email, &u.CreatedAt); err != nil {
			continue
		}
		u.TenantID = tenantID.String
		u.Email = email.String
		out = append(out, &u)
	}
	return out
}

func (m *MySQL) SaveTenant(t *platform.Tenant) {
	_, _ = m.db.Exec(`INSERT INTO tenants (id, name, slug, namespace, state, external_uuid, import_source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), slug=VALUES(slug), namespace=VALUES(namespace),
		state=VALUES(state), external_uuid=VALUES(external_uuid), import_source=VALUES(import_source)`,
		t.ID, t.Name, t.Slug, t.Namespace, t.State, nullStr(t.ExternalUUID), nullStr(t.ImportSource), t.CreatedAt)
}

func (m *MySQL) GetTenant(id string) (*platform.Tenant, bool) {
	row := m.db.QueryRow(`SELECT id, name, slug, namespace, state, external_uuid, import_source, created_at FROM tenants WHERE id=?`, id)
	var t platform.Tenant
	var ext, src sql.NullString
	if err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Namespace, &t.State, &ext, &src, &t.CreatedAt); err != nil {
		return nil, false
	}
	t.ExternalUUID = ext.String
	t.ImportSource = src.String
	return &t, true
}

func (m *MySQL) GetTenantBySlug(slug string) (*platform.Tenant, bool) {
	row := m.db.QueryRow(`SELECT id, name, slug, namespace, state, external_uuid, import_source, created_at FROM tenants WHERE slug=?`, slug)
	var t platform.Tenant
	var ext, src sql.NullString
	if err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Namespace, &t.State, &ext, &src, &t.CreatedAt); err != nil {
		return nil, false
	}
	t.ExternalUUID = ext.String
	t.ImportSource = src.String
	return &t, true
}

func (m *MySQL) ListTenants() []*platform.Tenant {
	rows, err := m.db.Query(`SELECT id, name, slug, namespace, state, external_uuid, import_source, created_at FROM tenants ORDER BY name`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.Tenant
	for rows.Next() {
		var t platform.Tenant
		var ext, src sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Namespace, &t.State, &ext, &src, &t.CreatedAt); err != nil {
			continue
		}
		t.ExternalUUID = ext.String
		t.ImportSource = src.String
		out = append(out, &t)
	}
	return out
}

func (m *MySQL) SaveVPC(v *platform.VPC) {
	_, _ = m.db.Exec(`INSERT INTO vpcs (id, tenant_id, name, cidr, namespace, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), cidr=VALUES(cidr), namespace=VALUES(namespace), state=VALUES(state)`,
		v.ID, v.TenantID, v.Name, v.CIDR, v.Namespace, v.State, v.CreatedAt)
}

func (m *MySQL) ListVPCs(tenantID string) []*platform.VPC {
	rows, err := m.db.Query(`SELECT id, tenant_id, name, cidr, namespace, state, created_at FROM vpcs WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanVPCRows(rows)
}

func (m *MySQL) GetVPC(id string) (*platform.VPC, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, name, cidr, namespace, state, created_at FROM vpcs WHERE id=?`, id)
	var v platform.VPC
	if err := row.Scan(&v.ID, &v.TenantID, &v.Name, &v.CIDR, &v.Namespace, &v.State, &v.CreatedAt); err != nil {
		return nil, false
	}
	return &v, true
}

func (m *MySQL) DeleteVPC(id string) {
	_, _ = m.db.Exec(`DELETE FROM vpcs WHERE id=?`, id)
}

func scanVPCRows(rows *sql.Rows) []*platform.VPC {
	var out []*platform.VPC
	for rows.Next() {
		var v platform.VPC
		if err := rows.Scan(&v.ID, &v.TenantID, &v.Name, &v.CIDR, &v.Namespace, &v.State, &v.CreatedAt); err != nil {
			continue
		}
		out = append(out, &v)
	}
	return out
}

func (m *MySQL) SaveSG(sg *platform.SecurityGroup) {
	rules, _ := json.Marshal(sg.Rules)
	_, _ = m.db.Exec(`INSERT INTO security_groups (id, tenant_id, vpc_id, name, description, rules_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), description=VALUES(description), rules_json=VALUES(rules_json)`,
		sg.ID, sg.TenantID, nullStr(sg.VPCID), sg.Name, nullStr(sg.Description), string(rules), sg.CreatedAt)
}

func (m *MySQL) ListSGs(tenantID string) []*platform.SecurityGroup {
	rows, err := m.db.Query(`SELECT id, tenant_id, vpc_id, name, description, rules_json, created_at FROM security_groups WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanSGRows(rows)
}

func (m *MySQL) GetSG(id string) (*platform.SecurityGroup, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, vpc_id, name, description, rules_json, created_at FROM security_groups WHERE id=?`, id)
	var sg platform.SecurityGroup
	var vpcID, desc sql.NullString
	var rulesJSON string
	if err := row.Scan(&sg.ID, &sg.TenantID, &vpcID, &sg.Name, &desc, &rulesJSON, &sg.CreatedAt); err != nil {
		return nil, false
	}
	sg.VPCID = vpcID.String
	sg.Description = desc.String
	_ = json.Unmarshal([]byte(rulesJSON), &sg.Rules)
	return &sg, true
}

func (m *MySQL) DeleteSG(id string) {
	_, _ = m.db.Exec(`DELETE FROM security_groups WHERE id=?`, id)
}

func scanSGRows(rows *sql.Rows) []*platform.SecurityGroup {
	var out []*platform.SecurityGroup
	for rows.Next() {
		var sg platform.SecurityGroup
		var vpcID, desc sql.NullString
		var rulesJSON string
		if err := rows.Scan(&sg.ID, &sg.TenantID, &vpcID, &sg.Name, &desc, &rulesJSON, &sg.CreatedAt); err != nil {
			continue
		}
		sg.VPCID = vpcID.String
		sg.Description = desc.String
		_ = json.Unmarshal([]byte(rulesJSON), &sg.Rules)
		out = append(out, &sg)
	}
	return out
}

func (m *MySQL) SaveNetwork(n *platform.Network) {
	nt := n.NetworkType
	if nt == "" {
		nt = platform.NetworkTypeIsolated
	}
	_, _ = m.db.Exec(`INSERT INTO networks (id, tenant_id, vpc_id, name, network_type, cidr, gateway, nad_namespace, nad_name, state, external_uuid, import_source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), network_type=VALUES(network_type), cidr=VALUES(cidr), gateway=VALUES(gateway),
		nad_namespace=VALUES(nad_namespace), nad_name=VALUES(nad_name), state=VALUES(state),
		external_uuid=VALUES(external_uuid), import_source=VALUES(import_source)`,
		n.ID, nullStr(n.TenantID), nullStr(n.VPCID), n.Name, nt, n.CIDR, nullStr(n.Gateway), nullStr(n.NADNamespace), nullStr(n.NADName),
		n.State, nullStr(n.ExternalUUID), nullStr(n.ImportSource), n.CreatedAt)
}

func (m *MySQL) GetSharedNetwork() (*platform.Network, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, vpc_id, name, network_type, cidr, gateway, nad_namespace, nad_name, state, external_uuid, import_source, created_at
		FROM networks WHERE network_type=? LIMIT 1`, platform.NetworkTypeShared)
	return scanNetworkRow(row)
}

func (m *MySQL) GetNetwork(id string) (*platform.Network, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, vpc_id, name, network_type, cidr, gateway, nad_namespace, nad_name, state, external_uuid, import_source, created_at
		FROM networks WHERE id=?`, id)
	return scanNetworkRow(row)
}

func scanNetworkRow(row *sql.Row) (*platform.Network, bool) {
	var n platform.Network
	var tenantID, vpcID, gw, nadNS, nadName, ext, src sql.NullString
	if err := row.Scan(&n.ID, &tenantID, &vpcID, &n.Name, &n.NetworkType, &n.CIDR, &gw, &nadNS, &nadName, &n.State, &ext, &src, &n.CreatedAt); err != nil {
		return nil, false
	}
	n.TenantID = tenantID.String
	n.VPCID = vpcID.String
	n.Gateway = gw.String
	n.NADNamespace = nadNS.String
	n.NADName = nadName.String
	n.ExternalUUID = ext.String
	n.ImportSource = src.String
	return &n, true
}

func (m *MySQL) DeleteNetwork(id string) {
	_, _ = m.db.Exec(`DELETE FROM networks WHERE id=?`, id)
}

func (m *MySQL) ListNetworks(tenantID string) []*platform.Network {
	rows, err := m.db.Query(`SELECT id, tenant_id, vpc_id, name, network_type, cidr, gateway, nad_namespace, nad_name, state, external_uuid, import_source, created_at
		FROM networks WHERE tenant_id=? OR network_type=?`, tenantID, platform.NetworkTypeShared)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.Network
	for rows.Next() {
		var n platform.Network
		var tenant, vpc, gw, nadNS, nadName, ext, src sql.NullString
		if err := rows.Scan(&n.ID, &tenant, &vpc, &n.Name, &n.NetworkType, &n.CIDR, &gw, &nadNS, &nadName, &n.State, &ext, &src, &n.CreatedAt); err != nil {
			continue
		}
		n.TenantID = tenant.String
		n.VPCID = vpc.String
		n.Gateway = gw.String
		n.NADNamespace = nadNS.String
		n.NADName = nadName.String
		n.ExternalUUID = ext.String
		n.ImportSource = src.String
		out = append(out, &n)
	}
	return out
}

func (m *MySQL) SaveVolume(v *platform.Volume) {
	_, _ = m.db.Exec(`INSERT INTO volumes (id, tenant_id, name, size_gi, namespace, pvc_name, state, vm_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE state=VALUES(state), vm_id=VALUES(vm_id)`,
		v.ID, v.TenantID, v.Name, v.SizeGi, v.Namespace, v.PVCName, v.State, nullStr(v.VMID), v.CreatedAt)
}

func (m *MySQL) ListVolumes(tenantID string) []*platform.Volume {
	rows, err := m.db.Query(`SELECT id, tenant_id, name, size_gi, namespace, pvc_name, state, vm_id, created_at FROM volumes WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.Volume
	for rows.Next() {
		var v platform.Volume
		var vmID sql.NullString
		if err := rows.Scan(&v.ID, &v.TenantID, &v.Name, &v.SizeGi, &v.Namespace, &v.PVCName, &v.State, &vmID, &v.CreatedAt); err != nil {
			continue
		}
		v.VMID = vmID.String
		out = append(out, &v)
	}
	return out
}

func (m *MySQL) GetVolume(id string) (*platform.Volume, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, name, size_gi, namespace, pvc_name, state, vm_id, created_at FROM volumes WHERE id=?`, id)
	var v platform.Volume
	var vmID sql.NullString
	if err := row.Scan(&v.ID, &v.TenantID, &v.Name, &v.SizeGi, &v.Namespace, &v.PVCName, &v.State, &vmID, &v.CreatedAt); err != nil {
		return nil, false
	}
	v.VMID = vmID.String
	return &v, true
}

func (m *MySQL) SaveSnapshot(s *platform.Snapshot) {
	_, _ = m.db.Exec(`INSERT INTO snapshots (id, tenant_id, volume_id, name, namespace, snapshot_uid, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE state=VALUES(state), snapshot_uid=VALUES(snapshot_uid)`,
		s.ID, s.TenantID, s.VolumeID, s.Name, s.Namespace, nullStr(s.SnapshotUID), s.State, s.CreatedAt)
}

func (m *MySQL) ListSnapshots(tenantID string) []*platform.Snapshot {
	rows, err := m.db.Query(`SELECT id, tenant_id, volume_id, name, namespace, snapshot_uid, state, created_at FROM snapshots WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.Snapshot
	for rows.Next() {
		var s platform.Snapshot
		var uid sql.NullString
		if err := rows.Scan(&s.ID, &s.TenantID, &s.VolumeID, &s.Name, &s.Namespace, &uid, &s.State, &s.CreatedAt); err != nil {
			continue
		}
		s.SnapshotUID = uid.String
		out = append(out, &s)
	}
	return out
}

func (m *MySQL) SaveVMSnapshot(s *platform.VMSnapshot) {
	_, _ = m.db.Exec(`INSERT INTO vm_snapshots (id, tenant_id, vm_id, vm_name, name, namespace, snapshot_uid, phase, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE phase=VALUES(phase), snapshot_uid=VALUES(snapshot_uid)`,
		s.ID, s.TenantID, s.VMID, s.VMName, s.Name, s.Namespace, nullStr(s.SnapshotUID), s.Phase, s.CreatedAt)
}

func (m *MySQL) ListVMSnapshots(tenantID string) []*platform.VMSnapshot {
	rows, err := m.db.Query(`SELECT id, tenant_id, vm_id, vm_name, name, namespace, snapshot_uid, phase, created_at FROM vm_snapshots WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.VMSnapshot
	for rows.Next() {
		var s platform.VMSnapshot
		var uid sql.NullString
		if err := rows.Scan(&s.ID, &s.TenantID, &s.VMID, &s.VMName, &s.Name, &s.Namespace, &uid, &s.Phase, &s.CreatedAt); err != nil {
			continue
		}
		s.SnapshotUID = uid.String
		out = append(out, &s)
	}
	return out
}

func (m *MySQL) GetVMSnapshot(id string) (*platform.VMSnapshot, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, vm_id, vm_name, name, namespace, snapshot_uid, phase, created_at FROM vm_snapshots WHERE id=?`, id)
	var s platform.VMSnapshot
	var uid sql.NullString
	if err := row.Scan(&s.ID, &s.TenantID, &s.VMID, &s.VMName, &s.Name, &s.Namespace, &uid, &s.Phase, &s.CreatedAt); err != nil {
		return nil, false
	}
	s.SnapshotUID = uid.String
	return &s, true
}

func (m *MySQL) DeleteVMSnapshot(id string) {
	_, _ = m.db.Exec(`DELETE FROM vm_snapshots WHERE id=?`, id)
}

func (m *MySQL) SaveVM(vm *platform.PlatformVM) {
	_, _ = m.db.Exec(`INSERT INTO vms (id, tenant_id, vpc_id, name, display_name, namespace, state, error_message,
		cpu, memory_mi, image, template, ip, hypervisor, zone, host_name, service_offering_id, external_uuid, import_source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE display_name=VALUES(display_name), state=VALUES(state), error_message=VALUES(error_message),
		cpu=VALUES(cpu), memory_mi=VALUES(memory_mi), image=VALUES(image), template=VALUES(template), ip=VALUES(ip),
		hypervisor=VALUES(hypervisor), zone=VALUES(zone), host_name=VALUES(host_name),
		service_offering_id=VALUES(service_offering_id), updated_at=VALUES(updated_at)`,
		vm.ID, vm.TenantID, nullStr(vm.VPCID), vm.Name, nullStr(vm.DisplayName), vm.Namespace, vm.State, nullStr(vm.ErrorMsg),
		vm.CPU, vm.MemoryMi, nullStr(vm.Image), nullStr(vm.Template), nullStr(vm.IP), nullStr(vm.Hypervisor),
		nullStr(vm.Zone), nullStr(vm.HostName), nullStr(vm.ServiceOfferingID), nullStr(vm.ExternalUUID), nullStr(vm.ImportSource),
		vm.CreatedAt, nullTime(vm.UpdatedAt))
	m.saveVMNics(vm.ID, vm.NICs)
}

func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

func (m *MySQL) saveVMNics(vmID string, nics []platform.VMNic) {
	_, _ = m.db.Exec(`DELETE FROM vm_nics WHERE vm_id=?`, vmID)
	for _, nic := range nics {
		id := NewID()
		_, _ = m.db.Exec(`INSERT INTO vm_nics (id, vm_id, name, ip, mac, nic_type, network_id, nad_namespace, nad_name)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, vmID, nic.Name, nullStr(nic.IP), nullStr(nic.MAC), nullStr(nic.Type),
			nullStr(nic.NetworkID), nullStr(nic.NADNamespace), nullStr(nic.NADName))
	}
}

func (m *MySQL) loadVMNics(vmID string) []platform.VMNic {
	rows, err := m.db.Query(`SELECT name, ip, mac, nic_type, network_id, nad_namespace, nad_name FROM vm_nics WHERE vm_id=?`, vmID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []platform.VMNic
	for rows.Next() {
		var n platform.VMNic
		var ip, mac, typ, netID, nadNS, nadName sql.NullString
		if err := rows.Scan(&n.Name, &ip, &mac, &typ, &netID, &nadNS, &nadName); err != nil {
			continue
		}
		n.IP = ip.String
		n.MAC = mac.String
		n.Type = typ.String
		n.NetworkID = netID.String
		n.NADNamespace = nadNS.String
		n.NADName = nadName.String
		out = append(out, n)
	}
	return out
}

func scanVM(row *sql.Row) (*platform.PlatformVM, bool) {
	var vm platform.PlatformVM
	var vpcID, display, errMsg, image, tmpl, ip, hyp, zone, host, offering, ext, src sql.NullString
	var updated sql.NullTime
	if err := row.Scan(&vm.ID, &vm.TenantID, &vpcID, &vm.Name, &display, &vm.Namespace, &vm.State, &errMsg,
		&vm.CPU, &vm.MemoryMi, &image, &tmpl, &ip, &hyp, &zone, &host, &offering, &ext, &src, &vm.CreatedAt, &updated); err != nil {
		return nil, false
	}
	vm.VPCID = vpcID.String
	vm.DisplayName = display.String
	vm.ErrorMsg = errMsg.String
	vm.Image = image.String
	vm.Template = tmpl.String
	vm.IP = ip.String
	vm.Hypervisor = hyp.String
	vm.Zone = zone.String
	vm.HostName = host.String
	vm.ServiceOfferingID = offering.String
	vm.ExternalUUID = ext.String
	vm.ImportSource = src.String
	if updated.Valid {
		vm.UpdatedAt = updated.Time
	}
	return &vm, true
}

const vmSelectCols = `id, tenant_id, vpc_id, name, display_name, namespace, state, error_message,
	cpu, memory_mi, image, template, ip, hypervisor, zone, host_name, service_offering_id, external_uuid, import_source, created_at, updated_at`

func (m *MySQL) GetVM(id string) (*platform.PlatformVM, bool) {
	row := m.db.QueryRow(`SELECT `+vmSelectCols+` FROM vms WHERE id=?`, id)
	vm, ok := scanVM(row)
	if !ok {
		return nil, false
	}
	vm.NICs = m.loadVMNics(vm.ID)
	return vm, true
}

func (m *MySQL) GetVMByName(tenantID, name string) (*platform.PlatformVM, bool) {
	row := m.db.QueryRow(`SELECT `+vmSelectCols+` FROM vms WHERE tenant_id=? AND name=?`, tenantID, name)
	vm, ok := scanVM(row)
	if !ok {
		return nil, false
	}
	vm.NICs = m.loadVMNics(vm.ID)
	return vm, true
}

func (m *MySQL) GetVMByExternalUUID(source, externalUUID string) (*platform.PlatformVM, bool) {
	if source == "" || externalUUID == "" {
		return nil, false
	}
	row := m.db.QueryRow(`SELECT `+vmSelectCols+` FROM vms WHERE import_source=? AND external_uuid=?`, source, externalUUID)
	vm, ok := scanVM(row)
	if !ok {
		return nil, false
	}
	vm.NICs = m.loadVMNics(vm.ID)
	return vm, true
}

func (m *MySQL) ListVMs(tenantID string) []*platform.PlatformVM {
	rows, err := m.db.Query(`SELECT `+vmSelectCols+` FROM vms WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.PlatformVM
	for rows.Next() {
		var vm platform.PlatformVM
		var vpcID, display, errMsg, image, tmpl, ip, hyp, zone, host, offering, ext, src sql.NullString
		var updated sql.NullTime
		if err := rows.Scan(&vm.ID, &vm.TenantID, &vpcID, &vm.Name, &display, &vm.Namespace, &vm.State, &errMsg,
			&vm.CPU, &vm.MemoryMi, &image, &tmpl, &ip, &hyp, &zone, &host, &offering, &ext, &src, &vm.CreatedAt, &updated); err != nil {
			continue
		}
		vm.VPCID = vpcID.String
		vm.DisplayName = display.String
		vm.ErrorMsg = errMsg.String
		vm.Image = image.String
		vm.Template = tmpl.String
		vm.IP = ip.String
		vm.Hypervisor = hyp.String
		vm.Zone = zone.String
		vm.HostName = host.String
		vm.ServiceOfferingID = offering.String
		vm.ExternalUUID = ext.String
		vm.ImportSource = src.String
		if updated.Valid {
			vm.UpdatedAt = updated.Time
		}
		vm.NICs = m.loadVMNics(vm.ID)
		out = append(out, &vm)
	}
	return out
}

func (m *MySQL) DeleteVM(id string) {
	_, _ = m.db.Exec(`DELETE FROM vms WHERE id=?`, id)
}

func (m *MySQL) SaveJob(j *platform.AsyncJob) {
	_, _ = m.db.Exec(`INSERT INTO async_jobs (id, tenant_id, job_type, status, payload, result, error_msg, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE status=VALUES(status), result=VALUES(result), error_msg=VALUES(error_msg), updated_at=VALUES(updated_at)`,
		j.ID, j.TenantID, j.Type, j.Status, nullStr(j.Payload), nullStr(j.Result), nullStr(j.Error), j.CreatedAt, j.UpdatedAt)
}

func (m *MySQL) GetJob(id string) (*platform.AsyncJob, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, job_type, status, payload, result, error_msg, created_at, updated_at FROM async_jobs WHERE id=?`, id)
	return scanJob(row)
}

func scanJob(row *sql.Row) (*platform.AsyncJob, bool) {
	var j platform.AsyncJob
	var payload, result, errMsg sql.NullString
	if err := row.Scan(&j.ID, &j.TenantID, &j.Type, &j.Status, &payload, &result, &errMsg, &j.CreatedAt, &j.UpdatedAt); err != nil {
		return nil, false
	}
	j.Payload = payload.String
	j.Result = result.String
	j.Error = errMsg.String
	return &j, true
}

func (m *MySQL) ListJobs(tenantID string) []*platform.AsyncJob {
	q := `SELECT id, tenant_id, job_type, status, payload, result, error_msg, created_at, updated_at FROM async_jobs`
	var rows *sql.Rows
	var err error
	if tenantID == "" {
		rows, err = m.db.Query(q + ` ORDER BY created_at DESC`)
	} else {
		rows, err = m.db.Query(q+` WHERE tenant_id=? ORDER BY created_at DESC`, tenantID)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.AsyncJob
	for rows.Next() {
		var j platform.AsyncJob
		var payload, result, errMsg sql.NullString
		if err := rows.Scan(&j.ID, &j.TenantID, &j.Type, &j.Status, &payload, &result, &errMsg, &j.CreatedAt, &j.UpdatedAt); err != nil {
			continue
		}
		j.Payload = payload.String
		j.Result = result.String
		j.Error = errMsg.String
		out = append(out, &j)
	}
	return out
}

func (m *MySQL) ListPendingJobs(limit int) []*platform.AsyncJob {
	if limit <= 0 {
		limit = 50
	}
	rows, err := m.db.Query(`SELECT id, tenant_id, job_type, status, payload, result, error_msg, created_at, updated_at
		FROM async_jobs WHERE status='pending' ORDER BY created_at ASC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.AsyncJob
	for rows.Next() {
		var j platform.AsyncJob
		var payload, result, errMsg sql.NullString
		if err := rows.Scan(&j.ID, &j.TenantID, &j.Type, &j.Status, &payload, &result, &errMsg, &j.CreatedAt, &j.UpdatedAt); err != nil {
			continue
		}
		j.Payload = payload.String
		j.Result = result.String
		j.Error = errMsg.String
		out = append(out, &j)
	}
	return out
}

func (m *MySQL) SaveServiceOffering(o *platform.ServiceOffering) {
	_, _ = m.db.Exec(`INSERT INTO service_offerings (id, name, display_name, cpu, memory_mi, storage_tags, state, external_uuid, import_source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE display_name=VALUES(display_name), cpu=VALUES(cpu), memory_mi=VALUES(memory_mi),
		storage_tags=VALUES(storage_tags), state=VALUES(state)`,
		o.ID, o.Name, o.DisplayName, o.CPU, o.MemoryMi, nullStr(o.StorageTags), o.State,
		nullStr(o.ExternalUUID), nullStr(o.ImportSource), o.CreatedAt)
}

func (m *MySQL) GetServiceOffering(id string) (*platform.ServiceOffering, bool) {
	row := m.db.QueryRow(`SELECT id, name, display_name, cpu, memory_mi, storage_tags, state, external_uuid, import_source, created_at FROM service_offerings WHERE id=?`, id)
	return scanOffering(row)
}

func scanOffering(row *sql.Row) (*platform.ServiceOffering, bool) {
	var o platform.ServiceOffering
	var tags, ext, src sql.NullString
	if err := row.Scan(&o.ID, &o.Name, &o.DisplayName, &o.CPU, &o.MemoryMi, &tags, &o.State, &ext, &src, &o.CreatedAt); err != nil {
		return nil, false
	}
	o.StorageTags = tags.String
	o.ExternalUUID = ext.String
	o.ImportSource = src.String
	return &o, true
}

func (m *MySQL) ListServiceOfferings(activeOnly bool) []*platform.ServiceOffering {
	q := `SELECT id, name, display_name, cpu, memory_mi, storage_tags, state, external_uuid, import_source, created_at FROM service_offerings`
	if activeOnly {
		q += ` WHERE state='Active'`
	}
	q += ` ORDER BY name`
	rows, err := m.db.Query(q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.ServiceOffering
	for rows.Next() {
		var o platform.ServiceOffering
		var tags, ext, src sql.NullString
		if err := rows.Scan(&o.ID, &o.Name, &o.DisplayName, &o.CPU, &o.MemoryMi, &tags, &o.State, &ext, &src, &o.CreatedAt); err != nil {
			continue
		}
		o.StorageTags = tags.String
		o.ExternalUUID = ext.String
		o.ImportSource = src.String
		out = append(out, &o)
	}
	return out
}

func (m *MySQL) SaveVMTemplate(t *platform.VMTemplate) {
	_, _ = m.db.Exec(`INSERT INTO vm_templates (id, name, display_name, image, os_type, hypervisor, state, external_uuid, import_source, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE display_name=VALUES(display_name), image=VALUES(image), os_type=VALUES(os_type), state=VALUES(state)`,
		t.ID, t.Name, t.DisplayName, t.Image, nullStr(t.OSType), t.Hypervisor, t.State,
		nullStr(t.ExternalUUID), nullStr(t.ImportSource), t.CreatedAt)
}

func (m *MySQL) GetVMTemplate(id string) (*platform.VMTemplate, bool) {
	row := m.db.QueryRow(`SELECT id, name, display_name, image, os_type, hypervisor, state, external_uuid, import_source, created_at FROM vm_templates WHERE id=?`, id)
	return scanTemplate(row)
}

func scanTemplate(row *sql.Row) (*platform.VMTemplate, bool) {
	var t platform.VMTemplate
	var osType, ext, src sql.NullString
	if err := row.Scan(&t.ID, &t.Name, &t.DisplayName, &t.Image, &osType, &t.Hypervisor, &t.State, &ext, &src, &t.CreatedAt); err != nil {
		return nil, false
	}
	t.OSType = osType.String
	t.ExternalUUID = ext.String
	t.ImportSource = src.String
	return &t, true
}

func (m *MySQL) ListVMTemplates(activeOnly bool) []*platform.VMTemplate {
	q := `SELECT id, name, display_name, image, os_type, hypervisor, state, external_uuid, import_source, created_at FROM vm_templates`
	if activeOnly {
		q += ` WHERE state='Active'`
	}
	q += ` ORDER BY name`
	rows, err := m.db.Query(q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.VMTemplate
	for rows.Next() {
		var t platform.VMTemplate
		var osType, ext, src sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.DisplayName, &t.Image, &osType, &t.Hypervisor, &t.State, &ext, &src, &t.CreatedAt); err != nil {
			continue
		}
		t.OSType = osType.String
		t.ExternalUUID = ext.String
		t.ImportSource = src.String
		out = append(out, &t)
	}
	return out
}

func (m *MySQL) SaveSSHKeyPair(k *platform.SSHKeyPair) {
	_, _ = m.db.Exec(`INSERT INTO ssh_key_pairs (id, tenant_id, name, public_key, fingerprint, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), public_key=VALUES(public_key), fingerprint=VALUES(fingerprint)`,
		k.ID, k.TenantID, k.Name, k.PublicKey, k.Fingerprint, k.CreatedAt)
}

func (m *MySQL) GetSSHKeyPair(id string) (*platform.SSHKeyPair, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, name, public_key, fingerprint, created_at FROM ssh_key_pairs WHERE id=?`, id)
	var k platform.SSHKeyPair
	if err := row.Scan(&k.ID, &k.TenantID, &k.Name, &k.PublicKey, &k.Fingerprint, &k.CreatedAt); err != nil {
		return nil, false
	}
	return &k, true
}

func (m *MySQL) ListSSHKeyPairs(tenantID string) []*platform.SSHKeyPair {
	rows, err := m.db.Query(`SELECT id, tenant_id, name, public_key, fingerprint, created_at FROM ssh_key_pairs WHERE tenant_id=? ORDER BY name`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.SSHKeyPair
	for rows.Next() {
		var k platform.SSHKeyPair
		if err := rows.Scan(&k.ID, &k.TenantID, &k.Name, &k.PublicKey, &k.Fingerprint, &k.CreatedAt); err != nil {
			continue
		}
		out = append(out, &k)
	}
	return out
}

func (m *MySQL) DeleteSSHKeyPair(id string) {
	_, _ = m.db.Exec(`DELETE FROM ssh_key_pairs WHERE id=?`, id)
}

func (m *MySQL) SaveAuditEvent(e *platform.AuditEvent) {
	_, _ = m.db.Exec(`INSERT INTO audit_events (id, actor_user_id, actor_role, target_tenant_id, action, method, path, resource_type, resource_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.ActorUserID, e.ActorRole, e.TargetTenantID, e.Action, e.Method, e.Path,
		nullStr(e.ResourceType), nullStr(e.ResourceID), e.CreatedAt)
}

func (m *MySQL) ListAuditEvents(targetTenantID string, limit int) []*platform.AuditEvent {
	if limit <= 0 {
		limit = 100
	}
	rows, err := m.db.Query(`SELECT id, actor_user_id, actor_role, target_tenant_id, action, method, path, resource_type, resource_id, created_at
		FROM audit_events WHERE target_tenant_id=? ORDER BY created_at DESC LIMIT ?`, targetTenantID, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.AuditEvent
	for rows.Next() {
		var e platform.AuditEvent
		var rt, rid sql.NullString
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.ActorRole, &e.TargetTenantID, &e.Action, &e.Method, &e.Path, &rt, &rid, &e.CreatedAt); err != nil {
			continue
		}
		e.ResourceType = rt.String
		e.ResourceID = rid.String
		out = append(out, &e)
	}
	return out
}

func (m *MySQL) AllocateIPAddress(networkID string) (*platform.IPAddress, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`SELECT id, network_id, address, status, vm_nic_id, created_at FROM ip_addresses
		WHERE network_id=? AND status='available' ORDER BY address LIMIT 1 FOR UPDATE`, networkID)
	var ip platform.IPAddress
	var vmNic sql.NullString
	if err := row.Scan(&ip.ID, &ip.NetworkID, &ip.Address, &ip.Status, &vmNic, &ip.CreatedAt); err != nil {
		return nil, fmt.Errorf("no available IP in pool")
	}
	ip.Status = "allocated"
	if _, err := tx.Exec(`UPDATE ip_addresses SET status='allocated' WHERE id=?`, ip.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ip, nil
}

func (m *MySQL) ReleaseIPAddressByVMNic(vmNicID string) {
	_, _ = m.db.Exec(`UPDATE ip_addresses SET status='available', vm_nic_id=NULL WHERE vm_nic_id=?`, vmNicID)
}
