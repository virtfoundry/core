package k8s

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StreamVMPodLogs tails the virt-launcher pod for a running VM (guest console / compute).
func (m *Manager) StreamVMPodLogs(ctx context.Context, namespace, vmName string, tailLines int64, follow bool) (io.ReadCloser, error) {
	pods, err := m.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubevirt.io/domain=%s,kubevirt.io=virt-launcher", vmName),
	})
	if err != nil {
		return nil, fmt.Errorf("list instance runtime: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("instance runtime not found — start the virtual machine to view logs")
	}

	pod := pods.Items[0]
	container := "compute"
	for _, c := range pod.Spec.Containers {
		if c.Name == "guest-console-log" {
			container = "guest-console-log"
			break
		}
	}

	opts := &corev1.PodLogOptions{
		Container: container,
		Follow:    follow,
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}

	stream, err := m.Clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, opts).Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream instance logs: %w", err)
	}
	return stream, nil
}
