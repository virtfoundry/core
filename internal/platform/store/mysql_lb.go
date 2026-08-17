package store

import (
	"github.com/virtfoundry/core/internal/platform"
)

func (m *MySQL) applyMigration007() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS load_balancers (
			id CHAR(36) PRIMARY KEY,
			tenant_id CHAR(36) NOT NULL,
			name VARCHAR(128) NOT NULL,
			description TEXT NULL,
			vip VARCHAR(64) NULL,
			namespace VARCHAR(128) NOT NULL,
			service_name VARCHAR(128) NOT NULL,
			state VARCHAR(32) NOT NULL,
			created_at DATETIME(3) NOT NULL,
			UNIQUE KEY uk_lb_tenant_name (tenant_id, name),
			KEY idx_lb_tenant (tenant_id)
		)`,
		`CREATE TABLE IF NOT EXISTS lb_listeners (
			id CHAR(36) PRIMARY KEY,
			load_balancer_id CHAR(36) NOT NULL,
			protocol VARCHAR(16) NOT NULL,
			port INT NOT NULL,
			target_group_id CHAR(36) NOT NULL,
			created_at DATETIME(3) NOT NULL,
			UNIQUE KEY uk_listener_lb_port (load_balancer_id, port),
			KEY idx_listener_lb (load_balancer_id)
		)`,
		`CREATE TABLE IF NOT EXISTS target_groups (
			id CHAR(36) PRIMARY KEY,
			tenant_id CHAR(36) NOT NULL,
			name VARCHAR(128) NOT NULL,
			protocol VARCHAR(16) NOT NULL,
			port INT NOT NULL,
			state VARCHAR(32) NOT NULL,
			created_at DATETIME(3) NOT NULL,
			UNIQUE KEY uk_tg_tenant_name (tenant_id, name),
			KEY idx_tg_tenant (tenant_id)
		)`,
		`CREATE TABLE IF NOT EXISTS targets (
			id CHAR(36) PRIMARY KEY,
			target_group_id CHAR(36) NOT NULL,
			vm_id CHAR(36) NOT NULL,
			vm_name VARCHAR(128) NULL,
			ip VARCHAR(64) NOT NULL,
			port INT NOT NULL DEFAULT 0,
			state VARCHAR(32) NOT NULL,
			created_at DATETIME(3) NOT NULL,
			UNIQUE KEY uk_target_tg_vm (target_group_id, vm_id),
			KEY idx_target_tg (target_group_id)
		)`,
	}
	for _, s := range stmts {
		if _, err := m.db.Exec(s); err != nil {
			return err
		}
	}
	_, _ = m.db.Exec(`INSERT IGNORE INTO schema_migrations (version) VALUES ('007_load_balancers')`)
	return nil
}

func (m *MySQL) SaveLoadBalancer(lb *platform.LoadBalancer) {
	_, _ = m.db.Exec(`INSERT INTO load_balancers (id, tenant_id, name, description, vip, namespace, service_name, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), description=VALUES(description), vip=VALUES(vip),
			namespace=VALUES(namespace), service_name=VALUES(service_name), state=VALUES(state)`,
		lb.ID, lb.TenantID, lb.Name, nullStr(lb.Description), nullStr(lb.VIP), lb.Namespace, lb.ServiceName, lb.State, lb.CreatedAt)
}

func (m *MySQL) GetLoadBalancer(id string) (*platform.LoadBalancer, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, name, COALESCE(description,''), COALESCE(vip,''), namespace, service_name, state, created_at
		FROM load_balancers WHERE id=?`, id)
	var lb platform.LoadBalancer
	if err := row.Scan(&lb.ID, &lb.TenantID, &lb.Name, &lb.Description, &lb.VIP, &lb.Namespace, &lb.ServiceName, &lb.State, &lb.CreatedAt); err != nil {
		return nil, false
	}
	return &lb, true
}

