package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/platform"
)

func (h *PlatformHandler) ListVPCs(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"vpcs": nonNilSlice(h.svc.ListVPCs(tid))})
}

func (h *PlatformHandler) VPCCIDRPlan(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, h.svc.PlanVPCCIDRs(tid))
}

func (h *PlatformHandler) NetworkCIDRPlan(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	vpcID := r.URL.Query().Get("vpc_id")
	if vpcID == "" {
		http.Error(w, `{"error":"vpc_id is required"}`, http.StatusBadRequest)
		return
	}
	prefix := 24
	if v := r.URL.Query().Get("prefix"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 16 && n <= 28 {
			prefix = n
		}
	}
	plan, err := h.svc.PlanSubnetCIDRs(tid, vpcID, prefix)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, plan)
}

func (h *PlatformHandler) CreateVPC(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
		CIDR string `json:"cidr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	vpc, defNet, err := h.svc.CreateVPCWithDefaultNet(r.Context(), tid, req.Name, req.CIDR)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"vpc": vpc, "default_network": defNet})
}

func (h *PlatformHandler) UpdateVPC(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	vpc, err := h.svc.UpdateVPC(r.Context(), tid, id, req.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"vpc": vpc})
}

func (h *PlatformHandler) DeleteVPC(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteVPC(r.Context(), tid, id); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) ListSecurityGroups(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"security_groups": nonNilSlice(h.svc.ListSecurityGroups(tid))})
}

func (h *PlatformHandler) CreateSecurityGroup(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name        string                       `json:"name"`
		Description string                       `json:"description"`
		VPCID       string                       `json:"vpc_id"`
		Rules       []platform.SecurityGroupRule `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	sg, err := h.svc.CreateSecurityGroup(r.Context(), tid, req.VPCID, req.Name, req.Description, req.Rules)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"security_group": sg})
}

func (h *PlatformHandler) UpdateSecurityGroup(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	var req struct {
		Name        string                       `json:"name"`
		Description string                       `json:"description"`
		Rules       []platform.SecurityGroupRule `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	sg, err := h.svc.UpdateSecurityGroup(r.Context(), tid, id, req.Name, req.Description, req.Rules)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"security_group": sg})
}

func (h *PlatformHandler) DeleteSecurityGroup(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteSecurityGroup(r.Context(), tid, id); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) ListNetworks(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"networks": nonNilSlice(h.svc.ListNetworks(tid))})
}

func (h *PlatformHandler) CreateNetwork(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name   string `json:"name"`
		CIDR   string `json:"cidr"`
		VPCID  string `json:"vpc_id"`
		Prefix int    `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	net, err := h.svc.CreateNetwork(r.Context(), tid, req.VPCID, req.Name, req.CIDR, req.Prefix)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"network": net})
}

func (h *PlatformHandler) UpdateNetwork(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	net, err := h.svc.UpdateNetwork(r.Context(), tid, id, req.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"network": net})
}

func (h *PlatformHandler) DeleteNetwork(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteNetwork(r.Context(), tid, id); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
