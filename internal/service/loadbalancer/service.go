package loadbalancer

import (
	"context"
	"fmt"
	"strings"

	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/service/shared"
	corev1 "k8s.io/api/core/v1"
)

const (
	stateActive   = "Active"
	stateCreating = "Creating"
	stateError    = "Error"
)

// Service manages AWS-style L4 load balancers backed by MetalLB Services.
type Service struct {
	store store.Repository
	k8s   *platformk8s.Manager
}

func New(st store.Repository, k8s *platformk8s.Manager) *Service {
	return &Service{store: st, k8s: k8s}
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
	vip, err := s.k8s.EnsureLBService(ctx, ns, svcName, lbID, nil)
	if err != nil {
		lb.State = stateError
		s.store.SaveLoadBalancer(lb)
		return lb, nil
	}
	if vip != "" {
		lb.VIP = vip
		lb.State = stateActive
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
	for _, l := range s.store.ListLBListeners(id) {
		s.store.DeleteLBListener(l.ID)
	}
	if err := s.k8s.DeleteLBService(ctx, lb.Namespace, lb.ServiceName); err != nil {
		return err
	}
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
	if err := s.syncLBService(ctx, lb); err != nil {
		return listener, err
	}
	if err := s.syncLBEndpoints(ctx, lb); err != nil {
		return listener, err
	}
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
	if err := s.syncLBService(ctx, lb); err != nil {
		return err
	}
	return s.syncLBEndpoints(ctx, lb)
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
	if err := s.syncTargetsForTargetGroup(ctx, tenantID, targetGroupID); err != nil {
		return target, err
	}
	return target, nil
}

func (s *Service) syncTargetsForTargetGroup(ctx context.Context, tenantID, targetGroupID string) error {
	for _, lb := range s.store.ListLoadBalancers(tenantID) {
		for _, l := range s.store.ListLBListeners(lb.ID) {
			if l.TargetGroupID == targetGroupID {
				if err := s.syncLBEndpoints(ctx, lb); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Service) syncLBService(ctx context.Context, lb *platform.LoadBalancer) error {
	listeners := s.store.ListLBListeners(lb.ID)
	ports := make([]platformk8s.LBServicePort, 0, len(listeners))
	for _, l := range listeners {
		tg, ok := s.store.GetTargetGroup(l.TargetGroupID)
		if !ok {
			continue
		}
		proto := corev1.ProtocolTCP
		if strings.EqualFold(l.Protocol, "udp") {
			proto = corev1.ProtocolUDP
		}
		ports = append(ports, platformk8s.LBServicePort{
			Name:       fmt.Sprintf("%s-%d", strings.ToLower(l.Protocol), l.Port),
			Port:       int32(l.Port),
			TargetPort: int32(tg.Port),
			Protocol:   proto,
		})
	}
	vip, err := s.k8s.EnsureLBService(ctx, lb.Namespace, lb.ServiceName, lb.ID, ports)
	if err != nil {
		lb.State = stateError
		s.store.SaveLoadBalancer(lb)
		return err
	}
	if vip != "" {
		lb.VIP = vip
	}
	if lb.State != stateError {
		lb.State = stateActive
	}
	s.store.SaveLoadBalancer(lb)
	return nil
}

func (s *Service) syncLBEndpoints(ctx context.Context, lb *platform.LoadBalancer) error {
	listeners := s.store.ListLBListeners(lb.ID)
	if len(listeners) == 0 {
		return s.k8s.SyncLBEndpoints(ctx, lb.Namespace, lb.ServiceName, nil)
	}
	seenIP := make(map[string]struct{})
	var addresses []corev1.EndpointAddress
	portMap := map[string]corev1.EndpointPort{}
	for _, l := range listeners {
		tg, ok := s.store.GetTargetGroup(l.TargetGroupID)
		if !ok {
			continue
		}
		proto := corev1.ProtocolTCP
		if strings.EqualFold(l.Protocol, "udp") {
			proto = corev1.ProtocolUDP
		}
		portName := fmt.Sprintf("%s-%d", strings.ToLower(l.Protocol), l.Port)
		portMap[portName] = corev1.EndpointPort{Name: portName, Port: int32(tg.Port), Protocol: proto}
		for _, t := range s.store.ListLBTargets(l.TargetGroupID) {
			if _, ok := seenIP[t.IP]; ok {
				continue
			}
			seenIP[t.IP] = struct{}{}
			addresses = append(addresses, corev1.EndpointAddress{IP: t.IP})
		}
	}
	if len(addresses) == 0 {
		return s.k8s.SyncLBEndpoints(ctx, lb.Namespace, lb.ServiceName, nil)
	}
	ports := make([]corev1.EndpointPort, 0, len(portMap))
	for _, p := range portMap {
		ports = append(ports, p)
	}
	subsets := []corev1.EndpointSubset{{
		Addresses: addresses,
		Ports:     ports,
	}}
	return s.k8s.SyncLBEndpoints(ctx, lb.Namespace, lb.ServiceName, subsets)
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