func (m *MySQL) ListLoadBalancers(tenantID string) []*platform.LoadBalancer {
	rows, err := m.db.Query(`SELECT id, tenant_id, name, COALESCE(description,''), COALESCE(vip,''), namespace, service_name, state, created_at
		FROM load_balancers WHERE tenant_id=? ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.LoadBalancer
	for rows.Next() {
		var lb platform.LoadBalancer
		if err := rows.Scan(&lb.ID, &lb.TenantID, &lb.Name, &lb.Description, &lb.VIP, &lb.Namespace, &lb.ServiceName, &lb.State, &lb.CreatedAt); err != nil {
			continue
		}
		out = append(out, &lb)
	}
	return out
}

func (m *MySQL) DeleteLoadBalancer(id string) {
	_, _ = m.db.Exec(`DELETE FROM lb_listeners WHERE load_balancer_id=?`, id)
	_, _ = m.db.Exec(`DELETE FROM load_balancers WHERE id=?`, id)
}

func (m *MySQL) SaveLBListener(l *platform.LBListener) {
	_, _ = m.db.Exec(`INSERT INTO lb_listeners (id, load_balancer_id, protocol, port, target_group_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE protocol=VALUES(protocol), port=VALUES(port), target_group_id=VALUES(target_group_id)`,
		l.ID, l.LoadBalancerID, l.Protocol, l.Port, l.TargetGroupID, l.CreatedAt)
}

func (m *MySQL) GetLBListener(id string) (*platform.LBListener, bool) {
	row := m.db.QueryRow(`SELECT id, load_balancer_id, protocol, port, target_group_id, created_at FROM lb_listeners WHERE id=?`, id)
	var l platform.LBListener
	if err := row.Scan(&l.ID, &l.LoadBalancerID, &l.Protocol, &l.Port, &l.TargetGroupID, &l.CreatedAt); err != nil {
		return nil, false
	}
	return &l, true
}

func (m *MySQL) ListLBListeners(loadBalancerID string) []*platform.LBListener {
	rows, err := m.db.Query(`SELECT id, load_balancer_id, protocol, port, target_group_id, created_at FROM lb_listeners WHERE load_balancer_id=?`, loadBalancerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.LBListener
	for rows.Next() {
		var l platform.LBListener
		if err := rows.Scan(&l.ID, &l.LoadBalancerID, &l.Protocol, &l.Port, &l.TargetGroupID, &l.CreatedAt); err != nil {
			continue
		}
		out = append(out, &l)
	}
	return out
}

func (m *MySQL) DeleteLBListener(id string) {
	_, _ = m.db.Exec(`DELETE FROM lb_listeners WHERE id=?`, id)
}

func (m *MySQL) SaveTargetGroup(tg *platform.TargetGroup) {
	_, _ = m.db.Exec(`INSERT INTO target_groups (id, tenant_id, name, protocol, port, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), protocol=VALUES(protocol), port=VALUES(port), state=VALUES(state)`,
		tg.ID, tg.TenantID, tg.Name, tg.Protocol, tg.Port, tg.State, tg.CreatedAt)
}

func (m *MySQL) GetTargetGroup(id string) (*platform.TargetGroup, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, name, protocol, port, state, created_at FROM target_groups WHERE id=?`, id)
	var tg platform.TargetGroup
	if err := row.Scan(&tg.ID, &tg.TenantID, &tg.Name, &tg.Protocol, &tg.Port, &tg.State, &tg.CreatedAt); err != nil {
		return nil, false
	}
	return &tg, true
}

func (m *MySQL) ListTargetGroups(tenantID string) []*platform.TargetGroup {
	rows, err := m.db.Query(`SELECT id, tenant_id, name, protocol, port, state, created_at FROM target_groups WHERE tenant_id=? ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.TargetGroup
	for rows.Next() {
		var tg platform.TargetGroup
		if err := rows.Scan(&tg.ID, &tg.TenantID, &tg.Name, &tg.Protocol, &tg.Port, &tg.State, &tg.CreatedAt); err != nil {
			continue
		}
		out = append(out, &tg)
	}
	return out
}

func (m *MySQL) DeleteTargetGroup(id string) {
	_, _ = m.db.Exec(`DELETE FROM targets WHERE target_group_id=?`, id)
	_, _ = m.db.Exec(`DELETE FROM target_groups WHERE id=?`, id)
}

func (m *MySQL) SaveTarget(t *platform.Target) {
	_, _ = m.db.Exec(`INSERT INTO targets (id, target_group_id, vm_id, vm_name, ip, port, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE vm_id=VALUES(vm_id), vm_name=VALUES(vm_name), ip=VALUES(ip), port=VALUES(port), state=VALUES(state)`,
		t.ID, t.TargetGroupID, t.VMID, nullStr(t.VMName), t.IP, t.Port, t.State, t.CreatedAt)
}

func (m *MySQL) GetTarget(id string) (*platform.Target, bool) {
	row := m.db.QueryRow(`SELECT id, target_group_id, vm_id, COALESCE(vm_name,''), ip, port, state, created_at FROM targets WHERE id=?`, id)
	var t platform.Target
	if err := row.Scan(&t.ID, &t.TargetGroupID, &t.VMID, &t.VMName, &t.IP, &t.Port, &t.State, &t.CreatedAt); err != nil {
		return nil, false
	}
	return &t, true
}

func (m *MySQL) ListTargets(targetGroupID string) []*platform.Target {
	rows, err := m.db.Query(`SELECT id, target_group_id, vm_id, COALESCE(vm_name,''), ip, port, state, created_at FROM targets WHERE target_group_id=?`, targetGroupID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.Target
	for rows.Next() {
		var t platform.Target
		if err := rows.Scan(&t.ID, &t.TargetGroupID, &t.VMID, &t.VMName, &t.IP, &t.Port, &t.State, &t.CreatedAt); err != nil {
			continue
		}
		out = append(out, &t)
	}
	return out
}

func (m *MySQL) DeleteTarget(id string) {
	_, _ = m.db.Exec(`DELETE FROM targets WHERE id=?`, id)
}