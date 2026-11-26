package store

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesStore_LoadBalancerRoundTrip(t *testing.T) {
	dyn := newTestDynamicClient()
	assignUIDsOnCreate(dyn)
	repo := &Kubernetes{dyn: dyn, clientset: kubefake.NewSimpleClientset()}
	repo.SaveTenant(&platform.Tenant{Name: "Acme", Slug: "acme", Namespace: "virtfoundry-tenant-acme", State: "active", CreatedAt: Now()})
	tenant, _ := repo.GetTenantBySlug("acme")

	tg := &platform.TargetGroup{ID: NewID(), TenantID: tenant.ID, Name: "web-tg", Protocol: "tcp", Port: 8080}
	repo.SaveTargetGroup(tg)
	lb := &platform.LoadBalancer{ID: NewID(), TenantID: tenant.ID, Name: "web", Description: "front"}
	repo.SaveLoadBalancer(lb)
	if lb.Namespace != "virtfoundry-tenant-acme" || lb.ServiceName != "lb-web" || lb.State != "Creating" {
		t.Fatalf("lb after save: %+v", lb)
	}

	l := &platform.LBListener{ID: NewID(), TenantID: tenant.ID, LoadBalancerID: lb.ID, Protocol: "tcp", Port: 80, TargetGroupID: tg.ID}
	repo.SaveLBListener(l)
	repo.SaveLBListener(l) // idempotent: same id replaces
	got := repo.ListLBListeners(lb.ID)
	if len(got) != 1 || got[0].Port != 80 || got[0].TargetGroupID != tg.ID || got[0].LoadBalancerID != lb.ID {
		t.Fatalf("listeners: %+v", got)
	}
	if _, ok := repo.GetLBListener(l.ID); !ok {
		t.Fatal("GetLBListener missed")
	}

	tgt := &platform.LBTarget{ID: NewID(), TenantID: tenant.ID, TargetGroupID: tg.ID, VMID: "vm1", IP: "10.0.0.4"}
	repo.SaveLBTarget(tgt)
	targets := repo.ListLBTargets(tg.ID)
	if len(targets) != 1 || targets[0].IP != "10.0.0.4" || targets[0].Port != 8080 {
		t.Fatalf("targets: %+v", targets)
	}

	// Saving the LoadBalancer again must not wipe its listeners.
	repo.SaveLoadBalancer(&platform.LoadBalancer{ID: lb.ID, TenantID: tenant.ID, Name: "web", Description: "edited"})
	if n := len(repo.ListLBListeners(lb.ID)); n != 1 {
		t.Fatalf("listeners wiped by metadata save: %d", n)
	}

	// Operator status becomes state and VIP.
	obj, ns, _ := repo.findNamespacedByID(mapping.LoadBalancerGVR, lb.ID)
	_ = unstructured.SetNestedField(obj.Object, "Ready", "status", "phase")
	_ = unstructured.SetNestedField(obj.Object, "10.0.50.7", "status", "vip")
	if _, err := repo.dyn.Resource(mapping.LoadBalancerGVR).Namespace(ns).Update(repo.ctx(), obj, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if cur, _ := repo.GetLoadBalancer(lb.ID); cur.State != "Active" || cur.VIP != "10.0.50.7" {
		t.Fatalf("status mapping: %+v", cur)
	}

	repo.DeleteLBTarget(tgt.ID)
	repo.DeleteLBListener(l.ID)
	if len(repo.ListLBTargets(tg.ID)) != 0 || len(repo.ListLBListeners(lb.ID)) != 0 {
		t.Fatal("entries not deleted")
	}
	repo.DeleteLoadBalancer(lb.ID)
	repo.DeleteTargetGroup(tg.ID)
	if len(repo.ListLoadBalancers(tenant.ID)) != 0 || len(repo.ListTargetGroups(tenant.ID)) != 0 {
		t.Fatal("CRs not deleted")
	}
}
