package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/platform"
)

func (h *PlatformHandler) ListSSHKeys(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	keys := h.svc.ListSSHKeys(tid)
	out := make([]map[string]interface{}, 0, len(keys))
	for _, k := range keys {
		out = append(out, publicSSHKey(k))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"ssh_keys": out})
}

// CreateSSHKey generates an Ed25519 keypair. private_key_pem is returned once in
// this response only (same pattern as API key secret) and is never persisted.
func (h *PlatformHandler) CreateSSHKey(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}
	out, err := h.svc.CreateSSHKey(tid, req.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"key":             publicSSHKey(out.Key),
		"private_key_pem": out.PrivateKey,
	})
}

func (h *PlatformHandler) RegisterSSHKey(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name      string `json:"name"`
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.PublicKey == "" {
		http.Error(w, `{"error":"name and public_key required"}`, http.StatusBadRequest)
		return
	}
	key, err := h.svc.RegisterSSHKey(tid, req.Name, req.PublicKey)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"key": publicSSHKey(key)})
}

func (h *PlatformHandler) DeleteSSHKey(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteSSHKey(tid, id); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// publicSSHKey is the list/detail DTO: name, fingerprint, and public_key only.
// Private key material is never included (it is only emitted once on Create).
func publicSSHKey(k *platform.SSHKeyPair) map[string]interface{} {
	if k == nil {
		return nil
	}
	return map[string]interface{}{
		"id": k.ID, "tenant_id": k.TenantID, "name": k.Name,
		"public_key": k.PublicKey, "fingerprint": k.Fingerprint, "created_at": k.CreatedAt,
	}
}
