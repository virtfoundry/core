package store

import (
	"github.com/virtfoundry/core/internal/platform"
)

func (m *Memory) SaveLoadBalancer(lb *platform.LoadBalancer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadBalancers[lb.ID] = lb
}

func (m *Memory) GetLoadBalancer(id string) (*platform.LoadBalancer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	lb, ok := m.loadBalancers[id]
	return lb, ok
}

func (m *Memory) ListLoadBalancers(tenantID string) []*platform.LoadBalancer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.LoadBalancer
	for _, lb := range m.loadBalancers {
		if lb.TenantID == tenantID {
			out = append(out, lb)
		}
	}
	return out
}

func (m *Memory) DeleteLoadBalancer(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for lid, l := range m.lbListeners {
		if l.LoadBalancerID == id {
			delete(m.lbListeners, lid)
		}
	}
	delete(m.loadBalancers, id)
}

func (m *Memory) SaveLBListener(l *platform.LBListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lbListeners[l.ID] = l
}

func (m *Memory) GetLBListener(id string) (*platform.LBListener, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l, ok := m.lbListeners[id]
	return l, ok
}

func (m *Memory) ListLBListeners(loadBalancerID string) []*platform.LBListener {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.LBListener
	for _, l := range m.lbListeners {
		if l.LoadBalancerID == loadBalancerID {
			out = append(out, l)
		}
	}
	return out
}

func (m *Memory) DeleteLBListener(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.lbListeners, id)
}

func (m *Memory) SaveTargetGroup(tg *platform.TargetGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targetGroups[tg.ID] = tg
}

func (m *Memory) GetTargetGroup(id string) (*platform.TargetGroup, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tg, ok := m.targetGroups[id]
	return tg, ok
}

func (m *Memory) ListTargetGroups(tenantID string) []*platform.TargetGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.TargetGroup
	for _, tg := range m.targetGroups {
		if tg.TenantID == tenantID {
			out = append(out, tg)
		}
	}
	return out
}

func (m *Memory) DeleteTargetGroup(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for tid, t := range m.targets {
		if t.TargetGroupID == id {
			delete(m.targets, tid)
		}
	}
	delete(m.targetGroups, id)
}

func (m *Memory) SaveTarget(t *platform.Target) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targets[t.ID] = t
}

func (m *Memory) GetTarget(id string) (*platform.Target, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.targets[id]
	return t, ok
}

func (m *Memory) ListTargets(targetGroupID string) []*platform.Target {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.Target
	for _, t := range m.targets {
		if t.TargetGroupID == targetGroupID {
			out = append(out, t)
		}
	}
	return out
}

func (m *Memory) DeleteTarget(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.targets, id)
}
