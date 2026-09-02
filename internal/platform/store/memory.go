package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/google/uuid"
)

type Memory struct {
	mu sync.RWMutex

	users            map[string]*platform.User
	usersByName      map[string]*platform.User
	tenants          map[string]*platform.Tenant
	vpcs             map[string]*platform.VPC
	securityGroups   map[string]*platform.SecurityGroup
	networks         map[string]*platform.Network
	auditEvents      []*platform.AuditEvent
	ipAddresses      map[string]*platform.IPAddress
	volumes          map[string]*platform.Volume
	snapshots        map[string]*platform.Snapshot
	vmSnapshots      map[string]*platform.VMSnapshot
	vms              map[string]*platform.PlatformVM
	jobs             map[string]*platform.AsyncJob
	serviceOfferings map[string]*platform.ServiceOffering
	vmTemplates      map[string]*platform.VMTemplate
	sshKeyPairs      map[string]*platform.SSHKeyPair
	targetGroups     map[string]*platform.TargetGroup
	loadBalancers    map[string]*platform.LoadBalancer
	lbListeners      map[string]*platform.LBListener
	lbTargets        map[string]*platform.LBTarget
	roles            map[string]*platform.RoleRecord
	rolePerms        map[string][]string
	apiKeys          map[string]*platform.APIKey
}

func NewMemory() *Memory {
	return &Memory{
		users:            make(map[string]*platform.User),
		usersByName:      make(map[string]*platform.User),
		tenants:          make(map[string]*platform.Tenant),
		vpcs:             make(map[string]*platform.VPC),
		securityGroups:   make(map[string]*platform.SecurityGroup),
		networks:         make(map[string]*platform.Network),
		auditEvents:      make([]*platform.AuditEvent, 0),
		ipAddresses:      make(map[string]*platform.IPAddress),
		volumes:          make(map[string]*platform.Volume),
		snapshots:        make(map[string]*platform.Snapshot),
		vmSnapshots:      make(map[string]*platform.VMSnapshot),
		vms:              make(map[string]*platform.PlatformVM),
		jobs:             make(map[string]*platform.AsyncJob),
		serviceOfferings: make(map[string]*platform.ServiceOffering),
		vmTemplates:      make(map[string]*platform.VMTemplate),
		sshKeyPairs:      make(map[string]*platform.SSHKeyPair),
		targetGroups:     make(map[string]*platform.TargetGroup),
		loadBalancers:    make(map[string]*platform.LoadBalancer),
		lbListeners:      make(map[string]*platform.LBListener),
		lbTargets:        make(map[string]*platform.LBTarget),
		roles:            make(map[string]*platform.RoleRecord),
		rolePerms:        make(map[string][]string),
		apiKeys:          make(map[string]*platform.APIKey),
	}
}

func (m *Memory) Close() error { return nil }

var _ Repository = (*Memory)(nil)

func (m *Memory) SaveUser(u *platform.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
	m.usersByName[u.Username] = u
}

func (m *Memory) GetUserByUsername(username string) (*platform.User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.usersByName[username]
	return u, ok
}

func (m *Memory) HasRootUser() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Role == platform.RoleRoot {
			return true
		}
	}
	return false
}

func (m *Memory) GetUser(id string) (*platform.User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	return u, ok
}

func (m *Memory) ListUsers() []*platform.User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*platform.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	return out
}

func (m *Memory) SaveTenant(t *platform.Tenant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[t.ID] = t
}

func (m *Memory) GetTenant(id string) (*platform.Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tenants[id]
	return t, ok
}

func (m *Memory) GetTenantBySlug(slug string) (*platform.Tenant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tenants {
		if t.Slug == slug {
			return t, true
		}
	}
	return nil, false
}

func (m *Memory) ListTenants() []*platform.Tenant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*platform.Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		out = append(out, t)
	}
	return out
}

func (m *Memory) DeleteTenant(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tenants, id)
}

