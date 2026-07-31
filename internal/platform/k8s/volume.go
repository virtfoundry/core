package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) CreatePVC(ctx context.Context, namespace, name, storageClass string, sizeGi int) (*corev1.PersistentVolumeClaim, error) {
	if storageClass == "" {
		storageClass = "standard"
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{LabelManagedBy: ManagedByValue},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClass,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", sizeGi)),
				},
			},
		},
	}

	return m.Clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
}

func (m *Manager) DeletePVC(ctx context.Context, namespace, name string) error {
	return m.Clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (m *Manager) ListPVCs(ctx context.Context, namespace string) (*corev1.PersistentVolumeClaimList, error) {
	return m.Clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: LabelManagedBy + "=" + ManagedByValue,
	})
}
