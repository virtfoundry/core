package k8s

import (
	"context"
	"fmt"

	"github.com/virtforge-cloud/virtforge/internal/platform/branding"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NADLabels optional metadata on a network attachment definition.
type NADLabels map[string]string

func (m *Manager) CreateNetworkAttachment(ctx context.Context, namespace, name, cidr string, extra NADLabels) error {
	gvr := schema.GroupVersionResource{
		Group:    "k8s.cni.cncf.io",
		Version:  "v1",
		Resource: "network-attachment-definitions",
	}

	config := fmt.Sprintf(`{
  "cniVersion": "0.3.1",
  "name": %q,
  "type": "bridge",
  "bridge": %q,
  "ipam": {
    "type": "host-local",
    "subnet": %q,
    "routes": [{ "dst": "0.0.0.0/0" }]
  }
}`, name, branding.BridgeName, cidr)

	labels := map[string]interface{}{
		LabelManagedBy: ManagedByValue,
	}
	for k, v := range extra {
		labels[k] = v
	}

	nad := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "k8s.cni.cncf.io/v1",
			"kind":       "NetworkAttachmentDefinition",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"config": config,
			},
		},
	}

	_, err := m.Dynamic.Resource(gvr).Namespace(namespace).Create(ctx, nad, metav1.CreateOptions{})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create network attachment: %w", err)
	}
	return nil
}

func (m *Manager) DeleteNetworkAttachment(ctx context.Context, namespace, name string) error {
	gvr := schema.GroupVersionResource{
		Group:    "k8s.cni.cncf.io",
		Version:  "v1",
		Resource: "network-attachment-definitions",
	}
	err := m.Dynamic.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("delete network attachment: %w", err)
	}
	return nil
}
