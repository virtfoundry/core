package mapping

import (
	"github.com/virtfoundry/core/internal/platform"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func InstanceCRName(vm *platform.PlatformVM) string {
	return SanitizeCRName(vm.Name)
}

// InstancePhaseToPlatformState maps CR status.phase to REST/UI VM state.
func InstancePhaseToPlatformState(phase string) string {
	switch phase {
	case "Ready":
		return "Running"
	case "Failed":
		return "Error"
	default:
		return phase
	}
}

// MergePlatformVM applies CR fields onto dst while preserving hypervisor-synced
// runtime fields when the CR status subresource has not been populated yet.
func MergePlatformVM(dst, prior, fromCR *platform.PlatformVM) {
	*dst = *fromCR
	if dst.State == "" || (dst.State == "Pending" && prior.State != "" && prior.State != "Pending") {
		dst.State = prior.State
	}
	if dst.CPU == 0 && prior.CPU > 0 {
		dst.CPU = prior.CPU
	}
	if dst.MemoryMi == 0 && prior.MemoryMi > 0 {
		dst.MemoryMi = prior.MemoryMi
	}
	if dst.IP == "" && prior.IP != "" {
		dst.IP = prior.IP
	}
	if dst.Image == "" && prior.Image != "" {
		dst.Image = prior.Image
	}
	if dst.Template == "" && prior.Template != "" {
		dst.Template = prior.Template
	}
	if dst.HostName == "" && prior.HostName != "" {
		dst.HostName = prior.HostName
	}
	if dst.ErrorMsg == "" && prior.ErrorMsg != "" {
		dst.ErrorMsg = prior.ErrorMsg
	}
	if len(dst.NICs) == 0 && len(prior.NICs) > 0 {
		dst.NICs = prior.NICs
	}
	if dst.UpdatedAt.IsZero() && !prior.UpdatedAt.IsZero() {
		dst.UpdatedAt = prior.UpdatedAt
	}
}

func InstanceToUnstructured(vm *platform.PlatformVM, tenantSlug, offeringCR, templateCR string, networkRefs map[string]string) *unstructured.Unstructured {
	obj := newObject("Instance", InstanceCRName(vm), "")
	obj.SetLabels(BaseLabels(tenantSlug))
	SetLegacyID(obj, vm.ID)
	display := vm.DisplayName
	if display == "" {
		display = vm.Name
	}
	spec := map[string]interface{}{
		"displayName": display,
	}
	if offeringCR != "" {
		spec["offeringRef"] = localRef(offeringCR)
	}
	if templateCR != "" {
		spec["templateRef"] = localRef(templateCR)
	}
	if len(vm.NICs) > 0 {
		nics := make([]interface{}, 0, len(vm.NICs))
		for _, nic := range vm.NICs {
			netCR := networkRefs[nic.NetworkID]
			if netCR == "" {
				continue
			}
			nics = append(nics, map[string]interface{}{
				"name":       nic.Name,
				"networkRef": localRef(netCR),
			})
		}
		if len(nics) > 0 {
			spec["nics"] = nics
		}
	}
	if vm.PowerState != "" {
		spec["powerState"] = vm.PowerState
	}
	if vm.DedicatedCPU {
		spec["dedicatedCPU"] = true
	}
	if imp := importMeta(vm.ExternalUUID, vm.ImportSource); imp != nil {
		spec["import"] = imp
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func InstanceFromUnstructured(obj *unstructured.Unstructured, tenantID string, resolveNetwork func(string) string) *platform.PlatformVM {
	vm := &platform.PlatformVM{
		ID:         ResourceID(obj),
		TenantID:   tenantID,
		Name:       obj.GetName(),
		Namespace:  obj.GetNamespace(),
		Hypervisor: "KubeVirt",
		CreatedAt:  obj.GetCreationTimestamp().Time,
	}
	display, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	vm.DisplayName = display
	if phase, ok, _ := unstructured.NestedString(obj.Object, "status", "phase"); ok && phase != "" {
		vm.State = InstancePhaseToPlatformState(phase)
	} else {
		vm.State = "Pending"
	}
	if ip, ok, _ := unstructured.NestedString(obj.Object, "status", "ip"); ok {
		vm.IP = ip
	}
	if errMsg, ok, _ := unstructured.NestedString(obj.Object, "status", "errorMessage"); ok {
		vm.ErrorMsg = errMsg
	}
	if ext, ok, _ := unstructured.NestedString(obj.Object, "spec", "import", "externalUUID"); ok {
		vm.ExternalUUID = ext
	}
	if src, ok, _ := unstructured.NestedString(obj.Object, "spec", "import", "source"); ok {
		vm.ImportSource = src
	}
	return vm
}

func DiskCRName(v *platform.Volume) string {
	return SanitizeCRName(v.Name)
}

func DiskToUnstructured(v *platform.Volume, tenantSlug, instanceCR string) *unstructured.Unstructured {
	obj := newObject("Disk", DiskCRName(v), "")
	obj.SetLabels(BaseLabels(tenantSlug))
	SetLegacyID(obj, v.ID)
	spec := map[string]interface{}{
		"name":   v.Name,
		"sizeGi": int64(v.SizeGi),
	}
	if instanceCR != "" {
		spec["instanceRef"] = localRef(instanceCR)
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func DiskFromUnstructured(obj *unstructured.Unstructured, tenantID string) *platform.Volume {
	v := &platform.Volume{
		ID:        ResourceID(obj),
		TenantID:  tenantID,
		Namespace: obj.GetNamespace(),
		State:     "active",
		CreatedAt: obj.GetCreationTimestamp().Time,
	}
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	size, _, _ := unstructured.NestedInt64(obj.Object, "spec", "sizeGi")
	pvc, _, _ := unstructured.NestedString(obj.Object, "status", "pvcName")
	if pvc == "" && name != "" {
		pvc = SanitizeCRName(name)
	}
	v.Name = name
	v.SizeGi = int(size)
	v.PVCName = pvc
	return v
}

func DiskSnapshotToUnstructured(s *platform.Snapshot, diskCR string) *unstructured.Unstructured {
	obj := newObject("DiskSnapshot", SanitizeCRName(s.Name), "")
	SetLegacyID(obj, s.ID)
	spec := map[string]interface{}{
		"name":    s.Name,
		"diskRef": localRef(diskCR),
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func DiskSnapshotFromUnstructured(obj *unstructured.Unstructured, tenantID, volumeID string) *platform.Snapshot {
	return &platform.Snapshot{
		ID:          ResourceID(obj),
		TenantID:    tenantID,
		VolumeID:    volumeID,
		Name:        stringFromSpec(obj, "name"),
		Namespace:   obj.GetNamespace(),
		SnapshotUID: stringFromStatus(obj, "volumeSnapshotName"),
		State:       stringFromStatus(obj, "phase"),
		CreatedAt:   obj.GetCreationTimestamp().Time,
	}
}

func InstanceSnapshotToUnstructured(s *platform.VMSnapshot, instanceCR string) *unstructured.Unstructured {
	obj := newObject("InstanceSnapshot", SanitizeCRName(s.Name), "")
	SetLegacyID(obj, s.ID)
	spec := map[string]interface{}{
		"name":        s.Name,
		"instanceRef": localRef(instanceCR),
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func InstanceSnapshotFromUnstructured(obj *unstructured.Unstructured, tenantID, vmID string) *platform.VMSnapshot {
	return &platform.VMSnapshot{
		ID:          ResourceID(obj),
		TenantID:    tenantID,
		VMID:        vmID,
		Name:        stringFromSpec(obj, "name"),
		Namespace:   obj.GetNamespace(),
		SnapshotUID: stringFromStatus(obj, "kubevirtSnapshotName"),
		Phase:       stringFromStatus(obj, "phase"),
		CreatedAt:   obj.GetCreationTimestamp().Time,
	}
}

func SSHKeyToUnstructured(k *platform.SSHKeyPair, tenantSlug string) *unstructured.Unstructured {
	obj := newObject("SSHKey", SanitizeCRName(k.Name), "")
	obj.SetLabels(BaseLabels(tenantSlug))
	SetLegacyID(obj, k.ID)
	spec := map[string]interface{}{"publicKey": k.PublicKey}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func SSHKeyFromUnstructured(obj *unstructured.Unstructured, tenantID string) *platform.SSHKeyPair {
	return &platform.SSHKeyPair{
		ID:          ResourceID(obj),
		TenantID:    tenantID,
		Name:        obj.GetName(),
		PublicKey:   stringFromSpec(obj, "publicKey"),
		Fingerprint: stringFromStatus(obj, "fingerprint"),
		CreatedAt:   obj.GetCreationTimestamp().Time,
	}
}

func stringFromSpec(obj *unstructured.Unstructured, key string) string {
	v, _, _ := unstructured.NestedString(obj.Object, "spec", key)
	return v
}

func stringFromStatus(obj *unstructured.Unstructured, key string) string {
	v, _, _ := unstructured.NestedString(obj.Object, "status", key)
	return v
}
