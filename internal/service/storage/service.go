package storage

import (
	"context"
	"fmt"

	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/service/shared"
)

// Service manages volumes and volume snapshots (PVC / VolumeSnapshot).
type Service struct {
	store         store.Repository
	k8s           *platformk8s.Manager
	storageClass  string
	snapshotClass string
}

func New(st store.Repository, k8s *platformk8s.Manager) *Service {
	return &Service{store: st, k8s: k8s}
}

func (s *Service) ConfigureStorage(defaultClass, snapshotClass string) {
	if defaultClass != "" {
		s.storageClass = defaultClass
	}
	if snapshotClass != "" {
		s.snapshotClass = snapshotClass
	}
}

func (s *Service) CreateVolume(ctx context.Context, tenantID, name string, sizeGi int) (*platform.Volume, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	pvcName := shared.SanitizeSlug(name)
	if _, err := s.k8s.CreatePVC(ctx, ns, pvcName, s.storageClass, sizeGi); err != nil {
		return nil, err
	}
	vol := &platform.Volume{
		ID: store.NewID(), TenantID: tenantID, Name: name, SizeGi: sizeGi,
		Namespace: ns, PVCName: pvcName, State: "ready", CreatedAt: store.Now(),
	}
	s.store.SaveVolume(vol)
	return vol, nil
}

func (s *Service) ListVolumes(tenantID string) []*platform.Volume {
	return s.store.ListVolumes(tenantID)
}

func (s *Service) DeleteVolume(ctx context.Context, tenantID, volumeID string) error {
	vol, ok := s.store.GetVolume(volumeID)
	if !ok || vol.TenantID != tenantID {
		return iaerrors.NewNotFoundError("volume", volumeID)
	}
	if vol.VMID != "" {
		return iaerrors.NewResourceInUseError("volume", "attached to a VM")
	}
	if err := s.k8s.DeletePVC(ctx, vol.Namespace, vol.PVCName); err != nil {
		return fmt.Errorf("delete pvc: %w", err)
	}
	s.store.DeleteVolume(volumeID)
	return nil
}

func (s *Service) CreateSnapshot(ctx context.Context, tenantID, volumeID, name string) (*platform.Snapshot, error) {
	vol, ok := s.store.GetVolume(volumeID)
	if !ok || vol.TenantID != tenantID {
		return nil, fmt.Errorf("volume not found")
	}
	snapName := shared.SanitizeSlug(name)
	created, err := s.k8s.CreateVolumeSnapshot(ctx, vol.Namespace, snapName, vol.PVCName, s.snapshotClass)
	if err != nil {
		return nil, err
	}
	snap := &platform.Snapshot{
		ID: store.NewID(), TenantID: tenantID, VolumeID: volumeID, Name: name,
		Namespace: vol.Namespace, SnapshotUID: string(created.GetUID()),
		State: "creating", CreatedAt: store.Now(),
	}
	s.store.SaveSnapshot(snap)
	if ready, err := s.k8s.WaitVolumeSnapshotReady(ctx, vol.Namespace, snapName, 180); err == nil && ready {
		snap.State = "ready"
		s.store.SaveSnapshot(snap)
	}
	return snap, nil
}

func (s *Service) ListSnapshots(tenantID string) []*platform.Snapshot {
	snaps := s.store.ListSnapshots(tenantID)
	for _, snap := range snaps {
		if snap.State == "ready" {
			continue
		}
		slug := shared.SanitizeSlug(snap.Name)
		if ready, err := s.k8s.VolumeSnapshotReady(context.Background(), snap.Namespace, slug); err == nil && ready {
			snap.State = "ready"
			s.store.SaveSnapshot(snap)
		}
	}
	return snaps
}
