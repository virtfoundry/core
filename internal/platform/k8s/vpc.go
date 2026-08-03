package k8s

import (
	"context"

	"github.com/virtfoundry/core/internal/platform/branding"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) EnsureVPCNamespace(ctx context.Context, tenantID, tenantSlug, vpcID, vpcName, cidr string) (string, error) {
	nsName := VPCNamespace(tenantSlug, vpcID)
	labels := map[string]string{
		LabelManagedBy: ManagedByValue,
		LabelTenantID:  tenantID,
		LabelVPCID:     vpcID,
		branding.LabelVPCName: vpcName,
		branding.LabelCIDR:    cidr,
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: labels},
	}
	if _, err := m.Clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil && !isAlreadyExists(err) {
		return "", err
	}
	return nsName, nil
}
