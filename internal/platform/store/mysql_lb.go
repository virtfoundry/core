package store

import (
	"database/sql"

	"github.com/virtfoundry/core/internal/platform"
)

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
	rows, err := m.db.Query(`SELECT id, tenant_id, name, protocol, port, state, created_at FROM target_groups WHERE tenant_id=?`, tenantID)
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
	_, _ = m.db.Exec(`DELETE FROM target_groups WHERE id=?`, id)
}

func (m *MySQL) SaveLoadBalancer(lb *platform.LoadBalancer) {
	_, _ = m.db.Exec(`INSERT INTO load_balancers (id, tenant_id, name, description, namespace, service_name, vip, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE name=VALUES(name), description=VALUES(description), namespace=VALUES(namespace),
		service_name=VALUES(service_name), vip=VALUES(vip), state=VALUES(state)`,
		lb.ID, lb.TenantID, lb.Name, lb.Description, lb.Namespace, lb.ServiceName, lb.VIP, lb.State, lb.CreatedAt)
}

func (m *MySQL) GetLoadBalancer(id string) (*platform.LoadBalancer, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, name, description, namespace, service_name, vip, state, created_at FROM load_balancers WHERE id=?`, id)
	var lb platform.LoadBalancer
	var desc, vip sql.NullString
	if err := row.Scan(&lb.ID, &lb.TenantID, &lb.Name, &desc, &lb.Namespace, &lb.ServiceName, &vip, &lb.State, &lb.CreatedAt); err != nil {
		return nil, false
	}
	lb.Description = desc.String
	lb.VIP = vip.String
	return &lb, true
}

func (m *MySQL) ListLoadBalancers(tenantID string) []*platform.LoadBalancer {
	rows, err := m.db.Query(`SELECT id, tenant_id, name, description, namespace, service_name, vip, state, created_at FROM load_balancers WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.LoadBalancer
	for rows.Next() {
		var lb platform.LoadBalancer
		var desc, vip sql.NullString
		if err := rows.Scan(&lb.ID, &lb.TenantID, &lb.Name, &desc, &lb.Namespace, &lb.ServiceName, &vip, &lb.State, &lb.CreatedAt); err != nil {
			continue
		}
		lb.Description = desc.String
		lb.VIP = vip.String
		out = append(out, &lb)
	}
	return out
}

func (m *MySQL) DeleteLoadBalancer(id string) {
	_, _ = m.db.Exec(`DELETE FROM load_balancers WHERE id=?`, id)
}

func (m *MySQL) SaveLBListener(l *platform.LBListener) {
	_, _ = m.db.Exec(`INSERT INTO lb_listeners (id, tenant_id, load_balancer_id, protocol, port, target_group_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE protocol=VALUES(protocol), port=VALUES(port), target_group_id=VALUES(target_group_id)`,
		l.ID, l.TenantID, l.LoadBalancerID, l.Protocol, l.Port, l.TargetGroupID, l.CreatedAt)
}

func (m *MySQL) GetLBListener(id string) (*platform.LBListener, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, load_balancer_id, protocol, port, target_group_id, created_at FROM lb_listeners WHERE id=?`, id)
	var l platform.LBListener
	if err := row.Scan(&l.ID, &l.TenantID, &l.LoadBalancerID, &l.Protocol, &l.Port, &l.TargetGroupID, &l.CreatedAt); err != nil {
		return nil, false
	}
	return &l, true
}

func (m *MySQL) ListLBListeners(loadBalancerID string) []*platform.LBListener {
	rows, err := m.db.Query(`SELECT id, tenant_id, load_balancer_id, protocol, port, target_group_id, created_at FROM lb_listeners WHERE load_balancer_id=?`, loadBalancerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.LBListener
	for rows.Next() {
		var l platform.LBListener
		if err := rows.Scan(&l.ID, &l.TenantID, &l.LoadBalancerID, &l.Protocol, &l.Port, &l.TargetGroupID, &l.CreatedAt); err != nil {
			continue
		}
		out = append(out, &l)
	}
	return out
}

func (m *MySQL) DeleteLBListener(id string) {
	_, _ = m.db.Exec(`DELETE FROM lb_listeners WHERE id=?`, id)
}

func (m *MySQL) SaveLBTarget(t *platform.LBTarget) {
	_, _ = m.db.Exec(`INSERT INTO lb_targets (id, tenant_id, target_group_id, vm_id, ip, port, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE ip=VALUES(ip), port=VALUES(port), state=VALUES(state)`,
		t.ID, t.TenantID, t.TargetGroupID, t.VMID, t.IP, t.Port, t.State, t.CreatedAt)
}

func (m *MySQL) GetLBTarget(id string) (*platform.LBTarget, bool) {
	row := m.db.QueryRow(`SELECT id, tenant_id, target_group_id, vm_id, ip, port, state, created_at FROM lb_targets WHERE id=?`, id)
	var t platform.LBTarget
	if err := row.Scan(&t.ID, &t.TenantID, &t.TargetGroupID, &t.VMID, &t.IP, &t.Port, &t.State, &t.CreatedAt); err != nil {
		return nil, false
	}
	return &t, true
}

func (m *MySQL) ListLBTargets(targetGroupID string) []*platform.LBTarget {
	rows, err := m.db.Query(`SELECT id, tenant_id, target_group_id, vm_id, ip, port, state, created_at FROM lb_targets WHERE target_group_id=?`, targetGroupID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*platform.LBTarget
	for rows.Next() {
		var t platform.LBTarget
		if err := rows.Scan(&t.ID, &t.TenantID, &t.TargetGroupID, &t.VMID, &t.IP, &t.Port, &t.State, &t.CreatedAt); err != nil {
			continue
		}
		out = append(out, &t)
	}
	return out
}

func (m *MySQL) DeleteLBTarget(id string) {
	_, _ = m.db.Exec(`DELETE FROM lb_targets WHERE id=?`, id)
}

func (m *MySQL) DeleteLBTargetsByTargetGroup(targetGroupID string) {
	_, _ = m.db.Exec(`DELETE FROM lb_targets WHERE target_group_id=?`, targetGroupID)
}
