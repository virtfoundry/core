package compute

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"

	"github.com/virtfoundry/core/internal/infra/hypervisor"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

type vmStateKey struct {
	tenantID string
	name     string
}

// Service manages VMs, VM snapshots and the compute catalog.
type Service struct {
	store  store.Repository
	k8s    *platformk8s.Manager
	kvBase *hypervisor.KubeVirtDriver
	hub    shared.EventBroadcaster

	vmStateMu       sync.Mutex
	vmStates        map[vmStateKey]string
	allowPodNetwork bool
	defaultNetwork  string
	storageClass    string
	windowsBootSizeGi int
	windowsISOSizeGi  int
}

func New(st store.Repository, k8s *platformk8s.Manager, kv *hypervisor.KubeVirtDriver, hub shared.EventBroadcaster) *Service {
	return &Service{
		store: st, k8s: k8s, kvBase: kv, hub: hub,
		vmStates: make(map[vmStateKey]string), allowPodNetwork: true, defaultNetwork: "pod",
	}
}

func (s *Service) ConfigureVMNetworking(defaultNetwork string, allowPodNetwork bool) {
	if defaultNetwork != "" {
		s.defaultNetwork = defaultNetwork
	}
	s.allowPodNetwork = allowPodNetwork
}

func (s *Service) ConfigureStorage(defaultClass string, bootSizeGi, isoSizeGi int) {
	if defaultClass != "" {
		s.storageClass = defaultClass
	}
	if bootSizeGi > 0 {
		s.windowsBootSizeGi = bootSizeGi
	}
	if isoSizeGi > 0 {
		s.windowsISOSizeGi = isoSizeGi
	}
}

// DeployVMInput is the payload for synchronous or async VM deployment.
type DeployVMInput struct {
	Name              string
	Image             string
	CPU               int
	MemoryMi          int64
	Start             bool
	ServiceOfferingID string
	TemplateID        string
	NetworkIDs        []string
	PublicIP          bool
	SecurityGroupIDs  []string
	DisplayName       string
	SSHKeyID          string
	DataVolumeID      string
	BootDiskSizeGi    int
	ExposeSSH         bool
}

// UpdateVMInput patches VM metadata and resources.
type UpdateVMInput struct {
	DisplayName       string
	CPU               int
	MemoryMi          int64
	ServiceOfferingID string
}

func (s *Service) ListVMTemplates(tenantID string) []*platform.VMTemplate {
	return s.ListVMTemplatesForTenant(tenantID)
}

