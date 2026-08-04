package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
	"github.com/virtfoundry/core/internal/service/identity"
)

type IAMHandler struct {
	store store.Repository
	svc   *service.PlatformService
}

func NewIAMHandler(st store.Repository, svc *service.PlatformService) *IAMHandler {
	return &IAMHandler{store: st, svc: svc}
}

func (h *IAMHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	users := h.svc.ListUsers(tid)
	out := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		out = append(out, publicUser(u))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"users": out})
}

func (h *IAMHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		RoleID   string `json:"role_id"`
		RoleName string `json:"role_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	actor := middleware.GetActor(r.Context())
	u, err := h.svc.CreateUser(tid, identity.CreateUserInput{
		Username: req.Username, Password: req.Password, Email: req.Email,
		RoleID: req.RoleID, RoleName: req.RoleName,
	}, actor)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"user": publicUser(u)})
}

func (h *IAMHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	var req struct {
		Email  string `json:"email"`
		RoleID string `json:"role_id"`
		State  string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	u, err := h.svc.UpdateUser(tid, id, req.Email, req.RoleID, req.State)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"user": publicUser(u)})
}

func (h *IAMHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := h.svc.DeleteUser(tid, mux.Vars(r)["id"]); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *IAMHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"roles": h.svc.ListRoles(tid)})
}

func (h *IAMHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	role, err := h.svc.CreateRole(tid, identity.CreateRoleInput{
		Name: req.Name, Description: req.Description, Permissions: req.Permissions,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"role": role})
}

func (h *IAMHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	role, err := h.svc.UpdateRole(tid, mux.Vars(r)["id"], req.Description, req.Permissions)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"role": role})
}

func (h *IAMHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := h.svc.DeleteRole(tid, mux.Vars(r)["id"]); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *IAMHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	actor := middleware.GetActor(r.Context())
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	admin := auth.HasPermission(actor.Permissions, auth.PermUsersWrite)
	keys := h.svc.ListAPIKeys(actor.UserID, tid, admin)
	out := make([]map[string]interface{}, 0, len(keys))
	for _, k := range keys {
		out = append(out, publicAPIKey(k))
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"api_keys": out})
}

func (h *IAMHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	actor := middleware.GetActor(r.Context())
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name          string   `json:"name"`
		ExpiresInDays int      `json:"expires_in_days"`
		Scopes        []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	res, err := h.svc.CreateAPIKey(actor.UserID, tid, identity.CreateAPIKeyInput{
		Name: req.Name, ExpiresInDays: req.ExpiresInDays, Scopes: req.Scopes,
	}, actor)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"api_key": publicAPIKey(res.Key),
		"secret":  res.Secret,
	})
}

func (h *IAMHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	actor := middleware.GetActor(r.Context())
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	admin := auth.HasPermission(actor.Permissions, auth.PermUsersWrite)
	if err := h.svc.RevokeAPIKey(actor.UserID, mux.Vars(r)["id"], admin); err != nil {
		respondError(w, err)
		return
	}
	_ = tid
	w.WriteHeader(http.StatusNoContent)
}

func (h *IAMHandler) tenantID(r *http.Request) (string, error) {
	claims := middleware.GetClaims(r.Context())
	tid := middleware.GetTenantID(r.Context())
	if tid == "" {
		tid = r.URL.Query().Get("tenant_id")
	}
	return h.svc.ResolveTenantID(claims, tid)
}

func publicAPIKey(k *platform.APIKey) map[string]interface{} {
	if k == nil {
		return nil
	}
	return map[string]interface{}{
		"id": k.ID, "user_id": k.UserID, "tenant_id": k.TenantID, "name": k.Name,
		"prefix": k.Prefix, "scopes": k.Scopes, "expires_at": k.ExpiresAt,
		"last_used_at": k.LastUsedAt, "revoked_at": k.RevokedAt, "created_at": k.CreatedAt,
	}
}
