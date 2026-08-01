// Package branding holds VirtForge Cloud naming used across API, worker, and K8s integration.
package branding

const (
	Domain = "virtforge.io"

	ManagedByValue = "virtforge-cloud"

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

	SystemNamespace       = "virtforge-system"
	TenantNamespacePrefix = "virtforge-tenant-"
	VPCNamespacePrefix    = "virtforge-vpc-"

	ResourceQuotaName  = "virtforge-quota"
	BridgeName         = "virtforge-br0"
	KubeVirtSecretName = "virtforge-kubevirt"

	DefaultRootPassword = "virtforge"
	ServiceName         = "virtforge-iaas"
	EmailDomain         = "virtforge.local"
	DBName              = "virtforge"

	LogLabelVM     = "virtforge_vm"
	LogLabelTenant = "virtforge_tenant"
)
