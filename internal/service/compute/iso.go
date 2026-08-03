package compute

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

const isoImportTimeout = 45 * time.Minute

func (s *Service) resolveISOPVC(tenantID string, tmpl *platform.VMTemplate) (string, error) {
	if tmpl.ISOVolumeID != "" {
		vol, ok := s.store.GetVolume(tmpl.ISOVolumeID)
		if !ok || vol.TenantID != tenantID {
			return "", fmt.Errorf("ISO volume not found for template")
		}
		return vol.PVCName, nil
	}
	if tmpl.ImportState != "" && tmpl.ImportState != "ready" {
		return "", fmt.Errorf("ISO import not ready (state=%s)", tmpl.ImportState)
	}
	dvName := templateISOName(tmpl.Name)
	return dvName, nil
}

func templateISOName(templateName string) string {
	return shared.SanitizeSlug(templateName) + "-iso"
}

func (s *Service) provisionWindowsDisks(ctx context.Context, ns, vmName string, tmpl *platform.VMTemplate, isoPVC string) (bootPVC string, err error) {
	storageClass := tmpl.StorageClass
	if storageClass == "" {
		storageClass = s.storageClass
	}
	bootSize := tmpl.BootDiskSizeGi
	if bootSize <= 0 {
		bootSize = s.windowsBootSizeGi
	}
	if bootSize <= 0 {
		bootSize = 32
	}

	bootPVC = vmName + "-boot"
	if err := s.k8s.CreateBlankDataVolume(ctx, ns, bootPVC, storageClass, bootSize); err != nil {
		return "", fmt.Errorf("create boot disk: %w", err)
	}
	if err := s.k8s.WaitDataVolumeReady(ctx, ns, bootPVC, 5*time.Minute); err != nil {
		return "", fmt.Errorf("wait boot disk: %w", err)
	}
	if isoPVC == "" {
		return "", fmt.Errorf("ISO PVC not available")
	}
	if phase, err := s.k8s.GetDataVolumePhase(ctx, ns, isoPVC); err == nil && phase != "" && phase != "Succeeded" && phase != "Ready" {
		if err := s.k8s.WaitDataVolumeReady(ctx, ns, isoPVC, isoImportTimeout); err != nil {
			return "", fmt.Errorf("wait ISO import: %w", err)
		}
	}
	return bootPVC, nil
}

func (s *Service) startISOImport(tenantID, templateID, ns string, tmpl *platform.VMTemplate) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), isoImportTimeout)
		defer cancel()

		dvName := templateISOName(tmpl.Name)
		isoSize := tmpl.ISOSizeGi
		if isoSize <= 0 {
			isoSize = s.windowsISOSizeGi
		}
		if isoSize <= 0 {
			isoSize = 8
		}
		storageClass := tmpl.StorageClass
		if storageClass == "" {
			storageClass = s.storageClass
		}

		t, ok := s.store.GetVMTemplate(templateID)
		if !ok {
			return
		}
		t.ImportState = "importing"
		s.store.SaveVMTemplate(t)

		if err := s.k8s.CreateHTTPImportDataVolume(ctx, ns, dvName, tmpl.Image, storageClass, isoSize); err != nil {
			s.markISOImportFailed(templateID, err)
			return
		}
		if err := s.k8s.WaitDataVolumeReady(ctx, ns, dvName, isoImportTimeout); err != nil {
			s.markISOImportFailed(templateID, err)
			return
		}

		vol := &platform.Volume{
			ID: store.NewID(), TenantID: tenantID, Name: dvName + "-iso",
			SizeGi: isoSize, Namespace: ns, PVCName: dvName,
			State: "ready", CreatedAt: store.Now(),
		}
		s.store.SaveVolume(vol)

		t, ok = s.store.GetVMTemplate(templateID)
		if !ok {
			return
		}
		t.ISOVolumeID = vol.ID
		t.ImportState = "ready"
		t.State = "Active"
		s.store.SaveVMTemplate(t)
	}()
}

func (s *Service) markISOImportFailed(templateID string, err error) {
	t, ok := s.store.GetVMTemplate(templateID)
	if !ok {
		return
	}
	t.ImportState = "failed"
	t.State = "Inactive"
	if t.Description == "" {
		t.Description = err.Error()
	} else {
		t.Description = strings.TrimSpace(t.Description + " — import failed: " + err.Error())
	}
	s.store.SaveVMTemplate(t)
}
