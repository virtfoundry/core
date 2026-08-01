package service

import (
	"context"
	"io"

	"github.com/virtforge-cloud/virtforge/internal/auth"
	"github.com/virtforge-cloud/virtforge/internal/config"
	"github.com/virtforge-cloud/virtforge/internal/infra/hypervisor"
	"github.com/virtforge-cloud/virtforge/internal/platform"
	cidrutil "github.com/virtforge-cloud/virtforge/internal/platform/cidr"
	"github.com/virtforge-cloud/virtforge/internal/platform/branding"
	platformk8s "github.com/virtforge-cloud/virtforge/internal/platform/k8s"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service/compute"
	"github.com/virtforge-cloud/virtforge/internal/service/identity"
	"github.com/virtforge-cloud/virtforge/internal/service/jobs"
	"github.com/virtforge-cloud/virtforge/internal/service/network"
	"github.com/virtforge-cloud/virtforge/internal/service/shared"
	"github.com/virtforge-cloud/virtforge/internal/service/sshkeys"
	"github.com/virtforge-cloud/virtforge/internal/service/storage"
	"github.com/virtforge-cloud/virtforge/internal/service/tenant"
)

// EventBroadcaster pushes realtime events to WebSocket clients.
type EventBroadcaster = shared.EventBroadcaster

// PlatformService is the facade over domain services. Handlers depend on this type.
type PlatformService struct {
	tenant   *tenant.Service
	identity *identity.Service
	network  *network.Service
	storage  *storage.Service
	compute  *compute.Service
	jobs     *jobs.Service
	sshkeys  *sshkeys.Service
}

func NewPlatformService(st store.Repository, k8s *platformk8s.Manager, kv *hypervisor.KubeVirtDriver, hub EventBroadcaster) *PlatformService {
	computeSvc := compute.New(st, k8s, kv, hub)
	return &PlatformService{
		tenant:   tenant.New(st, k8s),
		identity: identity.New(st),
		network:  network.New(st, k8s),
		storage:  storage.New(st, k8s),
		compute:  computeSvc,
		jobs:     jobs.New(st, computeSvc),
		sshkeys:  sshkeys.New(st),
	}
}

// --- identity ---

func (s *PlatformService) BootstrapRoot(username, password string) (*platform.User, error) {
	return s.identity.BootstrapRoot(username, password)
}

func (s *PlatformService) BootstrapRootDefaultTenant(ctx context.Context) (*platform.Tenant, error) {
	tenant, err := s.tenant.EnsureTenant(ctx, branding.DefaultTenantName, branding.DefaultTenantSlug)
	if err != nil {
		return nil, err
	}
	s.identity.LinkRootToTenant(tenant.ID)
	return tenant, nil
}

func (s *PlatformService) ResolveTenantID(claims *auth.Claims, requestedTenant string) (string, error) {
	return s.identity.ResolveTenantID(claims, requestedTenant)
}

func (s *PlatformService) BootstrapNetworking(ctx context.Context, cfg config.NetworkingConfig) error {
	s.network.ConfigureBridges(cfg.Isolated.BridgeName)
	s.compute.ConfigureVMNetworking(cfg.VM.DefaultNetwork, cfg.VM.AllowPodNetwork)
	return s.network.BootstrapSharedNetwork(ctx, cfg.Public)
}

// --- tenant ---

func (s *PlatformService) CreateTenant(ctx context.Context, name, slug, adminPassword string) (*platform.Tenant, *platform.User, error) {
	return s.tenant.CreateTenant(ctx, name, slug, adminPassword)
}

func (s *PlatformService) ListTenants() []*platform.Tenant {
	return s.tenant.ListTenants()
}

func (s *PlatformService) GetTenant(id string) (*platform.Tenant, bool) {
	return s.tenant.GetTenant(id)
}

// --- network ---

func (s *PlatformService) CreateVPC(ctx context.Context, tenantID, name, cidr string) (*platform.VPC, error) {
	vpc, _, err := s.network.CreateVPC(ctx, tenantID, name, cidr)
	return vpc, err
}

func (s *PlatformService) CreateVPCWithDefaultNet(ctx context.Context, tenantID, name, cidr string) (*platform.VPC, *platform.Network, error) {
	return s.network.CreateVPC(ctx, tenantID, name, cidr)
}

