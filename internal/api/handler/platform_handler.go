package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/virtforge-cloud/virtforge/internal/api/middleware"
	"github.com/virtforge-cloud/virtforge/internal/auth"
	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service"
)

func nonNilSlice[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

type PlatformHandler struct {
	auth  *auth.Service
	store store.Repository
	svc   *service.PlatformService
}

func NewPlatformHandler(authSvc *auth.Service, st store.Repository, svc *service.PlatformService) *PlatformHandler {
	return &PlatformHandler{auth: authSvc, store: st, svc: svc}
}

func (h *PlatformHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	user, ok := h.store.GetUserByUsername(req.Username)
	if !ok || !auth.CheckPassword(user.PasswordHash, req.Password) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}
	token, exp, err := h.auth.IssueToken(user)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token": token, "expires_at": exp,
		"user": map[string]interface{}{
			"id": user.ID, "username": user.Username, "role": user.Role, "tenant_id": user.TenantID,
		},
	})
}

func (h *PlatformHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	user, _ := h.store.GetUser(claims.UserID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id": claims.UserID, "username": claims.Username, "role": claims.Role, "tenant_id": claims.TenantID,
		"email": userEmail(user),
	})
}

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
		Name        string                      `json:"name"`
		Description string                      `json:"description"`
		VPCID       string                      `json:"vpc_id"`
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
		Name        string                      `json:"name"`
		Description string                      `json:"description"`
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

func (h *PlatformHandler) GetVMLogs(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	name := mux.Vars(r)["name"]
	tail := int64(200)
	if v := r.URL.Query().Get("tail"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			tail = n
		}
	}
	follow := r.URL.Query().Get("follow") == "1"

	stream, err := h.svc.StreamVMLogs(r.Context(), tid, name, tail, follow)
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
		DisplayName string `json:"display_name"`
		CPU         int    `json:"cpu"`
		MemoryMi    int64  `json:"memory_mi"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	vm, err := h.svc.UpdateVM(r.Context(), tid, name, service.UpdateVMInput{
		DisplayName: req.DisplayName,
		CPU:         req.CPU,
		MemoryMi:    req.MemoryMi,
	})
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"vm": vm})
}

func (h *PlatformHandler) ListServiceOfferings(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"service_offerings": h.svc.ListServiceOfferings()})
}

func (h *PlatformHandler) ListVMTemplates(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"vm_templates": h.svc.ListVMTemplates()})
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
		DataVolumeID      string   `json:"data_volume_id"`
		ExposeSSH         bool     `json:"expose_ssh"`
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
		SSHKeyID: req.SSHKeyID,
		DataVolumeID: req.DataVolumeID, ExposeSSH: req.ExposeSSH,
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
	var req struct{ Name string `json:"name"` }
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
	var req struct{ Name string `json:"name"` }
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
	var req struct{ Name string `json:"name"` }
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := h.svc.DeleteVM(r.Context(), tid, req.Name); err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *PlatformHandler) ListSSHKeys(w http.ResponseWriter, r *http.Request) {
	tid, err := h.tenantID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"ssh_keys": nonNilSlice(h.svc.ListSSHKeys(tid))})
}

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
	respondJSON(w, http.StatusCreated, out)
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
	respondJSON(w, http.StatusCreated, map[string]interface{}{"key": key})
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

func (h *PlatformHandler) tenantID(r *http.Request) (string, error) {
	claims := middleware.GetClaims(r.Context())
	tid := middleware.GetTenantID(r.Context())
	if tid == "" {
		tid = r.URL.Query().Get("tenant_id")
	}
	return h.svc.ResolveTenantID(claims, tid)
}

func publicUser(u *platform.User) map[string]interface{} {
	return map[string]interface{}{"id": u.ID, "username": u.Username, "role": u.Role, "tenant_id": u.TenantID}
}

func userEmail(u *platform.User) string {
	if u == nil {
		return ""
	}
	return u.Email
}
