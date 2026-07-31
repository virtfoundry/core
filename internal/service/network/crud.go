package network

import (
	"context"
	"fmt"

	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/service/shared"
)

func (s *Service) UpdateVPC(ctx context.Context, tenantID, vpcID, name string) (*platform.VPC, error) {
	vpc, ok := s.findVPC(tenantID, vpcID)
	if !ok {
		return nil, errVPCNotFound
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	vpc.Name = name
	s.store.SaveVPC(vpc)
	return vpc, nil
}

func (s *Service) DeleteVPC(ctx context.Context, tenantID, vpcID string) error {
	vpc, ok := s.findVPC(tenantID, vpcID)
	if !ok {
		return errVPCNotFound
	}
	nets := s.ListNetworksByVPC(tenantID, vpcID)
	for _, net := range nets {
		if s.isNetworkInUse(tenantID, net.ID) {
			return fmt.Errorf("cannot delete vpc: network %s is in use by a virtual machine", net.Name)
		}
	}
	for _, net := range nets {
		if err := s.deleteNetworkResources(ctx, net); err != nil {
			return err
		}
	}
	s.store.DeleteVPC(vpc.ID)
	return nil
}

func (s *Service) UpdateNetwork(ctx context.Context, tenantID, networkID, name string) (*platform.Network, error) {
	net, ok := s.store.GetNetwork(networkID)
	if !ok || net.TenantID != tenantID {
		return nil, fmt.Errorf("network not found")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	net.Name = name
	s.store.SaveNetwork(net)
	return net, nil
}

func (s *Service) DeleteNetwork(ctx context.Context, tenantID, networkID string) error {
	net, ok := s.store.GetNetwork(networkID)
	if !ok || net.TenantID != tenantID {
		return fmt.Errorf("network not found")
	}
	if s.isNetworkInUse(tenantID, networkID) {
		return fmt.Errorf("network is in use by a virtual machine")
	}
	if err := s.deleteNetworkResources(ctx, net); err != nil {
		return err
	}
	s.store.DeleteNetwork(networkID)
	return nil
}

func (s *Service) UpdateSecurityGroup(ctx context.Context, tenantID, sgID, name, desc string, rules []platform.SecurityGroupRule) (*platform.SecurityGroup, error) {
	sg, ok := s.store.GetSG(sgID)
	if !ok || sg.TenantID != tenantID {
		return nil, fmt.Errorf("security group not found")
	}
	if name != "" {
		sg.Name = name
	}
	sg.Description = desc
	if rules != nil {
		sg.Rules = rules
	}
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

func (s *Service) DeleteSecurityGroup(ctx context.Context, tenantID, sgID string) error {
	sg, ok := s.store.GetSG(sgID)
	if !ok || sg.TenantID != tenantID {
		return fmt.Errorf("security group not found")
	}
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return err
	}
	if err := s.k8s.DeleteSecurityGroup(ctx, ns, sg.ID); err != nil {
		return err
	}
	s.store.DeleteSG(sgID)
	return nil
}

func (s *Service) deleteNetworkResources(ctx context.Context, net *platform.Network) error {
	if net.NADNamespace != "" && net.NADName != "" {
		if err := s.k8s.DeleteNetworkAttachment(ctx, net.NADNamespace, net.NADName); err != nil {
			return err
		}
	}
	s.store.DeleteNetwork(net.ID)
	return nil
}

func (s *Service) isNetworkInUse(tenantID, networkID string) bool {
	for _, vm := range s.store.ListVMs(tenantID) {
		for _, nic := range vm.NICs {
			if nic.NetworkID == networkID {
				return true
			}
		}
	}
	return false
}
