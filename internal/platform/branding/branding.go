// Package branding holds VirtFoundry naming used across API and K8s integration.
package branding

import "strings"

const (
	Domain = "virtfoundry.io"

	ManagedByValue = "virtfoundry"

	LabelManagedBy    = Domain + "/managed-by"
	LabelTenantID     = Domain + "/tenant-id"
	LabelTenantSlug   = Domain + "/tenant-slug"
	LabelVPCID        = Domain + "/vpc-id"
	LabelVPCName      = Domain + "/vpc-name"
	LabelCIDR         = Domain + "/cidr"
	LabelNetworkRole  = Domain + "/network-role"
	LabelSGID         = Domain + "/sg-id"
	LabelSGName       = Domain + "/sg-name"
	LabelSG           = Domain + "/sg"
	LabelVM           = Domain + "/vm"
	LabelOS           = Domain + "/os"
	LabelLogSource    = Domain + "/log-source"
	LabelVMSSH        = Domain + "/vm-ssh"

	AppManagedByKey = "app.kubernetes.io/managed-by"

	SystemNamespace       = "virtfoundry-system"
	TenantNamespacePrefix = "virtfoundry-tenant-"
	VPCNamespacePrefix    = "virtfoundry-vpc-"

	ResourceQuotaName  = "virtfoundry-quota"
	BridgeName         = "virtfoundry-br0"
	KubeVirtSecretName = "virtfoundry-kubevirt"

	DefaultRootPassword     = "virtfoundry"
	DefaultTenantSlug       = "default"
	DefaultTenantName       = "Default"
	DefaultSecurityGroupName = "default"
	DefaultVPCName           = "default"
	DefaultVPCCIDR           = "10.0.0.0/16"
	ServiceName         = "virtfoundry-iaas"
	EmailDomain         = "virtfoundry.local"
	DBName              = "virtfoundry"

	LogLabelVM     = "virtfoundry_vm"
	LogLabelTenant = "virtfoundry_tenant"
)

// SGPodLabelKey returns the pod label key used by NetworkPolicy selectors for a security group.
func SGPodLabelKey(sgID string) string {
	id := sgID
	if len(id) > 8 {
		id = id[:8]
	}
	return LabelSG + "-" + strings.ToLower(id)
}