func (s *Service) DeployVM(ctx context.Context, tenantID string, in DeployVMInput) (*platform.PlatformVM, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	name := shared.SanitizeSlug(in.Name)
	if name == "" {
		return nil, fmt.Errorf("invalid vm name")
	}

	cpu, memMi, image := in.CPU, in.MemoryMi, in.Image
	var tmplDisplay, osType, cloudInitExtra string
	var deployTmpl *platform.VMTemplate
	if in.ServiceOfferingID != "" {
		if off, ok := s.store.GetServiceOffering(in.ServiceOfferingID); ok {
			cpu, memMi = off.CPU, off.MemoryMi
		}
	}
	if in.TemplateID != "" {
		tmpl, err := s.resolveTemplate(tenantID, in.TemplateID)
		if err != nil {
			return nil, err
		}
		if tmpl != nil {
			deployTmpl = tmpl
			image = tmpl.Image
			osType = tmpl.OSType
			cloudInitExtra = tmpl.CloudInitUserData
			tmplDisplay = tmpl.DisplayName
		}
	}
	if cpu <= 0 {
		cpu = 1
	}
	if memMi <= 0 {
		memMi = 1024
	}
	if image == "" {
		image = "quay.io/kubevirt/cirros-container-disk-demo"
	}

	networkIDs, err := s.resolveDeployNetworks(tenantID, in.PublicIP, in.NetworkIDs)
	if err != nil {
		return nil, err
	}
	if in.PublicIP {
		if err := s.validatePublicSecurityGroups(tenantID, in.SecurityGroupIDs); err != nil {
			return nil, err
		}
	}

	netSpecs, vmNics, err := s.buildVMNetworks(tenantID, networkIDs)
	if err != nil {
		return nil, err
	}

	kv := s.kvBase.WithNamespace(ns)
	spec := hypervisor.VMDeploySpec{
		Name: name, Namespace: ns,
		CPU: cpu, MemoryMi: memMi, Image: image, OSType: osType, Start: true,
		Networks: netSpecs,
		Labels:   sgLabels(in.SecurityGroupIDs),
		CloudInitExtra: cloudInitExtra,
	}
	if in.SSHKeyID != "" {
		if k, ok := s.store.GetSSHKeyPair(in.SSHKeyID); ok && k.TenantID == tenantID {
			spec.CloudInitSSHKeys = []string{k.PublicKey}
		}
	}
	if in.DataVolumeID != "" {
		vol, err := s.validateUnattachedVolume(tenantID, in.DataVolumeID)
		if err != nil {
			return nil, fmt.Errorf("data volume: %w", err)
		}
		spec.DataPVC = vol.PVCName
	}

	if deployTmpl != nil && strings.EqualFold(deployTmpl.SourceType, "iso") {
		isoPVC, err := s.resolveISOPVC(tenantID, deployTmpl)
		if err != nil {
			return nil, err
		}
		bootPVC, err := s.provisionWindowsDisks(ctx, ns, name, deployTmpl, isoPVC)
		if err != nil {
			return nil, err
		}
		spec.OSType = "windows"
		spec.BootPVC = bootPVC
		spec.InstallISO = isoPVC
		spec.Image = ""
	}

	if err := kv.CreateVM(ctx, spec); err != nil {
		return nil, err
	}

	info, _ := kv.GetVM(ctx, name)
	tenant, _ := s.store.GetTenant(tenantID)
	displayName := in.DisplayName
	if displayName == "" {
		displayName = name
	}
	vm := &platform.PlatformVM{
		ID: store.NewID(), TenantID: tenantID, Name: name, DisplayName: displayName, Namespace: ns,
		State: "Starting", CPU: cpu, MemoryMi: memMi, Image: image,
		Template: firstNonEmpty(tmplDisplay, templateLabel(image)), Hypervisor: "KubeVirt",
		ServiceOfferingID: in.ServiceOfferingID,
		NICs:              vmNics,
		CreatedAt:         store.Now(),
	}
	if tenant != nil {
		vm.Zone = tenant.Slug
	}
	if info != nil {
		s.applyVMInfo(vm, *info, tenant)
	}
	if in.ExposeSSH && info != nil && info.IP != "" {
		if _, err := s.k8s.EnsureVMSSHService(ctx, ns, name, info.IP, 0); err == nil {
			// NodePort created; IP sync will refresh state on next poll.
		}
	}
	s.store.SaveVM(vm)
	if in.DataVolumeID != "" {
		if vol, ok := s.store.GetVolume(in.DataVolumeID); ok {
			s.markVolumeAttached(vol, vm)
		}
	}
	s.broadcastVM("vm.created", vm)
	return vm, nil
}