func (s *PlatformService) ListVPCs(tenantID string) []*platform.VPC {
	return s.network.ListVPCs(tenantID)
}

func (s *PlatformService) UpdateVPC(ctx context.Context, tenantID, vpcID, name string) (*platform.VPC, error) {
	return s.network.UpdateVPC(ctx, tenantID, vpcID, name)
}

func (s *PlatformService) DeleteVPC(ctx context.Context, tenantID, vpcID string) error {
	return s.network.DeleteVPC(ctx, tenantID, vpcID)
}

func (s *PlatformService) CreateSecurityGroup(ctx context.Context, tenantID, vpcID, name, desc string, rules []platform.SecurityGroupRule) (*platform.SecurityGroup, error) {
	return s.network.CreateSecurityGroup(ctx, tenantID, vpcID, name, desc, rules)
}

func (s *PlatformService) ListSecurityGroups(tenantID string) []*platform.SecurityGroup {
	return s.network.ListSecurityGroups(tenantID)
}

func (s *PlatformService) UpdateSecurityGroup(ctx context.Context, tenantID, sgID, name, desc string, rules []platform.SecurityGroupRule) (*platform.SecurityGroup, error) {
	return s.network.UpdateSecurityGroup(ctx, tenantID, sgID, name, desc, rules)
}

func (s *PlatformService) DeleteSecurityGroup(ctx context.Context, tenantID, sgID string) error {
	return s.network.DeleteSecurityGroup(ctx, tenantID, sgID)
}

func (s *PlatformService) AddSGRules(ctx context.Context, tenantID, sgID string, rules []platform.SecurityGroupRule) (*platform.SecurityGroup, error) {
	return s.network.AddSGRules(ctx, tenantID, sgID, rules)
}

func (s *PlatformService) PlanVPCCIDRs(tenantID string) cidrutil.VPCPlan {
	return s.network.PlanVPCCIDRs(tenantID)
}

func (s *PlatformService) PlanSubnetCIDRs(tenantID, vpcID string, prefix int) (cidrutil.SubnetPlan, error) {
	return s.network.PlanSubnetCIDRs(tenantID, vpcID, prefix)
}

func (s *PlatformService) CreateNetwork(ctx context.Context, tenantID, vpcID, name, cidr string, prefix int) (*platform.Network, error) {
	return s.network.CreateNetwork(ctx, tenantID, vpcID, name, cidr, prefix)
}

func (s *PlatformService) ListNetworks(tenantID string) []*platform.Network {
	return s.network.ListNetworks(tenantID)
}

func (s *PlatformService) UpdateNetwork(ctx context.Context, tenantID, networkID, name string) (*platform.Network, error) {
	return s.network.UpdateNetwork(ctx, tenantID, networkID, name)
}

func (s *PlatformService) DeleteNetwork(ctx context.Context, tenantID, networkID string) error {
	return s.network.DeleteNetwork(ctx, tenantID, networkID)
}

// --- storage ---

func (s *PlatformService) CreateVolume(ctx context.Context, tenantID, name string, sizeGi int) (*platform.Volume, error) {
	return s.storage.CreateVolume(ctx, tenantID, name, sizeGi)
}

func (s *PlatformService) ListVolumes(tenantID string) []*platform.Volume {
	return s.storage.ListVolumes(tenantID)
}

func (s *PlatformService) CreateSnapshot(ctx context.Context, tenantID, volumeID, name string) (*platform.Snapshot, error) {
	return s.storage.CreateSnapshot(ctx, tenantID, volumeID, name)
}

func (s *PlatformService) ListSnapshots(tenantID string) []*platform.Snapshot {
	return s.storage.ListSnapshots(tenantID)
}

// --- compute ---

func (s *PlatformService) ListServiceOfferings() []*platform.ServiceOffering {
	return s.compute.ListServiceOfferings()
}

func (s *PlatformService) ListVMTemplates() []*platform.VMTemplate {
	return s.compute.ListVMTemplates()
}

func (s *PlatformService) DeployVM(ctx context.Context, tenantID string, in PlatformDeployVMInput) (*platform.PlatformVM, error) {
	return s.compute.DeployVM(ctx, tenantID, in)
}

