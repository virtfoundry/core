package storage

import (
	"context"
	"fmt"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform/store"
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

func (s *Service) CreateVolume(ctx context.Context, tenantID, name string, sizeGi int, tags []string) (*platform.Volume, error) {
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
		Namespace: ns, PVCName: pvcName, State: "ready", Tags: normalizeTags(tags), CreatedAt: store.Now(),
	}
	s.store.SaveVolume(vol)
	return vol, nil
}

// UpdateVolume applies tag changes to an existing volume. A nil tags pointer
// leaves the current tags untouched (same convention as UpdateVMInput.Tags).
func (s *Service) UpdateVolume(ctx context.Context, tenantID, volumeID string, tags *[]string) (*platform.Volume, error) {
	vol, ok := s.store.GetVolume(volumeID)
	if !ok || vol.TenantID != tenantID {
		return nil, iaerrors.NewNotFoundError("volume", volumeID)
	}
	if tags != nil {
		vol.Tags = normalizeTags(*tags)
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

func (s *Service) CreateSnapshot(ctx context.Context, tenantID, volumeID, name string, tags []string) (*platform.Snapshot, error) {
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
		State: "creating", Tags: normalizeTags(tags), CreatedAt: store.Now(),
	}
	s.store.SaveSnapshot(snap)
	if ready, err := s.k8s.WaitVolumeSnapshotReady(ctx, vol.Namespace, snapName, 180); err == nil && ready {
		snap.State = "ready"
		s.store.SaveSnapshot(snap)
	}
	return snap, nil
}

// UpdateSnapshot applies tag changes to an existing volume snapshot. A nil
// tags pointer leaves the current tags untouched.
func (s *Service) UpdateSnapshot(ctx context.Context, tenantID, snapshotID string, tags *[]string) (*platform.Snapshot, error) {
	var target *platform.Snapshot
	for _, snap := range s.store.ListSnapshots(tenantID) {
		if snap.ID == snapshotID {
			target = snap
			break
		}
	}
	if target == nil {
		return nil, iaerrors.NewNotFoundError("snapshot", snapshotID)
	}
	if tags != nil {
		target.Tags = normalizeTags(*tags)
	}
	s.store.SaveSnapshot(target)
	return target, nil
}

func (s *Service) ListSnapshots(tenantID string) []*platform.Snapshot {
	snaps := s.store.ListSnapshots(tenantID)
	if s.k8s == nil {
		return snaps
	}
	for _, snap := range snaps {
		if snap.State == "ready" {
			continue
		}
		slug := shared.SanitizeSlug(snap.Name)
		if ready, err := s.k8s.VolumeSnapshotReady(context.Background(), snap.Namespace, slug); err == nil && ready {
			// Readiness is response-only: do not write-on-read via SaveSnapshot.
			snap.State = "ready"
		}
	}
	return snaps
}

// normalizeTags trims whitespace and drops empty entries, matching the shared
// VM tags convention.
func normalizeTags(tags []string) []string {
	return shared.NormalizeTags(tags)
}
