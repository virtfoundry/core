package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func (h *PlatformHandler) ListVolumes(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"volumes": h.svc.ListVolumes(tid)})
}

func (h *PlatformHandler) CreateVolume(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name   string `json:"name"`
		SizeGi int    `json:"size_gi"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.SizeGi <= 0 {
		req.SizeGi = 10
	}
	vol, err := h.svc.CreateVolume(r.Context(), tid, req.Name, req.SizeGi)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"volume": vol})
}

func (h *PlatformHandler) DeleteVolume(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteVolume(r.Context(), tid, id); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) ListVMVolumes(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	respondJSON(w, http.StatusOK, map[string]interface{}{"volumes": nonNilSlice(h.svc.ListVolumesForVM(tid, name))})
}

func (h *PlatformHandler) AttachVolumeToVM(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	var req struct {
		VolumeID string `json:"volume_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.VolumeID == "" {
		http.Error(w, `{"error":"volume_id required"}`, http.StatusBadRequest)
		return
	}
	vol, err := h.svc.AttachVolumeToVM(r.Context(), tid, name, req.VolumeID)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"volume": vol})
}

func (h *PlatformHandler) DetachVolumeFromVM(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	vars := mux.Vars(r)
	vol, err := h.svc.DetachVolumeFromVM(r.Context(), tid, vars["name"], vars["volume_id"])
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"volume": vol})
}

func (h *PlatformHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"snapshots": h.svc.ListSnapshots(tid)})
}

func (h *PlatformHandler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		VolumeID string `json:"volume_id"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	snap, err := h.svc.CreateSnapshot(r.Context(), tid, req.VolumeID, req.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"snapshot": snap})
}
