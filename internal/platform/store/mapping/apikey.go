package mapping

import (
	"fmt"
	"time"

	"github.com/virtfoundry/core/internal/platform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func APIKeySecretName(crName string) string {
	return fmt.Sprintf("vf-apikey-%s", crName)
}

func APIKeySecret(crName, hash, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIKeySecretName(crName),
			Namespace: namespace,
			Labels:    map[string]string{LabelPartOf: PartOfValue},
		},
		StringData: map[string]string{SecretKeyAPIHash: hash},
	}
}

func APIKeyToUnstructured(k *platform.APIKey, userCR, crName string) *unstructured.Unstructured {
	obj := newObject("APIKey", crName, "")
	SetLegacyID(obj, k.ID)
	spec := map[string]interface{}{
		"userRef": localRef(userCR),
		"name":    k.Name,
		"prefix":  k.Prefix,
		"secretRef": map[string]interface{}{
			"name": APIKeySecretName(crName),
			"key":  SecretKeyAPIHash,
		},
	}
	if len(k.Scopes) > 0 {
		scopes := make([]interface{}, len(k.Scopes))
		for i, s := range k.Scopes {
			scopes[i] = s
		}
		spec["scopes"] = scopes
	}
	if k.ExpiresAt != nil {
		spec["expiresAt"] = k.ExpiresAt.Format(time.RFC3339)
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func APIKeyFromUnstructured(obj *unstructured.Unstructured) *platform.APIKey {
	k := &platform.APIKey{
		ID:        ResourceID(obj),
		Name:      stringFromSpec(obj, "name"),
		Prefix:    stringFromSpec(obj, "prefix"),
		CreatedAt: obj.GetCreationTimestamp().Time,
	}
	scopes, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "scopes")
	k.Scopes = scopes
	if revoked, ok, _ := unstructured.NestedString(obj.Object, "status", "revokedAt"); ok && revoked != "" {
		if t, err := time.Parse(time.RFC3339, revoked); err == nil {
			k.RevokedAt = &t
		}
	}
	if used, ok, _ := unstructured.NestedString(obj.Object, "status", "lastUsedAt"); ok && used != "" {
		if t, err := time.Parse(time.RFC3339, used); err == nil {
			k.LastUsedAt = &t
		}
	}
	return k
}
