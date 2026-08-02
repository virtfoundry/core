package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

var dataVolumeGVR = schema.GroupVersionResource{
	Group:    "cdi.kubevirt.io",
	Version:  "v1beta1",
	Resource: "datavolumes",
}

func (m *Manager) CreateBlankDataVolume(ctx context.Context, namespace, name, storageClass string, sizeGi int) error {
	if storageClass == "" {
		storageClass = "local-path"
	}
	obj := dataVolumeObject(namespace, name, map[string]interface{}{
		"source": map[string]interface{}{"blank": map[string]interface{}{}},
		"pvc": map[string]interface{}{
			"accessModes":      []interface{}{"ReadWriteOnce"},
			"storageClassName": storageClass,
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{"storage": fmt.Sprintf("%dGi", sizeGi)},
			},
		},
	})
	_, err := m.Dynamic.Resource(dataVolumeGVR).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create blank datavolume %s: %w", name, err)
	}
	return nil
}

func (m *Manager) CreateHTTPImportDataVolume(ctx context.Context, namespace, name, url, storageClass string, sizeGi int) error {
	if storageClass == "" {
		storageClass = "local-path"
	}
	obj := dataVolumeObject(namespace, name, map[string]interface{}{
		"source": map[string]interface{}{
			"http": map[string]interface{}{"url": url},
		},
		"pvc": map[string]interface{}{
			"accessModes":      []interface{}{"ReadWriteOnce"},
			"storageClassName": storageClass,
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{"storage": fmt.Sprintf("%dGi", sizeGi)},
			},
		},
	})
	_, err := m.Dynamic.Resource(dataVolumeGVR).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create iso datavolume %s: %w", name, err)
	}
	return nil
}

func dataVolumeObject(namespace, name string, spec map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cdi.kubevirt.io/v1beta1",
			"kind":       "DataVolume",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					LabelManagedBy: ManagedByValue,
				},
				"annotations": map[string]interface{}{
					"cdi.kubevirt.io/storage.bind.immediate.requested": "true",
				},
			},
			"spec": spec,
		},
	}
}

func (m *Manager) WaitDataVolumeReady(ctx context.Context, namespace, name string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		obj, err := m.Dynamic.Resource(dataVolumeGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
		switch phase {
		case "Succeeded", "Ready":
			return true, nil
		case "Failed", "Error":
			return false, fmt.Errorf("datavolume %s failed: phase=%s", name, phase)
		default:
			return false, nil
		}
	})
}

func (m *Manager) GetDataVolumePhase(ctx context.Context, namespace, name string) (string, error) {
	obj, err := m.Dynamic.Resource(dataVolumeGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	return phase, nil
}
