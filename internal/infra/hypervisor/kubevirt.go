package hypervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/platform/cloudinit"
	"github.com/virtfoundry/core/internal/platform/branding"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
)

const (
	defaultVMImage          = "quay.io/kubevirt/cirros-container-disk-demo"
	virtioContainerDisk     = "quay.io/kubevirt/virtio-container-disk:v1.8.4"
	windowsMachineType      = "q35"
)

// Cirros only drives the default VGA device; virtio-gpu yields a black VNC screen.
func videoDeviceForImage(image string) *kubevirtv1.VideoDevice {
	if strings.Contains(strings.ToLower(image), "cirros") {
		return nil
	}
	return &kubevirtv1.VideoDevice{Type: "virtio"}
}

// KubeVirtDriver implements Driver using KubeVirt CRDs.
type KubeVirtDriver struct {
	virtClient kubecli.KubevirtClient
	k8sClient  *kubernetes.Clientset
	namespace  string
}

type KubeVirtConfig struct {
	Kubeconfig string
	Namespace  string
}

func NewKubeVirtDriver(config KubeVirtConfig) (*KubeVirtDriver, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if config.Kubeconfig != "" {
		loadingRules.ExplicitPath = config.Kubeconfig
	}

	clientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("kubeconfig: %w", err)
	}

	virtClient, err := kubecli.GetKubevirtClientFromRESTConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("kubevirt client: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}

	namespace := config.Namespace
	if namespace == "" {
		namespace = "default"
	}

	return &KubeVirtDriver{
		virtClient: virtClient,
		k8sClient:  k8sClient,
		namespace:  namespace,
	}, nil
}

func (d *KubeVirtDriver) WithNamespace(ns string) *KubeVirtDriver {
	if ns == "" {
		return d
	}
	copy := *d
	copy.namespace = ns
	return &copy
}

func (d *KubeVirtDriver) VirtClient() kubecli.KubevirtClient {
	return d.virtClient
}

func (d *KubeVirtDriver) Namespace() string {
	return d.namespace
}

func (d *KubeVirtDriver) ListVMs(ctx context.Context) ([]VMInfo, error) {
	vms, err := d.virtClient.VirtualMachine(d.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list VMs: %w", err)
	}

	result := make([]VMInfo, 0, len(vms.Items))
	for _, vm := range vms.Items {
		info := d.vmToInfo(ctx, &vm)
		result = append(result, info)
	}
	return result, nil
}

func (d *KubeVirtDriver) CreateVM(ctx context.Context, spec VMDeploySpec) error {
	if spec.Name == "" {
		return fmt.Errorf("VM name is required")
	}

	ns := spec.Namespace
	if ns == "" {
		ns = d.namespace
	}

	image := spec.Image
	if image == "" {
		image = defaultVMImage
	}

	cpu := spec.CPU
	if cpu <= 0 {
		cpu = 1
	}

	memMi := spec.MemoryMi
	if memMi <= 0 {
		memMi = 512
	}

	runStrategy := kubevirtv1.RunStrategyHalted
	if spec.Start {
		runStrategy = kubevirtv1.RunStrategyAlways
	}

	networks, ifaces, defaultNet := buildNetworks(spec.Networks)
	annotations := map[string]string{
		"app.kubernetes.io/managed-by": branding.ManagedByValue,
	}
	if defaultNet != "" {
		annotations["v1.multus-cni.io/default-network"] = defaultNet
	}

	var vmiSpec kubevirtv1.VirtualMachineInstanceSpec
	if strings.EqualFold(spec.OSType, "windows") {
		if spec.BootPVC == "" {
			return fmt.Errorf("windows VM requires boot PVC (blank disk)")
		}
		vmiSpec = buildWindowsVMISpec(spec, ifaces, cpu, memMi)
	} else {
		vmiSpec = buildLinuxVMISpec(spec, ifaces, image, cpu, memMi)
	}
	vmiSpec.Networks = networks

	vm := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Namespace:   ns,
			Labels:      mergeLabels(map[string]string{branding.AppManagedByKey: branding.ManagedByValue, branding.LabelOS: strings.ToLower(spec.OSType)}, spec.Labels),
			Annotations: annotations,
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			RunStrategy: &runStrategy,
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergeLabels(map[string]string{
						"kubevirt.io/domain":    spec.Name,
						branding.LabelVM:        spec.Name,
						branding.LabelLogSource: "velas",
					}, spec.Labels),
					Annotations: copyStringMap(annotations),
				},
				Spec: vmiSpec,
			},
		},
	}

	_, err := d.virtClient.VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create VM %s: %w", spec.Name, err)
	}
	return nil
}