func (m *Memory) PurgeTenantData(tenantID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, u := range m.users {
		if u.TenantID == tenantID && u.Role != platform.RoleRoot {
			delete(m.usersByName, u.Username)
			delete(m.users, id)
		}
	}
	for id, k := range m.apiKeys {
		if k.TenantID == tenantID {
			delete(m.apiKeys, id)
		}
	}
	for id, r := range m.roles {
		if r.TenantID == tenantID && !r.IsSystem {
			delete(m.rolePerms, id)
			delete(m.roles, id)
		}
	}
	for id, vm := range m.vms {
		if vm.TenantID == tenantID {
			delete(m.vms, id)
		}
	}
	for id, v := range m.volumes {
		if v.TenantID == tenantID {
			delete(m.volumes, id)
		}
	}
	for id, s := range m.snapshots {
		if s.TenantID == tenantID {
			delete(m.snapshots, id)
		}
	}
	for id, s := range m.vmSnapshots {
		if s.TenantID == tenantID {
			delete(m.vmSnapshots, id)
		}
	}
	networkIDs := map[string]struct{}{}
	for id, n := range m.networks {
		if n.TenantID == tenantID {
			networkIDs[id] = struct{}{}
			delete(m.networks, id)
		}
	}
	for id, ip := range m.ipAddresses {
		if _, ok := networkIDs[ip.NetworkID]; ok {
			delete(m.ipAddresses, id)
		}
	}
	for id, v := range m.vpcs {
		if v.TenantID == tenantID {
			delete(m.vpcs, id)
		}
	}
	for id, sg := range m.securityGroups {
		if sg.TenantID == tenantID {
			delete(m.securityGroups, id)
		}
	}
	for id, k := range m.sshKeyPairs {
		if k.TenantID == tenantID {
			delete(m.sshKeyPairs, id)
		}
	}
	for id, j := range m.jobs {
		if j.TenantID == tenantID {
			delete(m.jobs, id)
		}
	}
	for id, t := range m.vmTemplates {
		if t.TenantID == tenantID {
			delete(m.vmTemplates, id)
		}
	}
	for id, tg := range m.targetGroups {
		if tg.TenantID == tenantID {
			delete(m.targetGroups, id)
		}
	}
	for id, lb := range m.loadBalancers {
		if lb.TenantID == tenantID {
			delete(m.loadBalancers, id)
		}
	}
	for id, l := range m.lbListeners {
		if l.TenantID == tenantID {
			delete(m.lbListeners, id)
		}
	}
	for id, t := range m.lbTargets {
		if t.TenantID == tenantID {
			delete(m.lbTargets, id)
		}
	}
}

func (m *Memory) SaveVPC(v *platform.VPC) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vpcs[v.ID] = v
}

func (m *Memory) ListVPCs(tenantID string) []*platform.VPC {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.VPC
	for _, v := range m.vpcs {
		if v.TenantID == tenantID {
			out = append(out, v)
		}
	}
	return out
}

func (m *Memory) GetVPC(id string) (*platform.VPC, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.vpcs[id]
	return v, ok
}

func (m *Memory) DeleteVPC(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.vpcs, id)
}

func (m *Memory) SaveSG(sg *platform.SecurityGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.securityGroups[sg.ID] = sg
}

func (m *Memory) ListSGs(tenantID string) []*platform.SecurityGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.SecurityGroup
	for _, sg := range m.securityGroups {
		if sg.TenantID == tenantID {
			out = append(out, sg)
		}
	}
	return out
}

func (m *Memory) GetSG(id string) (*platform.SecurityGroup, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sg, ok := m.securityGroups[id]
	return sg, ok
}

func (m *Memory) DeleteSG(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.securityGroups, id)
}

func (m *Memory) SaveNetwork(n *platform.Network) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.networks[n.ID] = n
}

func (m *Memory) ListNetworks(tenantID string) []*platform.Network {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.Network
	for _, n := range m.networks {
		if n.TenantID == tenantID || n.NetworkType == platform.NetworkTypeShared {
			out = append(out, n)
		}
	}
	return out
}

func (m *Memory) GetSharedNetwork() (*platform.Network, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, n := range m.networks {
		if n.NetworkType == platform.NetworkTypeShared {
			return n, true
		}
	}
	return nil, false
}

func (m *Memory) SaveAuditEvent(e *platform.AuditEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.auditEvents = append(m.auditEvents, e)
}

func (m *Memory) ListAuditEvents(targetTenantID string, limit int) []*platform.AuditEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	var out []*platform.AuditEvent
	for i := len(m.auditEvents) - 1; i >= 0 && len(out) < limit; i-- {
		if m.auditEvents[i].TargetTenantID == targetTenantID {
			out = append(out, m.auditEvents[i])
		}
	}
	return out
}

