package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TenantResources struct {
	Namespace string
	Quota     *corev1.ResourceQuota
}

func (m *Manager) EnsureTenantNamespace(ctx context.Context, tenantID, slug string, quota TenantQuotaSpec) (*TenantResources, error) {
	nsName := TenantNamespace(slug)
	labels := map[string]string{
		LabelManagedBy: ManagedByValue,
		LabelTenantID:  tenantID,
		"nimbus.io/tenant-slug": slug,
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   nsName,
			Labels: labels,
		},
	}
	_, err := m.Clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nimbus-quota",
			Namespace: nsName,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourcePods:   resource.MustParse(fmt.Sprintf("%d", quota.MaxVMs*2+10)),
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%d", quota.CPULimit)),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dGi", quota.MemoryGiLimit)),
				corev1.ResourcePersistentVolumeClaims: resource.MustParse(fmt.Sprintf("%d", quota.MaxVolumes)),
			},
		},
	}
	createdRQ, err := m.Clientset.CoreV1().ResourceQuotas(nsName).Create(ctx, rq, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return nil, fmt.Errorf("create quota: %w", err)
	}

	return &TenantResources{Namespace: nsName, Quota: createdRQ}, nil
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
