package mapping

import (
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const AnnLegacyID = "virtfoundry.io/legacy-id"

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
