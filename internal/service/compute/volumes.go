package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/service/shared"
)

func (s *Service) validateUnattachedVolume(tenantID, volumeID string) (*platform.Volume, error) {
	vol, ok := s.store.GetVolume(volumeID)
	if !ok || vol.TenantID != tenantID {
		return nil, fmt.Errorf("volume not found")
	}
	if vol.VMID != "" {
		return nil, fmt.Errorf("volume already attached")
	}
	return vol, nil
}

func (s *Service) markVolumeAttached(vol *platform.Volume, vm *platform.PlatformVM) {
	vol.VMID = vm.ID
	vol.State = "attached"
	s.store.SaveVolume(vol)
}

func (s *Service) markVolumeDetached(vol *platform.Volume) {
	vol.VMID = ""
	vol.State = "ready"
	s.store.SaveVolume(vol)
}

func (s *Service) vmDiskBus(ctx context.Context, ns, vmName string) string {
	kv := s.kvBase.WithNamespace(ns)
	vm, err := kv.GetRawVM(ctx, vmName)
	if err != nil {
		return "virtio"
	}
	if strings.EqualFold(vm.Labels[branding.LabelOS], "windows") {
		return "sata"
	}
	return "virtio"
}

func (s *Service) ListVolumesForVM(tenantID, vmName string) []*platform.Volume {
	vm, ok := s.store.GetVMByName(tenantID, vmName)
	if !ok {
		return nil
	}
	return s.store.ListVolumesByVMID(tenantID, vm.ID)
}

func (s *Service) AttachVolumeToVM(ctx context.Context, tenantID, vmName, volumeID string) (*platform.Volume, error) {
	vol, err := s.validateUnattachedVolume(tenantID, volumeID)
	if err != nil {
		return nil, err
	}
	vm, ok := s.store.GetVMByName(tenantID, vmName)
	if !ok {
		return nil, fmt.Errorf("vm not found")
	}
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	if vol.Namespace != ns {
		return nil, fmt.Errorf("volume namespace mismatch")
	}
	if strings.TrimSpace(vol.PVCName) == "" {
		return nil, fmt.Errorf("volume has no backing PVC")
	}
	bus := s.vmDiskBus(ctx, ns, vmName)
	kv := s.kvBase.WithNamespace(ns)
	if err := kv.AttachVolumeToVM(ctx, vmName, vol.PVCName, bus); err != nil {
		return nil, err
	}
	s.markVolumeAttached(vol, vm)
	return vol, nil
}

func (s *Service) DetachVolumeFromVM(ctx context.Context, tenantID, vmName, volumeID string) (*platform.Volume, error) {
	vol, ok := s.store.GetVolume(volumeID)
	if !ok || vol.TenantID != tenantID {
		return nil, fmt.Errorf("volume not found")
	}
	vm, ok := s.store.GetVMByName(tenantID, vmName)
	if !ok {
		return nil, fmt.Errorf("vm not found")
	}
	if vol.VMID == "" || vol.VMID != vm.ID {
		return nil, fmt.Errorf("volume not attached to this vm")
	}
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	kv := s.kvBase.WithNamespace(ns)
	if err := kv.DetachVolumeFromVM(ctx, vmName, vol.PVCName); err != nil {
		return nil, err
	}
	s.markVolumeDetached(vol)
	return vol, nil
}

func (s *Service) releaseVolumesForVM(tenantID, vmID string) {
	for _, vol := range s.store.ListVolumesByVMID(tenantID, vmID) {
		s.markVolumeDetached(vol)
	}
}
