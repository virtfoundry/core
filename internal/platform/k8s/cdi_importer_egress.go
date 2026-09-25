package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/virtfoundry/core/internal/platform/branding"
)

// CDI importer pod labels from containerized-data-importer
// (pkg/common: CDIComponentLabel + ImporterPodName; also labeled
// app=containerized-data-importer).
const (
	cdiComponentLabelKey = "cdi.kubevirt.io"
	cdiImporterComponent = "importer"
	kubeSystemNSLabel    = "kubernetes.io/metadata.name"
	kubeSystemNSName     = "kube-system"
	kubeDNSAppLabelKey   = "k8s-app"
	kubeDNSAppLabelValue = "kube-dns"
)

// privateIPv4Except blocks cluster-internal, RFC1918, link-local, and CGNAT
// destinations when paired with cidr 0.0.0.0/0. Service/pod CIDRs that fall
// inside these ranges are therefore unreachable from importer pods.
var privateIPv4Except = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"100.64.0.0/10",
}

// privateIPv6Except mirrors the IPv4 deny set for dual-stack clusters (ULA,
// link-local, loopback). Clusters that are IPv4-only ignore ::/0 peers.
var privateIPv6Except = []string{
	"fc00::/7",
	"fe80::/10",
	"::1/128",
}

func dnsPort(proto corev1.Protocol) networkingv1.NetworkPolicyPort {
	port := intstr.FromInt32(53)
	p := proto
	return networkingv1.NetworkPolicyPort{Protocol: &p, Port: &port}
}

func cdiImporterEgressPolicy(namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      branding.CDIImporterEgressPolicyName,
			Namespace: namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			// Prefer the CDI component label so uploadserver / other CDI pods
			// in the tenant NS are not restricted by this egress policy.
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					cdiComponentLabelKey: cdiImporterComponent,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// CoreDNS / kube-dns (label k8s-app=kube-dns is the
					// conventional selector on both).
					To: []networkingv1.NetworkPolicyPeer{{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{kubeSystemNSLabel: kubeSystemNSName},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{kubeDNSAppLabelKey: kubeDNSAppLabelValue},
						},
					}},
					Ports: []networkingv1.NetworkPolicyPort{
						dnsPort(corev1.ProtocolUDP),
						dnsPort(corev1.ProtocolTCP),
					},
				},
				{
					To: []networkingv1.NetworkPolicyPeer{{
						IPBlock: &networkingv1.IPBlock{
							CIDR:   "0.0.0.0/0",
							Except: append([]string(nil), privateIPv4Except...),
						},
					}},
				},
				{
					To: []networkingv1.NetworkPolicyPeer{{
						IPBlock: &networkingv1.IPBlock{
							CIDR:   "::/0",
							Except: append([]string(nil), privateIPv6Except...),
						},
					}},
				},
			},
		},
	}
}

// ensureCDIImporterEgressPolicy creates or updates the tenant-namespaced
// egress NetworkPolicy that selects CDI importer pods. A chart-release-NS
// policy cannot cover these pods — they run in the DataVolume (tenant) NS.
func (m *Manager) ensureCDIImporterEgressPolicy(ctx context.Context, namespace string) error {
	policy := cdiImporterEgressPolicy(namespace)
	_, err := m.Clientset.NetworkingV1().NetworkPolicies(namespace).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil {
		if !isAlreadyExists(err) {
			return fmt.Errorf("create CDI importer egress NetworkPolicy: %w", err)
		}
		existing, getErr := m.Clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, policy.Name, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("get CDI importer egress NetworkPolicy: %w", getErr)
		}
		policy.ResourceVersion = existing.ResourceVersion
		_, err = m.Clientset.NetworkingV1().NetworkPolicies(namespace).Update(ctx, policy, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("update CDI importer egress NetworkPolicy: %w", err)
		}
	}
	return nil
}
