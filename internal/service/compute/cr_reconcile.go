package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/service/shared"
)

const (
	instancePowerRunning = "Running"
	instancePowerHalted  = "Halted"
)

// SetOperatorReconcile enables CR-first VM lifecycle (operator reconciles KubeVirt).
func (s *Service) SetOperatorReconcile(enabled bool) {
	s.operatorReconcile = enabled
}

func (s *Service) canDeployViaOperator(deployTmpl *platform.VMTemplate, in DeployVMInput, networkIDs []string) bool {
	if !s.operatorReconcile {
		return false
	}
	if deployTmpl != nil && strings.EqualFold(deployTmpl.SourceType, "iso") {
		return false
	}
	if in.DataVolumeID != "" || in.PublicIP {
		return false
	}
	// Isolated Multus NICs are CR-first (Instance.spec.nics → operator).
	// Shared/public networks still need the legacy hypervisor path.
	if len(networkIDs) > 0 && !s.networksAllIsolated(networkIDs) {
		return false
	}
	// One-time cloud-init passwords are not on the Instance CR yet — keep
	// those deploys off the CR-first path. SSH keys go via sshKeyRefs.
	if strings.TrimSpace(in.CloudInitPassword) != "" {
		return false
	}
	return true
}

// networksAllIsolated reports whether every ID is an isolated tenant network.
// Unknown / missing IDs are not isolated (refuse CR-first).
func (s *Service) networksAllIsolated(networkIDs []string) bool {
	if s.store == nil || len(networkIDs) == 0 {
		return false
	}
	for _, id := range networkIDs {
		net, ok := s.store.GetNetwork(id)
		if !ok || net.NetworkType != platform.NetworkTypeIsolated {
			return false
		}
	}
	return true
}

// operatorDeployUnsupportedReason explains why a deploy cannot use the single
// CR actuator under operatorReconcile (core#131 — no CreateVM+SaveVM dual-write).
func (s *Service) operatorDeployUnsupportedReason(deployTmpl *platform.VMTemplate, in DeployVMInput, networkIDs []string) string {
	var reasons []string
	if deployTmpl != nil && strings.EqualFold(deployTmpl.SourceType, "iso") {
		reasons = append(reasons, "iso template")
	}
	if in.DataVolumeID != "" {
		reasons = append(reasons, "data_volume_id")
	}
	if in.PublicIP {
		reasons = append(reasons, "public_ip")
	}
	if len(networkIDs) > 0 && !s.networksAllIsolated(networkIDs) {
		reasons = append(reasons, "shared/public networks")
	}
	if strings.TrimSpace(in.CloudInitPassword) != "" {
		reasons = append(reasons, "cloud_init_password")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "unsupported deploy shape")
	}
	return "operator reconcile is enabled: refuse hypervisor dual-write for " +
		strings.Join(reasons, ", ") +
		"; use a CR-first-compatible deploy (SSH key, isolated networks, no public IP/shared networks/iso/password)"
}