// PatchMultusNetworkIPs sets static Multus IPs on the VM/VMI before the launcher pod is created.
func (d *KubeVirtDriver) PatchMultusNetworkIPs(ctx context.Context, name string, networks []VMNetworkSpec) error {
	ann := buildMultusNetworksAnnotation(networks)
	if ann == "" {
		return nil
	}
	ns := d.namespace

	for attempt := 0; attempt < 5; attempt++ {
		vm, err := d.virtClient.VirtualMachine(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get VM for multus patch: %w", err)
		}
		if vm.Spec.Template == nil {
			return fmt.Errorf("vm template missing")
		}
		if vm.Spec.Template.ObjectMeta.Annotations == nil {
			vm.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}
		vm.Spec.Template.ObjectMeta.Annotations["k8s.v1.cni.cncf.io/networks"] = ann
		_, err = d.virtClient.VirtualMachine(ns).Update(ctx, vm, metav1.UpdateOptions{})
		if err == nil {
			break
		}
		if !errors.IsConflict(err) || attempt == 4 {
			return fmt.Errorf("patch VM multus annotation: %w", err)
		}
	}

	vmi, err := d.virtClient.VirtualMachineInstance(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get VMI for multus patch: %w", err)
	}
	for attempt := 0; attempt < 5; attempt++ {
		if vmi.Annotations == nil {
			vmi.Annotations = map[string]string{}
		}
		vmi.Annotations["k8s.v1.cni.cncf.io/networks"] = ann
		_, err = d.virtClient.VirtualMachineInstance(ns).Update(ctx, vmi, metav1.UpdateOptions{})
		if err == nil {
			return nil
		}
		if !errors.IsConflict(err) || attempt == 4 {
			return fmt.Errorf("patch VMI multus annotation: %w", err)
		}
		vmi, err = d.virtClient.VirtualMachineInstance(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get VMI for multus patch retry: %w", err)
		}
	}
	return nil
}

func (d *KubeVirtDriver) StartVM(ctx context.Context, name string) error {
	return d.patchRunStrategy(ctx, name, kubevirtv1.RunStrategyAlways)
}

func (d *KubeVirtDriver) StopVM(ctx context.Context, name string) error {
	return d.patchRunStrategy(ctx, name, kubevirtv1.RunStrategyHalted)
}

func (d *KubeVirtDriver) RebootVM(ctx context.Context, name string) error {
	vmi, err := d.virtClient.VirtualMachineInstance(d.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get VMI: %w", err)
	}
	return d.virtClient.VirtualMachineInstance(d.namespace).Delete(ctx, vmi.Name, metav1.DeleteOptions{})
}

func (d *KubeVirtDriver) DeleteVM(ctx context.Context, name string) error {
	propagation := metav1.DeletePropagationForeground
	err := d.virtClient.VirtualMachine(d.namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete VM: %w", err)
	}
	return nil
}

