package k8s

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/branding"
)

func allowAllPeer() networkingv1.NetworkPolicyPeer {
	return networkingv1.NetworkPolicyPeer{
		IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
	}
}

func (m *Manager) ApplySecurityGroup(ctx context.Context, namespace string, sg *platform.SecurityGroup) error {
	policyName := sanitizeK8sName("sg-" + sg.ID[:8])

	var ingress []networkingv1.NetworkPolicyIngressRule
	var egress []networkingv1.NetworkPolicyEgressRule

	for _, rule := range sg.Rules {
		proto := corev1.ProtocolTCP
		isICMP := false
		switch strings.ToLower(rule.Protocol) {
		case "udp":
			proto = corev1.ProtocolUDP
		case "icmp":
			isICMP = true
		}

		var ports []networkingv1.NetworkPolicyPort
		if !isICMP && rule.PortFrom > 0 {
			portRange := networkingv1.NetworkPolicyPort{Protocol: &proto}
			from := intstr.FromInt32(int32(rule.PortFrom))
			portRange.Port = &from
			if rule.PortTo > rule.PortFrom {
				end := int32(rule.PortTo)
				portRange.EndPort = &end
			}
			ports = []networkingv1.NetworkPolicyPort{portRange}
		}

		peer := networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{CIDR: rule.CIDR},
		}

		switch strings.ToLower(rule.Direction) {
		case "egress":
			egress = append(egress, networkingv1.NetworkPolicyEgressRule{
				Ports: ports,
				To:    []networkingv1.NetworkPolicyPeer{peer},
			})
		default:
			ingress = append(ingress, networkingv1.NetworkPolicyIngressRule{
				Ports: ports,
				From:  []networkingv1.NetworkPolicyPeer{peer},
			})
		}
	}

	if len(ingress) == 0 {
		ingress = append(ingress, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{allowAllPeer()},
		})
	}
	if len(egress) == 0 {
		egress = append(egress, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{allowAllPeer()},
		})
	}

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: namespace,
			Labels: map[string]string{
				LabelManagedBy:       ManagedByValue,
				branding.LabelSGID:   sg.ID,
				branding.LabelSGName: sg.Name,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{branding.SGPodLabelKey(sg.ID): "true"},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress:     ingress,
			Egress:      egress,
		},
	}

	_, err := m.Clientset.NetworkingV1().NetworkPolicies(namespace).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil {
		if isAlreadyExists(err) {
			existing, getErr := m.Clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, policyName, metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("update security group policy: %w", getErr)
			}
			policy.ResourceVersion = existing.ResourceVersion
			_, err = m.Clientset.NetworkingV1().NetworkPolicies(namespace).Update(ctx, policy, metav1.UpdateOptions{})
		}
		if err != nil {
			return fmt.Errorf("apply security group policy: %w", err)
		}
	}
	return nil
}

func (m *Manager) DeleteSecurityGroup(ctx context.Context, namespace, sgID string) error {
	name := sanitizeK8sName("sg-" + sgID[:8])
	err := m.Clientset.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

func sanitizeK8sName(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 63 {
		return out[:63]
	}
	return out
}
