package config

import (
	"strings"
	"testing"
)

func TestApplyContainerImageAllowlistEnvOverridesPrefixes(t *testing.T) {
	t.Setenv(EnvAllowedContainerImagePrefixes, " registry.homelab/vf/ , quay.io/mirror/ ,, ")
	cfg := &Config{}
	cfg.Security.ContainerImageAllowlist.AllowedPrefixes = []string{"stale.example/"}

	ApplyContainerImageAllowlistEnv(cfg)

	got := strings.Join(cfg.Security.ContainerImageAllowlist.AllowedPrefixes, ",")
	if got != "registry.homelab/vf/,quay.io/mirror/" {
		t.Fatalf("AllowedPrefixes = %q", got)
	}
}

func TestApplyContainerImageAllowlistEnvKeepsConfigWhenEnvUnset(t *testing.T) {
	t.Setenv(EnvAllowedContainerImagePrefixes, "")
	cfg := &Config{}
	cfg.Security.ContainerImageAllowlist.AllowedPrefixes = []string{"registry.homelab/vf/"}

	ApplyContainerImageAllowlistEnv(cfg)

	if len(cfg.Security.ContainerImageAllowlist.AllowedPrefixes) != 1 ||
		cfg.Security.ContainerImageAllowlist.AllowedPrefixes[0] != "registry.homelab/vf/" {
		t.Fatalf("AllowedPrefixes = %v, want the YAML value", cfg.Security.ContainerImageAllowlist.AllowedPrefixes)
	}
}
