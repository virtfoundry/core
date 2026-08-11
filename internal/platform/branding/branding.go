// Package branding holds VirtFoundry naming used across API, worker, and K8s integration.
package branding

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	Domain = "virtfoundry.io"

	ManagedByValue = "virtfoundry"

	LabelManagedBy   = Domain + "/managed-by"
	LabelTenantID    = Domain + "/tenant-id"
	LabelTenantSlug  = Domain + "/tenant-slug"
	LabelVPCID       = Domain + "/vpc-id"
	LabelVPCName     = Domain + "/vpc-name"
	LabelCIDR        = Domain + "/cidr"
	LabelNetworkRole = Domain + "/network-role"
	LabelSGID        = Domain + "/sg-id"
	LabelSGName      = Domain + "/sg-name"
	LabelSG          = Domain + "/sg"
	LabelVM          = Domain + "/vm"
	LabelOS          = Domain + "/os"
	LabelLogSource   = Domain + "/log-source"
	LabelVMSSH       = Domain + "/vm-ssh"

	AppManagedByKey = "app.kubernetes.io/managed-by"

	SystemNamespace       = "virtfoundry-system"
	TenantNamespacePrefix = "virtfoundry-tenant-"
	VPCNamespacePrefix    = "virtfoundry-vpc-"

	ResourceQuotaName  = "virtfoundry-quota"
	BridgeName         = "virtfoundry-br0" // isolated L2; len 15 (IFNAMSIZ max)
	PublicBridgeName   = "vf-pub0"         // public/shared; never use "virtfoundry-pub0" (16 chars)
	MaxBridgeNameLen   = 15               // Linux IFNAMSIZ including NUL
	KubeVirtSecretName = "virtfoundry-kubevirt"

	DefaultRootPassword      = "virtfoundry"
	DefaultTenantSlug        = "default"
	DefaultTenantName        = "Default"
	DefaultSecurityGroupName = "default"
	DefaultVPCName           = "default"
	DefaultVPCCIDR           = "10.0.0.0/16"
	ServiceName              = "virtfoundry-iaas"
	EmailDomain              = "virtfoundry.local"
	DBName                   = "virtfoundry"

	LogLabelVM     = "virtfoundry_vm"
	LogLabelTenant = "virtfoundry_tenant"
)

// ValidateLinuxBridgeName rejects host interface names that exceed IFNAMSIZ.
// Multus/bridge CNI fails with "numerical result out of range" when the name is too long.
func ValidateLinuxBridgeName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("bridge name is empty")
	}
	if utf8.RuneCountInString(name) > MaxBridgeNameLen {
		return fmt.Errorf("bridge name %q exceeds Linux IFNAMSIZ (max %d chars); use e.g. %s", name, MaxBridgeNameLen, PublicBridgeName)
	}
	if name == "virtfoundry-pub0" {
		return fmt.Errorf("bridge name %q is invalid (16 chars); use %s", name, PublicBridgeName)
	}
	return nil
}

// SGPodLabelKey returns the pod label key used by NetworkPolicy selectors for a security group.
func SGPodLabelKey(sgID string) string {
	id := sgID
	if len(id) > 8 {
		id = id[:8]
	}
	return LabelSG + "-" + strings.ToLower(id)
}
