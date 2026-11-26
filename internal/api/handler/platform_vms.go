package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/service"
)

func (h *PlatformHandler) ListVMSnapshots(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	snaps, err := h.svc.ListVMSnapshots(r.Context(), tid)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"vm_snapshots": snaps})
}

func (h *PlatformHandler) CreateVMSnapshot(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		VMName string `json:"vm_name"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	snap, err := h.svc.CreateVMSnapshot(r.Context(), tid, req.VMName, req.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"vm_snapshot": snap})
}

func (h *PlatformHandler) DeleteVMSnapshot(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.DeleteVMSnapshot(r.Context(), tid, req.Name); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) RestoreVMSnapshot(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name   string `json:"name"`
		VMName string `json:"vm_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.RestoreVMSnapshot(r.Context(), tid, req.Name, req.VMName); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) ListVMs(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	vms, err := h.svc.ListVMs(r.Context(), tid)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"vms": vms})
}

func (h *PlatformHandler) GetVM(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	vm, err := h.svc.GetVM(r.Context(), tid, name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"vm":        vm,
		"velas_url": h.svc.VMLogExploreURL(tid, name),
	})
}

// parseVMLogTail returns a positive tail line count capped at maxVMLogTail.
func parseVMLogTail(raw string) int64 {
	tail := int64(200)
	if raw == "" {
		return tail
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return tail
	}
	if n > maxVMLogTail {
		return maxVMLogTail
	}
	return n
}

func (h *PlatformHandler) GetVMLogs(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	tail := parseVMLogTail(r.URL.Query().Get("tail"))
	follow := r.URL.Query().Get("follow") == "1"

	ctx := r.Context()
	if follow {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, vmLogFollowMaxAge)
		defer cancel()
	}

	stream, err := h.svc.StreamVMLogs(ctx, tid, name, tail, follow)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, sanitizeClientError(err.Error()), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if follow {
		w.Header().Set("Cache-Control", "no-cache")
	}
	io.Copy(w, stream)
}

func (h *PlatformHandler) UpdateVM(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	var req struct {
		DisplayName       string    `json:"display_name"`
		CPU               int       `json:"cpu"`
		MemoryMi          int64     `json:"memory_mi"`
		ServiceOfferingID string    `json:"service_offering_id"`
		Tags              *[]string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	vm, err := h.svc.UpdateVM(r.Context(), tid, name, service.UpdateVMInput{
		DisplayName:       req.DisplayName,
		CPU:               req.CPU,
		MemoryMi:          req.MemoryMi,
		ServiceOfferingID: req.ServiceOfferingID,
		Tags:              req.Tags,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"vm": vm})
}

func (h *PlatformHandler) DeployVM(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name              string   `json:"name"`
		DisplayName       string   `json:"display_name"`
		Image             string   `json:"image"`
		CPU               int      `json:"cpu"`
		MemoryMi          int64    `json:"memory_mi"`
		ServiceOfferingID string   `json:"service_offering_id"`
		TemplateID        string   `json:"template_id"`
		NetworkIDs        []string `json:"network_ids"`
		PublicIP          bool     `json:"public_ip"`
		SecurityGroupIDs  []string `json:"security_group_ids"`
		SSHKeyID          string   `json:"ssh_key_id"`
		CloudInitPassword string   `json:"cloud_init_password"`
		CloudInitUserData string   `json:"cloud_init_user_data"`
		DataVolumeID      string   `json:"data_volume_id"`
		ExposeSSH         bool     `json:"expose_ssh"`
		DedicatedCPU      bool     `json:"dedicated_cpu"`
		Tags              []string `json:"tags"`
		Async             bool     `json:"async"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	in := service.PlatformDeployVMInput{
		Name: req.Name, DisplayName: req.DisplayName, Image: req.Image,
		CPU: req.CPU, MemoryMi: req.MemoryMi, Start: true,
		ServiceOfferingID: req.ServiceOfferingID, TemplateID: req.TemplateID,
		NetworkIDs: req.NetworkIDs, PublicIP: req.PublicIP, SecurityGroupIDs: req.SecurityGroupIDs,
		SSHKeyID:          req.SSHKeyID,
		CloudInitPassword: req.CloudInitPassword,
		CloudInitUserData: req.CloudInitUserData,
		DataVolumeID:      req.DataVolumeID, ExposeSSH: req.ExposeSSH,
		DedicatedCPU: req.DedicatedCPU,
		Tags:         req.Tags,
	}
	if req.Async {
		payload, _ := json.Marshal(in)
		job := h.svc.EnqueueJob(tid, "deploy_vm", string(payload))
		respondJSON(w, http.StatusAccepted, map[string]interface{}{"job": job})
		return
	}
	vm, err := h.svc.DeployVM(r.Context(), tid, in)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{"vm": vm})
}

func (h *PlatformHandler) StartVM(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	vm, err := h.svc.StartVM(r.Context(), tid, req.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "vm": vm})
}

func (h *PlatformHandler) StopVM(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	vm, err := h.svc.StopVM(r.Context(), tid, req.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "vm": vm})
}

func (h *PlatformHandler) DeleteVM(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := h.svc.DeleteVM(r.Context(), tid, req.Name); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) ExposeVMSSH(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	var req struct {
		NodePort int32 `json:"node_port"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	port, err := h.svc.ExposeVMSSH(r.Context(), tid, name, req.NodePort)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"node_port": port})
}

func (h *PlatformHandler) GetVMSSH(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	info, err := h.svc.GetVMSSH(r.Context(), tid, name)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, info)
}
