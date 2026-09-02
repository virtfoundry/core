package store

import "github.com/virtfoundry/core/internal/platform"

func (k *Kubernetes) SaveTargetGroup(tg *platform.TargetGroup) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.targetGroups[tg.ID] = tg
}

func (k *Kubernetes) GetTargetGroup(id string) (*platform.TargetGroup, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	tg, ok := k.targetGroups[id]
	return tg, ok
}

func (k *Kubernetes) ListTargetGroups(tenantID string) []*platform.TargetGroup {
	k.mu.RLock()
	defer k.mu.RUnlock()
	var out []*platform.TargetGroup
	for _, tg := range k.targetGroups {
		if tg.TenantID == tenantID {
			out = append(out, tg)
		}
	}
	return out
}

func (k *Kubernetes) DeleteTargetGroup(id string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.targetGroups, id)
}

func (k *Kubernetes) SaveLoadBalancer(lb *platform.LoadBalancer) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.loadBalancers[lb.ID] = lb
}

func (k *Kubernetes) GetLoadBalancer(id string) (*platform.LoadBalancer, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	lb, ok := k.loadBalancers[id]
	return lb, ok
}

func (k *Kubernetes) ListLoadBalancers(tenantID string) []*platform.LoadBalancer {
	k.mu.RLock()
	defer k.mu.RUnlock()
	var out []*platform.LoadBalancer
	for _, lb := range k.loadBalancers {
		if lb.TenantID == tenantID {
			out = append(out, lb)
		}
	}
	return out
}

func (k *Kubernetes) DeleteLoadBalancer(id string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.loadBalancers, id)
}

func (k *Kubernetes) SaveLBListener(l *platform.LBListener) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.lbListeners[l.ID] = l
}

func (k *Kubernetes) GetLBListener(id string) (*platform.LBListener, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	l, ok := k.lbListeners[id]
	return l, ok
}

func (k *Kubernetes) ListLBListeners(loadBalancerID string) []*platform.LBListener {
	k.mu.RLock()
	defer k.mu.RUnlock()
	var out []*platform.LBListener
	for _, l := range k.lbListeners {
		if l.LoadBalancerID == loadBalancerID {
			out = append(out, l)
		}
	}
	return out
}

func (k *Kubernetes) DeleteLBListener(id string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.lbListeners, id)
}

func (k *Kubernetes) SaveLBTarget(t *platform.LBTarget) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.lbTargets[t.ID] = t
}

func (k *Kubernetes) GetLBTarget(id string) (*platform.LBTarget, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	t, ok := k.lbTargets[id]
	return t, ok
}

func (k *Kubernetes) ListLBTargets(targetGroupID string) []*platform.LBTarget {
	k.mu.RLock()
	defer k.mu.RUnlock()
	var out []*platform.LBTarget
	for _, t := range k.lbTargets {
		if t.TargetGroupID == targetGroupID {
			out = append(out, t)
		}
	}
	return out
}

func (k *Kubernetes) DeleteLBTarget(id string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.lbTargets, id)
}

func (k *Kubernetes) DeleteLBTargetsByTargetGroup(targetGroupID string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	for id, t := range k.lbTargets {
		if t.TargetGroupID == targetGroupID {
			delete(k.lbTargets, id)
		}
	}
}
