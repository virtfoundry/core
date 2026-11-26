package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/infra/hypervisor"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

func (s *Service) GetVM(ctx context.Context, tenantID, name string) (*platform.PlatformVM, error) {
	// CRD-first, cache cold: ListVMs is store-backed, so a direct lookup is equivalent.
	if _, cached := s.getVMListCache(tenantID); !cached && s.operatorReconcile {
		stored, ok := s.store.GetVMByName(tenantID, name)
		if !ok || stored == nil {
			return nil, fmt.Errorf("vm not found")
		}
		vms := clonePlatformVMs([]*platform.PlatformVM{stored})
		s.enrichVMsFromCatalog(vms)
		return vms[0], nil
	}
	vms, err := s.ListVMs(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, vm := range vms {
		if vm.Name == name {
			return vm, nil
		}
	}
	return nil, fmt.Errorf("vm not found")
}

func (s *Service) ListVMs(ctx context.Context, tenantID string) ([]*platform.PlatformVM, error) {
	if vms, ok := s.getVMListCache(tenantID); ok {
		return vms, nil
	}

	stored := s.store.ListVMs(tenantID)
	if s.operatorReconcile || storeVMsHaveObservedState(stored) {
		vms := clonePlatformVMs(stored)
		s.enrichVMsFromCatalog(vms)
		s.setVMListCache(tenantID, vms)
		return vms, nil
	}

	vms, err := s.listVMsFromKubeVirt(ctx, tenantID)
	if err != nil {
		if len(stored) > 0 {
			fallback := clonePlatformVMs(stored)
			s.enrichVMsFromCatalog(fallback)
			return fallback, nil
		}
		return nil, err
	}
	s.enrichVMsFromCatalog(vms)
	s.setVMListCache(tenantID, vms)
	return vms, nil
}

// enrichVMsFromCatalog fills CPU/memory/template display from Offering/Template
// refs. Instance CRs store offeringRef/templateRef names but not guest sizing,
// so operator-reconcile ListVMs would otherwise show 0 vCPU / 0 MiB.
func (s *Service) enrichVMsFromCatalog(vms []*platform.PlatformVM) {
	for _, vm := range vms {
		if vm == nil {
			continue
		}
		if (vm.CPU == 0 || vm.MemoryMi == 0) && vm.ServiceOfferingID != "" {
			if off := s.lookupOffering(vm.ServiceOfferingID); off != nil {
				if vm.CPU == 0 {
					vm.CPU = off.CPU
				}
				if vm.MemoryMi == 0 {
					vm.MemoryMi = off.MemoryMi
				}
			}
		}
		if vm.Template == "" && vm.TemplateRef != "" {
			if tmpl := s.lookupTemplate(vm.TemplateRef); tmpl != nil {
				if tmpl.DisplayName != "" {
					vm.Template = tmpl.DisplayName
				} else {
					vm.Template = tmpl.Name
				}
			} else {
				vm.Template = vm.TemplateRef
			}
		}
	}
}

func (s *Service) lookupOffering(idOrName string) *platform.ServiceOffering {
	if off, ok := s.store.GetServiceOffering(idOrName); ok {
		return off
	}
	if off, ok := s.store.GetServiceOfferingByName(idOrName); ok {
		return off
	}
	return nil
}

func (s *Service) lookupTemplate(idOrName string) *platform.VMTemplate {
	if tmpl, ok := s.store.GetVMTemplate(idOrName); ok {
		return tmpl
	}
	for _, tmpl := range s.store.ListVMTemplates(false) {
		if tmpl != nil && (tmpl.Name == idOrName || tmpl.ID == idOrName) {
			return tmpl
		}
	}
	return nil
}

func (s *Service) listVMsFromKubeVirt(ctx context.Context, tenantID string) ([]*platform.PlatformVM, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	kv := s.kvBase.WithNamespace(ns)
	infos, err := kv.ListVMs(ctx)
	if err != nil {
		return nil, err
	}

	tenant, _ := s.store.GetTenant(tenantID)
	byName := map[string]*platform.PlatformVM{}
	for _, stored := range s.store.ListVMs(tenantID) {
		byName[stored.Name] = stored
	}

	out := make([]*platform.PlatformVM, 0, len(infos))
	for _, info := range infos {
		vm, ok := byName[info.Name]
		if !ok {
			vm = &platform.PlatformVM{
				ID: store.NewID(), TenantID: tenantID, Name: info.Name, Namespace: ns,
				CreatedAt: info.Created,
			}
		}
		s.applyVMInfo(vm, info, tenant)
		out = append(out, vm)
	}
	return out, nil
}

func (s *Service) SyncAllVMStates(ctx context.Context) {
	for _, tenant := range s.store.ListTenants() {
		var vms []*platform.PlatformVM
		if s.operatorReconcile {
			// core#132: read Instance CR phase/status honestly — no KubeVirt
			// short-circuit, then broadcast when signature changes.
			vms = clonePlatformVMs(s.store.ListVMs(tenant.ID))
		} else {
			listed, err := s.listVMsFromKubeVirt(ctx, tenant.ID)
			if err != nil {
				continue
			}
			vms = listed
		}
		s.enrichVMsFromCatalog(vms)
		s.setVMListCache(tenant.ID, vms)
		for _, vm := range vms {
			key := vmStateKey{tenantID: tenant.ID, name: vm.Name}
			sig := vmStateSignature(vm)
			s.vmStateMu.Lock()
			prev, ok := s.vmStates[key]
			if !ok || prev != sig {
				s.vmStates[key] = sig
				s.vmStateMu.Unlock()
				if ok {
					s.broadcastVM(tenant.ID, "vm.updated", vm)
				}
				continue
			}
			s.vmStateMu.Unlock()
		}
	}
}

func (s *Service) ReconcileAll(ctx context.Context) {
	for _, tenant := range s.store.ListTenants() {
		ns, err := shared.TenantNamespace(s.store, tenant.ID)
		if err != nil {
			continue
		}
		kv := s.kvBase.WithNamespace(ns)
		infos, err := kv.ListVMs(ctx)
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, info := range infos {
			seen[info.Name] = true
			vm, ok := s.store.GetVMByName(tenant.ID, info.Name)
			if !ok {
				vm = &platform.PlatformVM{
					ID: store.NewID(), TenantID: tenant.ID, Name: info.Name,
					Namespace: ns, CreatedAt: info.Created,
				}
			}
			s.applyVMInfo(vm, info, tenant)
			_ = s.store.SaveVM(vm) // hypervisor path: VM already exists in KubeVirt
		}
		for _, stored := range s.store.ListVMs(tenant.ID) {
			if !seen[stored.Name] && stored.State != "Destroyed" {
				stored.State = "Destroyed"
				stored.UpdatedAt = store.Now()
				_ = s.store.SaveVM(stored)
				s.broadcastVM(tenant.ID, "vm.updated", stored)
			}
		}
		s.invalidateVMListCache(tenant.ID)
	}
}

// vmEvent is the realtime payload for VM events. It carries only id/name/state
// so the stream stays thin — namespace, IPs, host, cpu/memory and NICs stay out
// of WebSocket (inventory remains on REST). UI CN Phase 1 consumers merge-patch
// the list cache from these fields and may one-shot GET on vm.created when the
// row is incomplete.
type vmEvent struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	State string `json:"state,omitempty"`
}

