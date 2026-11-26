package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

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
		if snap.VMName == "" {
			snap.VMName = info.VMName
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