func (s *Service) GetVM(ctx context.Context, tenantID, name string) (*platform.PlatformVM, error) {
	vms, err := s.ListVMs(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, vm := range vms {
		if vm.Name == name {
			return vm, nil
		}
	}
	return nil, fmt.Errorf("vm not found")
}

func (s *Service) UpdateVM(ctx context.Context, tenantID, name string, in UpdateVMInput) (*platform.PlatformVM, error) {
	vm, ok := s.store.GetVMByName(tenantID, name)
	if !ok {
		if _, err := s.GetVM(ctx, tenantID, name); err != nil {
			return nil, fmt.Errorf("vm not found")
		}
		vm, _ = s.store.GetVMByName(tenantID, name)
	}
	if in.DisplayName != "" {
		vm.DisplayName = in.DisplayName
	}
	if in.ServiceOfferingID != "" {
		off, ok := s.store.GetServiceOffering(in.ServiceOfferingID)
		if !ok {
			return nil, fmt.Errorf("service offering not found")
		}
		in.CPU = off.CPU
		in.MemoryMi = off.MemoryMi
		vm.ServiceOfferingID = in.ServiceOfferingID
	}
	if in.CPU > 0 || in.MemoryMi > 0 {
		ns, err := shared.TenantNamespace(s.store, tenantID)
		if err != nil {
			return nil, err
		}
		cpu, mem := in.CPU, in.MemoryMi
		if cpu <= 0 {
			cpu = vm.CPU
		}
		if mem <= 0 {
			mem = vm.MemoryMi
		}
		if err := s.kvBase.WithNamespace(ns).UpdateVMResources(ctx, name, cpu, mem); err != nil {
			return nil, err
		}
		vm.CPU = cpu
		vm.MemoryMi = mem
	}
	vm.UpdatedAt = store.Now()
	s.store.SaveVM(vm)
	merged, _ := s.GetVM(ctx, tenantID, name)
	if merged != nil {
		s.broadcastVM("vm.updated", merged)
		return merged, nil
	}
	s.broadcastVM("vm.updated", vm)
	return vm, nil
}

func (s *Service) ListVMs(ctx context.Context, tenantID string) ([]*platform.PlatformVM, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	kv := s.kvBase.WithNamespace(ns)
	infos, err := kv.ListVMs(ctx)
	if err != nil {
		return s.store.ListVMs(tenantID), nil
	}

	tenant, _ := s.store.GetTenant(tenantID)
	byName := map[string]*platform.PlatformVM{}
	for _, stored := range s.store.ListVMs(tenantID) {
		byName[stored.Name] = stored
	}

	out := make([]*platform.PlatformVM, 0, len(infos))
	for _, info := range infos {
		vm, ok := byName[info.Name]
		if !ok {
			vm = &platform.PlatformVM{
				ID: store.NewID(), TenantID: tenantID, Name: info.Name, Namespace: ns,
				CreatedAt: info.Created,
			}
		}
		s.applyVMInfo(vm, info, tenant)
		s.store.SaveVM(vm)
		out = append(out, vm)
	}
	return out, nil
}

func (s *Service) SyncAllVMStates(ctx context.Context) {
	for _, tenant := range s.store.ListTenants() {
		vms, err := s.ListVMs(ctx, tenant.ID)
		if err != nil {
			continue
		}
		for _, vm := range vms {
			key := vmStateKey{tenantID: tenant.ID, name: vm.Name}
			sig := vmStateSignature(vm)
			s.vmStateMu.Lock()
			prev, ok := s.vmStates[key]
			if !ok || prev != sig {
				s.vmStates[key] = sig
				s.vmStateMu.Unlock()
				if ok {
					s.broadcastVM("vm.updated", vm)
				}
				continue
			}
			s.vmStateMu.Unlock()
		}
	}
}

func (s *Service) StartVM(ctx context.Context, tenantID, vmName string) (*platform.PlatformVM, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.kvBase.WithNamespace(ns).StartVM(ctx, vmName); err != nil {
		return nil, err
	}
	vm, err := s.GetVM(ctx, tenantID, vmName)
	if err != nil {
		return nil, err
	}
	s.broadcastVM("vm.updated", vm)
	return vm, nil
}

func (s *Service) StopVM(ctx context.Context, tenantID, vmName string) (*platform.PlatformVM, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.kvBase.WithNamespace(ns).StopVM(ctx, vmName); err != nil {
		return nil, err
	}
	vm, err := s.GetVM(ctx, tenantID, vmName)
	if err != nil {
		return nil, err
	}
	s.broadcastVM("vm.updated", vm)
	return vm, nil
}

func (s *Service) DeleteVM(ctx context.Context, tenantID, vmName string) error {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return err
	}
	if vm, ok := s.store.GetVMByName(tenantID, vmName); ok {
		for _, nic := range vm.NICs {
			if nic.NetworkID != "" && nic.IP != "" {
				s.store.ReleaseIPAddressByAddress(nic.NetworkID, nic.IP)
			}
		}
	}
	if err := s.kvBase.WithNamespace(ns).DeleteVM(ctx, vmName); err != nil {
		return err
	}
	if vm, ok := s.store.GetVMByName(tenantID, vmName); ok {
		s.releaseVolumesForVM(tenantID, vm.ID)
		s.store.DeleteVM(vm.ID)
	}
	key := vmStateKey{tenantID: tenantID, name: vmName}
	s.vmStateMu.Lock()
	delete(s.vmStates, key)
	s.vmStateMu.Unlock()
	s.broadcastVM("vm.deleted", map[string]string{"tenant_id": tenantID, "name": vmName})
	return nil
}

