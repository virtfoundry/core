package vks

// Cluster is the domain view of a virtfoundry.io VKSCluster.
type Cluster struct {
	Name                 string           `json:"name"`
	TenantID             string           `json:"tenant_id"`
	Namespace            string           `json:"namespace"`
	KubernetesVersion    string           `json:"kubernetes_version"`
	ControlPlane         ControlPlaneSpec `json:"control_plane"`
	Workers              WorkersSpec      `json:"workers"`
	Phase                string           `json:"phase,omitempty"`
	ControlPlaneEndpoint string           `json:"control_plane_endpoint,omitempty"`
	ReadyWorkers         int32            `json:"ready_workers,omitempty"`
	KubeconfigSecretRef  string           `json:"kubeconfig_secret_ref,omitempty"`
	TCPNamespace         string           `json:"tcp_namespace,omitempty"`
	TCPName              string           `json:"tcp_name,omitempty"`
	CreatedAt            string           `json:"created_at,omitempty"`
	Conditions           []Condition      `json:"conditions,omitempty"`
}

// Condition mirrors metav1.Condition for the console (GKE-style status).
type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type ControlPlaneSpec struct {
	ServiceType string `json:"service_type,omitempty"`
	Address     string `json:"address,omitempty"`
	Port        int32  `json:"port,omitempty"`
}

type LocalObjectRef struct {
	Name string `json:"name"`
}

type WorkersSpec struct {
	Count       int32            `json:"count"`
	TemplateRef LocalObjectRef   `json:"template_ref"`
	OfferingRef LocalObjectRef   `json:"offering_ref"`
	NetworkRef  LocalObjectRef   `json:"network_ref"`
	SSHKeyRefs  []LocalObjectRef `json:"ssh_key_refs,omitempty"`
}

// CreateInput is the validated create payload (mirrors CR + gRPC CreateClusterRequest).
type CreateInput struct {
	Name              string
	KubernetesVersion string
	ControlPlane      ControlPlaneSpec
	Workers           WorkersSpec
}

// Summary is the guest-cluster overview (namespaces, pods, nodes) for the console.
type Summary struct {
	Namespaces []NamespaceSummary `json:"namespaces"`
	PodTotals  PodTotals          `json:"pod_totals"`
	GuestNodes []GuestNode        `json:"guest_nodes"`
	Message    string             `json:"message,omitempty"`
}

// NamespaceSummary is a guest namespace with its pod count.
type NamespaceSummary struct {
	Name     string `json:"name"`
	PodCount int32  `json:"pod_count"`
}

// PodTotals aggregates pods across all guest namespaces by phase.
type PodTotals struct {
	Running int32 `json:"running"`
	Pending int32 `json:"pending"`
	Failed  int32 `json:"failed"`
	Other   int32 `json:"other"`
}

// GuestNode is a guest cluster node and its Ready condition.
type GuestNode struct {
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
}
