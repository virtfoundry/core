package sshkeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service/shared"
	"golang.org/x/crypto/ssh"
)

type CreateResult struct {
	Key        *platform.SSHKeyPair `json:"key"`
	PrivateKey string             `json:"private_key_pem"`
}

type Service struct {
	store store.Repository
}

func New(st store.Repository) *Service {
	return &Service{store: st}
}

func (s *Service) List(tenantID string) []*platform.SSHKeyPair {
	return s.store.ListSSHKeyPairs(tenantID)
}

func (s *Service) Get(tenantID, id string) (*platform.SSHKeyPair, bool) {
	k, ok := s.store.GetSSHKeyPair(id)
	if !ok || k.TenantID != tenantID {
		return nil, false
	}
	return k, true
}

func (s *Service) nameTaken(tenantID, name string) bool {
	for _, k := range s.store.ListSSHKeyPairs(tenantID) {
		if k.Name == name {
			return true
		}
	}
	return false
}

func (s *Service) Create(tenantID, name string) (*CreateResult, error) {
	name = shared.SanitizeSlug(name)
	if name == "" {
		return nil, fmt.Errorf("invalid key name")
	}
	if s.nameTaken(tenantID, name) {
		return nil, fmt.Errorf("ssh key name already exists")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("ssh public key: %w", err)
	}
	pubLine := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	fp := sha256Fingerprint(sshPub)

	key := &platform.SSHKeyPair{
		ID:          store.NewID(),
		TenantID:    tenantID,
		Name:        name,
		PublicKey:   pubLine,
		Fingerprint: fp,
		CreatedAt:   store.Now(),
	}
	s.store.SaveSSHKeyPair(key)

	privPEM, err := marshalOpenSSHPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return &CreateResult{Key: key, PrivateKey: privPEM}, nil
}

func (s *Service) Register(tenantID, name, publicKey string) (*platform.SSHKeyPair, error) {
	name = shared.SanitizeSlug(name)
	if name == "" {
		return nil, fmt.Errorf("invalid key name")
	}
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" {
		return nil, fmt.Errorf("public key required")
	}
	if s.nameTaken(tenantID, name) {
		return nil, fmt.Errorf("ssh key name already exists")
	}
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}
	pubLine := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
	fp := sha256Fingerprint(pub)

	key := &platform.SSHKeyPair{
		ID:          store.NewID(),
		TenantID:    tenantID,
		Name:        name,
		PublicKey:   pubLine,
		Fingerprint: fp,
		CreatedAt:   store.Now(),
	}
	s.store.SaveSSHKeyPair(key)
	return key, nil
}

func (s *Service) Delete(tenantID, id string) error {
	k, ok := s.Get(tenantID, id)
	if !ok {
		return fmt.Errorf("ssh key not found")
	}
	s.store.DeleteSSHKeyPair(k.ID)
	return nil
}

func (s *Service) PublicKeyLine(tenantID, id string) (string, error) {
	k, ok := s.Get(tenantID, id)
	if !ok {
		return "", fmt.Errorf("ssh key not found")
	}
	return k.PublicKey, nil
}

func sha256Fingerprint(pub ssh.PublicKey) string {
	sum := sha256.Sum256(pub.Marshal())
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

func marshalOpenSSHPrivateKey(priv ed25519.PrivateKey) (string, error) {
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(block)), nil
}
