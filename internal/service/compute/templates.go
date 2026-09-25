package compute

import (
	"context"
	"fmt"
	"strings"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

func looksLikeHTTPURL(image string) bool {
	lower := strings.ToLower(strings.TrimSpace(image))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

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
		{Name: "fedora-40", DisplayName: "Fedora 40", Image: "quay.io/containerdisks/fedora:40", OSType: "linux", SourceType: "container"},
	}
}

func platformTemplateNames(r store.Repository) map[string]struct{} {
	names := make(map[string]struct{})
	for _, t := range r.ListVMTemplates(false) {
		if t.TenantID == "" {
			names[t.Name] = struct{}{}
		}
	}
	return names
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
	platformNames := platformTemplateNames(s.store)
	now := store.Now()
	for _, want := range defaultTenantTemplates() {
		if _, exists := platformNames[want.Name]; exists {
			continue
		}
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
	for _, t := range s.store.ListVMTemplatesForTenant(tenantID, false) {
		// Include inactive templates while ISO CDI import is in progress (or failed).
		if t.State != "Active" && strings.TrimSpace(t.ImportState) == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (s *Service) CreateVMTemplate(ctx context.Context, tenantID string, in CreateVMTemplateInput) (*platform.VMTemplate, error) {
	name := shared.SanitizeSlug(in.Name)
	if name == "" {
		return nil, fmt.Errorf("invalid template name")
	}
	image := strings.TrimSpace(in.Image)
	sourceType := in.SourceType
	if sourceType == "" {
		// http(s) URLs are CDI imports, not container disks — defaulting them to
		// "container" would skip the ISO SSRF allowlist (core#95).
		if looksLikeHTTPURL(image) {
			sourceType = "iso"
		} else {
			sourceType = "container"
		}
	}
	osType := in.OSType
	if osType == "" {
		osType = "linux"
	}
	if sourceType == "container" {
		if image == "" && in.ISOVolumeID == "" {
			return nil, fmt.Errorf("image is required")
		}
		// Container disks are registry references pulled by the kubelet. An
		// http(s) URL here would be a confused-deputy path into CDI later.
		if looksLikeHTTPURL(image) {
			return nil, iaerrors.NewBadRequestError("http(s) image URLs require source_type=iso")
		}
	}
	if sourceType == "iso" && in.ISOVolumeID == "" {
		if image == "" {
			return nil, fmt.Errorf("iso_url or iso_volume_id is required")
		}
		if err := s.validateISOImportURL(image); err != nil {
			return nil, err
		}
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
		Description: in.Description, Image: image, SourceType: sourceType,
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
		trimmed := strings.TrimSpace(image)
		effectiveSource := t.SourceType
		if sourceType != "" {
			effectiveSource = sourceType
		}
		if looksLikeHTTPURL(trimmed) && !strings.EqualFold(effectiveSource, "iso") {
			return nil, iaerrors.NewBadRequestError("http(s) image URLs require source_type=iso")
		}
		if strings.EqualFold(effectiveSource, "iso") {
			if err := s.validateISOImportURL(trimmed); err != nil {
				return nil, err
			}
		}
		t.Image = trimmed
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