// UpdateVMResources changes CPU/RAM when the VM is stopped.
func (d *KubeVirtDriver) UpdateVMResources(ctx context.Context, name string, cpu int, memoryMi int64) error {
	for attempt := 0; attempt < 3; attempt++ {
		vm, err := d.virtClient.VirtualMachine(d.namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get VM: %w", err)
		}
		if vm.Spec.Template == nil {
			return fmt.Errorf("vm template missing")
		}
		state := mapVMState(vm)
		if state != "Stopped" && state != "Error" {
			return fmt.Errorf("vm must be stopped to resize (current: %s)", state)
		}
		reqs := vm.Spec.Template.Spec.Domain.Resources.Requests
		if reqs == nil {
			reqs = k8sv1.ResourceList{}
		}
		if cpu > 0 {
			reqs[k8sv1.ResourceCPU] = resourceQuantityCPU(cpu)
		}
		if memoryMi > 0 {
			reqs[k8sv1.ResourceMemory] = resourceQuantityMi(memoryMi)
		}
		vm.Spec.Template.Spec.Domain.Resources.Requests = reqs
		_, err = d.virtClient.VirtualMachine(d.namespace).Update(ctx, vm, metav1.UpdateOptions{})
		if err == nil {
			return nil
		}
		if !errors.IsConflict(err) {
			return fmt.Errorf("update VM resources: %w", err)
		}
	}
	return fmt.Errorf("update VM resources: conflict after retries")
}

func (d *KubeVirtDriver) GetVM(ctx context.Context, name string) (*VMInfo, error) {
	vm, err := d.virtClient.VirtualMachine(d.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get VM: %w", err)
	}
	info := d.vmToInfo(ctx, vm)
	return &info, nil
}

func (d *KubeVirtDriver) ListNodes(ctx context.Context) ([]NodeInfo, error) {
	nodes, err := d.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	result := make([]NodeInfo, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		state := "Down"
		for _, cond := range node.Status.Conditions {
			if cond.Type == k8sv1.NodeReady && cond.Status == k8sv1.ConditionTrue {
				state = "Up"
				break
			}
		}

		ip := ""
		for _, addr := range node.Status.Addresses {
			if addr.Type == k8sv1.NodeInternalIP {
				ip = addr.Address
				break
			}
		}

		cpu := node.Status.Capacity.Cpu().Value()
		memTotal := node.Status.Capacity.Memory().Value()

		result = append(result, NodeInfo{
			Name:        node.Name,
			IPAddress:   ip,
			State:       state,
			CPUCount:    cpu,
			MemoryTotal: memTotal,
			KubeVersion: node.Status.NodeInfo.KubeletVersion,
		})
	}
	return result, nil
}

func (d *KubeVirtDriver) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	nodes, err := d.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	var cpuTotal int64
	var memTotal int64
	upNodes := 0
	for _, n := range nodes {
		cpuTotal += n.CPUCount
		memTotal += n.MemoryTotal
		if n.State == "Up" {
			upNodes++
		}
	}

	state := "Disabled"
	if upNodes > 0 {
		state = "Enabled"
	}

	return &ClusterInfo{
		Name:        branding.KubeVirtSecretName,
		Hypervisor:  "KubeVirt",
		State:       state,
		NodeCount:   len(nodes),
		CPUCount:    cpuTotal,
		MemoryTotal: memTotal,
	}, nil
}

func (d *KubeVirtDriver) GetHostResources(ctx context.Context) (map[string]interface{}, error) {
	cluster, err := d.GetClusterInfo(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"cpu":    cluster.CPUCount,
		"memory": cluster.MemoryTotal,
		"nodes":  cluster.NodeCount,
	}, nil
}

func (d *KubeVirtDriver) CreateVolume(ctx context.Context, name string, size string) (*Volume, error) {
	pvc := &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: d.namespace,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Resources: k8sv1.VolumeResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceStorage: mustParseQuantity(size),
				},
			},
		},
	}

	created, err := d.k8sClient.CoreV1().PersistentVolumeClaims(d.namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create PVC: %w", err)
	}

	return &Volume{
		Name:   created.Name,
		Size:   size,
		PVName: string(created.UID),
		Status: string(created.Status.Phase),
	}, nil
}

