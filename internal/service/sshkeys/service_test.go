package sshkeys

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/virtfoundry/core/internal/platform/store"
)

func TestCreate_PrivateKeyOnceAndNeverStored(t *testing.T) {
	st := store.NewMemory()
	svc := New(st)
	tenantID := store.NewID()

	out, err := svc.Create(tenantID, "lab-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.Key == nil || out.Key.Name != "lab-key" {
		t.Fatalf("key = %+v", out.Key)
	}
	if out.Key.PublicKey == "" || out.Key.Fingerprint == "" {
		t.Fatal("expected public_key and fingerprint on create")
	}
	if !strings.Contains(out.PrivateKey, "BEGIN OPENSSH PRIVATE KEY") {
		snippet := out.PrivateKey
		if len(snippet) > 40 {
			snippet = snippet[:40]
		}
		t.Fatalf("private_key_pem missing OpenSSH PEM, got %q", snippet)
	}

	stored, ok := st.GetSSHKeyPair(out.Key.ID)
	if !ok {
		t.Fatal("expected key persisted")
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		t.Fatalf("marshal stored: %v", err)
	}
	body := string(raw)
	for _, leak := range []string{"private_key", "PRIVATE KEY", "BEGIN OPENSSH"} {
		if strings.Contains(body, leak) {
			t.Fatalf("stored SSH key must not contain %q: %s", leak, body)
		}
	}

	listed := svc.List(tenantID)
	if len(listed) != 1 {
		t.Fatalf("list len = %d, want 1", len(listed))
	}
	listRaw, err := json.Marshal(listed)
	if err != nil {
		t.Fatalf("marshal list: %v", err)
	}
	listBody := string(listRaw)
	for _, leak := range []string{"private_key", "PRIVATE KEY", "BEGIN OPENSSH"} {
		if strings.Contains(listBody, leak) {
			t.Fatalf("list must not contain %q: %s", leak, listBody)
		}
	}
}

func TestRegister_NoPrivateKeyMaterial(t *testing.T) {
	st := store.NewMemory()
	svc := New(st)
	tenantID := store.NewID()

	generated, err := svc.Create(tenantID, "seed")
	if err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	pub := generated.Key.PublicKey
	_ = svc.Delete(tenantID, generated.Key.ID)

	key, err := svc.Register(tenantID, "byo", pub)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	raw, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "private_key") || strings.Contains(string(raw), "PRIVATE KEY") {
		t.Fatalf("register response must not include private material: %s", raw)
	}
}
