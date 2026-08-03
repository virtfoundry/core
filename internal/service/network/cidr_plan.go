package network

import (
	cidrutil "github.com/virtfoundry/core/internal/platform/cidr"
)

func (s *Service) PlanVPCCIDRs(tenantID string) cidrutil.VPCPlan {
	var existing []string
	for _, v := range s.store.ListVPCs(tenantID) {
		existing = append(existing, v.CIDR)
	}
	return cidrutil.PlanVPC(existing)
}

func (s *Service) PlanSubnetCIDRs(tenantID, vpcID string, prefix int) (cidrutil.SubnetPlan, error) {
	vpc, ok := s.findVPC(tenantID, vpcID)
	if !ok {
		return cidrutil.SubnetPlan{}, errVPCNotFound
	}
	var used []string
	for _, n := range s.ListNetworksByVPC(tenantID, vpcID) {
		used = append(used, n.CIDR)
	}
	return cidrutil.PlanSubnet(vpc.CIDR, used, prefix)
}
