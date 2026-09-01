package mapping

import (
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func RoleCRName(roleName string) string {
	return strings.ReplaceAll(roleName, ".", "-")
}

func RoleToUnstructured(r *platform.RoleRecord, namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(Group + "/" + Version)
	obj.SetKind("Role")
	obj.SetName(RoleCRName(r.Name))
	obj.SetNamespace(namespace)
	obj.SetLabels(map[string]string{LabelPartOf: PartOfValue})
	if r.ID != "" {
		obj.SetAnnotations(map[string]string{AnnRoleID: r.ID})
	}

	spec := map[string]interface{}{
		"description": r.Description,
		"isSystem":    r.IsSystem,
	}
	if len(r.Permissions) > 0 {
		perms := make([]interface{}, len(r.Permissions))
		for i, p := range r.Permissions {
			perms[i] = p
		}
		spec["permissions"] = perms
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func RoleFromUnstructured(obj *unstructured.Unstructured) *platform.RoleRecord {
	r := &platform.RoleRecord{
		ID:        obj.GetAnnotations()[AnnRoleID],
		CreatedAt: obj.GetCreationTimestamp().Time,
	}
	if r.ID == "" {
		r.ID = string(obj.GetUID())
	}
	desc, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	isSystem, _, _ := unstructured.NestedBool(obj.Object, "spec", "isSystem")
	perms, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "permissions")
	r.Description = desc
	r.IsSystem = isSystem
	r.Permissions = perms
	if r.IsSystem {
		r.Name = roleNameFromCR(obj.GetName())
	} else {
		r.Name = obj.GetName()
	}
	if !r.IsSystem {
		r.TenantID = obj.GetNamespace()
	}
	return r
}

func roleNameFromCR(crName string) string {
	switch crName {
	case "platform-root":
		return platform.SystemRoleRoot
	case "platform-tenant-admin":
		return platform.SystemRoleTenantAdmin
	case "platform-tenant-operator":
		return platform.SystemRoleTenantOperator
	case "platform-tenant-viewer":
		return platform.SystemRoleTenantViewer
	default:
		return crName
	}
}

func RoleCRNameForID(roleID string) string {
	switch roleID {
	case "00000000-0000-4000-8000-000000000001":
		return RoleCRName(platform.SystemRoleRoot)
	case "00000000-0000-4000-8000-000000000002":
		return RoleCRName(platform.SystemRoleTenantAdmin)
	case "00000000-0000-4000-8000-000000000003":
		return RoleCRName(platform.SystemRoleTenantOperator)
	case "00000000-0000-4000-8000-000000000004":
		return RoleCRName(platform.SystemRoleTenantViewer)
	default:
		return ""
	}
}