func (m *Memory) AllocateIPAddress(networkID string) (*platform.IPAddress, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var best *platform.IPAddress
	for _, ip := range m.ipAddresses {
		if ip.NetworkID == networkID && ip.Status == "available" {
			if best == nil || ip.Address < best.Address {
				best = ip
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no available IP in pool")
	}
	best.Status = "allocated"
	return best, nil
}

func (m *Memory) ReleaseIPAddressByVMNic(vmNicID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ip := range m.ipAddresses {
		if ip.VMNicID == vmNicID {
			ip.Status = "available"
			ip.VMNicID = ""
		}
	}
}

func (m *Memory) ReleaseIPAddressByAddress(networkID, address string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ip := range m.ipAddresses {
		if ip.NetworkID == networkID && ip.Address == address {
			ip.Status = "available"
			ip.VMNicID = ""
		}
	}
}

func (m *Memory) SeedIPPool(networkID, start, end string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return seedIPRangeLocked(m, networkID, start, end)
}

func (m *Memory) saveIPAddressQuiet(networkID, addr string) {
	for _, existing := range m.ipAddresses {
		if existing.NetworkID == networkID && existing.Address == addr {
			return
		}
	}
	id := NewID()
	m.ipAddresses[id] = &platform.IPAddress{
		ID: id, NetworkID: networkID, Address: addr, Status: "available", CreatedAt: Now(),
	}
}

func (m *Memory) SaveVolume(v *platform.Volume) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.volumes[v.ID] = v
}

func (m *Memory) ListVolumes(tenantID string) []*platform.Volume {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.Volume
	for _, v := range m.volumes {
		if v.TenantID == tenantID {
			out = append(out, v)
		}
	}
	return out
}

func (m *Memory) GetVolume(id string) (*platform.Volume, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.volumes[id]
	return v, ok
}

func (m *Memory) ListVolumesByVMID(tenantID, vmID string) []*platform.Volume {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.Volume
	for _, v := range m.volumes {
		if v.TenantID == tenantID && v.VMID == vmID {
			out = append(out, v)
		}
	}
	return out
}

func (m *Memory) DeleteVolume(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.volumes, id)
}

func (m *Memory) SaveSnapshot(s *platform.Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[s.ID] = s
}

func (m *Memory) ListSnapshots(tenantID string) []*platform.Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.Snapshot
	for _, s := range m.snapshots {
		if s.TenantID == tenantID {
			out = append(out, s)
		}
	}
	return out
}

func (m *Memory) SaveVMSnapshot(s *platform.VMSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vmSnapshots[s.ID] = s
}

func (m *Memory) ListVMSnapshots(tenantID string) []*platform.VMSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.VMSnapshot
	for _, s := range m.vmSnapshots {
		if s.TenantID == tenantID {
			out = append(out, s)
		}
	}
	return out
}

func (m *Memory) GetVMSnapshot(id string) (*platform.VMSnapshot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.vmSnapshots[id]
	return s, ok
}

func (m *Memory) DeleteVMSnapshot(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.vmSnapshots, id)
}

func (m *Memory) SaveVM(vm *platform.PlatformVM) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vms[vm.ID] = vm
}

func (m *Memory) GetVM(id string) (*platform.PlatformVM, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	vm, ok := m.vms[id]
	return vm, ok
}

func (m *Memory) GetVMByName(tenantID, name string) (*platform.PlatformVM, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, vm := range m.vms {
		if vm.TenantID == tenantID && vm.Name == name {
			return vm, true
		}
	}
	return nil, false
}

func (m *Memory) ListVMs(tenantID string) []*platform.PlatformVM {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.PlatformVM
	for _, vm := range m.vms {
		if vm.TenantID == tenantID {
			out = append(out, vm)
		}
	}
	return out
}

func (m *Memory) DeleteVM(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.vms, id)
}

func (m *Memory) SaveJob(j *platform.AsyncJob) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j
}

func (m *Memory) GetJob(id string) (*platform.AsyncJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[id]
	return j, ok
}

func (m *Memory) ListJobs(tenantID string) []*platform.AsyncJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.AsyncJob
	for _, j := range m.jobs {
		if tenantID == "" || j.TenantID == tenantID {
			out = append(out, j)
		}
	}
	return out
}

func (m *Memory) ListPendingJobs(limit int) []*platform.AsyncJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.AsyncJob
	for _, j := range m.jobs {
		if j.Status == "pending" {
			out = append(out, j)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

func (m *Memory) GetNetwork(id string) (*platform.Network, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.networks[id]
	return n, ok
}

func (m *Memory) DeleteNetwork(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.networks, id)
}

func (m *Memory) GetVMByExternalUUID(source, externalUUID string) (*platform.PlatformVM, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, vm := range m.vms {
		if vm.ImportSource == source && vm.ExternalUUID == externalUUID {
			return vm, true
		}
	}
	return nil, false
}

func (m *Memory) SaveServiceOffering(o *platform.ServiceOffering) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceOfferings[o.ID] = o
}

func (m *Memory) GetServiceOffering(id string) (*platform.ServiceOffering, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.serviceOfferings[id]
	return o, ok
}

func (m *Memory) GetServiceOfferingByName(name string) (*platform.ServiceOffering, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, o := range m.serviceOfferings {
		if o.Name == name {
			return o, true
		}
	}
	return nil, false
}

