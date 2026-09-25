package config

import (
	"os"
	"testing"
)

func TestApplyAllowedOriginsEnvOverridesYAML(t *testing.T) {
	t.Setenv(EnvAllowedOrigins, "https://ui.a.test, https://ui.b.test")
	cfg := &Config{Security: SecurityConfig{AllowedOrigins: []string{"https://old.test"}}}
	ApplyAllowedOriginsEnv(cfg)
	want := []string{"https://ui.a.test", "https://ui.b.test"}
	if len(cfg.Security.AllowedOrigins) != len(want) {
		t.Fatalf("origins = %#v, want %#v", cfg.Security.AllowedOrigins, want)
	}
	for i := range want {
		if cfg.Security.AllowedOrigins[i] != want[i] {
			t.Fatalf("origins[%d] = %q, want %q", i, cfg.Security.AllowedOrigins[i], want[i])
		}
	}
}

func TestApplyAllowedOriginsEnvEmptyKeepsYAML(t *testing.T) {
	os.Unsetenv(EnvAllowedOrigins)
	cfg := &Config{Security: SecurityConfig{AllowedOrigins: []string{"https://ui.test"}}}
	ApplyAllowedOriginsEnv(cfg)
	if len(cfg.Security.AllowedOrigins) != 1 || cfg.Security.AllowedOrigins[0] != "https://ui.test" {
		t.Fatalf("origins changed unexpectedly: %#v", cfg.Security.AllowedOrigins)
	}
}
