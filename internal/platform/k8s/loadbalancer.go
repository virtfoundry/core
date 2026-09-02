package k8s

import (
	"context"
	"fmt"

	"github.com/virtfoundry/core/internal/platform/branding"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const LabelLoadBalancer = branding.Domain + "/load-balancer"

// LBServicePort maps a listener front port to a target port on backend VMs.
type LBServicePort struct {
	Name       string
	Port       int32
	TargetPort int32
	Protocol   corev1.Protocol
}

// EnsureLBService creates or updates a MetalLB LoadBalancer Service for tenant L4 VIP.
func (m *Manager) EnsureLBService(ctx context.Context, ns, svcName, lbID string, ports []LBServicePort) (string, error) {
	labels := map[string]string{
		LabelManagedBy:    ManagedByValue,
		LabelLoadBalancer: lbID,
		"app.kubernetes.io/name": svcName,
	}
	svcPorts := make([]corev1.ServicePort, 0, len(ports))
	for _, p := range ports {
		proto := p.Protocol
		if proto == "" {
			proto = corev1.ProtocolTCP
		}
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("%s-%d", proto, p.Port)
		}
		svcPorts = append(svcPorts, corev1.ServicePort{
			Name:       name,
			Port:       p.Port,
			TargetPort: intstr.FromInt32(p.TargetPort),
			Protocol:   proto,
		})
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeLoadBalancer,
			Ports: svcPorts,
		},
	}
	existing, err := m.Clientset.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err == nil {
		svc.Spec.ClusterIP = existing.Spec.ClusterIP
		svc.ResourceVersion = existing.ResourceVersion
		if _, err := m.Clientset.CoreV1().Services(ns).Update(ctx, svc, metav1.UpdateOptions{}); err != nil {
			return "", fmt.Errorf("update lb service: %w", err)
		}
	} else if errors.IsNotFound(err) {
		if _, err := m.Clientset.CoreV1().Services(ns).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
			return "", fmt.Errorf("create lb service: %w", err)
		}
	} else {
		return "", fmt.Errorf("get lb service: %w", err)
	}
	out, err := m.Clientset.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return loadBalancerVIP(out), nil
}

// DeleteLBService removes the LoadBalancer Service and its Endpoints.
func (m *Manager) DeleteLBService(ctx context.Context, ns, svcName string) error {
	if err := m.Clientset.CoreV1().Services(ns).Delete(ctx, svcName, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete lb service: %w", err)
	}
	if err := m.Clientset.CoreV1().Endpoints(ns).Delete(ctx, svcName, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete lb endpoints: %w", err)
	}
	return nil
}

// SyncLBEndpoints upserts manual Endpoints for guest VM IPs behind the LB Service.
func (m *Manager) SyncLBEndpoints(ctx context.Context, ns, svcName string, subsets []corev1.EndpointSubset) error {
	labels := map[string]string{
		LabelManagedBy: ManagedByValue,
		"app.kubernetes.io/name": svcName,
	}
	ep := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: ns, Labels: labels},
		Subsets:    subsets,
	}
	if _, err := m.Clientset.CoreV1().Endpoints(ns).Update(ctx, ep, metav1.UpdateOptions{}); err != nil {
		if errors.IsNotFound(err) {
			if _, err2 := m.Clientset.CoreV1().Endpoints(ns).Create(ctx, ep, metav1.CreateOptions{}); err2 != nil {
				return fmt.Errorf("create lb endpoints: %w", err2)
			}
			return nil
		}
		return fmt.Errorf("update lb endpoints: %w", err)
	}
	return nil
}

func loadBalancerVIP(svc *corev1.Service) string {
	if svc == nil {
		return ""
	}
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			return ing.IP
		}
		if ing.Hostname != "" {
			return ing.Hostname
		}
	}
	return ""
}
