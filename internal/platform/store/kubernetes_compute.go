package store

import (
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (k *Kubernetes) SaveVM(vm *platform.PlatformVM) {
	ns, ok := k.tenantNamespace(vm.TenantID)
	if !ok {
		return
	}
	offeringCR := ""
	if vm.ServiceOfferingID != "" {
		if o, ok := k.GetServiceOffering(vm.ServiceOfferingID); ok {
			offeringCR = mapping.SanitizeCRName(o.Name)
		}
	}
	templateCR := ""
	if vm.TemplateRef != "" {
		templateCR = mapping.SanitizeCRName(vm.TemplateRef)
	} else if vm.Template != "" {
		templateCR = mapping.SanitizeCRName(vm.Template)
	}
	networkRefs := map[string]string{}
	for _, nic := range vm.NICs {
		networkRefs[nic.NetworkID] = k.resolveNetworkCRName(nic.NetworkID)
	}
	slug := k.tenantSlug(vm.TenantID)
	prior := *vm
	k.saveNamespacedMapped(mapping.InstanceGVR, ns, func() *unstructured.Unstructured {
		return mapping.InstanceToUnstructured(vm, slug, offeringCR, templateCR, networkRefs)
	}, func(saved *unstructured.Unstructured) {
		fromCR := mapping.InstanceFromUnstructured(saved, vm.TenantID, nil)
		mapping.MergePlatformVM(vm, &prior, fromCR)
	})
}

func (k *Kubernetes) GetVM(id string) (*platform.PlatformVM, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.InstanceGVR, id)
	if !ok {
		return nil, false
	}
	return mapping.InstanceFromUnstructured(obj, k.tenantIDForNamespace(ns), nil), true
}

func (k *Kubernetes) GetVMByName(tenantID, name string) (*platform.PlatformVM, bool) {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil, false
	}
	obj, err := k.dyn.Resource(mapping.InstanceGVR).Namespace(ns).Get(k.ctx(), mapping.SanitizeCRName(name), metav1.GetOptions{})
	if err != nil {
		return nil, false
	}
	return mapping.InstanceFromUnstructured(obj, tenantID, nil), true
}

func (k *Kubernetes) GetVMByExternalUUID(source, externalUUID string) (*platform.PlatformVM, bool) {
	for _, obj := range k.listNamespacedAll(mapping.InstanceGVR) {
		src, _, _ := unstructured.NestedString(obj.Object, "spec", "import", "source")
		ext, _, _ := unstructured.NestedString(obj.Object, "spec", "import", "externalUUID")
		if src == source && ext == externalUUID {
			return mapping.InstanceFromUnstructured(&obj, k.tenantIDForNamespace(obj.GetNamespace()), nil), true
		}
	}
	return nil, false
}

func (k *Kubernetes) ListVMs(tenantID string) []*platform.PlatformVM {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.InstanceGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.PlatformVM, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapping.InstanceFromUnstructured(&list.Items[i], tenantID, nil))
	}
	return out
}

func (k *Kubernetes) DeleteVM(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.InstanceGVR, id); ok {
		k.deleteNamespaced(mapping.InstanceGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) SaveVolume(v *platform.Volume) {
	ns, ok := k.tenantNamespace(v.TenantID)
	if !ok {
		return
	}
	instanceCR := ""
	if v.VMID != "" {
		if vm, ok := k.GetVM(v.VMID); ok {
			instanceCR = mapping.InstanceCRName(vm)
		}
	}
	slug := k.tenantSlug(v.TenantID)
	prior := *v
	k.saveNamespacedMapped(mapping.DiskGVR, ns, func() *unstructured.Unstructured {
		return mapping.DiskToUnstructured(v, slug, instanceCR)
	}, func(saved *unstructured.Unstructured) {
		fromCR := mapping.DiskFromUnstructured(saved, v.TenantID)
		*v = *fromCR
		if v.PVCName == "" && prior.PVCName != "" {
			v.PVCName = prior.PVCName
		}
		if v.VMID == "" && prior.VMID != "" {
			v.VMID = prior.VMID
			v.State = prior.State
		}
	})
}

func (k *Kubernetes) ListVolumes(tenantID string) []*platform.Volume {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.DiskGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.Volume, 0, len(list.Items))
	for i := range list.Items {
		vol := mapping.DiskFromUnstructured(&list.Items[i], tenantID)
		if vol.VMID == "" {
			vol.VMID = k.vmIDFromInstanceRef(tenantID, &list.Items[i])
			if vol.VMID != "" {
				vol.State = "attached"
			}
		}
		out = append(out, vol)
	}
	return out
}

func (k *Kubernetes) ListVolumesByVMID(tenantID, vmID string) []*platform.Volume {
	var out []*platform.Volume
	for _, v := range k.ListVolumes(tenantID) {
		if v.VMID == vmID {
			out = append(out, v)
		}
	}
	return out
}

func (k *Kubernetes) GetVolume(id string) (*platform.Volume, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.DiskGVR, id)
	if !ok {
		return nil, false
	}
	return mapping.DiskFromUnstructured(obj, k.tenantIDForNamespace(ns)), true
}

