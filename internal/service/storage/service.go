package storage

import (
	"context"
	"fmt"

	platformk8s "github.com/virtforge-cloud/virtforge/internal/platform/k8s"
	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service/shared"
)

// Service manages volumes and volume snapshots (PVC / VolumeSnapshot).
type Service struct {
	store store.Repository
	k8s   *platformk8s.Manager
}

func New(st store.Repository, k8s *platformk8s.Manager) *Service {
	return &Service{store: st, k8s: k8s}
}

func (s *Service) CreateVolume(ctx context.Context, tenantID, name string, sizeGi int) (*platform.Volume, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	pvcName := shared.SanitizeSlug(name)
	if _, err := s.k8s.CreatePVC(ctx, ns, pvcName, "", sizeGi); err != nil {
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

func (s *Service) CreateSnapshot(ctx context.Context, tenantID, volumeID, name string) (*platform.Snapshot, error) {
	vol, ok := s.store.GetVolume(volumeID)
	if !ok || vol.TenantID != tenantID {
		return nil, fmt.Errorf("volume not found")
	}
	snapName := shared.SanitizeSlug(name)
	created, err := s.k8s.CreateVolumeSnapshot(ctx, vol.Namespace, snapName, vol.PVCName, "")
	if err != nil {
		return nil, err
	}
	snap := &platform.Snapshot{
		ID: store.NewID(), TenantID: tenantID, VolumeID: volumeID, Name: name,
		Namespace: vol.Namespace, SnapshotUID: string(created.GetUID()),
		State: "creating", CreatedAt: store.Now(),
	}
	s.store.SaveSnapshot(snap)
	return snap, nil
}

func (s *Service) ListSnapshots(tenantID string) []*platform.Snapshot {
	return s.store.ListSnapshots(tenantID)
}
