package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/service"
	vkssvc "github.com/virtfoundry/core/internal/service/vks"
)

// VKSHandler is the transitional REST shim over internal/service/vks.
type VKSHandler struct {
	platform *service.PlatformService
	vks      *vkssvc.Service
}

func NewVKSHandler(platform *service.PlatformService, vks *vkssvc.Service) *VKSHandler {
	return &VKSHandler{platform: platform, vks: vks}
}

func (h *VKSHandler) ListClusters(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	list, err := h.vks.List(r.Context(), tid)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"clusters": list})
}

func (h *VKSHandler) GetCluster(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	c, err := h.vks.Get(r.Context(), tid, name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"cluster": c})
}

func (h *VKSHandler) CreateCluster(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name              string                   `json:"name"`
		KubernetesVersion string                   `json:"kubernetes_version"`
		ControlPlane      vkssvc.ControlPlaneSpec  `json:"control_plane"`
		Workers           vkssvc.WorkersSpec       `json:"workers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	c, err := h.vks.Create(r.Context(), tid, vkssvc.CreateInput{
		Name:              req.Name,
		KubernetesVersion: req.KubernetesVersion,
		ControlPlane:      req.ControlPlane,
		Workers:           req.Workers,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"cluster": c})
}

func (h *VKSHandler) DeleteCluster(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	if err := h.vks.Delete(r.Context(), tid, name); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleting"})
}

func (h *VKSHandler) GetClusterSummary(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	sum, err := h.vks.GetSummary(r.Context(), tid, name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"summary": sum})
}

func (h *VKSHandler) GetKubeconfig(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	raw, err := h.vks.GetKubeconfig(r.Context(), tid, name)
	if err != nil {
		respondError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.kubeconfig"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (h *VKSHandler) tenantID(r *http.Request) (string, error) {
	claims := middleware.GetClaims(r.Context())
	tid := middleware.GetTenantID(r.Context())
	if tid == "" {
		tid = r.URL.Query().Get("tenant_id")
	}
	return h.platform.ResolveTenantID(claims, tid)
}
