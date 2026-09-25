package store

import (
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/cloudinit"
)

// SeedCatalog inserts default service offerings and VM templates when empty,
// then ensures platform-specific catalog entries (e.g. Windows) exist.
//
// defaultPassword is only baked into seed Linux templates when the caller
// supplies a non-empty value (config.VM.DefaultPassword /
// VIRTFOUNDRY_VM_DEFAULT_PASSWORD). There is no built-in fallback: an empty
// password leaves CloudInitUserData empty so deploy must supply an SSH key
// (or an explicit one-time password). See issue #97.
func SeedCatalog(r Repository, defaultPassword string) error {
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
			{ID: NewID(), Name: "ubuntu-2204", DisplayName: "Ubuntu 22.04", Image: "quay.io/containerdisks/ubuntu:22.04", OSType: "linux", SourceType: "container", Hypervisor: "KubeVirt", State: "Active", CreatedAt: now, CloudInitUserData: ubuntuDefaultUserData(defaultPassword)},
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
	// Dedicated offerings: Guaranteed QoS (request=limit=cores), no CPU overcommit.
	ensureOffering(r, platform.ServiceOffering{
		Name: "small-dedicated", DisplayName: "Small Dedicated (1 vCPU, 1 GiB)",
		CPU: 1, MemoryMi: 1024, DedicatedCPU: true, State: "Active", CreatedAt: now,
	})
	ensureOffering(r, platform.ServiceOffering{
		Name: "medium-dedicated", DisplayName: "Medium Dedicated (2 vCPU, 4 GiB)",
		CPU: 2, MemoryMi: 4096, DedicatedCPU: true, State: "Active", CreatedAt: now,
	})
	ensureOffering(r, platform.ServiceOffering{
		Name: "large-dedicated", DisplayName: "Large Dedicated (4 vCPU, 8 GiB)",
		CPU: 4, MemoryMi: 8192, DedicatedCPU: true, State: "Active", CreatedAt: now,
	})
	ensureTemplate(r, platform.VMTemplate{
		Name: "windows-server-2022", DisplayName: "Windows Server 2022 Eval",
		Image: "windows-server-2022-eval", OSType: "windows", SourceType: "iso", Hypervisor: "KubeVirt", State: "Active", CreatedAt: now,
	})
	// Strip the historical insecure default (password: ubuntu) from ubuntu-2204
	// templates. Only re-seed password user-data when the operator explicitly
	// configured VIRTFOUNDRY_VM_DEFAULT_PASSWORD / vm.default_password.
	for _, t := range r.ListVMTemplates(false) {
		if t.Name != "ubuntu-2204" {
			continue
		}
		ud := strings.TrimSpace(t.CloudInitUserData)
		if isInsecureDefaultUbuntuUserData(ud) {
			t.CloudInitUserData = ubuntuDefaultUserData(defaultPassword)
			r.SaveVMTemplate(t)
			continue
		}
		if ud == "" && strings.TrimSpace(defaultPassword) != "" {
			t.CloudInitUserData = ubuntuDefaultUserData(defaultPassword)
			r.SaveVMTemplate(t)
		}
	}
	return nil
}

// isInsecureDefaultUbuntuUserData detects the pre-#97 seed payload that baked
// password: ubuntu with ssh_pwauth and NOPASSWD sudo.
func isInsecureDefaultUbuntuUserData(userData string) bool {
	ud := strings.ToLower(userData)
	return strings.Contains(ud, "password: ubuntu") && strings.Contains(ud, "ssh_pwauth")
}

// ubuntuDefaultUserData returns a #cloud-config payload that creates the `ubuntu`
// user with an explicit one-time password, or "" when password is empty (no
// default). Deploy must then supply an SSH key or cloud_init_password.
func ubuntuDefaultUserData(password string) string {
	password = strings.TrimSpace(password)
	if password == "" {
		return ""
	}
	out, err := cloudinit.BuildLinuxUserData(cloudinit.LinuxConfig{Password: password})
	if err != nil {
		return ""
	}
	return out
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
