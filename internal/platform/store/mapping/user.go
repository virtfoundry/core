package mapping

import (
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func UserCRName(username string) string {
	return strings.ReplaceAll(username, "_", "-")
}

func UserSecretName(userCRName string) string {
	return fmt.Sprintf("vf-user-%s", userCRName)
}

func UserToUnstructured(u *platform.User, roleCRName, tenantCRName string) *unstructured.Unstructured {
	crName := UserCRName(u.Username)
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(Group + "/" + Version)
	obj.SetKind("User")
	obj.SetName(crName)
	obj.SetLabels(map[string]string{LabelPartOf: PartOfValue})

	spec := map[string]interface{}{
		"username": u.Username,
		"roleRef": map[string]interface{}{
			"name": roleCRName,
		},
		"secretRef": map[string]interface{}{
			"name": UserSecretName(crName),
			"key":  SecretKeyPasswordHash,
		},
	}
	if u.Email != "" {
		spec["email"] = u.Email
	}
	if u.State != "" {
		spec["state"] = u.State
	}
	if tenantCRName != "" {
		spec["tenantRef"] = map[string]interface{}{"name": tenantCRName}
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func UserFromUnstructured(obj *unstructured.Unstructured, passwordHash string) *platform.User {
	u := &platform.User{
		ID:           string(obj.GetUID()),
		PasswordHash: passwordHash,
		CreatedAt:    obj.GetCreationTimestamp().Time,
	}
	username, _, _ := unstructured.NestedString(obj.Object, "spec", "username")
	email, _, _ := unstructured.NestedString(obj.Object, "spec", "email")
	state, _, _ := unstructured.NestedString(obj.Object, "spec", "state")
	roleRef, _, _ := unstructured.NestedString(obj.Object, "spec", "roleRef", "name")
	u.Username = username
	u.Email = email
	u.State = state
	u.RoleID = roleIDFromCRName(roleRef)
	u.Role = legacyRoleFromCRName(roleRef)
	return u
}

func UserSecret(crName, passwordHash string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      UserSecretName(crName),
			Namespace: SystemNamespace,
			Labels:    map[string]string{LabelPartOf: PartOfValue},
		},
		StringData: map[string]string{
			SecretKeyPasswordHash: passwordHash,
		},
	}
}

func roleIDFromCRName(roleCRName string) string {
	switch roleCRName {
	case RoleCRName(platform.SystemRoleRoot):
		return "00000000-0000-4000-8000-000000000001"
	case RoleCRName(platform.SystemRoleTenantAdmin):
		return "00000000-0000-4000-8000-000000000002"
	case RoleCRName(platform.SystemRoleTenantOperator):
		return "00000000-0000-4000-8000-000000000003"
	case RoleCRName(platform.SystemRoleTenantViewer):
		return "00000000-0000-4000-8000-000000000004"
	default:
		return ""
	}
}

func legacyRoleFromCRName(roleCRName string) platform.Role {
	switch roleCRName {
	case RoleCRName(platform.SystemRoleRoot):
		return platform.RoleRoot
	case RoleCRName(platform.SystemRoleTenantAdmin):
		return platform.RoleTenantAdmin
	default:
		return platform.RoleUser
	}
}
