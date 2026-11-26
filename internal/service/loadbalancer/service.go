package loadbalancer

import (
	"context"
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/service/shared"
)

const (
	stateActive   = "Active"
	stateCreating = "Creating"
)

// Service manages AWS-style L4 load balancers. It writes LoadBalancer and
// TargetGroup CRs; the operator reconciles them into a Service and EndpointSlices.
type Service struct {
	store store.Repository
}

func New(st store.Repository) *Service {
	return &Service{store: st}
}

func (s *Service) CreateTargetGroup(tenantID, name, protocol string, port int) (*platform.TargetGroup, error) {
	if name == "" {
		return nil, iaerrors.NewBadRequestError("name is required")
	}
	if port <= 0 {
		return nil, iaerrors.NewBadRequestError("port must be positive")
	}
	proto := strings.ToLower(strings.TrimSpace(protocol))
	if proto == "" {
		proto = "tcp"
	}
	tg := &platform.TargetGroup{
		ID:        store.NewID(),
		TenantID:  tenantID,
		Name:      name,
		Protocol:  proto,
		Port:      port,
		State:     stateActive,
		CreatedAt: store.Now(),
	}
	s.store.SaveTargetGroup(tg)
	return tg, nil
}

func (s *Service) ListTargetGroups(tenantID string) []*platform.TargetGroup {
	return s.store.ListTargetGroups(tenantID)
}

func (s *Service) DeleteTargetGroup(ctx context.Context, tenantID, id string) error {
	tg, ok := s.store.GetTargetGroup(id)
	if !ok || tg.TenantID != tenantID {
		return iaerrors.NewNotFoundError("target group", id)
	}
	s.store.DeleteLBTargetsByTargetGroup(id)
	s.store.DeleteTargetGroup(id)
	return nil
}

func (s *Service) CreateLoadBalancer(ctx context.Context, tenantID, name, description string) (*platform.LoadBalancer, error) {
	if name == "" {
		return nil, iaerrors.NewBadRequestError("name is required")
	}
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	svcName := "lb-" + shared.SanitizeSlug(name)
	if svcName == "" || svcName == "lb-" {
		svcName = "lb-" + store.NewID()[:8]
	}
	lbID := store.NewID()
	lb := &platform.LoadBalancer{
		ID:          lbID,
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Namespace:   ns,
		ServiceName: svcName,
		State:       stateCreating,
		CreatedAt:   store.Now(),
	}
	s.store.SaveLoadBalancer(lb)
	return lb, nil
}

func (s *Service) ListLoadBalancers(tenantID string) []*platform.LoadBalancer {
	return s.store.ListLoadBalancers(tenantID)
}

func (s *Service) DeleteLoadBalancer(ctx context.Context, tenantID, id string) error {
	lb, ok := s.store.GetLoadBalancer(id)
	if !ok || lb.TenantID != tenantID {
		return iaerrors.NewNotFoundError("load balancer", id)
	}
	// Listeners are part of the CR; the operator-owned Service and EndpointSlices
	// are garbage-collected with it.
	s.store.DeleteLoadBalancer(id)
	return nil
}

func (s *Service) CreateListener(ctx context.Context, tenantID, lbID, protocol string, port int, targetGroupID string) (*platform.LBListener, error) {
	if port <= 0 {
		return nil, iaerrors.NewBadRequestError("port must be positive")
	}
	lb, ok := s.store.GetLoadBalancer(lbID)
	if !ok || lb.TenantID != tenantID {
		return nil, iaerrors.NewNotFoundError("load balancer", lbID)
	}
	tg, ok := s.store.GetTargetGroup(targetGroupID)
	if !ok || tg.TenantID != tenantID {
		return nil, iaerrors.NewNotFoundError("target group", targetGroupID)
	}
	proto := strings.ToLower(strings.TrimSpace(protocol))
	if proto == "" {
		proto = tg.Protocol
	}
	listener := &platform.LBListener{
		ID:             store.NewID(),
		TenantID:       tenantID,
		LoadBalancerID: lbID,
		Protocol:       proto,
		Port:           port,
		TargetGroupID:  targetGroupID,
		CreatedAt:      store.Now(),
	}
	s.store.SaveLBListener(listener)
	return listener, nil
}

func (s *Service) DeleteListener(ctx context.Context, tenantID, lbID, listenerID string) error {
	lb, ok := s.store.GetLoadBalancer(lbID)
	if !ok || lb.TenantID != tenantID {
		return iaerrors.NewNotFoundError("load balancer", lbID)
	}
	listener, ok := s.store.GetLBListener(listenerID)
	if !ok || listener.LoadBalancerID != lbID || listener.TenantID != tenantID {
		return iaerrors.NewNotFoundError("listener", listenerID)
	}
	s.store.DeleteLBListener(listenerID)
	return nil
}

func (s *Service) RegisterTarget(ctx context.Context, tenantID, targetGroupID, vmID string) (*platform.LBTarget, error) {
	tg, ok := s.store.GetTargetGroup(targetGroupID)
	if !ok || tg.TenantID != tenantID {
		return nil, iaerrors.NewNotFoundError("target group", targetGroupID)
	}
	vm, ok := s.store.GetVM(vmID)
	if !ok || vm.TenantID != tenantID {
		return nil, iaerrors.NewNotFoundError("vm", vmID)
	}
	ip := resolveVMIP(vm)
	if ip == "" {
		return nil, iaerrors.NewBadRequestError("vm has no reachable IP")
	}
	target := &platform.LBTarget{
		ID:            store.NewID(),
		TenantID:      tenantID,
		TargetGroupID: targetGroupID,
		VMID:          vmID,
		IP:            ip,
		Port:          tg.Port,
		State:         "registered",
		CreatedAt:     store.Now(),
	}
	s.store.SaveLBTarget(target)
	return target, nil
}

func resolveVMIP(vm *platform.PlatformVM) string {
	if vm.IP != "" && !strings.HasPrefix(vm.IP, "10.233.") {
		return vm.IP
	}
	for _, nic := range vm.NICs {
		if nic.IP != "" {
			return nic.IP
		}
	}
	return vm.IP
}