func mapVMSnapshotPhase(phase string) string {
	switch phase {
	case "Succeeded":
		return "ready"
	case "InProgress", "":
		return "creating"
	case "Failed":
		return "failed"
	default:
		return strings.ToLower(phase)
	}
}

func (s *Service) CreateVMSnapshot(ctx context.Context, tenantID, vmName, name string) (*platform.VMSnapshot, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	vm, ok := s.store.GetVMByName(tenantID, vmName)
	if !ok {
		return nil, fmt.Errorf("vm not found")
	}
	snapName := shared.SanitizeSlug(name)
	if snapName == "" {
		return nil, fmt.Errorf("invalid snapshot name")
	}
	kv := s.kvBase.WithNamespace(ns)
	if err := kv.CreateVMSnapshot(ctx, vmName, snapName); err != nil {
		return nil, err
	}
	snap := &platform.VMSnapshot{
		ID: store.NewID(), TenantID: tenantID, VMID: vm.ID, VMName: vmName,
		Name: snapName, Namespace: ns, Phase: "creating", CreatedAt: store.Now(),
	}
	s.store.SaveVMSnapshot(snap)
	return snap, nil
}

func (s *Service) ListVMSnapshots(ctx context.Context, tenantID string) ([]*platform.VMSnapshot, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	kv := s.kvBase.WithNamespace(ns)
	infos, err := kv.ListVMSnapshots(ctx)
	if err != nil {
		return s.store.ListVMSnapshots(tenantID), nil
	}
	byName := map[string]*platform.VMSnapshot{}
	for _, stored := range s.store.ListVMSnapshots(tenantID) {
		byName[stored.Name] = stored
	}
	out := make([]*platform.VMSnapshot, 0, len(infos))
	for _, info := range infos {
		snap, ok := byName[info.Name]
		if !ok {
			vmID := ""
			if vm, found := s.store.GetVMByName(tenantID, info.VMName); found {
				vmID = vm.ID
			}
			snap = &platform.VMSnapshot{
				ID: store.NewID(), TenantID: tenantID, VMID: vmID, VMName: info.VMName,
				Name: info.Name, Namespace: ns, CreatedAt: info.Created,
			}
		}
		snap.Phase = mapVMSnapshotPhase(info.Phase)
		s.store.SaveVMSnapshot(snap)
		out = append(out, snap)
	}
	return out, nil
}

func (s *Service) DeleteVMSnapshot(ctx context.Context, tenantID, snapName string) error {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return err
	}
	if err := s.kvBase.WithNamespace(ns).DeleteVMSnapshot(ctx, snapName); err != nil {
		return err
	}
	for _, snap := range s.store.ListVMSnapshots(tenantID) {
		if snap.Name == snapName {
			s.store.DeleteVMSnapshot(snap.ID)
			break
		}
	}
	return nil
}