func (s *Service) deployVMViaOperator(
	ctx context.Context,
	tenantID string,
	in DeployVMInput,
	name, ns string,
	cpu int,
	memMi int64,
	image string,
	dedicated bool,
	deployTmpl *platform.VMTemplate,
	tmplDisplay string,
	networkIDs []string,
) (*platform.PlatformVM, error) {
	tenant, _ := s.store.GetTenant(tenantID)
	displayName := in.DisplayName
	if displayName == "" {
		displayName = name
	}
	templateRef := ""
	if deployTmpl != nil {
		templateRef = deployTmpl.Name
	}
	var sshKeyRefs []string
	if in.SSHKeyID != "" {
		k, ok := s.store.GetSSHKeyPair(in.SSHKeyID)
		if !ok || k.TenantID != tenantID {
			return nil, iaerrors.NewBadRequestError("ssh_key_id not found in this tenant")
		}
		crName := mapping.SanitizeCRName(k.Name)
		if crName == "" {
			return nil, iaerrors.NewBadRequestError("ssh_key_id has empty name")
		}
		sshKeyRefs = []string{crName}
	}
	vm := &platform.PlatformVM{
		ID:                store.NewID(),
		TenantID:          tenantID,
		Name:              name,
		DisplayName:       displayName,
		Namespace:         ns,
		State:             "Pending",
		PowerState:        instancePowerRunning,
		CPU:               cpu,
		MemoryMi:          memMi,
		Image:             image,
		Template:          firstNonEmpty(tmplDisplay, templateLabel(image)),
		TemplateRef:       templateRef,
		DedicatedCPU:      dedicated,
		SSHKeyRefs:        sshKeyRefs,
		NICs:              s.buildOperatorVMNics(tenantID, networkIDs),
		Hypervisor:        "KubeVirt",
		ServiceOfferingID: in.ServiceOfferingID,
		CreatedAt:         store.Now(),
	}
	// Only Instance override — empty leaves Template.spec.cloudInitUserData to the operator.
	if ud := strings.TrimSpace(in.CloudInitUserData); ud != "" {
		vm.CloudInitUserData = ud
	}
	if tenant != nil {
		vm.Zone = tenant.Slug
	}
	s.store.SaveVM(vm)
	s.invalidateVMListCache(tenantID)
	s.broadcastVM(tenantID, "vm.created", vm)
	return vm, nil
}

// buildOperatorVMNics maps resolved network IDs to Instance Multus NICs.
// Unlike buildVMNetworks, this does not require NAD fields yet — the operator
// resolves NAD from Network status when reconciling Instance.spec.nics.
func (s *Service) buildOperatorVMNics(tenantID string, networkIDs []string) []platform.VMNic {
	if len(networkIDs) == 0 {
		return nil
	}
	nics := make([]platform.VMNic, 0, len(networkIDs))
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
		nics = append(nics, platform.VMNic{
			Name:      ifaceName,
			Type:      "multus",
			NetworkID: net.ID,
		})
	}
	return nics
}

func (s *Service) setVMPowerState(ctx context.Context, tenantID, vmName, power string) (*platform.PlatformVM, error) {
	vm, ok := s.store.GetVMByName(tenantID, vmName)
	if !ok {
		if _, err := s.GetVM(ctx, tenantID, vmName); err != nil {
			return nil, fmt.Errorf("vm not found")
		}
		vm, ok = s.store.GetVMByName(tenantID, vmName)
		if !ok {
			return nil, fmt.Errorf("vm not found")
		}
	}
	vm.PowerState = power
	vm.UpdatedAt = store.Now()
	s.store.SaveVM(vm)
	s.invalidateVMListCache(tenantID)
	merged, err := s.GetVM(ctx, tenantID, vmName)
	if err != nil {
		s.broadcastVM(tenantID, "vm.updated", vm)
		return vm, nil
	}
	s.broadcastVM(tenantID, "vm.updated", merged)
	return merged, nil
}

func (s *Service) deleteVMViaOperator(ctx context.Context, tenantID, vmName string) error {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return err
	}
	_ = ns
	if vm, ok := s.store.GetVMByName(tenantID, vmName); ok {
		for _, nic := range vm.NICs {
			if nic.NetworkID != "" && nic.IP != "" {
				s.store.ReleaseIPAddressByAddress(nic.NetworkID, nic.IP)
			}
		}
		s.releaseVolumesForVM(tenantID, vm.ID)
		s.store.DeleteVM(vm.ID)
	}
	key := vmStateKey{tenantID: tenantID, name: vmName}
	s.vmStateMu.Lock()
	delete(s.vmStates, key)
	s.vmStateMu.Unlock()
	s.invalidateVMListCache(tenantID)
	s.broadcastVMDeleted(tenantID, vmName)
	return nil
}
