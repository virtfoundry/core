package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	EnvAllowInsecureDefaults = "VF_ALLOW_INSECURE_DEFAULTS"

	MinJWTSecretLen    = 32
	MinRootPasswordLen = 12

	knownJWTSecretDev      = "change-me-in-production"
	knownJWTSecretExample  = "dev-secret-change-in-prod"
	knownRootPasswordLocal = "virtfoundry"
)

var (
	ErrJWTSecretRequired        = errors.New("JWT_SECRET is required")
	ErrJWTSecretTooShort        = fmt.Errorf("JWT_SECRET must be at least %d characters of random data", MinJWTSecretLen)
	ErrJWTSecretKnownDefault    = errors.New("JWT_SECRET is a known default; generate a random secret (e.g. `openssl rand -base64 32`) and inject it via env or Kubernetes Secret")
	ErrRootPasswordRequired     = errors.New("ROOT_PASSWORD is required (min length); the server can also generate a one-time secret at first boot, or set VF_ALLOW_INSECURE_DEFAULTS=1 for local dev")
	ErrRootPasswordTooShort     = fmt.Errorf("ROOT_PASSWORD must be at least %d characters", MinRootPasswordLen)
	ErrRootPasswordKnownDefault = fmt.Errorf("ROOT_PASSWORD cannot be %q (known default); set a real password or use VF_ALLOW_INSECURE_DEFAULTS=1", knownRootPasswordLocal)
)

var knownJWTSecrets = map[string]struct{}{
	knownJWTSecretDev:     {},
	knownJWTSecretExample: {},
}

func AllowInsecureDefaults() bool {
	return os.Getenv(EnvAllowInsecureDefaults) == "1"
}

func Validate(cfg *Config) error {
	if AllowInsecureDefaults() {
		return nil
	}
	secret := cfg.Security.JWTSecret
	if secret == "" {
		return ErrJWTSecretRequired
	}
	if _, isDefault := knownJWTSecrets[secret]; isDefault {
		return ErrJWTSecretKnownDefault
	}
	if len(secret) < MinJWTSecretLen {
		return ErrJWTSecretTooShort
	}
	return nil
}

func ValidateRootPassword(password string) error {
	if AllowInsecureDefaults() {
		return nil
	}
	if password == "" {
		return ErrRootPasswordRequired
	}
	if strings.EqualFold(password, knownRootPasswordLocal) {
		return ErrRootPasswordKnownDefault
	}
	if len(password) < MinRootPasswordLen {
		return ErrRootPasswordTooShort
	}
	return nil
}

func GenerateRootPassword() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate root password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