func (s *Service) RestoreVMSnapshot(ctx context.Context, tenantID, snapName, vmName string) error {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return err
	}
	if vmName == "" {
		for _, snap := range s.store.ListVMSnapshots(tenantID) {
			if snap.Name == snapName {
				vmName = snap.VMName
				break
			}
		}
	}
	if vmName == "" {
		return fmt.Errorf("vm name required for restore")
	}
	return s.kvBase.WithNamespace(ns).RestoreVMSnapshot(ctx, snapName, vmName)
}

func (s *Service) ReconcileAll(ctx context.Context) {
	for _, tenant := range s.store.ListTenants() {
		ns, err := shared.TenantNamespace(s.store, tenant.ID)
		if err != nil {
			continue
		}
		kv := s.kvBase.WithNamespace(ns)
		infos, err := kv.ListVMs(ctx)
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, info := range infos {
			seen[info.Name] = true
			vm, ok := s.store.GetVMByName(tenant.ID, info.Name)
			if !ok {
				vm = &platform.PlatformVM{
					ID: store.NewID(), TenantID: tenant.ID, Name: info.Name,
					Namespace: ns, CreatedAt: info.Created,
				}
			}
			s.applyVMInfo(vm, info, tenant)
			s.store.SaveVM(vm)
		}
		for _, stored := range s.store.ListVMs(tenant.ID) {
			if !seen[stored.Name] && stored.State != "Destroyed" {
				stored.State = "Destroyed"
				stored.UpdatedAt = store.Now()
				s.store.SaveVM(stored)
				s.broadcastVM("vm.updated", stored)
			}
		}
	}
}

func (s *Service) broadcastVM(eventType string, payload interface{}) {
	if s.hub != nil {
		s.hub.Broadcast(eventType, payload)
	}
}

func (s *Service) applyVMInfo(vm *platform.PlatformVM, info hypervisor.VMInfo, tenant *platform.Tenant) {
	priorNics := make(map[string]platform.VMNic, len(vm.NICs))
	for _, nic := range vm.NICs {
		priorNics[nic.Name] = nic
	}

	storedPublicIP := ""
	for _, nic := range vm.NICs {
		if nic.Name == "public" && nic.IP != "" {
			storedPublicIP = nic.IP
		}
	}
	if storedPublicIP == "" {
		if existing, ok := s.store.GetVMByName(vm.TenantID, vm.Name); ok {
			for _, nic := range existing.NICs {
				if nic.Name == "public" && nic.IP != "" {
					storedPublicIP = nic.IP
				}
				if _, seen := priorNics[nic.Name]; !seen {
					priorNics[nic.Name] = nic
				}
			}
			if storedPublicIP == "" && strings.HasPrefix(existing.IP, "10.0.50.") {
				storedPublicIP = existing.IP
			}
		}
	}
	if storedPublicIP == "" && strings.HasPrefix(vm.IP, "10.0.50.") {
		storedPublicIP = vm.IP
	}

	vm.State = info.State
	vm.IP = info.IP
	if storedPublicIP != "" && (vm.IP == "" || strings.HasPrefix(vm.IP, "10.233.")) {
		vm.IP = storedPublicIP
	}
	vm.ErrorMsg = info.ErrorMsg
	vm.CPU = info.CPU
	vm.MemoryMi = info.MemoryMi
	if info.Image != "" {
		vm.Image = info.Image
		vm.Template = templateLabel(info.Image)
	}
	vm.HostName = info.NodeName
	vm.Hypervisor = "KubeVirt"
	if tenant != nil {
		vm.Zone = tenant.Slug
	}
	vm.NICs = nil
	merged := map[string]bool{}
	for _, n := range info.NICs {
		nic := platform.VMNic{Name: n.Name, IP: n.IP, MAC: n.MAC, Type: n.Type}
		if prior, ok := priorNics[n.Name]; ok {
			if nic.IP == "" {
				nic.IP = prior.IP
			}
			nic.NetworkID = prior.NetworkID
			nic.NADNamespace = prior.NADNamespace
			nic.NADName = prior.NADName
		}
		vm.NICs = append(vm.NICs, nic)
		merged[n.Name] = true
	}
	for name, prior := range priorNics {
		if !merged[name] {
			vm.NICs = append(vm.NICs, prior)
		}
	}
	if storedPublicIP == "" {
		for _, nic := range vm.NICs {
			if nic.Name == "public" && nic.IP != "" {
				storedPublicIP = nic.IP
			}
		}
	}
	if storedPublicIP != "" && (vm.IP == "" || strings.HasPrefix(vm.IP, "10.233.")) {
		vm.IP = storedPublicIP
	}
	if vm.DisplayName == "" {
		vm.DisplayName = vm.Name
	}
	vm.UpdatedAt = store.Now()
}

