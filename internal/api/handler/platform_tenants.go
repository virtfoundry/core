package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func (h *PlatformHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"tenants": h.svc.ListTenants()})
}

func (h *PlatformHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		AdminPassword string `json:"admin_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	tenant, user, err := h.svc.CreateTenant(r.Context(), req.Name, req.Slug, req.AdminPassword)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"tenant": tenant, "admin_user": publicUser(user)})
}

func (h *PlatformHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteTenant(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "id": id})
}
