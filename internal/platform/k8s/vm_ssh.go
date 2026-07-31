package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	LabelVMSSH = "nimbus.io/vm-ssh"
)

// EnsureVMSSHService exposes guest SSH via NodePort targeting the VM pod network IP.
func (m *Manager) EnsureVMSSHService(ctx context.Context, ns, vmName, vmIP string, nodePort int32) (int32, error) {
	svcName := vmName + "-ssh"
	labels := map[string]string{
		LabelManagedBy: ManagedByValue,
		LabelVMSSH:     vmName,
		"app.kubernetes.io/name": vmName,
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{{
				Name:       "ssh",
				Port:       22,
				TargetPort: intstr.FromInt32(22),
				NodePort:   nodePort,
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
	existing, err := m.Clientset.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err == nil {
		svc.Spec.ClusterIP = existing.Spec.ClusterIP
		svc.ResourceVersion = existing.ResourceVersion
		if _, err := m.Clientset.CoreV1().Services(ns).Update(ctx, svc, metav1.UpdateOptions{}); err != nil {
			return 0, fmt.Errorf("update ssh service: %w", err)
		}
	} else {
		if _, err := m.Clientset.CoreV1().Services(ns).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
			return 0, fmt.Errorf("create ssh service: %w", err)
		}
	}
	ep := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: ns, Labels: labels},
		Subsets: []corev1.EndpointSubset{{
			Addresses: []corev1.EndpointAddress{{IP: vmIP}},
			Ports:     []corev1.EndpointPort{{Name: "ssh", Port: 22, Protocol: corev1.ProtocolTCP}},
		}},
	}
	if _, err := m.Clientset.CoreV1().Endpoints(ns).Update(ctx, ep, metav1.UpdateOptions{}); err != nil {
		if _, err2 := m.Clientset.CoreV1().Endpoints(ns).Create(ctx, ep, metav1.CreateOptions{}); err2 != nil {
			return 0, fmt.Errorf("upsert ssh endpoints: %w", err2)
		}
	}
	out, err := m.Clientset.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	for _, p := range out.Spec.Ports {
		if p.NodePort != 0 {
			return p.NodePort, nil
		}
	}
	return nodePort, nil
}

// GetVMSSHNodePort returns the NodePort for an existing VM SSH service, if any.
func (m *Manager) GetVMSSHNodePort(ctx context.Context, ns, vmName string) (int32, bool, error) {
	svcName := vmName + "-ssh"
	svc, err := m.Clientset.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("get ssh service: %w", err)
	}
	for _, p := range svc.Spec.Ports {
		if p.NodePort != 0 {
			return p.NodePort, true, nil
		}
	}
	return 0, false, nil
}