func (d *KubeVirtDriver) GetStoragePool(ctx context.Context) (map[string]interface{}, error) {
	pvcs, err := d.k8sClient.CoreV1().PersistentVolumeClaims(d.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var total int64
	var used int64
	for _, pvc := range pvcs.Items {
		req := pvc.Spec.Resources.Requests.Storage()
		if req != nil {
			total += req.Value()
			if pvc.Status.Phase == k8sv1.ClaimBound {
				used += req.Value()
			}
		}
	}

	return map[string]interface{}{
		"capacity":  total,
		"used":      used,
		"available": total - used,
		"count":     len(pvcs.Items),
	}, nil
}

func (d *KubeVirtDriver) patchRunStrategy(ctx context.Context, name string, strategy kubevirtv1.VirtualMachineRunStrategy) error {
	for attempt := 0; attempt < 3; attempt++ {
		vm, err := d.virtClient.VirtualMachine(d.namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get VM: %w", err)
		}

		vm.Spec.RunStrategy = &strategy
		_, err = d.virtClient.VirtualMachine(d.namespace).Update(ctx, vm, metav1.UpdateOptions{})
		if err == nil {
			return nil
		}
		if !errors.IsConflict(err) {
			return fmt.Errorf("update VM runStrategy: %w", err)
		}
	}
	return fmt.Errorf("update VM runStrategy: conflict after retries")
}

func (d *KubeVirtDriver) vmToInfo(ctx context.Context, vm *kubevirtv1.VirtualMachine) VMInfo {
	cpu := 1
	memMi := int64(512)
	image := ""

	if vm.Spec.Template != nil {
		reqs := vm.Spec.Template.Spec.Domain.Resources.Requests
		if cpuQty, ok := reqs[k8sv1.ResourceCPU]; ok {
			cpu = int(cpuQty.Value())
		}
		if memQty, ok := reqs[k8sv1.ResourceMemory]; ok {
			memMi = memQty.Value() / (1024 * 1024)
		}
		for _, vol := range vm.Spec.Template.Spec.Volumes {
			if vol.ContainerDisk != nil && vol.ContainerDisk.Image != "" {
				image = vol.ContainerDisk.Image
				break
			}
		}
	}

	info := VMInfo{
		Name:      vm.Name,
		Namespace: vm.Namespace,
		State:     mapVMState(vm),
		ErrorMsg:  vmErrorMessage(vm),
		CPU:       cpu,
		MemoryMi:  memMi,
		Image:     image,
		Created:   vm.CreationTimestamp.Time,
	}

	vmi, err := d.virtClient.VirtualMachineInstance(vm.Namespace).Get(ctx, vm.Name, metav1.GetOptions{})
	if err == nil {
		info.State = mapVMIState(vmi)
		info.NodeName = vmi.Status.NodeName
		for _, iface := range vmi.Status.Interfaces {
			nicType := "multus"
			if iface.Name == "default" {
				nicType = "pod"
			}
			info.NICs = append(info.NICs, VMNicInfo{
				Name: iface.Name, IP: iface.IP, MAC: iface.MAC, Type: nicType,
			})
		}
		info.IP = preferGuestIP(vmi.Status.Interfaces)
		if msg := vmiErrorMessage(vmi); msg != "" {
			info.ErrorMsg = msg
		}
	}

	return info
}

// preferGuestIP returns the routable guest IP (public/macvlan) instead of the pod masquerade address.
func preferGuestIP(ifaces []kubevirtv1.VirtualMachineInstanceNetworkInterface) string {
	for _, iface := range ifaces {
		if iface.Name == "public" && iface.IP != "" {
			return iface.IP
		}
	}
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.IP, "10.0.50.") {
			return iface.IP
		}
	}
	for _, iface := range ifaces {
		if iface.IP != "" && iface.Name != "default" {
			return iface.IP
		}
	}
	for _, iface := range ifaces {
		if iface.IP != "" {
			return iface.IP
		}
	}
	return ""
}

