package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/virtfoundry/core/internal/platform/branding"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const LabelLoadBalancer = branding.LabelLoadBalancer

// LBPort describes one front-end listener on the LoadBalancer Service.
type LBPort struct {
	Name       string
	Port       int32
	TargetPort int32
	Protocol   corev1.Protocol
}

// LBEndpoint is a backend address for manual Endpoints.
type LBEndpoint struct {
	IP   string
	Port int32
	Name string
}

// EnsureLBService creates or updates a Service type LoadBalancer with empty selector.
func (m *Manager) EnsureLBService(ctx context.Context, ns, name, lbID string, ports []LBPort) (*corev1.Service, error) {
	if len(ports) == 0 {
		ports = []LBPort{{Name: "placeholder", Port: 65535, TargetPort: 65535, Protocol: corev1.ProtocolTCP}}
	}
	labels := map[string]string{
		LabelManagedBy:                ManagedByValue,
		LabelLoadBalancer:             lbID,
		"app.kubernetes.io/name":      name,
		"app.kubernetes.io/component": "load-balancer",
	}
	svcPorts := make([]corev1.ServicePort, 0, len(ports))
	for _, p := range ports {
		proto := p.Protocol
		if proto == "" {
			proto = corev1.ProtocolTCP
		}
		pname := p.Name
		if pname == "" {
			pname = fmt.Sprintf("p-%d", p.Port)
		}
		svcPorts = append(svcPorts, corev1.ServicePort{
			Name:       pname,
			Port:       p.Port,
			TargetPort: intstr.FromString(pname),
			Protocol:   proto,
		})
	}
	allocNodePorts := false
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:                          corev1.ServiceTypeLoadBalancer,
			AllocateLoadBalancerNodePorts: &allocNodePorts,
			Ports:                         svcPorts,
		},
	}
	existing, err := m.Clientset.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		svc.Spec.ClusterIP = existing.Spec.ClusterIP
		svc.ResourceVersion = existing.ResourceVersion
		if existing.Spec.HealthCheckNodePort != 0 {
			svc.Spec.HealthCheckNodePort = existing.Spec.HealthCheckNodePort
		}
		out, uerr := m.Clientset.CoreV1().Services(ns).Update(ctx, svc, metav1.UpdateOptions{})
		if uerr != nil {
			return nil, fmt.Errorf("update lb service: %w", uerr)
		}
		return out, nil
	}
	if !errors.IsNotFound(err) {
		return nil, fmt.Errorf("get lb service: %w", err)
	}
	out, err := m.Clientset.CoreV1().Services(ns).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		// Older clusters may reject AllocateLoadBalancerNodePorts=false — retry without.
		svc.Spec.AllocateLoadBalancerNodePorts = nil
		out, err = m.Clientset.CoreV1().Services(ns).Create(ctx, svc, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("create lb service: %w", err)
		}
	}
	return out, nil
}

// WaitLBVIP polls Service status until MetalLB assigns an external IP or timeout.
func (m *Manager) WaitLBVIP(ctx context.Context, ns, name string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for {
		svc, err := m.Clientset.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				return ing.IP, nil
			}
			if ing.Hostname != "" {
				return ing.Hostname, nil
			}
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for load balancer VIP on %s/%s", ns, name)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// SyncLBEndpoints upserts Endpoints for the LB Service (guest Multus IPs).
func (m *Manager) SyncLBEndpoints(ctx context.Context, ns, name, lbID string, endpoints []LBEndpoint) error {
	labels := map[string]string{
		LabelManagedBy:    ManagedByValue,
		LabelLoadBalancer: lbID,
	}
	ep := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
	}
	if len(endpoints) > 0 {
		addrSet := map[string]struct{}{}
		portMap := map[string]int32{}
		for _, e := range endpoints {
			if e.IP == "" {
				continue
			}
			addrSet[e.IP] = struct{}{}
			pname := e.Name
			if pname == "" {
				pname = fmt.Sprintf("p-%d", e.Port)
			}
			portMap[pname] = e.Port
		}
		subset := corev1.EndpointSubset{}
		for ip := range addrSet {
			subset.Addresses = append(subset.Addresses, corev1.EndpointAddress{IP: ip})
		}
		for pname, port := range portMap {
			subset.Ports = append(subset.Ports, corev1.EndpointPort{
				Name: pname, Port: port, Protocol: corev1.ProtocolTCP,
			})
		}
		if len(subset.Addresses) > 0 && len(subset.Ports) > 0 {
			ep.Subsets = []corev1.EndpointSubset{subset}
		}
	}
	_, err := m.Clientset.CoreV1().Endpoints(ns).Update(ctx, ep, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = m.Clientset.CoreV1().Endpoints(ns).Create(ctx, ep, metav1.CreateOptions{})
		}
		if err != nil {
			return fmt.Errorf("upsert lb endpoints: %w", err)
		}
	}
	return nil
}

// DeleteLB removes the Service and Endpoints for a load balancer.
func (m *Manager) DeleteLB(ctx context.Context, ns, name string) error {
	err := m.Clientset.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete lb service: %w", err)
	}
	err = m.Clientset.CoreV1().Endpoints(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete lb endpoints: %w", err)
	}
	return nil
}

// LBServiceName returns a DNS-safe Service name for a load balancer ID.
func LBServiceName(lbID string) string {
	id := strings.ToLower(strings.ReplaceAll(lbID, "-", ""))
	if len(id) > 8 {
		id = id[:8]
	}
	return "vf-lb-" + id
}