func (s *PlatformService) GetVM(ctx context.Context, tenantID, name string) (*platform.PlatformVM, error) {
	return s.compute.GetVM(ctx, tenantID, name)
}

func (s *PlatformService) UpdateVM(ctx context.Context, tenantID, name string, in UpdateVMInput) (*platform.PlatformVM, error) {
	return s.compute.UpdateVM(ctx, tenantID, name, in)
}

func (s *PlatformService) ListVMs(ctx context.Context, tenantID string) ([]*platform.PlatformVM, error) {
	return s.compute.ListVMs(ctx, tenantID)
}

func (s *PlatformService) SyncAllVMStates(ctx context.Context) {
	s.compute.SyncAllVMStates(ctx)
}

func (s *PlatformService) StartVM(ctx context.Context, tenantID, vmName string) (*platform.PlatformVM, error) {
	return s.compute.StartVM(ctx, tenantID, vmName)
}

func (s *PlatformService) StopVM(ctx context.Context, tenantID, vmName string) (*platform.PlatformVM, error) {
	return s.compute.StopVM(ctx, tenantID, vmName)
}

func (s *PlatformService) DeleteVM(ctx context.Context, tenantID, vmName string) error {
	return s.compute.DeleteVM(ctx, tenantID, vmName)
}

func (s *PlatformService) CreateVMSnapshot(ctx context.Context, tenantID, vmName, name string) (*platform.VMSnapshot, error) {
	return s.compute.CreateVMSnapshot(ctx, tenantID, vmName, name)
}

func (s *PlatformService) ListVMSnapshots(ctx context.Context, tenantID string) ([]*platform.VMSnapshot, error) {
	return s.compute.ListVMSnapshots(ctx, tenantID)
}

func (s *PlatformService) DeleteVMSnapshot(ctx context.Context, tenantID, snapName string) error {
	return s.compute.DeleteVMSnapshot(ctx, tenantID, snapName)
}

func (s *PlatformService) RestoreVMSnapshot(ctx context.Context, tenantID, snapName, vmName string) error {
	return s.compute.RestoreVMSnapshot(ctx, tenantID, snapName, vmName)
}

func (s *PlatformService) ReconcileAll(ctx context.Context) {
	s.compute.ReconcileAll(ctx)
}

// --- ssh keys ---

func (s *PlatformService) ListSSHKeys(tenantID string) []*platform.SSHKeyPair {
	return s.sshkeys.List(tenantID)
}

func (s *PlatformService) CreateSSHKey(tenantID, name string) (*sshkeys.CreateResult, error) {
	return s.sshkeys.Create(tenantID, name)
}

func (s *PlatformService) RegisterSSHKey(tenantID, name, publicKey string) (*platform.SSHKeyPair, error) {
	return s.sshkeys.Register(tenantID, name, publicKey)
}

func (s *PlatformService) DeleteSSHKey(tenantID, id string) error {
	return s.sshkeys.Delete(tenantID, id)
}

func (s *PlatformService) ExposeVMSSH(ctx context.Context, tenantID, vmName string, nodePort int32) (int32, error) {
	return s.compute.ExposeVMSSH(ctx, tenantID, vmName, nodePort)
}

func (s *PlatformService) GetVMSSH(ctx context.Context, tenantID, vmName string) (*compute.VMSSHInfo, error) {
	return s.compute.GetVMSSH(ctx, tenantID, vmName)
}

// --- jobs ---

func (s *PlatformService) EnqueueJob(tenantID, jobType, payload string) *platform.AsyncJob {
	return s.jobs.Enqueue(tenantID, jobType, payload)
}

func (s *PlatformService) ProcessPendingJobs(ctx context.Context) {
	s.jobs.ProcessPending(ctx)
}

func (s *PlatformService) StreamVMLogs(ctx context.Context, tenantID, vmName string, tailLines int64, follow bool) (io.ReadCloser, error) {
	return s.compute.StreamLogs(ctx, tenantID, vmName, tailLines, follow)
}

func (s *PlatformService) VMLogExploreURL(tenantID, vmName string) string {
	return s.compute.LogExploreURL(tenantID, vmName)
}
