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
	if in.DataVolumeID != "" || in.PublicIP || len(networkIDs) > 0 {
		return false
	}
	// One-time cloud-init passwords are not on the Instance CR yet — keep
	// those deploys off the CR-first path. SSH keys go via sshKeyRefs.
	if strings.TrimSpace(in.CloudInitPassword) != "" {
		return false
	}
	return true
}

// operatorDeployUnsupportedReason explains why a deploy cannot use the single
// CR actuator under operatorReconcile (core#131 — no CreateVM+SaveVM dual-write).
func operatorDeployUnsupportedReason(deployTmpl *platform.VMTemplate, in DeployVMInput, networkIDs []string) string {
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
	if len(networkIDs) > 0 {
		reasons = append(reasons, "extra networks")
	}
	if strings.TrimSpace(in.CloudInitPassword) != "" {
		reasons = append(reasons, "cloud_init_password")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "unsupported deploy shape")
	}
	return "operator reconcile is enabled: refuse hypervisor dual-write for " +
		strings.Join(reasons, ", ") +
		"; use a CR-first-compatible deploy (SSH key, no public IP/extra networks/iso/password)"
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
		Hypervisor:        "KubeVirt",
		ServiceOfferingID: in.ServiceOfferingID,
		CreatedAt:         store.Now(),
	}
	if tenant != nil {
		vm.Zone = tenant.Slug
	}
	s.store.SaveVM(vm)
	s.invalidateVMListCache(tenantID)
	s.broadcastVM(tenantID, "vm.created", vm)
	return vm, nil
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
