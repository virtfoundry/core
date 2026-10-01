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
	Count       int32           `json:"count"`
	TemplateRef LocalObjectRef  `json:"template_ref"`
	OfferingRef LocalObjectRef  `json:"offering_ref"`
	NetworkRef  LocalObjectRef  `json:"network_ref"`
	SSHKeyRefs  []LocalObjectRef `json:"ssh_key_refs,omitempty"`
}

// CreateInput is the validated create payload (mirrors CR + gRPC CreateClusterRequest).
type CreateInput struct {
	Name              string
	KubernetesVersion string
	ControlPlane      ControlPlaneSpec
	Workers           WorkersSpec
}
