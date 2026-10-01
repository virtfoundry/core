package compute

import (
	"fmt"
	"os"
	"strings"
)

// EnvAllowedContainerImagePrefixes overrides the built-in ContainerDisk allowlist
// (comma-separated image reference prefixes). Empty / unset → defaults.
const EnvAllowedContainerImagePrefixes = "VIRTFOUNDRY_ALLOWED_CONTAINER_IMAGE_PREFIXES"

// defaultContainerImagePrefixes are the registries VirtFoundry ships in the
// catalog (cirros demo + quay containerdisks). Operators with a private mirror
// must set VIRTFOUNDRY_ALLOWED_CONTAINER_IMAGE_PREFIXES explicitly — configuring
// the env replaces the built-in list (same model as the ISO allowlist).
var defaultContainerImagePrefixes = []string{
	"quay.io/containerdisks/",
	"quay.io/kubevirt/",
	"ghcr.io/virtfoundry/",
}

// ConfigureContainerImageAllowlist stores an optional prefix override used by
// DeployVM. An empty slice keeps env / built-in defaults.
func (s *Service) ConfigureContainerImageAllowlist(prefixes []string) {
	s.containerImagePrefixes = prefixes
}

// EffectiveContainerImagePrefixes resolves the active allowlist:
// non-empty override → env CSV → built-in defaults.
func EffectiveContainerImagePrefixes(override []string) []string {
	if len(override) > 0 {
		return override
	}
	if env := strings.TrimSpace(os.Getenv(EnvAllowedContainerImagePrefixes)); env != "" {
		return parseContainerImageAllowlist(env)
	}
	return defaultContainerImagePrefixes
}

func parseContainerImageAllowlist(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// ValidateContainerDiskImage rejects ContainerDisk refs that are not on the
// allowlist (issue #134). HTTP(S) URLs must use the ISO/CDI path — never land
// as containerDisk.image.
func ValidateContainerDiskImage(image string, override []string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return fmt.Errorf("container disk image is empty")
	}
	lower := strings.ToLower(image)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return fmt.Errorf("container disk image %q looks like an HTTP(S) URL; use an ISO template / CDI import instead", image)
	}
	allowed := EffectiveContainerImagePrefixes(override)
	for _, prefix := range allowed {
		if prefix != "" && strings.HasPrefix(image, prefix) {
			return nil
		}
	}
	return fmt.Errorf("container disk image %q is not on the allowlist (allowed prefixes: %s)",
		image, strings.Join(allowed, ", "))
}

// BootstrapContainerImageAllowlist returns the effective prefixes for startup
// logging (same resolution as DeployVM validation).
func BootstrapContainerImageAllowlist(override []string) []string {
	return EffectiveContainerImagePrefixes(override)
}
