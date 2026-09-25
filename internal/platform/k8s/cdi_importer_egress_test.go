package k8s

import (
	"context"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/virtfoundry/core/internal/platform/branding"
)

func TestEnsureTenantNamespaceCreatesCDIImporterEgressPolicy(t *testing.T) {
	cs := fake.NewSimpleClientset()
	m := &Manager{Clientset: cs}
	ctx := context.Background()

	res, err := m.EnsureTenantNamespace(ctx, "tenant-1", "acme", DefaultTenantQuota())
	if err != nil {
		t.Fatalf("EnsureTenantNamespace: %v", err)
	}
	if res.Namespace != "virtfoundry-tenant-acme" {
		t.Fatalf("namespace = %q", res.Namespace)
	}

	got := mustGetImporterEgress(t, cs, res.Namespace)
	assertImporterEgressPolicy(t, got)
}

func TestEnsureTenantNamespaceUpdatesExistingCDIImporterEgressPolicy(t *testing.T) {
	cs := fake.NewSimpleClientset()
	m := &Manager{Clientset: cs}
	ctx := context.Background()

	ns := "virtfoundry-tenant-acme"
	stale := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      branding.CDIImporterEgressPolicyName,
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"stale": "true"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
		},
	}
	if _, err := cs.NetworkingV1().NetworkPolicies(ns).Create(ctx, stale, metav1.CreateOptions{}); err != nil {
		t.Fatalf("seed stale policy: %v", err)
	}

	if _, err := m.EnsureTenantNamespace(ctx, "tenant-1", "acme", DefaultTenantQuota()); err != nil {
		t.Fatalf("EnsureTenantNamespace: %v", err)
	}

	got := mustGetImporterEgress(t, cs, ns)
	assertImporterEgressPolicy(t, got)
	if got.Spec.PodSelector.MatchLabels["stale"] == "true" {
		t.Fatal("expected stale podSelector to be replaced")
	}
}

func mustGetImporterEgress(t *testing.T, cs *fake.Clientset, ns string) *networkingv1.NetworkPolicy {
	t.Helper()
	got, err := cs.NetworkingV1().NetworkPolicies(ns).Get(
		context.Background(), branding.CDIImporterEgressPolicyName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get NetworkPolicy: %v", err)
	}
	return got
}

func assertImporterEgressPolicy(t *testing.T, got *networkingv1.NetworkPolicy) {
	t.Helper()

	if got.Labels[LabelManagedBy] != ManagedByValue {
		t.Fatalf("managed-by label = %q", got.Labels[LabelManagedBy])
	}
	if got.Spec.PodSelector.MatchLabels[cdiComponentLabelKey] != cdiImporterComponent {
		t.Fatalf("podSelector = %#v", got.Spec.PodSelector.MatchLabels)
	}
	if len(got.Spec.PolicyTypes) != 1 || got.Spec.PolicyTypes[0] != networkingv1.PolicyTypeEgress {
		t.Fatalf("policyTypes = %#v", got.Spec.PolicyTypes)
	}
	if len(got.Spec.Egress) != 3 {
		t.Fatalf("egress rules = %d, want 3 (dns + ipv4 + ipv6)", len(got.Spec.Egress))
	}

	dns := got.Spec.Egress[0]
	if dns.To[0].NamespaceSelector == nil ||
		dns.To[0].NamespaceSelector.MatchLabels[kubeSystemNSLabel] != kubeSystemNSName {
		t.Fatalf("dns namespaceSelector = %#v", dns.To[0].NamespaceSelector)
	}
	if dns.To[0].PodSelector == nil ||
		dns.To[0].PodSelector.MatchLabels[kubeDNSAppLabelKey] != kubeDNSAppLabelValue {
		t.Fatalf("dns podSelector = %#v", dns.To[0].PodSelector)
	}
	if len(dns.Ports) != 2 {
		t.Fatalf("dns ports = %d, want UDP+TCP 53", len(dns.Ports))
	}

	v4 := got.Spec.Egress[1].To[0].IPBlock
	if v4 == nil || v4.CIDR != "0.0.0.0/0" {
		t.Fatalf("ipv4 cidr = %#v", v4)
	}
	wantExcept := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "100.64.0.0/10",
	}
	if len(v4.Except) != len(wantExcept) {
		t.Fatalf("ipv4 except = %#v", v4.Except)
	}
	for i, cidr := range wantExcept {
		if v4.Except[i] != cidr {
			t.Fatalf("ipv4 except[%d] = %q, want %q", i, v4.Except[i], cidr)
		}
	}

	v6 := got.Spec.Egress[2].To[0].IPBlock
	if v6 == nil || v6.CIDR != "::/0" {
		t.Fatalf("ipv6 cidr = %#v", v6)
	}
	wantV6 := []string{"fc00::/7", "fe80::/10", "::1/128"}
	if len(v6.Except) != len(wantV6) {
		t.Fatalf("ipv6 except = %#v", v6.Except)
	}
}