func mapVMState(vm *kubevirtv1.VirtualMachine) string {
	switch vm.Status.PrintableStatus {
	case kubevirtv1.VirtualMachineStatusRunning:
		return "Running"
	case kubevirtv1.VirtualMachineStatusStopped:
		return "Stopped"
	case kubevirtv1.VirtualMachineStatusStarting, kubevirtv1.VirtualMachineStatusProvisioning:
		return "Starting"
	case kubevirtv1.VirtualMachineStatusStopping:
		return "Stopping"
	case kubevirtv1.VirtualMachineStatusTerminating:
		return "Stopping"
	case kubevirtv1.VirtualMachineStatusCrashLoopBackOff,
		kubevirtv1.VirtualMachineStatusUnknown,
		kubevirtv1.VirtualMachineStatusUnschedulable,
		kubevirtv1.VirtualMachineStatusErrImagePull,
		kubevirtv1.VirtualMachineStatusImagePullBackOff,
		kubevirtv1.VirtualMachineStatusPvcNotFound,
		kubevirtv1.VirtualMachineStatusDataVolumeError:
		return "Error"
	}

	if vm.Spec.RunStrategy != nil {
		switch *vm.Spec.RunStrategy {
		case kubevirtv1.RunStrategyHalted, kubevirtv1.RunStrategyManual:
			return "Stopped"
		case kubevirtv1.RunStrategyAlways, kubevirtv1.RunStrategyRerunOnFailure:
			if vm.Status.Ready {
				return "Running"
			}
			return "Starting"
		}
	}
	return "Stopped"
}

func mapVMIState(vmi *kubevirtv1.VirtualMachineInstance) string {
	for _, cond := range vmi.Status.Conditions {
		if cond.Type == kubevirtv1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled) && cond.Status == k8sv1.ConditionFalse {
			if cond.Reason == "Unschedulable" {
				return "Error"
			}
		}
	}

	switch vmi.Status.Phase {
	case kubevirtv1.Pending, kubevirtv1.Scheduling, kubevirtv1.Scheduled:
		return "Starting"
	case kubevirtv1.Running:
		return "Running"
	case kubevirtv1.Succeeded:
		return "Stopped"
	case kubevirtv1.Failed:
		return "Error"
	default:
		return "Stopped"
	}
}

func resourceQuantityMi(mi int64) resource.Quantity {
	return resource.MustParse(fmt.Sprintf("%dMi", mi))
}

func resourceQuantityCPU(cores int) resource.Quantity {
	return resource.MustParse(fmt.Sprintf("%d", cores))
}

func mustParseQuantity(size string) resource.Quantity {
	return resource.MustParse(size)
}

func vmErrorMessage(vm *kubevirtv1.VirtualMachine) string {
	if vm.Status.StartFailure != nil && vm.Status.StartFailure.ConsecutiveFailCount > 0 {
		if vm.Status.PrintableStatus == kubevirtv1.VirtualMachineStatusCrashLoopBackOff {
			return dockerDesktopVMHint()
		}
	}
	for _, c := range vm.Status.Conditions {
		if c.Status == k8sv1.ConditionFalse && c.Message != "" && c.Message != "VMI does not exist" {
			return c.Message
		}
	}
	return ""
}

func vmiErrorMessage(vmi *kubevirtv1.VirtualMachineInstance) string {
	for _, c := range vmi.Status.Conditions {
		if c.Status == k8sv1.ConditionFalse && c.Message != "" {
			if strings.Contains(c.Message, "tap") || strings.Contains(c.Message, "MTU") || strings.Contains(c.Message, "network") {
				return dockerDesktopVMHint()
			}
			return c.Message
		}
	}
	if vmi.Status.Phase == kubevirtv1.Failed {
		return dockerDesktopVMHint()
	}
	return ""
}

func dockerDesktopVMHint() string {
	return "Falha de rede virtual (tap/MTU). Docker Desktop no Mac não suporta VMs KubeVirt — use um cluster Linux com KVM (ex: homelab)."
}

