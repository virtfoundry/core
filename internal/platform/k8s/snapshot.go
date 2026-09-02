package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var volumeSnapshotGVR = schema.GroupVersionResource{
	Group:    "snapshot.storage.k8s.io",
	Version:  "v1",
	Resource: "volumesnapshots",
}

func (m *Manager) CreateVolumeSnapshot(ctx context.Context, namespace, name, pvcName, snapshotClass string) (*unstructured.Unstructured, error) {
	spec := map[string]interface{}{
		"source": map[string]interface{}{
			"persistentVolumeClaimName": pvcName,
		},
	}
	if snapshotClass != "" {
		spec["volumeSnapshotClassName"] = snapshotClass
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "snapshot.storage.k8s.io/v1",
			"kind":       "VolumeSnapshot",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					LabelManagedBy: ManagedByValue,
				},
			},
			"spec": spec,
		},
	}

	created, err := m.Dynamic.Resource(volumeSnapshotGVR).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create volumesnapshot: %w", err)
	}
	return created, nil
}

func (m *Manager) DeleteVolumeSnapshot(ctx context.Context, namespace, name string) error {
	return m.Dynamic.Resource(volumeSnapshotGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (m *Manager) ListVolumeSnapshots(ctx context.Context, namespace string) (*unstructured.UnstructuredList, error) {
	return m.Dynamic.Resource(volumeSnapshotGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}