func vmStateSignature(vm *platform.PlatformVM) string {
	return vm.State + "|" + vm.IP + "|" + vm.ErrorMsg
}

func templateLabel(image string) string {
	if i := strings.LastIndex(image, "/"); i >= 0 {
		image = image[i+1:]
	}
	if i := strings.Index(image, ":"); i >= 0 {
		image = image[:i]
	}
	return image
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (s *Service) buildVMNetworks(tenantID string, networkIDs []string) ([]hypervisor.VMNetworkSpec, []platform.VMNic, error) {
	if len(networkIDs) == 0 {
		if s.allowPodNetwork {
			return nil, []platform.VMNic{{Name: "default", Type: "pod"}}, nil
		}
		return nil, nil, nil
	}
	var specs []hypervisor.VMNetworkSpec
	var nics []platform.VMNic
	for i, netID := range networkIDs {
		net, ok := s.store.GetNetwork(netID)
		if !ok {
			continue
		}
		if net.NetworkType != platform.NetworkTypeShared && net.TenantID != tenantID {
			continue
		}
		ifaceName := shared.SanitizeSlug(net.Name)
		if ifaceName == "" {
			ifaceName = fmt.Sprintf("net%d", i)
		}
		if net.NADName != "" && net.NADNamespace != "" {
			spec := hypervisor.VMNetworkSpec{
				Name: ifaceName, NADNamespace: net.NADNamespace, NADName: net.NADName,
			}
			nic := platform.VMNic{
				Name: ifaceName, Type: "multus", NetworkID: net.ID,
				NADNamespace: net.NADNamespace, NADName: net.NADName,
			}
			if net.NetworkType == platform.NetworkTypeShared {
				spec.MACAddress = randomMAC()
				ip, err := s.store.AllocateIPAddress(net.ID)
				if err != nil {
					return nil, nil, fmt.Errorf("allocate public IP: %w", err)
				}
				spec.StaticIP = ip.Address
				spec.PrefixLen = 24
				nic.IP = ip.Address
				if net.Gateway != "" {
					spec.Gateway = net.Gateway
					spec.DNS = []string{net.Gateway}
				}
			}
			specs = append(specs, spec)
			nics = append(nics, nic)
		}
	}
	if len(specs) == 0 {
		if s.allowPodNetwork {
			return nil, []platform.VMNic{{Name: "default", Type: "pod"}}, nil
		}
		return nil, nil, nil
	}
	if !s.allowPodNetwork {
		return specs, nics, nil
	}
	// Pod NIC must not be named "default" — tenant subnets use that name and KubeVirt requires unique network names.
	specs = append([]hypervisor.VMNetworkSpec{{Name: "pod", Default: true}}, specs...)
	nics = append([]platform.VMNic{{Name: "pod", Type: "pod"}}, nics...)
	return specs, nics, nil
}

func randomMAC() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "02:00:00:00:00:01"
	}
	b[0] = (b[0] | 0x02) & 0xfe
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}

