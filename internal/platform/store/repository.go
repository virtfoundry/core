package store

import "github.com/virtforge-cloud/virtforge/internal/platform"

// Repository persists platform state. Memory and MySQL both implement it.
type Repository interface {
	SaveUser(u *platform.User)
	GetUserByUsername(username string) (*platform.User, bool)
	HasRootUser() bool
	GetUser(id string) (*platform.User, bool)
	ListUsers() []*platform.User

	SaveTenant(t *platform.Tenant)
	GetTenant(id string) (*platform.Tenant, bool)
	ListTenants() []*platform.Tenant

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
	GetNetwork(id string) (*platform.Network, bool)
	DeleteNetwork(id string)

	SaveVolume(v *platform.Volume)
	ListVolumes(tenantID string) []*platform.Volume
	GetVolume(id string) (*platform.Volume, bool)

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
	ListServiceOfferings(activeOnly bool) []*platform.ServiceOffering

	SaveVMTemplate(t *platform.VMTemplate)
	GetVMTemplate(id string) (*platform.VMTemplate, bool)
	ListVMTemplates(activeOnly bool) []*platform.VMTemplate

	SaveSSHKeyPair(k *platform.SSHKeyPair)
	GetSSHKeyPair(id string) (*platform.SSHKeyPair, bool)
	ListSSHKeyPairs(tenantID string) []*platform.SSHKeyPair
	DeleteSSHKeyPair(id string)

	Close() error
}
