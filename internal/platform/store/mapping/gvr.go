package mapping

import (
	"github.com/virtfoundry/core/internal/platform/branding"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "virtfoundry.io"
	Version = "v1alpha1"

	SystemNamespace = "virtfoundry-system"

	LabelPartOf        = branding.LabelPartOf
	LabelSlug          = "virtfoundry.io/slug"
	LabelRoleID        = "virtfoundry.io/role-id"
	LabelTenant        = branding.LabelTenant
	LabelPlatformOwned = "virtfoundry.io/platform-owned"

	PartOfValue = branding.PartOfValue

	AnnRoleID = "virtfoundry.io/role-id"

	SecretKeyPasswordHash = "password_hash"
	SecretKeyAPIHash      = "secret_hash"
)

var (
	TenantGVR           = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "tenants"}
	UserGVR             = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "users"}
	RoleGVR             = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "roles"}
	VPCGVR              = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "vpcs"}
	SecurityGroupGVR    = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "securitygroups"}
	NetworkGVR          = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "networks"}
	OfferingGVR         = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "offerings"}
	TemplateGVR         = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "templates"}
	InstanceGVR         = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "instances"}
	DiskGVR             = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "disks"}
	DiskSnapshotGVR     = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "disksnapshots"}
	InstanceSnapshotGVR = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "instancesnapshots"}
	IPAddressGVR        = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "ipaddresses"}
	APIKeyGVR           = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "apikeys"}
	SSHKeyGVR           = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "sshkeys"}
	VKSClusterGVR       = schema.GroupVersionResource{Group: Group, Version: Version, Resource: "vksclusters"}
)
