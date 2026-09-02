package k8s

import (
	"context"
	"fmt"
	"time"

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

// VolumeSnapshotReady reports whether a VolumeSnapshot exists and is ReadyToUse.
func (m *Manager) VolumeSnapshotReady(ctx context.Context, namespace, name string) (bool, error) {
	obj, err := m.Dynamic.Resource(volumeSnapshotGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	ready, found, _ := unstructured.NestedBool(obj.Object, "status", "readyToUse")
	return found && ready, nil
}

// WaitVolumeSnapshotReady polls until the VolumeSnapshot is ReadyToUse or timeout seconds elapse.
func (m *Manager) WaitVolumeSnapshotReady(ctx context.Context, namespace, name string, timeoutSec int) (bool, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		obj, err := m.Dynamic.Resource(volumeSnapshotGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		ready, found, _ := unstructured.NestedBool(obj.Object, "status", "readyToUse")
		if found && ready {
			return true, nil
		}
		if msg, found, _ := unstructured.NestedString(obj.Object, "status", "error", "message"); found && msg != "" {
			return false, fmt.Errorf("%s", msg)
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return false, nil
}

func (m *Manager) ListVolumeSnapshots(ctx context.Context, namespace string) (*unstructured.UnstructuredList, error) {
	return m.Dynamic.Resource(volumeSnapshotGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}
