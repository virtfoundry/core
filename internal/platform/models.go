package platform

import "time"

type Role string

const (
	RoleRoot        Role = "root"
	RoleTenantAdmin Role = "tenant_admin"
	RoleUser        Role = "user"
)

const (
	SystemRoleRoot           = "platform.root"
	SystemRoleTenantAdmin    = "tenant.admin"
	SystemRoleTenantOperator = "tenant.operator"
	SystemRoleTenantViewer   = "tenant.viewer"
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	RoleID       string    `json:"role_id,omitempty"`
	TenantID     string    `json:"tenant_id,omitempty"`
	Email        string    `json:"email,omitempty"`
	State        string    `json:"state,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type RoleRecord struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsSystem    bool      `json:"is_system"`
	Permissions []string  `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	TenantID   string     `json:"tenant_id,omitempty"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	SecretHash string     `json:"-"`
	Scopes     []string   `json:"scopes,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Tenant struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Namespace    string    `json:"namespace"`
	State        string    `json:"state"`
	ExternalUUID string    `json:"external_uuid,omitempty"`
	ImportSource string    `json:"import_source,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type VPC struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	CIDR      string    `json:"cidr"`
	Namespace string    `json:"namespace"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type SecurityGroup struct {
	ID          string              `json:"id"`
	TenantID    string              `json:"tenant_id"`
	VPCID       string              `json:"vpc_id,omitempty"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Rules       []SecurityGroupRule `json:"rules"`
	CreatedAt   time.Time           `json:"created_at"`
}

type SecurityGroupRule struct {
	Direction string `json:"direction"` // ingress, egress
	Protocol  string `json:"protocol"`  // tcp, udp, icmp, all
	PortFrom  int    `json:"port_from,omitempty"`
	PortTo    int    `json:"port_to,omitempty"`
	CIDR      string `json:"cidr"`
}

type Network struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id,omitempty"`
	VPCID        string    `json:"vpc_id,omitempty"`
	Name         string    `json:"name"`
	NetworkType  string    `json:"network_type,omitempty"` // isolated | shared
	CIDR         string    `json:"cidr"`
	Gateway      string    `json:"gateway,omitempty"`
	NADNamespace string    `json:"nad_namespace,omitempty"`
	NADName      string    `json:"nad_name,omitempty"`
	State        string    `json:"state"`
	ExternalUUID string    `json:"external_uuid,omitempty"`
	ImportSource string    `json:"import_source,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

const (
	NetworkTypeIsolated = "isolated"
	NetworkTypeShared   = "shared"
	SharedNetworkID     = "00000000-0000-4000-8000-000000000001"
)

type AuditEvent struct {
	ID             string    `json:"id"`
	ActorUserID    string    `json:"actor_user_id"`
	ActorRole      string    `json:"actor_role"`
	TargetTenantID string    `json:"target_tenant_id"`
	Action         string    `json:"action"`
	Method         string    `json:"method"`
	Path           string    `json:"path"`
	ResourceType   string    `json:"resource_type,omitempty"`
	ResourceID     string    `json:"resource_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type IPAddress struct {
	ID        string    `json:"id"`
	NetworkID string    `json:"network_id"`
	Address   string    `json:"address"`
	Status    string    `json:"status"` // available | allocated
	VMNicID   string    `json:"vm_nic_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Volume struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	SizeGi    int       `json:"size_gi"`
	Namespace string    `json:"namespace"`
	PVCName   string    `json:"pvc_name"`
	State     string    `json:"state"`
	VMID      string    `json:"vm_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Snapshot struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	VolumeID    string    `json:"volume_id"`
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	SnapshotUID string    `json:"snapshot_uid,omitempty"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
}

type VMSnapshot struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	VMID        string    `json:"vm_id"`
	VMName      string    `json:"vm_name"`
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	SnapshotUID string    `json:"snapshot_uid,omitempty"`
	Phase       string    `json:"phase"`
	CreatedAt   time.Time `json:"created_at"`
}

type PlatformVM struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	VPCID             string    `json:"vpc_id,omitempty"`
	Name              string    `json:"name"`
	DisplayName       string    `json:"display_name,omitempty"`
	Namespace         string    `json:"namespace"`
	State             string    `json:"state"`
	ErrorMsg          string    `json:"error_message,omitempty"`
	CPU               int       `json:"cpu"`
	MemoryMi          int64     `json:"memory_mi"`
	Image             string    `json:"image,omitempty"`
	Template          string    `json:"template,omitempty"`
	IP                string    `json:"ip,omitempty"`
	Hypervisor        string    `json:"hypervisor,omitempty"`
	Zone              string    `json:"zone,omitempty"`
	HostName          string    `json:"host_name,omitempty"`
	ServiceOfferingID string    `json:"service_offering_id,omitempty"`
	TemplateRef       string    `json:"template_ref,omitempty"`
	PowerState        string    `json:"power_state,omitempty"`
	DedicatedCPU      bool      `json:"dedicated_cpu,omitempty"`
	ExternalUUID      string    `json:"external_uuid,omitempty"`
	ImportSource      string    `json:"import_source,omitempty"`
	NICs              []VMNic   `json:"nics,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

type VMNic struct {
	Name         string `json:"name"`
	IP           string `json:"ip,omitempty"`
	MAC          string `json:"mac,omitempty"`
	Type         string `json:"type,omitempty"`
	NetworkID    string `json:"network_id,omitempty"`
	NADNamespace string `json:"nad_namespace,omitempty"`
	NADName      string `json:"nad_name,omitempty"`
}

type ServiceOffering struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	CPU          int       `json:"cpu"`
	MemoryMi     int64     `json:"memory_mi"`
	DedicatedCPU bool      `json:"dedicated_cpu"`
	StorageTags  string    `json:"storage_tags,omitempty"`
	State        string    `json:"state"`
	ExternalUUID string    `json:"external_uuid,omitempty"`
	ImportSource string    `json:"import_source,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type VMTemplate struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id,omitempty"` // empty = platform catalog
	Name              string    `json:"name"`
	DisplayName       string    `json:"display_name"`
	Description       string    `json:"description,omitempty"`
	Image             string    `json:"image"`                 // container disk URL or ISO HTTP URL
	SourceType        string    `json:"source_type,omitempty"` // container, iso
	OSType            string    `json:"os_type,omitempty"`
	CloudInitUserData string    `json:"cloud_init_user_data,omitempty"`
	ISOVolumeID       string    `json:"iso_volume_id,omitempty"`
	ISOSizeGi         int       `json:"iso_size_gi,omitempty"`
	BootDiskSizeGi    int       `json:"boot_disk_size_gi,omitempty"`
	StorageClass      string    `json:"storage_class,omitempty"`
	ImportState       string    `json:"import_state,omitempty"` // pending, importing, ready, failed
	Hypervisor        string    `json:"hypervisor"`
	State             string    `json:"state"`
	ExternalUUID      string    `json:"external_uuid,omitempty"`
	ImportSource      string    `json:"import_source,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type AsyncJob struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Payload   string    `json:"payload,omitempty"`
	Result    string    `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SSHKeyPair struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
}

type TenantQuota struct {
	MaxVMs        int `json:"max_vms"`
	MaxVolumes    int `json:"max_volumes"`
	MaxSnapshots  int `json:"max_snapshots"`
	MaxVPCs       int `json:"max_vpcs"`
	CPULimit      int `json:"cpu_limit"`
	MemoryGiLimit int `json:"memory_gi_limit"`
}
