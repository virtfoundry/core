package network

import (
	"context"
	"fmt"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/service/shared"
)

func (s *Service) UpdateVPC(ctx context.Context, tenantID, vpcID, name string, tags *[]string) (*platform.VPC, error) {
	vpc, ok := s.findVPC(tenantID, vpcID)
	if !ok {
		return nil, errVPCNotFound
	}
	if name != "" {
		vpc.Name = name
	}
	if tags != nil {
		vpc.Tags = shared.NormalizeTags(*tags)
	}
	s.store.SaveVPC(vpc)
	return vpc, nil
}

func (s *Service) DeleteVPC(ctx context.Context, tenantID, vpcID string) error {
	vpc, ok := s.findVPC(tenantID, vpcID)
	if !ok {
		return errVPCNotFound
	}
	if vpc.Name == branding.DefaultVPCName {
		return iaerrors.NewBadRequestError("cannot delete the default VPC")
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

func (s *Service) UpdateNetwork(ctx context.Context, tenantID, networkID, name string, tags *[]string) (*platform.Network, error) {
	net, ok := s.store.GetNetwork(networkID)
	if !ok || net.TenantID != tenantID {
		return nil, fmt.Errorf("network not found")
	}
	if name != "" {
		net.Name = name
	}
	if tags != nil {
		net.Tags = shared.NormalizeTags(*tags)
	}
	s.store.SaveNetwork(net)
	return net, nil
}

func (s *Service) DeleteNetwork(ctx context.Context, tenantID, networkID string) error {
	net, ok := s.store.GetNetwork(networkID)
	if !ok || net.TenantID != tenantID {
		return fmt.Errorf("network not found")
	}
	if err := s.refuseProtectedDefaultNetwork(tenantID, net); err != nil {
		return err
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

func (s *Service) refuseProtectedDefaultNetwork(tenantID string, net *platform.Network) error {
	if net.Name != "default" {
		return nil
	}
	vpc, ok := s.findVPC(tenantID, net.VPCID)
	if !ok || vpc.Name != branding.DefaultVPCName {
		return nil
	}
	return iaerrors.NewBadRequestError("cannot delete the default subnet of the default VPC")
}

func (s *Service) UpdateSecurityGroup(ctx context.Context, tenantID, sgID, name string, desc *string, rules []platform.SecurityGroupRule, tags *[]string) (*platform.SecurityGroup, error) {
	sg, ok := s.store.GetSG(sgID)
	if !ok || sg.TenantID != tenantID {
		return nil, fmt.Errorf("security group not found")
	}
	if name != "" {
		sg.Name = name
	}
	if desc != nil {
		sg.Description = *desc
	}
	if rules != nil {
		sg.Rules = rules
	}
	if tags != nil {
		sg.Tags = shared.NormalizeTags(*tags)
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
