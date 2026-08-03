package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const APIKeyPrefix = "vfd_live_"

// GenerateAPIKey returns full secret (show once), public prefix, and bcrypt hash.
func GenerateAPIKey() (full, prefix, hash string, err error) {
	var secretBytes [24]byte
	if _, err = rand.Read(secretBytes[:]); err != nil {
		return "", "", "", err
	}
	secret := hex.EncodeToString(secretBytes[:])
	var prefixBytes [4]byte
	if _, err = rand.Read(prefixBytes[:]); err != nil {
		return "", "", "", err
	}
	prefix = hex.EncodeToString(prefixBytes[:])
	full = fmt.Sprintf("%s%s_%s", APIKeyPrefix, prefix, secret)
	sum := sha256.Sum256([]byte(full))
	hashBytes, err := bcrypt.GenerateFromPassword(sum[:], bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", err
	}
	return full, prefix, string(hashBytes), nil
}

// VerifyAPIKey compares a presented key with stored bcrypt hash of SHA256(key).
func VerifyAPIKey(presented, hash string) bool {
	sum := sha256.Sum256([]byte(presented))
	return bcrypt.CompareHashAndPassword([]byte(hash), sum[:]) == nil
}

// ParseAPIKeyPrefix extracts lookup prefix from vfd_live_<prefix>_<secret>.
func ParseAPIKeyPrefix(token string) (prefix string, ok bool) {
	if !strings.HasPrefix(token, APIKeyPrefix) {
		return "", false
	}
	rest := strings.TrimPrefix(token, APIKeyPrefix)
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0], true
}

// IsAPIKeyToken detects VirtFoundry API key bearer tokens.
func IsAPIKeyToken(token string) bool {
	return strings.HasPrefix(token, APIKeyPrefix)
}
