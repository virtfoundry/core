package mapping

import (
	"fmt"

	"github.com/virtfoundry/core/internal/platform"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TenantCRName(slug string) string {
	return slug
}

func TenantNamespace(slug string) string {
	return fmt.Sprintf("virtfoundry-tenant-%s", slug)
}

func TenantToUnstructured(t *platform.Tenant) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(Group + "/" + Version)
	obj.SetKind("Tenant")
	obj.SetName(TenantCRName(t.Slug))
	obj.SetLabels(map[string]string{
		LabelPartOf: PartOfValue,
		LabelSlug:   t.Slug,
	})

	spec := map[string]interface{}{
		"name": t.Name,
		"slug": t.Slug,
	}
	if t.ExternalUUID != "" || t.ImportSource != "" {
		imp := map[string]interface{}{}
		if t.ExternalUUID != "" {
			imp["externalUUID"] = t.ExternalUUID
		}
		if t.ImportSource != "" {
			imp["source"] = t.ImportSource
		}
		spec["import"] = imp
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func TenantFromUnstructured(obj *unstructured.Unstructured) *platform.Tenant {
	t := &platform.Tenant{
		ID:        string(obj.GetUID()),
		CreatedAt: obj.GetCreationTimestamp().Time,
		State:     "active",
	}
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	slug, _, _ := unstructured.NestedString(obj.Object, "spec", "slug")
	t.Name = name
	t.Slug = slug
	if ns, ok, _ := unstructured.NestedString(obj.Object, "status", "namespace"); ok && ns != "" {
		t.Namespace = ns
	} else if slug != "" {
		t.Namespace = TenantNamespace(slug)
	}
	if ext, ok, _ := unstructured.NestedString(obj.Object, "spec", "import", "externalUUID"); ok {
		t.ExternalUUID = ext
	}
	if src, ok, _ := unstructured.NestedString(obj.Object, "spec", "import", "source"); ok {
		t.ImportSource = src
	}
	return t
}
