package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
)

func newSSHKeyTestHandler(t *testing.T) (*PlatformHandler, string) {
	t.Helper()
	st := store.NewMemory()
	tenantID := store.NewID()
	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "t", Slug: "t", Namespace: "ns-t", State: "active",
	})
	svc := service.NewPlatformService(st, nil, nil, nil)
	h := NewPlatformHandler(auth.NewService("test-secret", 3600), st, svc, nil)
	return h, tenantID
}

func withSSHTenantClaims(r *http.Request, tenantID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ContextClaims, &auth.Claims{
		Role: platform.RoleTenantAdmin, TenantID: tenantID,
	})
	ctx = context.WithValue(ctx, middleware.ContextTenant, tenantID)
	return r.WithContext(ctx)
}

func TestCreateSSHKey_PrivateKeyOnlyOnce(t *testing.T) {
	h, tid := newSSHKeyTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ssh-keys", strings.NewReader(`{"name":"once"}`))
	req = withSSHTenantClaims(req, tid)
	rec := httptest.NewRecorder()
	h.CreateSSHKey(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}

	var created map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	var pem string
	if err := json.Unmarshal(created["private_key_pem"], &pem); err != nil || pem == "" {
		t.Fatalf("expected private_key_pem on create, got %v", created["private_key_pem"])
	}
	if !strings.Contains(pem, "BEGIN OPENSSH PRIVATE KEY") {
		snippet := pem
		if len(snippet) > 48 {
			snippet = snippet[:48]
		}
		t.Fatalf("unexpected pem: %q", snippet)
	}
	keyJSON := string(created["key"])
	for _, leak := range []string{"private_key", "PRIVATE KEY", "BEGIN OPENSSH"} {
		if strings.Contains(keyJSON, leak) {
			t.Fatalf("create key DTO must not embed private material (%s): %s", leak, keyJSON)
		}
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/ssh-keys", nil)
	listReq = withSSHTenantClaims(listReq, tid)
	listRec := httptest.NewRecorder()
	h.ListSSHKeys(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listRec.Code, listRec.Body.String())
	}
	listBody := listRec.Body.String()
	for _, leak := range []string{"private_key", "PRIVATE KEY", "BEGIN OPENSSH"} {
		if strings.Contains(listBody, leak) {
			t.Fatalf("list must never return private material (%s): %s", leak, listBody)
		}
	}
	if !strings.Contains(listBody, `"name":"once"`) {
		t.Fatalf("list missing key name: %s", listBody)
	}
}

func TestSSHKeys_NoPrivateKeyRefetchRoute(t *testing.T) {
	h, tid := newSSHKeyTestHandler(t)
	out, err := h.svc.CreateSSHKey(tid, "blocked")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/ssh-keys/{id}", h.DeleteSSHKey).Methods(http.MethodDelete)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/ssh-keys/"+out.Key.ID, nil)
	getReq = withSSHTenantClaims(getReq, tid)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /ssh-keys/{id} status = %d, want 405 (no private-key refetch)", getRec.Code)
	}

	privReq := httptest.NewRequest(http.MethodGet, "/api/v1/ssh-keys/"+out.Key.ID+"/private", nil)
	privReq = withSSHTenantClaims(privReq, tid)
	privRec := httptest.NewRecorder()
	r.ServeHTTP(privRec, privReq)
	if privRec.Code == http.StatusOK {
		t.Fatalf("private-key subresource must not exist; got 200: %s", privRec.Body.String())
	}
}

func TestRegisterSSHKey_PublicOnly(t *testing.T) {
	h, tid := newSSHKeyTestHandler(t)
	seed, err := h.svc.CreateSSHKey(tid, "seed")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	pub := seed.Key.PublicKey
	_ = h.svc.DeleteSSHKey(tid, seed.Key.ID)

	body, _ := json.Marshal(map[string]string{"name": "byo", "public_key": pub})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ssh-keys/register", bytes.NewReader(body))
	req = withSSHTenantClaims(req, tid)
	rec := httptest.NewRecorder()
	h.RegisterSSHKey(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "private_key") || strings.Contains(rec.Body.String(), "PRIVATE KEY") {
		t.Fatalf("register must not return private material: %s", rec.Body.String())
	}
}

func TestPublicSSHKey_OmitsSecrets(t *testing.T) {
	m := publicSSHKey(&platform.SSHKeyPair{
		ID: "id", TenantID: "t", Name: "n", PublicKey: "ssh-ed25519 AAAA", Fingerprint: "SHA256:x",
	})
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, "private") {
		t.Fatalf("publicSSHKey leaked private field: %s", s)
	}
	if !strings.Contains(s, `"public_key"`) || !strings.Contains(s, `"fingerprint"`) {
		t.Fatalf("expected public fields: %s", s)
	}
}