func (d *KubeVirtDriver) CreateVMSnapshot(ctx context.Context, vmName, snapName string) error {
	ns := d.namespace
	policy := snapshotv1.VirtualMachineSnapshotContentDelete
	snap := &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapName,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": branding.ManagedByValue,
				branding.LabelVM: vmName,
			},
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: k8sv1.TypedLocalObjectReference{
				APIGroup: strPtr("kubevirt.io"),
				Kind:     "VirtualMachine",
				Name:     vmName,
			},
			DeletionPolicy: &policy,
		},
	}
	_, err := d.virtClient.VirtualMachineSnapshot(ns).Create(ctx, snap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create vm snapshot %s: %w", snapName, err)
	}
	return nil
}

func (d *KubeVirtDriver) ListVMSnapshots(ctx context.Context) ([]VMSnapshotInfo, error) {
	list, err := d.virtClient.VirtualMachineSnapshot(d.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: branding.AppManagedByKey + "=" + branding.ManagedByValue,
	})
	if err != nil {
		return nil, fmt.Errorf("list vm snapshots: %w", err)
	}
	out := make([]VMSnapshotInfo, 0, len(list.Items))
	for _, item := range list.Items {
		info := VMSnapshotInfo{
			Name:      item.Name,
			Namespace: item.Namespace,
			Phase:     string(snapshotv1.PhaseUnset),
		}
		if item.Status != nil {
			info.Phase = string(item.Status.Phase)
			if item.Status.CreationTime != nil {
				info.Created = item.Status.CreationTime.Time
			}
		}
		if item.Labels != nil {
			info.VMName = item.Labels[branding.LabelVM]
		}
		out = append(out, info)
	}
	return out, nil
}

func (d *KubeVirtDriver) DeleteVMSnapshot(ctx context.Context, name string) error {
	err := d.virtClient.VirtualMachineSnapshot(d.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete vm snapshot %s: %w", name, err)
	}
	return nil
}

func (d *KubeVirtDriver) RestoreVMSnapshot(ctx context.Context, snapName, targetVM string) error {
	ns := d.namespace
	policy := snapshotv1.VirtualMachineRestoreStopTarget
	restore := &snapshotv1.VirtualMachineRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapName + "-restore",
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": branding.ManagedByValue,
			},
		},
		Spec: snapshotv1.VirtualMachineRestoreSpec{
			Target: k8sv1.TypedLocalObjectReference{
				APIGroup: strPtr("kubevirt.io"),
				Kind:     "VirtualMachine",
				Name:     targetVM,
			},
			VirtualMachineSnapshotName: snapName,
			TargetReadinessPolicy:      &policy,
		},
	}
	_, err := d.virtClient.VirtualMachineRestore(ns).Create(ctx, restore, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("restore vm from snapshot %s: %w", snapName, err)
	}
	return nil
}

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

func buildLinuxVMISpec(spec VMDeploySpec, ifaces []kubevirtv1.Interface, image string, cpu int, memMi int64) kubevirtv1.VirtualMachineInstanceSpec {
	disks := []kubevirtv1.Disk{
		{Name: "containerdisk", DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "virtio"}}},
		{Name: "cloudinitdisk", DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "virtio"}}},
	}
	volumes := []kubevirtv1.Volume{
		{
			Name: "containerdisk",
			VolumeSource: kubevirtv1.VolumeSource{
				ContainerDisk: &kubevirtv1.ContainerDiskSource{Image: image},
			},
		},
		{
			Name: "cloudinitdisk",
			VolumeSource: kubevirtv1.VolumeSource{
				CloudInitNoCloud: buildCloudInitSource(spec),
			},
		},
	}
	if spec.DataPVC != "" {
		disks = append(disks, kubevirtv1.Disk{
			Name: "datadisk", DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "virtio"}},
		})
		volumes = append(volumes, pvcVolume("datadisk", spec.DataPVC))
	}
	devices := kubevirtv1.Devices{
		Disks:      disks,
		Interfaces: ifaces,
	}
	if video := videoDeviceForImage(image); video != nil {
		devices.Video = video
	}
	return kubevirtv1.VirtualMachineInstanceSpec{
		Domain: kubevirtv1.DomainSpec{
			Devices: devices,
			Resources: kubevirtv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resourceQuantityMi(memMi),
					k8sv1.ResourceCPU:    resourceQuantityCPU(cpu),
				},
			},
		},
		Volumes: volumes,
	}
}

