package compute

import (
	"time"

	"github.com/virtfoundry/core/internal/platform"
)

const vmListCacheTTL = 20 * time.Second

type vmListCacheEntry struct {
	vms []*platform.PlatformVM
	at  time.Time
}

func (s *Service) getVMListCache(tenantID string) ([]*platform.PlatformVM, bool) {
	s.vmListCacheMu.RLock()
	defer s.vmListCacheMu.RUnlock()
	entry, ok := s.vmListCache[tenantID]
	if !ok || time.Since(entry.at) > vmListCacheTTL {
		return nil, false
	}
	out := make([]*platform.PlatformVM, len(entry.vms))
	copy(out, entry.vms)
	return out, true
}

func (s *Service) setVMListCache(tenantID string, vms []*platform.PlatformVM) {
	copied := make([]*platform.PlatformVM, len(vms))
	copy(copied, vms)
	s.vmListCacheMu.Lock()
	if s.vmListCache == nil {
		s.vmListCache = make(map[string]vmListCacheEntry)
	}
	s.vmListCache[tenantID] = vmListCacheEntry{vms: copied, at: time.Now()}
	s.vmListCacheMu.Unlock()
}

func (s *Service) invalidateVMListCache(tenantID string) {
	s.vmListCacheMu.Lock()
	delete(s.vmListCache, tenantID)
	s.vmListCacheMu.Unlock()
}

func storeVMsHaveObservedState(vms []*platform.PlatformVM) bool {
	if len(vms) == 0 {
		return false
	}
	for _, vm := range vms {
		if vm.State == "" || vm.State == "Pending" {
			return false
		}
	}
	return true
}

func clonePlatformVMs(vms []*platform.PlatformVM) []*platform.PlatformVM {
	out := make([]*platform.PlatformVM, len(vms))
	for i, vm := range vms {
		if vm == nil {
			continue
		}
		copy := *vm
		out[i] = &copy
	}
	return out
}
