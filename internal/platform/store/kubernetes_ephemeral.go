package store

import (
	"time"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (k *Kubernetes) SaveAPIKey(key *platform.APIKey) {
	ns := mapping.SystemNamespace
	if key.TenantID != "" {
		if tns, ok := k.tenantNamespace(key.TenantID); ok {
			ns = tns
		}
	}
	userCR := ""
	if u, ok := k.GetUser(key.UserID); ok {
		userCR = mapping.UserCRName(u.Username)
	}
	crName := mapping.SanitizeCRName(key.Name)
	if crName == "" {
		crName = mapping.SanitizeCRName(key.Prefix)
	}
	obj := mapping.APIKeyToUnstructured(key, userCR, crName)
	secret := mapping.APIKeySecret(crName, key.SecretHash, ns)
	_, _ = k.clientset.CoreV1().Secrets(ns).Create(k.ctx(), secret, metav1.CreateOptions{})
	k.saveNamespacedMapped(mapping.APIKeyGVR, ns, func() *unstructured.Unstructured { return obj }, func(saved *unstructured.Unstructured) {
		*key = *k.apiKeyFromCR(saved, ns)
	})
}

func (k *Kubernetes) GetAPIKey(id string) (*platform.APIKey, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.APIKeyGVR, id)
	if !ok {
		return nil, false
	}
	return k.apiKeyFromCR(obj, ns), true
}

func (k *Kubernetes) GetAPIKeyByPrefix(prefix string) (*platform.APIKey, bool) {
	for _, obj := range k.listNamespacedAll(mapping.APIKeyGVR) {
		p, _, _ := unstructured.NestedString(obj.Object, "spec", "prefix")
		if p != prefix {
			continue
		}
		key := k.apiKeyFromCR(&obj, obj.GetNamespace())
		if key.RevokedAt == nil {
			return key, true
		}
	}
	return nil, false
}

func (k *Kubernetes) ListAPIKeys(userID string) []*platform.APIKey {
	var out []*platform.APIKey
	u, ok := k.GetUser(userID)
	if !ok {
		return nil
	}
	userCR := mapping.UserCRName(u.Username)
	for _, obj := range k.listNamespacedAll(mapping.APIKeyGVR) {
		ref, _, _ := unstructured.NestedString(obj.Object, "spec", "userRef", "name")
		if ref == userCR {
			out = append(out, k.apiKeyFromCR(&obj, obj.GetNamespace()))
		}
	}
	return out
}

func (k *Kubernetes) ListAPIKeysByTenant(tenantID string) []*platform.APIKey {
	var out []*platform.APIKey
	for _, obj := range k.listNamespacedAll(mapping.APIKeyGVR) {
		key := k.apiKeyFromCR(&obj, obj.GetNamespace())
		if key.TenantID == tenantID {
			out = append(out, key)
		}
	}
	return out
}

func (k *Kubernetes) DeleteAPIKey(id string) {
	key, ok := k.GetAPIKey(id)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key.RevokedAt = &now
	k.SaveAPIKey(key)
}

func (k *Kubernetes) TouchAPIKeyLastUsed(id string) {
	key, ok := k.GetAPIKey(id)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key.LastUsedAt = &now
	k.SaveAPIKey(key)
}

func (k *Kubernetes) apiKeyFromCR(obj *unstructured.Unstructured, ns string) *platform.APIKey {
	key := mapping.APIKeyFromUnstructured(obj)
	secretRef, _, _ := unstructured.NestedString(obj.Object, "spec", "secretRef", "name")
	if secretRef == "" {
		secretRef = mapping.APIKeySecretName(obj.GetName())
	}
	if sec, err := k.clientset.CoreV1().Secrets(ns).Get(k.ctx(), secretRef, metav1.GetOptions{}); err == nil {
		key.SecretHash = string(sec.Data[mapping.SecretKeyAPIHash])
	}
	userRef, _, _ := unstructured.NestedString(obj.Object, "spec", "userRef", "name")
	for _, u := range k.ListUsers() {
		if mapping.UserCRName(u.Username) == userRef {
			key.UserID = u.ID
			key.TenantID = u.TenantID
			break
		}
	}
	return key
}

func (k *Kubernetes) SaveAuditEvent(e *platform.AuditEvent) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.auditEvents = append(k.auditEvents, e)
}

func (k *Kubernetes) ListAuditEvents(targetTenantID string, limit int) []*platform.AuditEvent {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	var out []*platform.AuditEvent
	for i := len(k.auditEvents) - 1; i >= 0 && len(out) < limit; i-- {
		if k.auditEvents[i].TargetTenantID == targetTenantID {
			out = append(out, k.auditEvents[i])
		}
	}
	return out
}

func (k *Kubernetes) SaveJob(j *platform.AsyncJob) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.jobs[j.ID] = j
}

func (k *Kubernetes) GetJob(id string) (*platform.AsyncJob, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	j, ok := k.jobs[id]
	return j, ok
}

func (k *Kubernetes) ListJobs(tenantID string) []*platform.AsyncJob {
	k.mu.RLock()
	defer k.mu.RUnlock()
	var out []*platform.AsyncJob
	for _, j := range k.jobs {
		if tenantID == "" || j.TenantID == tenantID {
			out = append(out, j)
		}
	}
	return out
}

func (k *Kubernetes) ListPendingJobs(limit int) []*platform.AsyncJob {
	k.mu.RLock()
	defer k.mu.RUnlock()
	var out []*platform.AsyncJob
	for _, j := range k.jobs {
		if j.Status != "pending" {
			continue
		}
		out = append(out, j)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func (k *Kubernetes) PurgeTenantData(tenantID string) {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return
	}
	gvrs := []schema.GroupVersionResource{
		mapping.InstanceGVR, mapping.DiskGVR, mapping.DiskSnapshotGVR, mapping.InstanceSnapshotGVR,
		mapping.NetworkGVR, mapping.SecurityGroupGVR, mapping.VPCGVR, mapping.TemplateGVR,
		mapping.SSHKeyGVR, mapping.APIKeyGVR, mapping.IPAddressGVR,
	}
	for _, gvr := range gvrs {
		list, err := k.dyn.Resource(gvr).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
		if err != nil {
			continue
		}
		for i := range list.Items {
			k.deleteNamespaced(gvr, ns, list.Items[i].GetName())
		}
	}
	for _, u := range k.ListUsersByTenant(tenantID) {
		k.DeleteUser(u.ID)
	}
}