// ExposeVMSSH creates a NodePort Service targeting the VM guest IP on port 22.
func (s *Service) ExposeVMSSH(ctx context.Context, tenantID, vmName string, nodePort int32) (int32, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return 0, err
	}
	info, err := s.kvBase.WithNamespace(ns).GetVM(ctx, vmName)
	if err != nil {
		return 0, err
	}
	if info.IP == "" {
		return 0, fmt.Errorf("vm has no IP yet")
	}
	return s.k8s.EnsureVMSSHService(ctx, ns, vmName, info.IP, nodePort)
}

type VMSSHInfo struct {
	Exposed  bool   `json:"exposed"`
	NodePort int32  `json:"node_port,omitempty"`
	VMIP     string `json:"vm_ip,omitempty"`
}

// GetVMSSH returns NodePort SSH exposure for a VM, if configured.
func (s *Service) GetVMSSH(ctx context.Context, tenantID, vmName string) (*VMSSHInfo, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	info, err := s.kvBase.WithNamespace(ns).GetVM(ctx, vmName)
	if err != nil {
		return nil, err
	}
	port, ok, err := s.k8s.GetVMSSHNodePort(ctx, ns, vmName)
	if err != nil {
		return nil, err
	}
	out := &VMSSHInfo{Exposed: ok, NodePort: port, VMIP: info.IP}
	return out, nil
}

func (s *Service) resolveDeployNetworks(tenantID string, publicIP bool, networkIDs []string) ([]string, error) {
	ids := append([]string{}, networkIDs...)

	if len(ids) == 0 {
		defID, err := s.tenantDefaultNetworkID(tenantID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, defID)
	} else if publicIP && !s.hasIsolatedNetwork(ids) {
		defID, err := s.tenantDefaultNetworkID(tenantID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, defID)
	}

	if publicIP {
		sharedNet, ok := s.store.GetSharedNetwork()
		if !ok {
			return nil, fmt.Errorf("public network is not configured")
		}
		ids = append([]string{sharedNet.ID}, ids...)
	}
	for _, id := range ids {
		net, ok := s.store.GetNetwork(id)
		if !ok {
			return nil, fmt.Errorf("network not found: %s", id)
		}
		if net.NetworkType != platform.NetworkTypeShared && net.TenantID != tenantID {
			return nil, fmt.Errorf("network not available for tenant")
		}
	}
	return ids, nil
}

func (s *Service) tenantDefaultNetworkID(tenantID string) (string, error) {
	for _, vpc := range s.store.ListVPCs(tenantID) {
		if vpc.Name != branding.DefaultVPCName {
			continue
		}
		for _, net := range s.store.ListNetworks(tenantID) {
			if net.VPCID == vpc.ID && net.Name == "default" {
				return net.ID, nil
			}
		}
	}
	return "", fmt.Errorf("default VPC not provisioned for tenant")
}

func (s *Service) hasIsolatedNetwork(networkIDs []string) bool {
	for _, id := range networkIDs {
		net, ok := s.store.GetNetwork(id)
		if ok && net.NetworkType == platform.NetworkTypeIsolated {
			return true
		}
	}
	return false
}

func (s *Service) validatePublicSecurityGroups(tenantID string, sgIDs []string) error {
	if len(sgIDs) == 0 {
		return fmt.Errorf("security group required for public IP access")
	}
	for _, id := range sgIDs {
		sg, ok := s.store.GetSG(id)
		if !ok || sg.TenantID != tenantID {
			return fmt.Errorf("security group not found")
		}
	}
	return nil
}

func sgLabels(sgIDs []string) map[string]string {
	if len(sgIDs) == 0 {
		return nil
	}
	labels := make(map[string]string, len(sgIDs))
	for _, id := range sgIDs {
		labels[branding.SGPodLabelKey(id)] = "true"
	}
	return labels
}