func buildWindowsVMISpec(spec VMDeploySpec, ifaces []kubevirtv1.Interface, cpu int, memMi int64) kubevirtv1.VirtualMachineInstanceSpec {
	disks := []kubevirtv1.Disk{
		{Name: "bootdisk", DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "sata"}}},
	}
	volumes := []kubevirtv1.Volume{
		pvcVolume("bootdisk", spec.BootPVC),
	}
	if spec.DataPVC != "" {
		disks = append(disks, kubevirtv1.Disk{
			Name: "datadisk", DiskDevice: kubevirtv1.DiskDevice{Disk: &kubevirtv1.DiskTarget{Bus: "sata"}},
		})
		volumes = append(volumes, pvcVolume("datadisk", spec.DataPVC))
	}
	if spec.InstallISO != "" {
		disks = append(disks, kubevirtv1.Disk{
			Name: "installiso", BootOrder: uintPtr(1),
			// UEFI boots Windows ISO via SATA cdrom (ide-cd). SATA disk exposes ide-hd and is not bootable.
			DiskDevice: kubevirtv1.DiskDevice{CDRom: &kubevirtv1.CDRomTarget{Bus: "sata"}},
		})
		volumes = append(volumes, pvcVolume("installiso", spec.InstallISO))
	}
	disks = append(disks, kubevirtv1.Disk{
		Name: "virtiodrivers",
		DiskDevice: kubevirtv1.DiskDevice{CDRom: &kubevirtv1.CDRomTarget{Bus: "sata"}},
	})
	volumes = append(volumes, kubevirtv1.Volume{
		Name: "virtiodrivers",
		VolumeSource: kubevirtv1.VolumeSource{
			ContainerDisk: &kubevirtv1.ContainerDiskSource{Image: virtioContainerDisk},
		},
	})

	winIfaces := make([]kubevirtv1.Interface, len(ifaces))
	for i, iface := range ifaces {
		winIfaces[i] = iface
		winIfaces[i].Model = "e1000"
	}

	return kubevirtv1.VirtualMachineInstanceSpec{
		Domain: kubevirtv1.DomainSpec{
			Machine: &kubevirtv1.Machine{Type: windowsMachineType},
			Firmware: &kubevirtv1.Firmware{
				Bootloader: &kubevirtv1.Bootloader{
					EFI: &kubevirtv1.EFI{SecureBoot: boolPtr(false)},
				},
			},
			Devices: kubevirtv1.Devices{
				Disks:      disks,
				Interfaces: winIfaces,
			},
			Resources: kubevirtv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resourceQuantityMi(memMi),
					k8sv1.ResourceCPU:    resourceQuantityCPU(cpu),
				},
			},
			Features: &kubevirtv1.Features{
				Hyperv: &kubevirtv1.FeatureHyperv{},
			},
		},
		Volumes: volumes,
	}
}

func pvcVolume(name, claim string) kubevirtv1.Volume {
	return kubevirtv1.Volume{
		Name: name,
		VolumeSource: kubevirtv1.VolumeSource{
			PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claim},
			},
		},
	}
}

func uintPtr(n uint) *uint { return &n }