func (k *Kubernetes) DeleteVolume(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.DiskGVR, id); ok {
		k.deleteNamespaced(mapping.DiskGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) SaveSnapshot(s *platform.Snapshot) {
	ns, ok := k.tenantNamespace(s.TenantID)
	if !ok {
		return
	}
	diskCR := ""
	if vol, ok := k.GetVolume(s.VolumeID); ok {
		diskCR = mapping.DiskCRName(vol)
	}
	prior := *s
	k.saveNamespacedMapped(mapping.DiskSnapshotGVR, ns, func() *unstructured.Unstructured {
		return mapping.DiskSnapshotToUnstructured(s, diskCR)
	}, func(saved *unstructured.Unstructured) {
		fromCR := mapping.DiskSnapshotFromUnstructured(saved, s.TenantID, s.VolumeID)
		*s = *fromCR
		if s.State == "" && prior.State != "" {
			s.State = prior.State
		}
	})
}

func (k *Kubernetes) ListSnapshots(tenantID string) []*platform.Snapshot {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.DiskSnapshotGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.Snapshot, 0, len(list.Items))
	for i := range list.Items {
		volID := k.volumeIDFromDiskRef(tenantID, &list.Items[i])
		out = append(out, mapping.DiskSnapshotFromUnstructured(&list.Items[i], tenantID, volID))
	}
	return out
}

func (k *Kubernetes) SaveVMSnapshot(s *platform.VMSnapshot) {
	ns, ok := k.tenantNamespace(s.TenantID)
	if !ok {
		return
	}
	instanceCR := ""
	if vm, ok := k.GetVM(s.VMID); ok {
		instanceCR = mapping.InstanceCRName(vm)
	}
	k.saveNamespacedMapped(mapping.InstanceSnapshotGVR, ns, func() *unstructured.Unstructured {
		return mapping.InstanceSnapshotToUnstructured(s, instanceCR)
	}, func(saved *unstructured.Unstructured) {
		*s = *mapping.InstanceSnapshotFromUnstructured(saved, s.TenantID, s.VMID)
	})
}

func (k *Kubernetes) ListVMSnapshots(tenantID string) []*platform.VMSnapshot {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.InstanceSnapshotGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.VMSnapshot, 0, len(list.Items))
	for i := range list.Items {
		vmID := k.vmIDFromInstanceRef(tenantID, &list.Items[i])
		out = append(out, mapping.InstanceSnapshotFromUnstructured(&list.Items[i], tenantID, vmID))
	}
	return out
}

func (k *Kubernetes) GetVMSnapshot(id string) (*platform.VMSnapshot, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.InstanceSnapshotGVR, id)
	if !ok {
		return nil, false
	}
	tenantID := k.tenantIDForNamespace(ns)
	return mapping.InstanceSnapshotFromUnstructured(obj, tenantID, k.vmIDFromInstanceRef(tenantID, obj)), true
}

func (k *Kubernetes) DeleteVMSnapshot(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.InstanceSnapshotGVR, id); ok {
		k.deleteNamespaced(mapping.InstanceSnapshotGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) SaveSSHKeyPair(key *platform.SSHKeyPair) {
	ns, ok := k.tenantNamespace(key.TenantID)
	if !ok {
		return
	}
	slug := k.tenantSlug(key.TenantID)
	k.saveNamespacedMapped(mapping.SSHKeyGVR, ns, func() *unstructured.Unstructured {
		return mapping.SSHKeyToUnstructured(key, slug)
	}, func(saved *unstructured.Unstructured) {
		*key = *mapping.SSHKeyFromUnstructured(saved, key.TenantID)
	})
}

func (k *Kubernetes) GetSSHKeyPair(id string) (*platform.SSHKeyPair, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.SSHKeyGVR, id)
	if !ok {
		return nil, false
	}
	return mapping.SSHKeyFromUnstructured(obj, k.tenantIDForNamespace(ns)), true
}

func (k *Kubernetes) ListSSHKeyPairs(tenantID string) []*platform.SSHKeyPair {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.SSHKeyGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.SSHKeyPair, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapping.SSHKeyFromUnstructured(&list.Items[i], tenantID))
	}
	return out
}

func (k *Kubernetes) DeleteSSHKeyPair(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.SSHKeyGVR, id); ok {
		k.deleteNamespaced(mapping.SSHKeyGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) volumeIDFromDiskRef(tenantID string, obj *unstructured.Unstructured) string {
	ref, _, _ := unstructured.NestedString(obj.Object, "spec", "diskRef", "name")
	if ref == "" {
		return ""
	}
	for _, v := range k.ListVolumes(tenantID) {
		if mapping.DiskCRName(v) == ref {
			return v.ID
		}
	}
	return ""
}

func (k *Kubernetes) vmIDFromInstanceRef(tenantID string, obj *unstructured.Unstructured) string {
	ref, _, _ := unstructured.NestedString(obj.Object, "spec", "instanceRef", "name")
	if ref == "" {
		return ""
	}
	for _, vm := range k.ListVMs(tenantID) {
		if mapping.InstanceCRName(vm) == ref {
			return vm.ID
		}
	}
	return ""
}
