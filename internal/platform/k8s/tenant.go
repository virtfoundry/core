package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"go.uber.org/zap"

	"github.com/virtfoundry/core/internal/pkg/logger"
	"github.com/virtfoundry/core/internal/platform/branding"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type TenantResources struct {
	Namespace string
	Quota     *corev1.ResourceQuota
}

// tenantNamespaceLabels is the full label set for a virtfoundry-tenant-*
// namespace. app.kubernetes.io/part-of and virtfoundry.io/tenant are the
// ownership contract with virtfoundry-operator: it refuses to adopt a
// namespace without both (operator/internal/controller/tenant_namespace.go,
// assertTenantNamespaceOwned), fails the Tenant terminally, and rejects every
// Instance in it with "namespace %q is missing label
// app.kubernetes.io/part-of=virtfoundry".
//
// app.kubernetes.io/managed-by is deliberately omitted: the operator owns that
// key and rewrites it to "virtfoundry-operator" in stampNamespace.
func tenantNamespaceLabels(tenantID, slug string) map[string]string {
	return map[string]string{
		LabelManagedBy:           ManagedByValue,
		LabelTenantID:            tenantID,
		branding.LabelTenantSlug: slug,
		branding.LabelPartOf:     branding.PartOfValue,
		branding.LabelTenant:     slug,
	}
}

func (m *Manager) EnsureTenantNamespace(ctx context.Context, tenantID, slug string, quota TenantQuotaSpec) (*TenantResources, error) {
	nsName := TenantNamespace(slug)
	labels := tenantNamespaceLabels(tenantID, slug)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nsName,
			Labels: labels,
		},
	}
	_, err := m.Clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	switch {
	case err == nil:
		// Created with the full label set.
	case isAlreadyExists(err):
		// Namespaces created before the operator label contract existed are
		// missing those labels, and Create cannot heal them. Backfill so an
		// upgrade converges instead of leaving the Tenant terminally Failed.
		if err := m.backfillTenantNamespaceLabels(ctx, nsName, labels); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      branding.ResourceQuotaName,
			Namespace: nsName,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourcePods:                   resource.MustParse(fmt.Sprintf("%d", quota.MaxVMs*2+10)),
				corev1.ResourceCPU:                    resource.MustParse(fmt.Sprintf("%d", quota.CPULimit)),
				corev1.ResourceMemory:                 resource.MustParse(fmt.Sprintf("%dGi", quota.MemoryGiLimit)),
				corev1.ResourcePersistentVolumeClaims: resource.MustParse(fmt.Sprintf("%d", quota.MaxVolumes)),
			},
		},
	}
	createdRQ, err := m.Clientset.CoreV1().ResourceQuotas(nsName).Create(ctx, rq, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return nil, fmt.Errorf("create quota: %w", err)
	}
	if isAlreadyExists(err) {
		createdRQ, err = m.Clientset.CoreV1().ResourceQuotas(nsName).Get(ctx, branding.ResourceQuotaName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("get quota: %w", err)
		}
	}

	if err := m.ensureCDIImporterEgressPolicy(ctx, nsName); err != nil {
		return nil, err
	}

	return &TenantResources{Namespace: nsName, Quota: createdRQ}, nil
}

// backfillTenantNamespaceLabels adds labels an existing tenant namespace is
// missing. A JSON merge patch merges metadata.labels key by key, so labels the
// API does not manage (including the ones the operator stamps) are preserved.
func (m *Manager) backfillTenantNamespaceLabels(ctx context.Context, nsName string, want map[string]string) error {
	ns, err := m.Clientset.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get namespace %s: %w", nsName, err)
	}
	// A namespace being torn down cannot take new labels, and the Tenant record
	// is about to be recreated with a fresh namespace.
	if ns.DeletionTimestamp != nil {
		return nil
	}

	missing := map[string]string{}
	for k, v := range want {
		if ns.Labels[k] != v {
			missing[k] = v
		}
	}
	if len(missing) == 0 {
		return nil
	}

	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"labels": missing},
	})
	if err != nil {
		return fmt.Errorf("marshal labels for namespace %s: %w", nsName, err)
	}
	if _, err := m.Clientset.CoreV1().Namespaces().Patch(
		ctx, nsName, types.MergePatchType, patch, metav1.PatchOptions{},
	); err != nil {
		// The backfill is a best-effort migration for namespaces created before the
		// ownership contract existed. The API deliberately has no namespaces/patch
		// grant (least privilege; the operator owns tenant namespaces), so a legacy
		// namespace is labelled by hand once. Failing here would crash-loop API
		// startup through the default tenant bootstrap, so warn instead.
		if errors.IsForbidden(err) {
			logger.Warn("tenant namespace is missing operator ownership labels and the API cannot patch namespaces; label it manually (see helm-charts docs: operator recovery)",
				zap.String("namespace", nsName), zap.Strings("missing_labels", missingKeys(missing)), zap.Error(err))
			return nil
		}
		return fmt.Errorf("patch labels on namespace %s: %w", nsName, err)
	}
	return nil
}

func missingKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type TenantQuotaSpec struct {
	MaxVMs        int
	MaxVolumes    int
	CPULimit      int
	MemoryGiLimit int
}

func DefaultTenantQuota() TenantQuotaSpec {
	return TenantQuotaSpec{MaxVMs: 20, MaxVolumes: 50, CPULimit: 32, MemoryGiLimit: 64}
}

func isAlreadyExists(err error) bool {
	return errors.IsAlreadyExists(err)
}

func isNotFound(err error) bool {
	return errors.IsNotFound(err)
}

func (m *Manager) DeleteTenantNamespace(ctx context.Context, namespace string) error {
	if namespace == "" {
		return nil
	}
	err := m.Clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("delete namespace %s: %w", namespace, err)
	}
	return nil
}
