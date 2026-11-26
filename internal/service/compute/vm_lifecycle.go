package compute

import (
	"context"
	"fmt"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/service/shared"
)

func (s *Service) StartVM(ctx context.Context, tenantID, vmName string) (*platform.PlatformVM, error) {
	// CRD-first (core#131): with operatorReconcile, only Instance.spec.powerState.
	// The operator syncs KubeVirt RunStrategy — never patch KV in the same flow.
	if s.operatorReconcile {
		return s.setVMPowerState(ctx, tenantID, vmName, instancePowerRunning)
	}
	if err := s.patchKubeVirtRunStrategy(ctx, tenantID, vmName, true); err != nil {
		return nil, err
	}
	s.invalidateVMListCache(tenantID)
	vm, err := s.GetVM(ctx, tenantID, vmName)
	if err != nil {
		return nil, err
	}
	s.broadcastVM(tenantID, "vm.updated", vm)
	return vm, nil
}

func (s *Service) StopVM(ctx context.Context, tenantID, vmName string) (*platform.PlatformVM, error) {
	if s.operatorReconcile {
		return s.setVMPowerState(ctx, tenantID, vmName, instancePowerHalted)
	}
	if err := s.patchKubeVirtRunStrategy(ctx, tenantID, vmName, false); err != nil {
		return nil, err
	}
	s.invalidateVMListCache(tenantID)
	vm, err := s.GetVM(ctx, tenantID, vmName)
	if err != nil {
		return nil, err
	}
	s.broadcastVM(tenantID, "vm.updated", vm)
	return vm, nil
}

func (s *Service) patchKubeVirtRunStrategy(ctx context.Context, tenantID, vmName string, start bool) error {
	if s.kvBase == nil {
		return nil
	}
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return err
	}
	kv := s.kvBase.WithNamespace(ns)
	if start {
		return kv.StartVM(ctx, vmName)
	}
	return kv.StopVM(ctx, vmName)
}

func (s *Service) DeleteVM(ctx context.Context, tenantID, vmName string) error {
	if s.operatorReconcile {
		return s.deleteVMViaOperator(ctx, tenantID, vmName)
	}
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
	s.invalidateVMListCache(tenantID)
	s.broadcastVMDeleted(tenantID, vmName)
	return nil
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
