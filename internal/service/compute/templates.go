package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service/shared"
)

// CreateVMTemplateInput registers a tenant OS image (container disk or ISO).
type CreateVMTemplateInput struct {
	Name              string
	DisplayName       string
	Description       string
	Image             string // container URL or ISO HTTP URL
	SourceType        string
	OSType            string
	CloudInitUserData string
	ISOVolumeID       string
	ISOSizeGi         int
	BootDiskSizeGi    int
	StorageClass      string
}

func defaultTenantTemplates() []platform.VMTemplate {
	return []platform.VMTemplate{
		{Name: "cirros", DisplayName: "Cirros (demo)", Image: "quay.io/kubevirt/cirros-container-disk-demo", OSType: "linux", SourceType: "container"},
		{Name: "ubuntu-2204", DisplayName: "Ubuntu 22.04", Image: "quay.io/containerdisks/ubuntu:22.04", OSType: "linux", SourceType: "container"},
		{Name: "fedora-39", DisplayName: "Fedora 39", Image: "quay.io/kubevirt/fedora-container-disk-demo", OSType: "linux", SourceType: "container"},
	}
}

func (s *Service) EnsureDefaultTemplates(tenantID string) error {
	hasOwn := false
	for _, t := range s.store.ListVMTemplates(false) {
		if t.TenantID == tenantID {
			hasOwn = true
			break
		}
	}
	if hasOwn {
		return nil
	}
	now := store.Now()
	for _, want := range defaultTenantTemplates() {
		t := want
		t.ID = store.NewID()
		t.TenantID = tenantID
		t.Hypervisor = "KubeVirt"
		t.State = "Active"
		t.ImportState = "ready"
		t.CreatedAt = now
		s.store.SaveVMTemplate(&t)
	}
	return nil
}

func (s *Service) ListVMTemplatesForTenant(tenantID string) []*platform.VMTemplate {
	var out []*platform.VMTemplate
	for _, t := range s.store.ListVMTemplates(true) {
		if t.TenantID == "" || t.TenantID == tenantID {
			out = append(out, t)
		}
	}
	return out
}

func (s *Service) CreateVMTemplate(ctx context.Context, tenantID string, in CreateVMTemplateInput) (*platform.VMTemplate, error) {
	name := shared.SanitizeSlug(in.Name)
	if name == "" {
		return nil, fmt.Errorf("invalid template name")
	}
	sourceType := in.SourceType
	if sourceType == "" {
		sourceType = "container"
	}
	osType := in.OSType
	if osType == "" {
		osType = "linux"
	}
	if sourceType == "container" && strings.TrimSpace(in.Image) == "" && in.ISOVolumeID == "" {
		return nil, fmt.Errorf("image is required")
	}
	if sourceType == "iso" && in.ISOVolumeID == "" && strings.TrimSpace(in.Image) == "" {
		return nil, fmt.Errorf("iso_url or iso_volume_id is required")
	}
	for _, existing := range s.store.ListVMTemplates(false) {
		if existing.TenantID == tenantID && existing.Name == name {
			return nil, fmt.Errorf("template name already exists")
		}
	}
	displayName := in.DisplayName
	if displayName == "" {
		displayName = name
	}

	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}

	t := &platform.VMTemplate{
		ID: store.NewID(), TenantID: tenantID, Name: name, DisplayName: displayName,
		Description: in.Description, Image: strings.TrimSpace(in.Image), SourceType: sourceType,
		OSType: osType, CloudInitUserData: strings.TrimSpace(in.CloudInitUserData),
		ISOVolumeID: in.ISOVolumeID, ISOSizeGi: in.ISOSizeGi, BootDiskSizeGi: in.BootDiskSizeGi,
		StorageClass: in.StorageClass, Hypervisor: "KubeVirt", CreatedAt: store.Now(),
	}

	if sourceType == "iso" {
		t.OSType = "windows"
		if t.ISOVolumeID != "" {
			if vol, ok := s.store.GetVolume(t.ISOVolumeID); !ok || vol.TenantID != tenantID {
				return nil, fmt.Errorf("iso volume not found")
			}
			t.ImportState = "ready"
			t.State = "Active"
		} else {
			t.ImportState = "importing"
			t.State = "Inactive"
			s.store.SaveVMTemplate(t)
			s.startISOImport(tenantID, t.ID, ns, t)
			return t, nil
		}
	} else {
		t.ImportState = "ready"
		t.State = "Active"
	}

	s.store.SaveVMTemplate(t)
	return t, nil
}

func (s *Service) UpdateVMTemplate(tenantID, id, displayName, description, image, sourceType, osType, cloudInit, state string) (*platform.VMTemplate, error) {
	t, ok := s.store.GetVMTemplate(id)
	if !ok || t.TenantID != tenantID {
		return nil, fmt.Errorf("template not found")
	}
	if displayName != "" {
		t.DisplayName = displayName
	}
	t.Description = description
	if image != "" {
		t.Image = strings.TrimSpace(image)
	}
	if sourceType != "" {
		t.SourceType = sourceType
	}
	if osType != "" {
		t.OSType = osType
	}
	t.CloudInitUserData = strings.TrimSpace(cloudInit)
	if state != "" {
		t.State = state
	}
	s.store.SaveVMTemplate(t)
	return t, nil
}

func (s *Service) DeleteVMTemplate(tenantID, id string) error {
	t, ok := s.store.GetVMTemplate(id)
	if !ok || t.TenantID != tenantID {
		return fmt.Errorf("template not found")
	}
	s.store.DeleteVMTemplate(id)
	return nil
}

func (s *Service) resolveTemplate(tenantID, templateID string) (*platform.VMTemplate, error) {
	if templateID == "" {
		return nil, nil
	}
	t, ok := s.store.GetVMTemplate(templateID)
	if !ok {
		return nil, fmt.Errorf("template not found")
	}
	if t.TenantID != "" && t.TenantID != tenantID {
		return nil, fmt.Errorf("template not found")
	}
	if t.State != "Active" {
		return nil, fmt.Errorf("template is not active")
	}
	if strings.EqualFold(t.SourceType, "iso") && t.ImportState != "" && t.ImportState != "ready" {
		return nil, fmt.Errorf("ISO template import not ready (state=%s)", t.ImportState)
	}
	return t, nil
}
