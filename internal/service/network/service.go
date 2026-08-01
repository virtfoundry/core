package network

import (
	"context"
	"errors"
	"fmt"

	platformk8s "github.com/virtforge-cloud/virtforge/internal/platform/k8s"
	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/branding"
	cidrutil "github.com/virtforge-cloud/virtforge/internal/platform/cidr"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service/shared"
)

var errVPCNotFound = errors.New("vpc not found")

// Service manages VPCs, private networks and security groups.
type Service struct {
	store          store.Repository
	k8s            *platformk8s.Manager
	isolatedBridge string
}

func New(st store.Repository, k8s *platformk8s.Manager) *Service {
	return &Service{store: st, k8s: k8s, isolatedBridge: branding.BridgeName}
}

func (s *Service) ConfigureBridges(isolated string) {
	if isolated != "" {
		s.isolatedBridge = isolated
	}
}

func (s *Service) CreateVPC(ctx context.Context, tenantID, name, vpcCIDR string) (*platform.VPC, *platform.Network, error) {
	if _, ok := s.store.GetTenant(tenantID); !ok {
		return nil, nil, fmt.Errorf("tenant not found")
	}
	var existing []string
	for _, v := range s.store.ListVPCs(tenantID) {
		existing = append(existing, v.CIDR)
	}
	if vpcCIDR == "" {
		plan := cidrutil.PlanVPC(existing)
		for _, s := range plan.Suggestions {
			if s.Available {
				vpcCIDR = s.CIDR
				break
			}
		}
		if vpcCIDR == "" {
			vpcCIDR = "10.0.0.0/16"
		}
	}
	if err := cidrutil.ValidateVPC(vpcCIDR, existing); err != nil {
		return nil, nil, err
	}

	tenantNS, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, nil, err
	}

	vpcID := store.NewID()
	vpcSlug := shared.SanitizeSlug(name)
	if vpcSlug == "" {
		return nil, nil, fmt.Errorf("invalid vpc name")
	}

	defaultNetCIDR, err := cidrutil.AllocateSubnet(vpcCIDR, nil, 24)
	if err != nil {
		return nil, nil, err
	}

	nadName := fmt.Sprintf("%s-default", vpcSlug)
	labels := platformk8s.NADLabels{
		platformk8s.LabelTenantID: tenantID,
		platformk8s.LabelVPCID:    vpcID,
		branding.LabelVPCName:     name,
		branding.LabelNetworkRole: "default",
	}
	if err := s.k8s.CreateNetworkAttachment(ctx, tenantNS, nadName, defaultNetCIDR, s.isolatedBridge, labels); err != nil {
		return nil, nil, err
	}

	vpc := &platform.VPC{
		ID: vpcID, TenantID: tenantID, Name: name, CIDR: vpcCIDR,
		Namespace: tenantNS, State: "active", CreatedAt: store.Now(),
	}
	s.store.SaveVPC(vpc)

	defNet := &platform.Network{
		ID: store.NewID(), TenantID: tenantID, VPCID: vpcID,
		Name: "default", CIDR: defaultNetCIDR, NetworkType: platform.NetworkTypeIsolated,
		NADNamespace: tenantNS, NADName: nadName,
		State: "active", CreatedAt: store.Now(),
	}
	s.store.SaveNetwork(defNet)
	return vpc, defNet, nil
}

func (s *Service) ListVPCs(tenantID string) []*platform.VPC {
	return s.store.ListVPCs(tenantID)
}

func (s *Service) ListNetworksByVPC(tenantID, vpcID string) []*platform.Network {
	var out []*platform.Network
	for _, n := range s.store.ListNetworks(tenantID) {
		if n.VPCID == vpcID {
			out = append(out, n)
		}
	}
	return out
}

func (s *Service) CreateSecurityGroup(ctx context.Context, tenantID, vpcID, name, desc string, rules []platform.SecurityGroupRule) (*platform.SecurityGroup, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}

	sg := &platform.SecurityGroup{
		ID: store.NewID(), TenantID: tenantID, VPCID: vpcID,
		Name: name, Description: desc, Rules: rules, CreatedAt: store.Now(),
	}
	if len(sg.Rules) == 0 {
		sg.Rules = []platform.SecurityGroupRule{{Direction: "ingress", Protocol: "tcp", PortFrom: 22, CIDR: "0.0.0.0/0"}}
	}
	if err := s.k8s.ApplySecurityGroup(ctx, ns, sg); err != nil {
		return nil, err
	}
	s.store.SaveSG(sg)
	return sg, nil
}

func (s *Service) ListSecurityGroups(tenantID string) []*platform.SecurityGroup {
	return s.store.ListSGs(tenantID)
}

func (s *Service) AddSGRules(ctx context.Context, tenantID, sgID string, rules []platform.SecurityGroupRule) (*platform.SecurityGroup, error) {
	sg, ok := s.store.GetSG(sgID)
	if !ok || sg.TenantID != tenantID {
		return nil, fmt.Errorf("security group not found")
	}
	sg.Rules = append(sg.Rules, rules...)
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.k8s.ApplySecurityGroup(ctx, ns, sg); err != nil {
		return nil, err
	}
	s.store.SaveSG(sg)
	return sg, nil
}

func (s *Service) CreateNetwork(ctx context.Context, tenantID, vpcID, name, subnetCIDR string, prefix int) (*platform.Network, error) {
	if vpcID == "" {
		return nil, fmt.Errorf("vpc_id is required")
	}
	vpc, ok := s.findVPC(tenantID, vpcID)
	if !ok {
		return nil, fmt.Errorf("vpc not found")
	}

	tenantNS, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}

	existing := s.ListNetworksByVPC(tenantID, vpcID)
	var used []string
	for _, n := range existing {
		used = append(used, n.CIDR)
	}

	if prefix <= 0 {
		prefix = 24
	}
	if subnetCIDR == "" {
		subnetCIDR, err = cidrutil.AllocateSubnet(vpc.CIDR, used, prefix)
		if err != nil {
			return nil, err
		}
	}
	if err := cidrutil.ValidateSubnet(vpc.CIDR, subnetCIDR, used); err != nil {
		return nil, err
	}

	vpcSlug := shared.SanitizeSlug(vpc.Name)
	netSlug := shared.SanitizeSlug(name)
	if netSlug == "" {
		return nil, fmt.Errorf("invalid network name")
	}
	nadName := fmt.Sprintf("%s-%s", vpcSlug, netSlug)

	labels := platformk8s.NADLabels{
		platformk8s.LabelTenantID: tenantID,
		platformk8s.LabelVPCID:    vpcID,
		branding.LabelVPCName: vpc.Name,
	}
	if err := s.k8s.CreateNetworkAttachment(ctx, tenantNS, nadName, subnetCIDR, s.isolatedBridge, labels); err != nil {
		return nil, err
	}

	net := &platform.Network{
		ID: store.NewID(), TenantID: tenantID, VPCID: vpcID,
		Name: name, CIDR: subnetCIDR, NetworkType: platform.NetworkTypeIsolated,
		NADNamespace: tenantNS, NADName: nadName,
		State: "active", CreatedAt: store.Now(),
	}
	s.store.SaveNetwork(net)
	return net, nil
}

func (s *Service) ListNetworks(tenantID string) []*platform.Network {
	return s.store.ListNetworks(tenantID)
}

func (s *Service) findVPC(tenantID, vpcID string) (*platform.VPC, bool) {
	for _, v := range s.store.ListVPCs(tenantID) {
		if v.ID == vpcID {
			return v, true
		}
	}
	return nil, false
}
