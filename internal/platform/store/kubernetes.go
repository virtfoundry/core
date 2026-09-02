package store

import (
	"fmt"
	"os"
	"sync"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var _ Repository = (*Kubernetes)(nil)

// Kubernetes persists platform state in virtfoundry.io CRDs.
type Kubernetes struct {
	dyn       dynamic.Interface
	clientset kubernetes.Interface

	mu            sync.RWMutex
	jobs          map[string]*platform.AsyncJob
	targetGroups  map[string]*platform.TargetGroup
	loadBalancers map[string]*platform.LoadBalancer
	lbListeners   map[string]*platform.LBListener
	lbTargets     map[string]*platform.LBTarget
	auditEvents   []*platform.AuditEvent
	tenantCacheMu sync.RWMutex
	tenantCache   *cachedTenants
}

// KubernetesOptions configures the CRD-backed store.
type KubernetesOptions struct {
	Kubeconfig  string
	InCluster   bool
	RESTConfig  *rest.Config
	Dynamic     dynamic.Interface
	Clientset   kubernetes.Interface
}

// NewKubernetes opens a Repository backed by virtfoundry.io CRDs.
func NewKubernetes(opts KubernetesOptions) (*Kubernetes, error) {
	cfg := opts.RESTConfig
	var err error
	if cfg == nil {
		if opts.InCluster || os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			cfg, err = rest.InClusterConfig()
		} else {
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			if opts.Kubeconfig != "" {
				loadingRules.ExplicitPath = opts.Kubeconfig
			}
			cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}).ClientConfig()
		}
		if err != nil {
			return nil, fmt.Errorf("kubernetes store config: %w", err)
		}
	}

	dyn := opts.Dynamic
	if dyn == nil {
		dyn, err = dynamic.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("kubernetes store dynamic client: %w", err)
		}
	}

	cs := opts.Clientset
	if cs == nil {
		cs, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("kubernetes store clientset: %w", err)
		}
	}

	return &Kubernetes{
		dyn:           dyn,
		clientset:     cs,
		jobs:          make(map[string]*platform.AsyncJob),
		targetGroups:  make(map[string]*platform.TargetGroup),
		loadBalancers: make(map[string]*platform.LoadBalancer),
		lbListeners:   make(map[string]*platform.LBListener),
		lbTargets:     make(map[string]*platform.LBTarget),
		auditEvents:   make([]*platform.AuditEvent, 0),
	}, nil
}

func (k *Kubernetes) Close() error { return nil }

func (k *Kubernetes) systemNS() string {
	return mapping.SystemNamespace
}

func (k *Kubernetes) SeedIAM() error {
	if err := SeedIAM(k); err != nil {
		return err
	}
	return nil
}