func buildNetworks(specs []VMNetworkSpec) ([]kubevirtv1.Network, []kubevirtv1.Interface, string) {
	if len(specs) == 0 {
		return []kubevirtv1.Network{
				{Name: "default", NetworkSource: kubevirtv1.NetworkSource{Pod: &kubevirtv1.PodNetwork{}}},
			},
			[]kubevirtv1.Interface{{
				Name: "default",
				InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
					Masquerade: &kubevirtv1.InterfaceMasquerade{},
				},
			}},
			""
	}

	var networks []kubevirtv1.Network
	var ifaces []kubevirtv1.Interface
	var defaultNet string
	hasPodNet := false
	for _, sn := range specs {
		if sn.Default || (sn.NADName == "" && sn.NADNamespace == "") {
			hasPodNet = true
			break
		}
	}

	for i, sn := range specs {
		name := sn.Name
		if name == "" {
			name = fmt.Sprintf("net%d", i)
		}
		iface := kubevirtv1.Interface{Name: name}
		if sn.MACAddress != "" {
			iface.MacAddress = sn.MACAddress
		}
		if sn.Default || (sn.NADName == "" && sn.NADNamespace == "") {
			networks = append(networks, kubevirtv1.Network{
				Name: name, NetworkSource: kubevirtv1.NetworkSource{Pod: &kubevirtv1.PodNetwork{}},
			})
			iface.InterfaceBindingMethod = kubevirtv1.InterfaceBindingMethod{
				Masquerade: &kubevirtv1.InterfaceMasquerade{},
			}
		} else {
			nadRef := fmt.Sprintf("%s/%s", sn.NADNamespace, sn.NADName)
			networks = append(networks, kubevirtv1.Network{
				Name: name,
				NetworkSource: kubevirtv1.NetworkSource{
					Multus: &kubevirtv1.MultusNetwork{NetworkName: nadRef},
				},
			})
			iface.InterfaceBindingMethod = kubevirtv1.InterfaceBindingMethod{
				Bridge: &kubevirtv1.InterfaceBridge{},
			}
			if !hasPodNet && defaultNet == "" {
				defaultNet = nadRef
			}
		}
		ifaces = append(ifaces, iface)
	}
	return networks, ifaces, defaultNet
}

func buildCloudInitSource(spec VMDeploySpec) *kubevirtv1.CloudInitNoCloudSource {
	src := &kubevirtv1.CloudInitNoCloudSource{
		UserData: cloudinit.BuildLinuxUserData(cloudinit.LinuxConfig{
			SSHPublicKeys:  spec.CloudInitSSHKeys,
			Password:       spec.CloudInitPassword,
			ExtraUserData:  spec.CloudInitExtra,
			FormatDataDisk: spec.FormatDataDisk || spec.DataPVC != "",
		}),
	}
	var netIfaces []cloudinit.NetworkInterfaceConfig
	for _, sn := range spec.Networks {
		if sn.MACAddress == "" || sn.Default || sn.NADName == "" {
			continue
		}
		cfg := cloudinit.NetworkInterfaceConfig{
			MACAddress: sn.MACAddress,
			Address:    sn.StaticIP,
			PrefixLen:  sn.PrefixLen,
			Gateway:    sn.Gateway,
			DNS:        sn.DNS,
		}
		netIfaces = append(netIfaces, cfg)
	}
	if networkData := cloudinit.BuildNetworkData(netIfaces); networkData != "" {
		src.NetworkData = networkData
	}
	return src
}

func buildMultusNetworksAnnotation(specs []VMNetworkSpec) string {
	type multusNet struct {
		Name      string   `json:"name"`
		Namespace string   `json:"namespace,omitempty"`
		MAC       string   `json:"mac,omitempty"`
		IPs       []string `json:"ips,omitempty"`
	}
	var nets []multusNet
	for _, sn := range specs {
		if sn.NADName == "" || sn.StaticIP == "" {
			continue
		}
		prefix := sn.PrefixLen
		if prefix <= 0 {
			prefix = 24
		}
		entry := multusNet{
			Name: sn.NADName,
			IPs:  []string{fmt.Sprintf("%s/%d", sn.StaticIP, prefix)},
		}
		if sn.NADNamespace != "" {
			entry.Namespace = sn.NADNamespace
		}
		if sn.MACAddress != "" {
			entry.MAC = sn.MACAddress
		}
		nets = append(nets, entry)
	}
	if len(nets) == 0 {
		return ""
	}
	b, err := json.Marshal(nets)
	if err != nil {
		return ""
	}
	return string(b)
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergeLabels(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
