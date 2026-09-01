package mapping

import (
	"github.com/virtfoundry/core/internal/platform"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func OfferingToUnstructured(o *platform.ServiceOffering) *unstructured.Unstructured {
	obj := newObject("Offering", SanitizeCRName(o.Name), "")
	obj.SetLabels(map[string]string{LabelPartOf: PartOfValue})
	SetLegacyID(obj, o.ID)
	spec := map[string]interface{}{
		"displayName":  o.DisplayName,
		"cpu":          int64(o.CPU),
		"memoryMi":     o.MemoryMi,
		"dedicatedCPU": o.DedicatedCPU,
	}
	if o.StorageTags != "" {
		spec["storageTags"] = o.StorageTags
	}
	if imp := importMeta(o.ExternalUUID, o.ImportSource); imp != nil {
		spec["import"] = imp
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func OfferingFromUnstructured(obj *unstructured.Unstructured) *platform.ServiceOffering {
	o := &platform.ServiceOffering{
		ID:        ResourceID(obj),
		Name:      obj.GetName(),
		State:     "Active",
		CreatedAt: obj.GetCreationTimestamp().Time,
	}
	display, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	cpu, _, _ := unstructured.NestedInt64(obj.Object, "spec", "cpu")
	mem, _, _ := unstructured.NestedInt64(obj.Object, "spec", "memoryMi")
	ded, _, _ := unstructured.NestedBool(obj.Object, "spec", "dedicatedCPU")
	tags, _, _ := unstructured.NestedString(obj.Object, "spec", "storageTags")
	o.DisplayName = display
	o.CPU = int(cpu)
	o.MemoryMi = mem
	o.DedicatedCPU = ded
	o.StorageTags = tags
	if ext, ok, _ := unstructured.NestedString(obj.Object, "spec", "import", "externalUUID"); ok {
		o.ExternalUUID = ext
	}
	if src, ok, _ := unstructured.NestedString(obj.Object, "spec", "import", "source"); ok {
		o.ImportSource = src
	}
	return o
}

func TemplateNamespace(t *platform.VMTemplate) string {
	if t.TenantID == "" {
		return SystemNamespace
	}
	return ""
}

func TemplateCRName(t *platform.VMTemplate) string {
	return SanitizeCRName(t.Name)
}

func TemplateToUnstructured(t *platform.VMTemplate) *unstructured.Unstructured {
	obj := newObject("Template", TemplateCRName(t), "")
	obj.SetLabels(map[string]string{LabelPartOf: PartOfValue})
	SetLegacyID(obj, t.ID)
	spec := map[string]interface{}{
		"displayName": t.DisplayName,
		"image":       t.Image,
		"sourceType":  t.SourceType,
	}
	if t.OSType != "" {
		spec["osType"] = t.OSType
	}
	if t.CloudInitUserData != "" {
		spec["cloudInitUserData"] = t.CloudInitUserData
	}
	if t.ISOSizeGi > 0 {
		spec["isoSizeGi"] = int64(t.ISOSizeGi)
	}
	if t.BootDiskSizeGi > 0 {
		spec["bootDiskSizeGi"] = int64(t.BootDiskSizeGi)
	}
	if t.StorageClass != "" {
		spec["storageClass"] = t.StorageClass
	}
	if imp := importMeta(t.ExternalUUID, t.ImportSource); imp != nil {
		spec["import"] = imp
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func TemplateFromUnstructured(obj *unstructured.Unstructured, tenantID string) *platform.VMTemplate {
	t := &platform.VMTemplate{
		ID:         ResourceID(obj),
		TenantID:   tenantID,
		Name:       obj.GetName(),
		State:      "Active",
		Hypervisor: "KubeVirt",
		CreatedAt:  obj.GetCreationTimestamp().Time,
	}
	display, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	image, _, _ := unstructured.NestedString(obj.Object, "spec", "image")
	srcType, _, _ := unstructured.NestedString(obj.Object, "spec", "sourceType")
	osType, _, _ := unstructured.NestedString(obj.Object, "spec", "osType")
	cloud, _, _ := unstructured.NestedString(obj.Object, "spec", "cloudInitUserData")
	isoSize, _, _ := unstructured.NestedInt64(obj.Object, "spec", "isoSizeGi")
	bootSize, _, _ := unstructured.NestedInt64(obj.Object, "spec", "bootDiskSizeGi")
	sc, _, _ := unstructured.NestedString(obj.Object, "spec", "storageClass")
	importState, _, _ := unstructured.NestedString(obj.Object, "status", "importState")
	t.DisplayName = display
	t.Image = image
	t.SourceType = srcType
	t.OSType = osType
	t.CloudInitUserData = cloud
	t.ISOSizeGi = int(isoSize)
	t.BootDiskSizeGi = int(bootSize)
	t.StorageClass = sc
	t.ImportState = importState
	return t
}
