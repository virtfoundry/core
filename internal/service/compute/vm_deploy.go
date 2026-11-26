package compute

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/infra/hypervisor"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

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
	CloudInitPassword string // optional one-time guest password; enables password SSH
	CloudInitUserData string // optional deploy userdata; overrides template when non-empty
	DataVolumeID      string
	BootDiskSizeGi    int
	ExposeSSH         bool
	DedicatedCPU      bool     // Guaranteed CPU (request=limit); default shares via KubeVirt ratio
	Tags              []string // user-defined labels applied to the Instance CR
}

// UpdateVMInput patches VM metadata and resources.
type UpdateVMInput struct {
	DisplayName       string
	CPU               int
	MemoryMi          int64
	ServiceOfferingID string
	// Tags is nil when the caller did not include tags in the request; a
	// non-nil value (including an empty slice) replaces the VM's tags.
	Tags *[]string
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
	dedicated := in.DedicatedCPU
	var tmplDisplay, osType, cloudInitExtra string
	var deployTmpl *platform.VMTemplate
	if in.ServiceOfferingID != "" {
		off, ok := s.store.GetServiceOffering(in.ServiceOfferingID)
		if !ok {
			return nil, iaerrors.NewBadRequestError("service_offering_id not found")
		}
		cpu, memMi = off.CPU, off.MemoryMi
		if off.DedicatedCPU {
			dedicated = true
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
			if looksLikeInsecureDefaultUbuntuUserData(cloudInitExtra) {
				// Do not append historical password:ubuntu seed payloads on top of
				// deploy-time SSH keys (duplicate ssh_pwauth would re-enable password SSH).
				cloudInitExtra = ""
			}
			tmplDisplay = tmpl.DisplayName
		}
	}
	if err := requireOperatorTemplate(s.operatorReconcile, deployTmpl); err != nil {
		return nil, err
	}
	// Deploy userdata overrides template (operator Instance > Template; hypervisor CloudInitExtra).
	if ud := strings.TrimSpace(in.CloudInitUserData); ud != "" {
		cloudInitExtra = ud
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

	cloudInitPassword := strings.TrimSpace(in.CloudInitPassword)
	var sshKeys []string
	if in.SSHKeyID != "" {
		k, ok := s.store.GetSSHKeyPair(in.SSHKeyID)
		if !ok || k.TenantID != tenantID {
			return nil, iaerrors.NewBadRequestError("ssh_key_id not found in this tenant")
		}
		sshKeys = []string{k.PublicKey}
	}
	if needsLinuxGuestAuth(osType, deployTmpl) {
		if len(sshKeys) == 0 && cloudInitPassword == "" {
			return nil, iaerrors.NewBadRequestError("linux VM requires ssh_key_id or cloud_init_password")
		}
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
		DedicatedCPU:      dedicated,
		Networks:          netSpecs,
		Labels:            sgLabels(in.SecurityGroupIDs),
		CloudInitExtra:    cloudInitExtra,
		CloudInitPassword: cloudInitPassword,
		CloudInitSSHKeys:  sshKeys,
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

	// ContainerDisk allowlist (issue #134): skip when ISO path cleared Image.
	if img := strings.TrimSpace(spec.Image); img != "" {
		if err := ValidateContainerDiskImage(img, s.containerImagePrefixes); err != nil {
			return nil, iaerrors.NewBadRequestError(err.Error())
		}
	}

	if s.canDeployViaOperator(deployTmpl, in, in.NetworkIDs) {
		return s.deployVMViaOperator(ctx, tenantID, in, name, ns, cpu, memMi, image, dedicated, deployTmpl, tmplDisplay, networkIDs)
	}
	// core#131: with operatorReconcile, never CreateVM + SaveVM in the same flow.
	// Unsupported shapes stay blocked until they are CR-first (one actuator).
	if s.operatorReconcile {
		return nil, iaerrors.NewBadRequestError(s.operatorDeployUnsupportedReason(deployTmpl, in, in.NetworkIDs))
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
		Tags:              in.Tags,
		CreatedAt:         store.Now(),
	}
	if deployTmpl != nil {
		vm.TemplateRef = deployTmpl.Name
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
	_ = s.store.SaveVM(vm) // hypervisor path: VM already exists in KubeVirt
	if in.DataVolumeID != "" {
		if vol, ok := s.store.GetVolume(in.DataVolumeID); ok {
			s.markVolumeAttached(vol, vm)
		}
	}
	s.invalidateVMListCache(tenantID)
	s.broadcastVM(tenantID, "vm.created", vm)
	return vm, nil
}

func (s *Service) UpdateVM(ctx context.Context, tenantID, name string, in UpdateVMInput) (*platform.PlatformVM, error) {
	vm, ok := s.store.GetVMByName(tenantID, name)
	if !ok {
		// The store may not expose this VM by its Instance name (e.g. a stale
		// list cache or a KubeVirt-only legacy VM). GetVM enumerates the
		// tenant, so use its result instead of re-querying the store (which
		// used to return nil and panic on the field access below).
		found, err := s.GetVM(ctx, tenantID, name)
		if err != nil {
			return nil, fmt.Errorf("vm not found")
		}
		vm = found
	}
	if vm == nil {
		return nil, fmt.Errorf("vm not found")
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
	if in.Tags != nil {
		vm.Tags = *in.Tags
	}
	vm.UpdatedAt = store.Now()
	_ = s.store.SaveVM(vm) // hypervisor path: VM already exists in KubeVirt
	s.invalidateVMListCache(tenantID)
	merged, _ := s.GetVM(ctx, tenantID, name)
	if merged != nil {
		s.broadcastVM(tenantID, "vm.updated", merged)
		return merged, nil
	}
	s.broadcastVM(tenantID, "vm.updated", vm)
	return vm, nil
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
	var defaultVPCFound bool
	for _, vpc := range s.store.ListVPCs(tenantID) {
		if vpc.Name != branding.DefaultVPCName {
			continue
		}
		defaultVPCFound = true
		for _, net := range s.store.ListNetworks(tenantID) {
			if net.VPCID == vpc.ID && net.Name == "default" {
				return net.ID, nil
			}
		}
	}
	if defaultVPCFound {
		return "", fmt.Errorf("default network missing for default VPC")
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

func needsLinuxGuestAuth(osType string, tmpl *platform.VMTemplate) bool {
	if strings.EqualFold(osType, "windows") {
		return false
	}
	if tmpl != nil && strings.EqualFold(tmpl.SourceType, "iso") {
		return false
	}
	if tmpl != nil && strings.EqualFold(tmpl.OSType, "windows") {
		return false
	}
	return true
}

func looksLikeInsecureDefaultUbuntuUserData(userData string) bool {
	ud := strings.ToLower(userData)
	return strings.Contains(ud, "password: ubuntu") && strings.Contains(ud, "ssh_pwauth")
}