func (m *Memory) ListServiceOfferings(activeOnly bool) []*platform.ServiceOffering {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.ServiceOffering
	for _, o := range m.serviceOfferings {
		if activeOnly && o.State != "Active" {
			continue
		}
		out = append(out, o)
	}
	return out
}

func (m *Memory) DeleteServiceOffering(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.serviceOfferings, id)
}

func (m *Memory) SaveVMTemplate(t *platform.VMTemplate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vmTemplates[t.ID] = t
}

func (m *Memory) GetVMTemplate(id string) (*platform.VMTemplate, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.vmTemplates[id]
	return t, ok
}

func (m *Memory) DeleteVMTemplate(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.vmTemplates, id)
}

func (m *Memory) ListVMTemplates(activeOnly bool) []*platform.VMTemplate {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.VMTemplate
	for _, t := range m.vmTemplates {
		if activeOnly && t.State != "Active" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (m *Memory) ListVMTemplatesForTenant(tenantID string, activeOnly bool) []*platform.VMTemplate {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.VMTemplate
	for _, t := range m.vmTemplates {
		if t.TenantID != "" && t.TenantID != tenantID {
			continue
		}
		if activeOnly && t.State != "Active" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (m *Memory) SaveSSHKeyPair(k *platform.SSHKeyPair) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sshKeyPairs[k.ID] = k
}

func (m *Memory) GetSSHKeyPair(id string) (*platform.SSHKeyPair, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.sshKeyPairs[id]
	return k, ok
}

func (m *Memory) ListSSHKeyPairs(tenantID string) []*platform.SSHKeyPair {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.SSHKeyPair
	for _, k := range m.sshKeyPairs {
		if k.TenantID == tenantID {
			out = append(out, k)
		}
	}
	return out
}

func (m *Memory) DeleteSSHKeyPair(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sshKeyPairs, id)
}

func (m *Memory) SaveTargetGroup(tg *platform.TargetGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targetGroups[tg.ID] = tg
}

func (m *Memory) GetTargetGroup(id string) (*platform.TargetGroup, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tg, ok := m.targetGroups[id]
	return tg, ok
}

func (m *Memory) ListTargetGroups(tenantID string) []*platform.TargetGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.TargetGroup
	for _, tg := range m.targetGroups {
		if tg.TenantID == tenantID {
			out = append(out, tg)
		}
	}
	return out
}

func (m *Memory) DeleteTargetGroup(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.targetGroups, id)
}

func (m *Memory) SaveLoadBalancer(lb *platform.LoadBalancer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadBalancers[lb.ID] = lb
}

func (m *Memory) GetLoadBalancer(id string) (*platform.LoadBalancer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	lb, ok := m.loadBalancers[id]
	return lb, ok
}

func (m *Memory) ListLoadBalancers(tenantID string) []*platform.LoadBalancer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.LoadBalancer
	for _, lb := range m.loadBalancers {
		if lb.TenantID == tenantID {
			out = append(out, lb)
		}
	}
	return out
}

func (m *Memory) DeleteLoadBalancer(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.loadBalancers, id)
}

func (m *Memory) SaveLBListener(l *platform.LBListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lbListeners[l.ID] = l
}

func (m *Memory) GetLBListener(id string) (*platform.LBListener, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l, ok := m.lbListeners[id]
	return l, ok
}

func (m *Memory) ListLBListeners(loadBalancerID string) []*platform.LBListener {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.LBListener
	for _, l := range m.lbListeners {
		if l.LoadBalancerID == loadBalancerID {
			out = append(out, l)
		}
	}
	return out
}

func (m *Memory) DeleteLBListener(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.lbListeners, id)
}

func (m *Memory) SaveLBTarget(t *platform.LBTarget) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lbTargets[t.ID] = t
}

func (m *Memory) GetLBTarget(id string) (*platform.LBTarget, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.lbTargets[id]
	return t, ok
}

func (m *Memory) ListLBTargets(targetGroupID string) []*platform.LBTarget {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*platform.LBTarget
	for _, t := range m.lbTargets {
		if t.TargetGroupID == targetGroupID {
			out = append(out, t)
		}
	}
	return out
}

func (m *Memory) DeleteLBTarget(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.lbTargets, id)
}

func (m *Memory) DeleteLBTargetsByTargetGroup(targetGroupID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, t := range m.lbTargets {
		if t.TargetGroupID == targetGroupID {
			delete(m.lbTargets, id)
		}
	}
}

func NewID() string {
	return uuid.New().String()
}

func Now() time.Time {
	return time.Now().UTC()
}
