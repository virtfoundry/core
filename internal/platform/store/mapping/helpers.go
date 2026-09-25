package mapping

import (
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	AnnLegacyID = "virtfoundry.io/legacy-id"
	// AnnAllowPodNetwork opts an Instance into the KubeVirt pod network
	// (masquerade). Must match operator annotationAllowPodNetwork — Instances
	// without Multus spec.nics need this after operator 0.7.2.
	AnnAllowPodNetwork = "virtfoundry.io/allow-pod-network"
)

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// SanitizeCRName normalizes a string for use as metadata.name.
func SanitizeCRName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func SetLegacyID(obj *unstructured.Unstructured, id string) {
	if id == "" {
		return
	}
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	ann[AnnLegacyID] = id
	obj.SetAnnotations(ann)
}

func LegacyID(obj *unstructured.Unstructured) string {
	if obj == nil {
		return ""
	}
	return obj.GetAnnotations()[AnnLegacyID]
}

func ResourceID(obj *unstructured.Unstructured) string {
	if uid := string(obj.GetUID()); uid != "" {
		return uid
	}
	return LegacyID(obj)
}

func MatchesID(obj *unstructured.Unstructured, id string) bool {
	if id == "" {
		return false
	}
	return string(obj.GetUID()) == id || LegacyID(obj) == id
}

func BaseLabels(tenantSlug string) map[string]string {
	labels := map[string]string{LabelPartOf: PartOfValue}
	if tenantSlug != "" {
		labels[LabelTenant] = tenantSlug
	}
	return labels
}

func importMeta(extUUID, source string) map[string]interface{} {
	if extUUID == "" && source == "" {
		return nil
	}
	imp := map[string]interface{}{}
	if extUUID != "" {
		imp["externalUUID"] = extUUID
	}
	if source != "" {
		imp["source"] = source
	}
	return imp
}

func localRef(name string) map[string]interface{} {
	return map[string]interface{}{"name": name}
}

// MergeUnstructuredSpec copies spec keys from src onto dst without deleting
// keys that exist only on dst. Stop/start must set powerState without wiping
// offeringRef/nics written at deploy.
func MergeUnstructuredSpec(dst, src *unstructured.Unstructured) {
	if dst == nil || src == nil {
		return
	}
	srcSpec, ok, _ := unstructured.NestedMap(src.Object, "spec")
	if !ok || len(srcSpec) == 0 {
		return
	}
	dstSpec, _, _ := unstructured.NestedMap(dst.Object, "spec")
	if dstSpec == nil {
		dstSpec = map[string]interface{}{}
	}
	for k, v := range srcSpec {
		dstSpec[k] = v
	}
	_ = unstructured.SetNestedMap(dst.Object, dstSpec, "spec")
}

func setSpecField(obj *unstructured.Unstructured, key string, val interface{}) {
	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec == nil {
		spec = map[string]interface{}{}
	}
	spec[key] = val
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
}

func newObject(kind, name, namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(Group + "/" + Version)
	obj.SetKind(kind)
	obj.SetName(name)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	return obj
}
