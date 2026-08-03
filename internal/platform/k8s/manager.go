package k8s

import (
	"fmt"

	"github.com/virtfoundry/core/internal/platform/branding"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubevirt "kubevirt.io/client-go/kubecli"
)

const (
	LabelManagedBy = branding.LabelManagedBy
	LabelTenantID  = branding.LabelTenantID
	LabelVPCID     = branding.LabelVPCID
	ManagedByValue = branding.ManagedByValue
)

type Manager struct {
	Clientset kubernetes.Interface
	Virt      kubevirt.KubevirtClient
	Dynamic   dynamic.Interface
	Config    *rest.Config
}

type Options struct {
	Kubeconfig string
	InCluster  bool
}

func NewManager(opts Options) (*Manager, error) {
	var cfg *rest.Config
	var err error

	if opts.InCluster {
		cfg, err = rest.InClusterConfig()
	} else {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		if opts.Kubeconfig != "" {
			loadingRules.ExplicitPath = opts.Kubeconfig
		}
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}).ClientConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("kubeconfig: %w", err)
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	virt, err := kubevirt.GetKubevirtClientFromRESTConfig(cfg)
	if err != nil {
		return nil, err
	}

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Manager{Clientset: cs, Virt: virt, Dynamic: dyn, Config: cfg}, nil
}

func TenantNamespace(slug string) string {
	return branding.TenantNamespacePrefix + slug
}

func VPCNamespace(tenantSlug, vpcID string) string {
	return fmt.Sprintf("%s%s-%s", branding.VPCNamespacePrefix, tenantSlug, vpcID[:8])
}
