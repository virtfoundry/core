package compute

import (
	"sync"

	"github.com/virtfoundry/core/internal/infra/hypervisor"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/importurl"
	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

type vmStateKey struct {
	tenantID string
	name     string
}

// Service manages VMs, VM snapshots and the compute catalog.
type Service struct {
	store  store.Repository
	k8s    *platformk8s.Manager
	kvBase *hypervisor.KubeVirtDriver
	hub    shared.EventBroadcaster

	vmStateMu              sync.Mutex
	vmStates               map[vmStateKey]string
	vmListCacheMu          sync.RWMutex
	vmListCache            map[string]vmListCacheEntry
	allowPodNetwork        bool
	defaultNetwork         string
	storageClass           string
	windowsBootSizeGi      int
	windowsISOSizeGi       int
	operatorReconcile      bool
	isoImport              *importurl.Policy
	containerImagePrefixes []string
}

func New(st store.Repository, k8s *platformk8s.Manager, kv *hypervisor.KubeVirtDriver, hub shared.EventBroadcaster) *Service {
	return &Service{
		store: st, k8s: k8s, kvBase: kv, hub: hub,
		vmStates: make(map[vmStateKey]string), allowPodNetwork: true, defaultNetwork: "pod",
		isoImport: importurl.NewPolicy(nil),
	}
}

func (s *Service) ConfigureVMNetworking(defaultNetwork string, allowPodNetwork bool) {
	if defaultNetwork != "" {
		s.defaultNetwork = defaultNetwork
	}
	s.allowPodNetwork = allowPodNetwork
}

func (s *Service) ConfigureStorage(defaultClass string, bootSizeGi, isoSizeGi int) {
	if defaultClass != "" {
		s.storageClass = defaultClass
	}
	if bootSizeGi > 0 {
		s.windowsBootSizeGi = bootSizeGi
	}
	if isoSizeGi > 0 {
		s.windowsISOSizeGi = isoSizeGi
	}
}

func (s *Service) ListVMTemplates(tenantID string) []*platform.VMTemplate {
	return s.ListVMTemplatesForTenant(tenantID)
}