func (s *Service) broadcastVM(tenantID, eventType string, vm *platform.PlatformVM) {
	if s.hub == nil || vm == nil {
		return
	}
	s.hub.BroadcastTenant(tenantID, eventType, vmEvent{ID: vm.ID, Name: vm.Name, State: vm.State})
}

func (s *Service) broadcastVMDeleted(tenantID, vmName string) {
	if s.hub == nil {
		return
	}
	s.hub.BroadcastTenant(tenantID, "vm.deleted", vmEvent{Name: vmName})
}

func (s *Service) applyVMInfo(vm *platform.PlatformVM, info hypervisor.VMInfo, tenant *platform.Tenant) {
	priorNics := make(map[string]platform.VMNic, len(vm.NICs))
	for _, nic := range vm.NICs {
		priorNics[nic.Name] = nic
	}

	storedPublicIP := ""
	for _, nic := range vm.NICs {
		if nic.Name == "public" && nic.IP != "" {
			storedPublicIP = nic.IP
		}
	}
	if storedPublicIP == "" {
		if existing, ok := s.store.GetVMByName(vm.TenantID, vm.Name); ok {
			for _, nic := range existing.NICs {
				if nic.Name == "public" && nic.IP != "" {
					storedPublicIP = nic.IP
				}
				if _, seen := priorNics[nic.Name]; !seen {
					priorNics[nic.Name] = nic
				}
			}
			if storedPublicIP == "" && strings.HasPrefix(existing.IP, "10.0.50.") {
				storedPublicIP = existing.IP
			}
		}
	}
	if storedPublicIP == "" && strings.HasPrefix(vm.IP, "10.0.50.") {
		storedPublicIP = vm.IP
	}

	vm.State = info.State
	vm.IP = info.IP
	if storedPublicIP != "" && (vm.IP == "" || strings.HasPrefix(vm.IP, "10.233.")) {
		vm.IP = storedPublicIP
	}
	vm.ErrorMsg = info.ErrorMsg
	vm.CPU = info.CPU
	vm.MemoryMi = info.MemoryMi
	if info.Image != "" {
		vm.Image = info.Image
		vm.Template = templateLabel(info.Image)
	}
	vm.HostName = info.NodeName
	vm.Hypervisor = "KubeVirt"
	if tenant != nil {
		vm.Zone = tenant.Slug
	}
	vm.NICs = nil
	merged := map[string]bool{}
	for _, n := range info.NICs {
		nic := platform.VMNic{Name: n.Name, IP: n.IP, MAC: n.MAC, Type: n.Type}
		if prior, ok := priorNics[n.Name]; ok {
			if nic.IP == "" {
				nic.IP = prior.IP
			}
			nic.NetworkID = prior.NetworkID
			nic.NADNamespace = prior.NADNamespace
			nic.NADName = prior.NADName
		}
		vm.NICs = append(vm.NICs, nic)
		merged[n.Name] = true
	}
	for name, prior := range priorNics {
		if !merged[name] {
			vm.NICs = append(vm.NICs, prior)
		}
	}
	if storedPublicIP == "" {
		for _, nic := range vm.NICs {
			if nic.Name == "public" && nic.IP != "" {
				storedPublicIP = nic.IP
			}
		}
	}
	if storedPublicIP != "" && (vm.IP == "" || strings.HasPrefix(vm.IP, "10.233.")) {
		vm.IP = storedPublicIP
	}
	if vm.DisplayName == "" {
		vm.DisplayName = vm.Name
	}
	vm.UpdatedAt = store.Now()
}

func vmStateSignature(vm *platform.PlatformVM) string {
	return vm.State + "|" + vm.IP + "|" + vm.ErrorMsg
}

func templateLabel(image string) string {
	if i := strings.LastIndex(image, "/"); i >= 0 {
		image = image[i+1:]
	}
	if i := strings.Index(image, ":"); i >= 0 {
		image = image[:i]
	}
	return image
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
