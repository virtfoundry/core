package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
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
	return true
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
		Hypervisor:        "KubeVirt",
		ServiceOfferingID: in.ServiceOfferingID,
		CreatedAt:         store.Now(),
	}
	if tenant != nil {
		vm.Zone = tenant.Slug
	}
	s.store.SaveVM(vm)
	s.invalidateVMListCache(tenantID)
	s.broadcastVM("vm.created", vm)
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
		s.broadcastVM("vm.updated", vm)
		return vm, nil
	}
	s.broadcastVM("vm.updated", merged)
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
	s.broadcastVM("vm.deleted", map[string]string{"tenant_id": tenantID, "name": vmName})
	return nil
}
