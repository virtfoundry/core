package config

import (
	"strings"
	"testing"
)

func TestValidate_AllowsInsecureDefaults(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "1")
	cfg := &Config{Security: SecurityConfig{JWTSecret: ""}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected nil under VF_ALLOW_INSECURE_DEFAULTS, got %v", err)
	}
	cfg = &Config{Security: SecurityConfig{JWTSecret: knownJWTSecretDev}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected known default to be allowed under flag, got %v", err)
	}
}

func TestValidate_RejectsEmpty(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	cfg := &Config{Security: SecurityConfig{JWTSecret: ""}}
	err := Validate(cfg)
	if err != ErrJWTSecretRequired {
		t.Fatalf("expected ErrJWTSecretRequired, got %v", err)
	}
}

func TestValidate_RejectsKnownDefaults(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	for _, s := range []string{knownJWTSecretDev, knownJWTSecretExample} {
		cfg := &Config{Security: SecurityConfig{JWTSecret: s}}
		if err := Validate(cfg); err != ErrJWTSecretKnownDefault {
			t.Fatalf("expected ErrJWTSecretKnownDefault for %q, got %v", s, err)
		}
	}
}

func TestValidate_RejectsTooShort(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	cfg := &Config{Security: SecurityConfig{JWTSecret: strings.Repeat("a", MinJWTSecretLen-1)}}
	if err := Validate(cfg); err != ErrJWTSecretTooShort {
		t.Fatalf("expected ErrJWTSecretTooShort, got %v", err)
	}
}

func TestValidate_AcceptsStrongSecret(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	cfg := &Config{Security: SecurityConfig{JWTSecret: strings.Repeat("x", MinJWTSecretLen)}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("expected nil for strong secret, got %v", err)
	}
}

func TestValidateRootPassword_AllowsInsecureDefaults(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "1")
	if err := ValidateRootPassword(""); err != nil {
		t.Fatalf("expected nil under flag, got %v", err)
	}
	if err := ValidateRootPassword(knownRootPasswordLocal); err != nil {
		t.Fatalf("expected known default to be allowed under flag, got %v", err)
	}
}

func TestValidateRootPassword_RejectsEmpty(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	if err := ValidateRootPassword(""); err != ErrRootPasswordRequired {
		t.Fatalf("expected ErrRootPasswordRequired, got %v", err)
	}
}

func TestValidateRootPassword_RejectsKnownDefault(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	for _, p := range []string{knownRootPasswordLocal, "VirtFoundry", "VIRTFOUNDRY"} {
		if err := ValidateRootPassword(p); err != ErrRootPasswordKnownDefault {
			t.Fatalf("expected ErrRootPasswordKnownDefault for %q, got %v", p, err)
		}
	}
}

func TestValidateRootPassword_RejectsTooShort(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	if err := ValidateRootPassword(strings.Repeat("a", MinRootPasswordLen-1)); err != ErrRootPasswordTooShort {
		t.Fatalf("expected ErrRootPasswordTooShort, got %v", err)
	}
}

func TestValidateRootPassword_AcceptsStrongPassword(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	if err := ValidateRootPassword(strings.Repeat("a", MinRootPasswordLen)); err != nil {
		t.Fatalf("expected nil for strong password, got %v", err)
	}
}

func TestGenerateRootPassword(t *testing.T) {
	pw, err := GenerateRootPassword()
	if err != nil {
		t.Fatalf("GenerateRootPassword: %v", err)
	}
	if pw == "" {
		t.Fatal("expected non-empty password")
	}
	if len(pw) < 16 {
		t.Fatalf("expected length >= 16, got %d", len(pw))
	}
	pw2, err := GenerateRootPassword()
	if err != nil {
		t.Fatalf("GenerateRootPassword second call: %v", err)
	}
	if pw == pw2 {
		t.Fatal("expected distinct passwords on each call")
	}
}

func TestAllowInsecureDefaults(t *testing.T) {
	t.Setenv(EnvAllowInsecureDefaults, "")
	if AllowInsecureDefaults() {
		t.Fatal("expected false when env is unset")
	}
	t.Setenv(EnvAllowInsecureDefaults, "1")
	if !AllowInsecureDefaults() {
		t.Fatal("expected true when env=1")
	}
	t.Setenv(EnvAllowInsecureDefaults, "true")
	if AllowInsecureDefaults() {
		t.Fatal("expected false for non-`1` values")
	}
}
