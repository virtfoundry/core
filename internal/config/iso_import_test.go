package config

import (
	"strings"
	"testing"
)

func TestApplyISOImportEnvOverridesAllowedHosts(t *testing.T) {
	t.Setenv(EnvISOAllowedHosts, " iso.example.com , *.cdn.example.com ,, ")
	cfg := &Config{}
	cfg.Security.ISOImport.AllowedHosts = []string{"stale.example.com"}

	ApplyISOImportEnv(cfg)

	got := strings.Join(cfg.Security.ISOImport.AllowedHosts, ",")
	if got != "iso.example.com,*.cdn.example.com" {
		t.Fatalf("AllowedHosts = %q", got)
	}
}

func TestApplyISOImportEnvKeepsConfigWhenEnvUnset(t *testing.T) {
	cfg := &Config{}
	cfg.Security.ISOImport.AllowedHosts = []string{"iso.example.com"}

	ApplyISOImportEnv(cfg)

	if len(cfg.Security.ISOImport.AllowedHosts) != 1 || cfg.Security.ISOImport.AllowedHosts[0] != "iso.example.com" {
		t.Fatalf("AllowedHosts = %v, want the YAML value", cfg.Security.ISOImport.AllowedHosts)
	}
	if cfg.Security.ISOImport.DisableHTTPImport {
		t.Fatal("DisableHTTPImport = true, want false")
	}
}

func TestApplyISOImportEnvDisablesHTTPImport(t *testing.T) {
	t.Setenv(EnvISODisableHTTPImport, "1")
	cfg := &Config{}

	ApplyISOImportEnv(cfg)

	if !cfg.Security.ISOImport.DisableHTTPImport {
		t.Fatal("DisableHTTPImport = false, want true")
	}
}
