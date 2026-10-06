package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/virtfoundry/core/internal/platform/branding"
)

// The operator refuses to adopt a tenant namespace that is not labelled
// part-of=virtfoundry + virtfoundry.io/tenant={slug}; it then fails the Tenant
// terminally and rejects every Instance in it. These tests pin the labels the
// API must write so the two components agree on who owns the namespace.
func assertOperatorTenantLabels(t *testing.T, labels map[string]string, slug string) {
	t.Helper()

	if got := labels[branding.LabelPartOf]; got != branding.PartOfValue {
		t.Errorf("%s = %q, want %q", branding.LabelPartOf, got, branding.PartOfValue)
	}
	if got := labels[branding.LabelTenant]; got != slug {
		t.Errorf("%s = %q, want %q", branding.LabelTenant, got, slug)
	}
}

func mustGetNamespace(t *testing.T, cs *fake.Clientset, name string) *corev1.Namespace {
	t.Helper()
	ns, err := cs.CoreV1().Namespaces().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get namespace %s: %v", name, err)
	}
	return ns
}

func TestEnsureTenantNamespaceStampsOperatorLabelsOnCreate(t *testing.T) {
	cs := fake.NewSimpleClientset()
	m := &Manager{Clientset: cs}

	res, err := m.EnsureTenantNamespace(context.Background(), "tenant-1", "acme", DefaultTenantQuota())
	if err != nil {
		t.Fatalf("EnsureTenantNamespace: %v", err)
	}

	ns := mustGetNamespace(t, cs, res.Namespace)
	assertOperatorTenantLabels(t, ns.Labels, "acme")

	if got := ns.Labels[LabelManagedBy]; got != ManagedByValue {
		t.Errorf("%s = %q, want %q", LabelManagedBy, got, ManagedByValue)
	}
	if got := ns.Labels[LabelTenantID]; got != "tenant-1" {
		t.Errorf("%s = %q, want %q", LabelTenantID, got, "tenant-1")
	}
	if got := ns.Labels[branding.LabelTenantSlug]; got != "acme" {
		t.Errorf("%s = %q, want %q", branding.LabelTenantSlug, got, "acme")
	}
}

// Regression: the default tenant namespace is created by the API before the
// Tenant CR exists, so the operator always sees it pre-existing. Before the
// labels were added the API wrote only virtfoundry.io/* keys and the operator
// rejected the namespace on every boot.
func TestEnsureTenantNamespaceBackfillsOperatorLabelsOnExisting(t *testing.T) {
	const nsName = "virtfoundry-tenant-default"

	// Exactly the state left behind by the API before this fix, plus a label the
	// operator owns, which the backfill must not clobber.
	existing := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				LabelManagedBy:           ManagedByValue,
				LabelTenantID:            "tenant-1",
				branding.LabelTenantSlug: "default",
				branding.AppManagedByKey: "virtfoundry-operator",
				"unrelated":              "keep-me",
			},
		},
	}
	cs := fake.NewSimpleClientset(existing)
	m := &Manager{Clientset: cs}

	if _, err := m.EnsureTenantNamespace(context.Background(), "tenant-1", "default", DefaultTenantQuota()); err != nil {
		t.Fatalf("EnsureTenantNamespace: %v", err)
	}

	ns := mustGetNamespace(t, cs, nsName)
	assertOperatorTenantLabels(t, ns.Labels, "default")

	if got := ns.Labels[branding.AppManagedByKey]; got != "virtfoundry-operator" {
		t.Errorf("operator-owned %s = %q, want it preserved", branding.AppManagedByKey, got)
	}
	if got := ns.Labels["unrelated"]; got != "keep-me" {
		t.Errorf("foreign label = %q, want it preserved", got)
	}
}

// A namespace mid-deletion cannot take new labels, so the backfill must skip
// it rather than error: the boot sweep re-ensures every tenant and one
// terminating namespace must not fail the whole API startup.
func TestEnsureTenantNamespaceToleratesTerminatingNamespace(t *testing.T) {
	const nsName = "virtfoundry-tenant-acme"

	now := metav1.Now()
	terminating := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              nsName,
			DeletionTimestamp: &now,
			Finalizers:        []string{"kubernetes"},
			Labels:            map[string]string{LabelManagedBy: ManagedByValue},
		},
	}
	cs := fake.NewSimpleClientset(terminating)
	m := &Manager{Clientset: cs}

	if _, err := m.EnsureTenantNamespace(context.Background(), "tenant-1", "acme", DefaultTenantQuota()); err != nil {
		t.Fatalf("EnsureTenantNamespace on terminating namespace: %v", err)
	}

	ns := mustGetNamespace(t, cs, nsName)
	if got := ns.Labels[branding.LabelPartOf]; got != "" {
		t.Errorf("%s = %q, want backfill skipped on a terminating namespace", branding.LabelPartOf, got)
	}
}

// A missing namespaces/patch grant must not abort startup: the backfill is best
// effort, and EnsureTenantNamespace still has to return the tenant resources.
func TestEnsureTenantNamespaceToleratesForbiddenLabelBackfill(t *testing.T) {
	cs := fake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   TenantNamespace("legacy"),
			Labels: map[string]string{LabelManagedBy: ManagedByValue},
		},
	})
	cs.PrependReactor("patch", "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Resource: "namespaces"}, TenantNamespace("legacy"), nil)
	})
	m := &Manager{Clientset: cs}

	res, err := m.EnsureTenantNamespace(context.Background(), "tenant-id", "legacy", TenantQuotaSpec{})
	if err != nil {
		t.Fatalf("EnsureTenantNamespace with forbidden backfill = %v, want nil", err)
	}
	if res == nil || res.Namespace != TenantNamespace("legacy") {
		t.Fatalf("resources = %#v, want namespace %s", res, TenantNamespace("legacy"))
	}
}
