package store

import "github.com/virtfoundry/core/internal/platform"

// Repository persists platform state. Memory and Kubernetes (CRD) backends implement it.
type Repository interface {
	SaveUser(u *platform.User)
	GetUserByUsername(username string) (*platform.User, bool)
	HasRootUser() bool
	GetUser(id string) (*platform.User, bool)
	ListUsers() []*platform.User
	ListUsersByTenant(tenantID string) []*platform.User
	DeleteUser(id string)

	SaveRole(r *platform.RoleRecord)
	GetRole(id string) (*platform.RoleRecord, bool)
	GetRoleByName(tenantID, name string) (*platform.RoleRecord, bool)
	ListRoles(tenantID string) []*platform.RoleRecord
	DeleteRole(id string)
	GetRolePermissions(roleID string) ([]string, bool)
	SetRolePermissions(roleID string, perms []string)

	SaveAPIKey(k *platform.APIKey)
	GetAPIKey(id string) (*platform.APIKey, bool)
	GetAPIKeyByPrefix(prefix string) (*platform.APIKey, bool)
	ListAPIKeys(userID string) []*platform.APIKey
	ListAPIKeysByTenant(tenantID string) []*platform.APIKey
	DeleteAPIKey(id string)
	TouchAPIKeyLastUsed(id string)

	SeedIAM() error

	SaveTenant(t *platform.Tenant)
	GetTenant(id string) (*platform.Tenant, bool)
	GetTenantBySlug(slug string) (*platform.Tenant, bool)
	ListTenants() []*platform.Tenant
	DeleteTenant(id string)
	PurgeTenantData(tenantID string)

	SaveVPC(v *platform.VPC)
	GetVPC(id string) (*platform.VPC, bool)
	ListVPCs(tenantID string) []*platform.VPC
	DeleteVPC(id string)

	SaveSG(sg *platform.SecurityGroup)
	ListSGs(tenantID string) []*platform.SecurityGroup
	GetSG(id string) (*platform.SecurityGroup, bool)
	DeleteSG(id string)

	SaveNetwork(n *platform.Network)
	ListNetworks(tenantID string) []*platform.Network
	GetSharedNetwork() (*platform.Network, bool)
	GetNetwork(id string) (*platform.Network, bool)
	DeleteNetwork(id string)

	SaveAuditEvent(e *platform.AuditEvent)
	ListAuditEvents(targetTenantID string, limit int) []*platform.AuditEvent

	AllocateIPAddress(networkID string) (*platform.IPAddress, error)
	ReleaseIPAddressByVMNic(vmNicID string)
	ReleaseIPAddressByAddress(networkID, address string)
	SeedIPPool(networkID, start, end string) error

	SaveVolume(v *platform.Volume)
	ListVolumes(tenantID string) []*platform.Volume
	ListVolumesByVMID(tenantID, vmID string) []*platform.Volume
	GetVolume(id string) (*platform.Volume, bool)
	DeleteVolume(id string)

	SaveSnapshot(s *platform.Snapshot)
	ListSnapshots(tenantID string) []*platform.Snapshot

	SaveVMSnapshot(s *platform.VMSnapshot)
	ListVMSnapshots(tenantID string) []*platform.VMSnapshot
	GetVMSnapshot(id string) (*platform.VMSnapshot, bool)
	DeleteVMSnapshot(id string)

	SaveVM(vm *platform.PlatformVM)
	GetVM(id string) (*platform.PlatformVM, bool)
	GetVMByName(tenantID, name string) (*platform.PlatformVM, bool)
	GetVMByExternalUUID(source, externalUUID string) (*platform.PlatformVM, bool)
	ListVMs(tenantID string) []*platform.PlatformVM
	DeleteVM(id string)

	SaveJob(j *platform.AsyncJob)
	GetJob(id string) (*platform.AsyncJob, bool)
	ListJobs(tenantID string) []*platform.AsyncJob
	ListPendingJobs(limit int) []*platform.AsyncJob

	SaveServiceOffering(o *platform.ServiceOffering)
	GetServiceOffering(id string) (*platform.ServiceOffering, bool)
	GetServiceOfferingByName(name string) (*platform.ServiceOffering, bool)
	ListServiceOfferings(activeOnly bool) []*platform.ServiceOffering
	DeleteServiceOffering(id string)

	SaveVMTemplate(t *platform.VMTemplate)
	GetVMTemplate(id string) (*platform.VMTemplate, bool)
	ListVMTemplates(activeOnly bool) []*platform.VMTemplate
	ListVMTemplatesForTenant(tenantID string, activeOnly bool) []*platform.VMTemplate
	DeleteVMTemplate(id string)

	SaveSSHKeyPair(k *platform.SSHKeyPair)
	GetSSHKeyPair(id string) (*platform.SSHKeyPair, bool)
	ListSSHKeyPairs(tenantID string) []*platform.SSHKeyPair
	DeleteSSHKeyPair(id string)

	SaveTargetGroup(tg *platform.TargetGroup)
	GetTargetGroup(id string) (*platform.TargetGroup, bool)
	ListTargetGroups(tenantID string) []*platform.TargetGroup
	DeleteTargetGroup(id string)

	SaveLoadBalancer(lb *platform.LoadBalancer)
	GetLoadBalancer(id string) (*platform.LoadBalancer, bool)
	ListLoadBalancers(tenantID string) []*platform.LoadBalancer
	DeleteLoadBalancer(id string)

	SaveLBListener(l *platform.LBListener)
	GetLBListener(id string) (*platform.LBListener, bool)
	ListLBListeners(loadBalancerID string) []*platform.LBListener
	DeleteLBListener(id string)

	SaveLBTarget(t *platform.LBTarget)
	GetLBTarget(id string) (*platform.LBTarget, bool)
	ListLBTargets(targetGroupID string) []*platform.LBTarget
	DeleteLBTarget(id string)
	DeleteLBTargetsByTargetGroup(targetGroupID string)

	Close() error
}
