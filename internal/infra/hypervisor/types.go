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
	Start             bool
	Networks          []VMNetworkSpec
	CloudInitSSHKeys  []string
	CloudInitPassword string
	FormatDataDisk    bool
}

// VMNetworkSpec describes a NIC backed by pod network or Multus NAD.
type VMNetworkSpec struct {
	Name         string
	NADNamespace string
	NADName      string
	Default      bool // use pod masquerade network
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
	Name       string
	Hypervisor string
	State      string
	NodeCount  int
	CPUCount   int64
	MemoryTotal int64
}

// VMSnapshotInfo represents a KubeVirt VirtualMachineSnapshot.
type VMSnapshotInfo struct {
	Name      string
	Namespace string
	VMName    string
	Phase     string
	Created   time.Time
}
