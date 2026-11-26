package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/service"
)

func (h *PlatformHandler) ListServiceOfferings(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("include_inactive") != "true"
	respondJSON(w, http.StatusOK, map[string]interface{}{"service_offerings": h.svc.ListServiceOfferings(activeOnly)})
}

func (h *PlatformHandler) CreateServiceOffering(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		DisplayName  string `json:"display_name"`
		CPU          int    `json:"cpu"`
		MemoryMi     int64  `json:"memory_mi"`
		DedicatedCPU bool   `json:"dedicated_cpu"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	off, err := h.svc.CreateServiceOffering(service.CreateServiceOfferingInput{
		Name: req.Name, DisplayName: req.DisplayName, CPU: req.CPU, MemoryMi: req.MemoryMi,
		DedicatedCPU: req.DedicatedCPU,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"service_offering": off})
}

func (h *PlatformHandler) UpdateServiceOffering(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req struct {
		DisplayName  string `json:"display_name"`
		CPU          int    `json:"cpu"`
		MemoryMi     int64  `json:"memory_mi"`
		State        string `json:"state"`
		DedicatedCPU *bool  `json:"dedicated_cpu"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	off, err := h.svc.UpdateServiceOffering(id, service.UpdateServiceOfferingInput{
		DisplayName: req.DisplayName, CPU: req.CPU, MemoryMi: req.MemoryMi, State: req.State,
		DedicatedCPU: req.DedicatedCPU,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"service_offering": off})
}

func (h *PlatformHandler) DeleteServiceOffering(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteServiceOffering(id); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) ListVMTemplates(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"vm_templates": h.svc.ListVMTemplates(tid)})
}

func (h *PlatformHandler) CreateVMTemplate(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name              string `json:"name"`
		DisplayName       string `json:"display_name"`
		Description       string `json:"description"`
		Image             string `json:"image"`
		SourceType        string `json:"source_type"`
		OSType            string `json:"os_type"`
		CloudInitUserData string `json:"cloud_init_user_data"`
		ISOVolumeID       string `json:"iso_volume_id"`
		ISOSizeGi         int    `json:"iso_size_gi"`
		BootDiskSizeGi    int    `json:"boot_disk_size_gi"`
		StorageClass      string `json:"storage_class"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	tmpl, err := h.svc.CreateVMTemplate(r.Context(), tid, service.CreateVMTemplateInput{
		Name: req.Name, DisplayName: req.DisplayName, Description: req.Description,
		Image: req.Image, SourceType: req.SourceType, OSType: req.OSType,
		CloudInitUserData: req.CloudInitUserData, ISOVolumeID: req.ISOVolumeID,
		ISOSizeGi: req.ISOSizeGi, BootDiskSizeGi: req.BootDiskSizeGi, StorageClass: req.StorageClass,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"vm_template": tmpl})
}

func (h *PlatformHandler) UpdateVMTemplate(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	var req struct {
		DisplayName       string `json:"display_name"`
		Description       string `json:"description"`
		Image             string `json:"image"`
		SourceType        string `json:"source_type"`
		OSType            string `json:"os_type"`
		CloudInitUserData string `json:"cloud_init_user_data"`
		State             string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	tmpl, err := h.svc.UpdateVMTemplate(tid, id, req.DisplayName, req.Description, req.Image, req.SourceType, req.OSType, req.CloudInitUserData, req.State)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"vm_template": tmpl})
}

func (h *PlatformHandler) DeleteVMTemplate(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteVMTemplate(tid, id); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
