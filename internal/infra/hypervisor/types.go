package hypervisor

import "time"

// VMDeploySpec defines parameters for creating a VM.
type VMDeploySpec struct {
	Name       string
	Namespace  string
	CPU        int
	MemoryMi   int64
	Image      string
	OSType     string // linux (default) or windows
	BootPVC    string // windows: blank PVC for OS install
	DataPVC    string // optional second disk (e.g. IOPS tests)
	InstallISO string // windows: PVC/DataVolume with Windows ISO
	Start      bool
	// DedicatedCPU pins Guaranteed QoS (request=limit=cores). Default false lets
	// KubeVirt cpuAllocationRatio overcommit the virt-launcher CPU request.
	DedicatedCPU      bool
	Networks          []VMNetworkSpec
	CloudInitSSHKeys  []string
	CloudInitPassword string
	CloudInitExtra    string // optional #cloud-config fragment merged at deploy
	FormatDataDisk    bool
	Labels            map[string]string // applied to VM and VMI template (e.g. security group)
}

// VMNetworkSpec describes a NIC backed by pod network or Multus NAD.
type VMNetworkSpec struct {
	Name         string
	NADNamespace string
	NADName      string
	Default      bool // use pod masquerade network
	StaticIP     string
	PrefixLen    int
	Gateway      string
	DNS          []string
	MACAddress   string
}

// VMInfo represents a virtual machine in the hypervisor.
type VMInfo struct {
	Name      string
	Namespace string
	State     string
	ErrorMsg  string
	CPU       int
	MemoryMi  int64
	Image     string
	IP        string
	NodeName  string
	NICs      []VMNicInfo
	Created   time.Time
}

type VMNicInfo struct {
	Name string
	IP   string
	MAC  string
	Type string
}

// Volume represents a persistent volume claim.
type Volume struct {
	Name   string
	Size   string
	PVName string
	Status string
}

// NodeInfo represents a Kubernetes node (CloudStack host).
type NodeInfo struct {
	Name         string
	IPAddress    string
	State        string
	CPUCount     int64
	MemoryTotal  int64
	MemoryUsed   int64
	StorageTotal int64
	KubeVersion  string
}

// ClusterInfo represents the KubeVirt cluster.
type ClusterInfo struct {
	Name        string
	Hypervisor  string
	State       string
	NodeCount   int
	CPUCount    int64
	MemoryTotal int64
}

type ClusterMetrics struct {
	Nodes               int       `json:"nodes"`
	NodesReady          int       `json:"nodes_ready"`
	CPUCapacityMilli    int64     `json:"cpu_capacity_millicores"`
	CPUAllocatableMilli int64     `json:"cpu_allocatable_millicores"`
	MemoryCapacity      int64     `json:"memory_capacity_bytes"`
	MemoryAllocatable   int64     `json:"memory_allocatable_bytes"`
	KubeletVersions     []string  `json:"kubelet_versions"`
	OSImages            []string  `json:"os_images"`
	OSArchitectures     []string  `json:"os_architectures"`
	CollectedAt         time.Time `json:"collected_at"`

	// Usage is populated when metrics-server is reachable. Available is false when
	// the API is absent, the call is Forbidden, or the cluster has no Ready nodes
	// to report. The dashboard renders a degraded state instead of failing the
	// summary when Available is false.
	Usage *ClusterUsage `json:"usage,omitempty"`
}

// ClusterUsage is the sum of metrics.k8s.io/v1beta1 NodeMetrics across all
// nodes, served by metrics-server. Tier 2 of the dashboard host metrics spec.
type ClusterUsage struct {
	CPUUsageMilli int64     `json:"cpu_usage_millicores"`
	MemoryUsage   int64     `json:"memory_usage_bytes"`
	WindowSeconds int64     `json:"window_seconds"`
	CollectedAt   time.Time `json:"collected_at"`
}

// StorageSummary aggregates tenant-scoped PVC capacity from the cluster.
// TotalBytes is the sum of every PVC's spec.resources.requests.storage
// (capacity the user has reserved); UsedBytes is the subset of those PVCs
// whose status.phase is Bound (actually attached to a pod). AvailableBytes is
// Total - Used and is informational — there is no global "free" pool in K8s.
type StorageSummary struct {
	TotalBytes     int64 `json:"total_bytes"`
	UsedBytes      int64 `json:"used_bytes"`
	AvailableBytes int64 `json:"available_bytes"`
	Count          int   `json:"count"`
}

// AddonHealth is the status of one critical VirtFoundry dependency on the
// cluster (KubeVirt, CDI, Multus, metrics-server, networking). The dashboard
// renders a single coloured dot per addon.
type AddonHealth struct {
	// Name is a stable identifier ("kubevirt", "cdi", "multus",
	// "metrics-server", "networking") used by the UI for i18n lookup.
	Name string `json:"name"`
	// Status is one of: "ok" (installed and reachable), "absent" (CRD/API
	// not registered on the cluster), "unknown" (Forbidden or transient
	// failure — the UI shows a neutral dot in this case).
	Status string `json:"status"`
	// Detail is a short human-readable reason, sanitised server-side. It is
	// only populated when Status != "ok".
	Detail string `json:"detail,omitempty"`
}

// AddonsHealth is the per-check summary of every addon the dashboard tracks.
// CheckedAt lets the UI show how stale the dots are.
type AddonsHealth struct {
	Addons    []AddonHealth `json:"addons"`
	CheckedAt time.Time     `json:"checked_at"`
}

// VMSnapshotInfo represents a KubeVirt VirtualMachineSnapshot.
type VMSnapshotInfo struct {
	Name      string
	Namespace string
	VMName    string
	Phase     string
	Created   time.Time
}
