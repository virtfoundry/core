package store

import "github.com/virtfoundry/core/internal/platform"

// SeedCatalog inserts default service offerings and VM templates when empty,
// then ensures platform-specific catalog entries (e.g. Windows) exist.
func SeedCatalog(r Repository) error {
	if len(r.ListServiceOfferings(false)) == 0 {
		now := Now()
		defaults := []platform.ServiceOffering{
			{ID: NewID(), Name: "small", DisplayName: "Small (1 vCPU, 1 GiB)", CPU: 1, MemoryMi: 1024, State: "Active", CreatedAt: now},
			{ID: NewID(), Name: "medium", DisplayName: "Medium (2 vCPU, 4 GiB)", CPU: 2, MemoryMi: 4096, State: "Active", CreatedAt: now},
			{ID: NewID(), Name: "large", DisplayName: "Large (4 vCPU, 8 GiB)", CPU: 4, MemoryMi: 8192, State: "Active", CreatedAt: now},
		}
		for i := range defaults {
			r.SaveServiceOffering(&defaults[i])
		}
	}
	if len(r.ListVMTemplates(false)) == 0 {
		now := Now()
		defaults := []platform.VMTemplate{
			{ID: NewID(), Name: "cirros", DisplayName: "Cirros (demo)", Image: "quay.io/kubevirt/cirros-container-disk-demo", OSType: "linux", SourceType: "container", Hypervisor: "KubeVirt", State: "Active", CreatedAt: now},
			{ID: NewID(), Name: "ubuntu-2204", DisplayName: "Ubuntu 22.04", Image: "quay.io/containerdisks/ubuntu:22.04", OSType: "linux", SourceType: "container", Hypervisor: "KubeVirt", State: "Active", CreatedAt: now},
		}
		for i := range defaults {
			r.SaveVMTemplate(&defaults[i])
		}
	}

	now := Now()
	ensureOffering(r, platform.ServiceOffering{
		Name: "windows-large", DisplayName: "Windows Large (4 vCPU, 16 GiB)",
		CPU: 4, MemoryMi: 16384, State: "Active", CreatedAt: now,
	})
	ensureTemplate(r, platform.VMTemplate{
		Name: "windows-server-2022", DisplayName: "Windows Server 2022 Eval",
		Image: "windows-server-2022-eval", OSType: "windows", SourceType: "iso", Hypervisor: "KubeVirt", State: "Active", CreatedAt: now,
	})
	return nil
}

func ensureOffering(r Repository, want platform.ServiceOffering) {
	for _, existing := range r.ListServiceOfferings(false) {
		if existing.Name == want.Name {
			want.ID = existing.ID
			want.CreatedAt = existing.CreatedAt
			want.ExternalUUID = existing.ExternalUUID
			want.ImportSource = existing.ImportSource
			r.SaveServiceOffering(&want)
			return
		}
	}
	if want.ID == "" {
		want.ID = NewID()
	}
	if want.CreatedAt.IsZero() {
		want.CreatedAt = Now()
	}
	r.SaveServiceOffering(&want)
}

func ensureTemplate(r Repository, want platform.VMTemplate) {
	for _, existing := range r.ListVMTemplates(false) {
		if existing.Name == want.Name {
			want.ID = existing.ID
			want.CreatedAt = existing.CreatedAt
			want.ExternalUUID = existing.ExternalUUID
			want.ImportSource = existing.ImportSource
			r.SaveVMTemplate(&want)
			return
		}
	}
	if want.ID == "" {
		want.ID = NewID()
	}
	if want.CreatedAt.IsZero() {
		want.CreatedAt = Now()
	}
	r.SaveVMTemplate(&want)
}
