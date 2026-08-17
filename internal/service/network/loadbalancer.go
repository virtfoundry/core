package network

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
	corev1 "k8s.io/api/core/v1"
)

// CreateTargetGroup registers an AWS-style target group (default instance port).
func (s *Service) CreateTargetGroup(tenantID, name, protocol string, port int) (*platform.TargetGroup, error) {
	if _, ok := s.store.GetTenant(tenantID); !ok {
		return nil, fmt.Errorf("tenant not found")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if protocol == "" {
		protocol = "tcp"
	}
	if !strings.EqualFold(protocol, "tcp") {
		return nil, fmt.Errorf("only tcp is supported in v1")
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("port must be 1-65535")
	}
	for _, existing := range s.store.ListTargetGroups(tenantID) {
		if existing.Name == name {
			return nil, fmt.Errorf("target group %q already exists", name)
		}
	}
	tg := &platform.TargetGroup{
		ID: store.NewID(), TenantID: tenantID, Name: name,
		Protocol: strings.ToLower(protocol), Port: port, State: "active",
		CreatedAt: store.Now(),
	}
	s.store.SaveTargetGroup(tg)
	return tg, nil
}

func (s *Service) ListTargetGroups(tenantID string) []*platform.TargetGroup {
	return s.store.ListTargetGroups(tenantID)
}

func (s *Service) GetTargetGroup(tenantID, id string) (*platform.TargetGroup, error) {
	tg, ok := s.store.GetTargetGroup(id)
	if !ok || tg.TenantID != tenantID {
		return nil, fmt.Errorf("target group not found")
	}
	return tg, nil
}

func (s *Service) DeleteTargetGroup(ctx context.Context, tenantID, id string) error {
	tg, err := s.GetTargetGroup(tenantID, id)
	if err != nil {
		return err
	}
	for _, lb := range s.store.ListLoadBalancers(tenantID) {
		for _, l := range s.store.ListLBListeners(lb.ID) {
			if l.TargetGroupID == tg.ID {
				return fmt.Errorf("target group is attached to load balancer %q listener on port %d", lb.Name, l.Port)
			}
		}
	}
	s.store.DeleteTargetGroup(tg.ID)
	return nil
}

// RegisterTarget adds a VM to a target group using a reachable NIC IP (public preferred).
func (s *Service) RegisterTarget(ctx context.Context, tenantID, tgID, vmID string, port int) (*platform.Target, error) {
	tg, err := s.GetTargetGroup(tenantID, tgID)
	if err != nil {
		return nil, err
	}
	vm, ok := s.store.GetVM(vmID)
	if !ok || vm.TenantID != tenantID {
		return nil, fmt.Errorf("vm not found")
	}
	ip := resolveVMTargetIP(vm)
	if ip == "" {
		return nil, fmt.Errorf("vm %q has no reachable NIC IP (attach a public or private Multus network)", vm.Name)
	}
	for _, t := range s.store.ListTargets(tg.ID) {
		if t.VMID == vm.ID {
			return nil, fmt.Errorf("vm already registered in this target group")
		}
	}
	t := &platform.Target{
		ID: store.NewID(), TargetGroupID: tg.ID, VMID: vm.ID, VMName: vm.Name,
		IP: ip, Port: port, State: "healthy", CreatedAt: store.Now(),
	}
	s.store.SaveTarget(t)
	_ = s.resyncLBsUsingTG(ctx, tenantID, tg.ID)
	return t, nil
}

func (s *Service) DeregisterTarget(ctx context.Context, tenantID, tgID, targetID string) error {
	tg, err := s.GetTargetGroup(tenantID, tgID)
	if err != nil {
		return err
	}
	t, ok := s.store.GetTarget(targetID)
	if !ok || t.TargetGroupID != tg.ID {
		return fmt.Errorf("target not found")
	}
	s.store.DeleteTarget(targetID)
	_ = s.resyncLBsUsingTG(ctx, tenantID, tg.ID)
	return nil
}

func (s *Service) ListTargets(tenantID, tgID string) ([]*platform.Target, error) {
	if _, err := s.GetTargetGroup(tenantID, tgID); err != nil {
		return nil, err
	}
	return s.store.ListTargets(tgID), nil
}

// CreateLoadBalancer allocates a MetalLB VIP via Service type LoadBalancer.
func (s *Service) CreateLoadBalancer(ctx context.Context, tenantID, name, description string) (*platform.LoadBalancer, error) {
	if _, ok := s.store.GetTenant(tenantID); !ok {
		return nil, fmt.Errorf("tenant not found")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	for _, existing := range s.store.ListLoadBalancers(tenantID) {
		if existing.Name == name {
			return nil, fmt.Errorf("load balancer %q already exists", name)
		}
	}
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	id := store.NewID()
	svcName := platformk8s.LBServiceName(id)
	lb := &platform.LoadBalancer{
		ID: id, TenantID: tenantID, Name: name, Description: description,
		Namespace: ns, ServiceName: svcName, State: "Creating", CreatedAt: store.Now(),
	}
	s.store.SaveLoadBalancer(lb)

	if _, err := s.k8s.EnsureLBService(ctx, ns, svcName, id, nil); err != nil {
		lb.State = "Error"
		s.store.SaveLoadBalancer(lb)
		return nil, err
	}
	vip, err := s.k8s.WaitLBVIP(ctx, ns, svcName, 60*time.Second)
	if err != nil {
		lb.State = "Error"
		s.store.SaveLoadBalancer(lb)
		return lb, fmt.Errorf("load balancer created but VIP pending: %w", err)
	}
	lb.VIP = vip
	lb.State = "Active"
	s.store.SaveLoadBalancer(lb)
	_ = s.k8s.SyncLBEndpoints(ctx, ns, svcName, id, nil)
	return lb, nil
}

func (s *Service) ListLoadBalancers(tenantID string) []*platform.LoadBalancer {
	return s.store.ListLoadBalancers(tenantID)
}

func (s *Service) GetLoadBalancer(tenantID, id string) (*platform.LoadBalancer, error) {
	lb, ok := s.store.GetLoadBalancer(id)
	if !ok || lb.TenantID != tenantID {
		return nil, fmt.Errorf("load balancer not found")
	}
	return lb, nil
}

func (s *Service) DeleteLoadBalancer(ctx context.Context, tenantID, id string) error {
	lb, err := s.GetLoadBalancer(tenantID, id)
	if err != nil {
		return err
	}
	if err := s.k8s.DeleteLB(ctx, lb.Namespace, lb.ServiceName); err != nil {
		return err
	}
	s.store.DeleteLoadBalancer(lb.ID)
	return nil
}

// CreateLBListener attaches a front-end port to a target group.
func (s *Service) CreateLBListener(ctx context.Context, tenantID, lbID, protocol string, port int, tgID string) (*platform.LBListener, error) {
	lb, err := s.GetLoadBalancer(tenantID, lbID)
	if err != nil {
		return nil, err
	}
	tg, err := s.GetTargetGroup(tenantID, tgID)
	if err != nil {
		return nil, err
	}
	if protocol == "" {
		protocol = "tcp"
	}
	if !strings.EqualFold(protocol, "tcp") {
		return nil, fmt.Errorf("only tcp is supported in v1")
	}
	if port < 1 || port > 65534 {
		return nil, fmt.Errorf("port must be 1-65534")
	}
	for _, existing := range s.store.ListLBListeners(lb.ID) {
		if existing.Port == port {
			return nil, fmt.Errorf("listener on port %d already exists", port)
		}
	}
	l := &platform.LBListener{
		ID: store.NewID(), LoadBalancerID: lb.ID, Protocol: strings.ToLower(protocol),
		Port: port, TargetGroupID: tg.ID, CreatedAt: store.Now(),
	}
	s.store.SaveLBListener(l)
	if err := s.syncLBDataplane(ctx, lb); err != nil {
		s.store.DeleteLBListener(l.ID)
		return nil, err
	}
	return l, nil
}

func (s *Service) ListLBListeners(tenantID, lbID string) ([]*platform.LBListener, error) {
	if _, err := s.GetLoadBalancer(tenantID, lbID); err != nil {
		return nil, err
	}
	return s.store.ListLBListeners(lbID), nil
}

func (s *Service) DeleteLBListener(ctx context.Context, tenantID, lbID, listenerID string) error {
	lb, err := s.GetLoadBalancer(tenantID, lbID)
	if err != nil {
		return err
	}
	l, ok := s.store.GetLBListener(listenerID)
	if !ok || l.LoadBalancerID != lb.ID {
		return fmt.Errorf("listener not found")
	}
	s.store.DeleteLBListener(listenerID)
	return s.syncLBDataplane(ctx, lb)
}

func (s *Service) syncLBDataplane(ctx context.Context, lb *platform.LoadBalancer) error {
	listeners := s.store.ListLBListeners(lb.ID)
	ports := make([]platformk8s.LBPort, 0, len(listeners))
	var endpoints []platformk8s.LBEndpoint
	for _, l := range listeners {
		pname := fmt.Sprintf("p-%d", l.Port)
		tg, ok := s.store.GetTargetGroup(l.TargetGroupID)
		if !ok {
			continue
		}
		ports = append(ports, platformk8s.LBPort{
			Name: pname, Port: int32(l.Port), TargetPort: int32(tg.Port), Protocol: corev1.ProtocolTCP,
		})
		for _, t := range s.store.ListTargets(tg.ID) {
			tp := tg.Port
			if t.Port > 0 {
				tp = t.Port
			}
			endpoints = append(endpoints, platformk8s.LBEndpoint{
				IP: t.IP, Port: int32(tp), Name: pname,
			})
		}
	}
	if _, err := s.k8s.EnsureLBService(ctx, lb.Namespace, lb.ServiceName, lb.ID, ports); err != nil {
		return err
	}
	return s.k8s.SyncLBEndpoints(ctx, lb.Namespace, lb.ServiceName, lb.ID, endpoints)
}

func (s *Service) resyncLBsUsingTG(ctx context.Context, tenantID, tgID string) error {
	for _, lb := range s.store.ListLoadBalancers(tenantID) {
		for _, l := range s.store.ListLBListeners(lb.ID) {
			if l.TargetGroupID == tgID {
				if err := s.syncLBDataplane(ctx, lb); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func resolveVMTargetIP(vm *platform.PlatformVM) string {
	var public, private, any string
	for _, nic := range vm.NICs {
		if nic.IP == "" {
			continue
		}
		any = nic.IP
		name := strings.ToLower(nic.Name)
		if name == "public" || nic.NetworkID == platform.SharedNetworkID || nic.Type == "shared" {
			public = nic.IP
			continue
		}
		if private == "" {
			private = nic.IP
		}
	}
	if public != "" {
		return public
	}
	if private != "" {
		return private
	}
	if vm.IP != "" && !strings.HasPrefix(vm.IP, "10.233.") {
		return vm.IP
	}
	return any
}
