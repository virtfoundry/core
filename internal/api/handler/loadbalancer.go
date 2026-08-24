package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func (h *PlatformHandler) ListLoadBalancers(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"load_balancers": nonNilSlice(h.svc.ListLoadBalancers(tid))})
}

func (h *PlatformHandler) CreateLoadBalancer(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	lb, err := h.svc.CreateLoadBalancer(r.Context(), tid, req.Name, req.Description)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"load_balancer": lb})
}

func (h *PlatformHandler) GetLoadBalancer(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	lb, err := h.svc.GetLoadBalancer(tid, mux.Vars(r)["id"])
	if err != nil {
		respondError(w, err)
		return
	}
	listeners, _ := h.svc.ListLBListeners(tid, lb.ID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"load_balancer": lb,
		"listeners":     nonNilSlice(listeners),
	})
}

func (h *PlatformHandler) DeleteLoadBalancer(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := h.svc.DeleteLoadBalancer(r.Context(), tid, mux.Vars(r)["id"]); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) CreateLBListener(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Protocol      string `json:"protocol"`
		Port          int    `json:"port"`
		TargetGroupID string `json:"target_group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	l, err := h.svc.CreateLBListener(r.Context(), tid, mux.Vars(r)["id"], req.Protocol, req.Port, req.TargetGroupID)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"listener": l})
}

func (h *PlatformHandler) ListLBListeners(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	listeners, err := h.svc.ListLBListeners(tid, mux.Vars(r)["id"])
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"listeners": nonNilSlice(listeners)})
}

func (h *PlatformHandler) DeleteLBListener(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	vars := mux.Vars(r)
	if err := h.svc.DeleteLBListener(r.Context(), tid, vars["id"], vars["lid"]); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) ListTargetGroups(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"target_groups": nonNilSlice(h.svc.ListTargetGroups(tid))})
}

func (h *PlatformHandler) CreateTargetGroup(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name     string `json:"name"`
		Protocol string `json:"protocol"`
		Port     int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	tg, err := h.svc.CreateTargetGroup(tid, req.Name, req.Protocol, req.Port)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"target_group": tg})
}

func (h *PlatformHandler) GetTargetGroup(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	tg, err := h.svc.GetTargetGroup(tid, mux.Vars(r)["id"])
	if err != nil {
		respondError(w, err)
		return
	}
	targets, _ := h.svc.ListTargets(tid, tg.ID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"target_group": tg,
		"targets":      nonNilSlice(targets),
	})
}

func (h *PlatformHandler) DeleteTargetGroup(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := h.svc.DeleteTargetGroup(r.Context(), tid, mux.Vars(r)["id"]); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) RegisterTarget(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		VMID string `json:"vm_id"`
		Port int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	t, err := h.svc.RegisterTarget(r.Context(), tid, mux.Vars(r)["id"], req.VMID, req.Port)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"target": t})
}

func (h *PlatformHandler) ListTargets(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	targets, err := h.svc.ListTargets(tid, mux.Vars(r)["id"])
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"targets": nonNilSlice(targets)})
}

func (h *PlatformHandler) DeregisterTarget(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	vars := mux.Vars(r)
	if err := h.svc.DeregisterTarget(r.Context(), tid, vars["id"], vars["tid"]); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
