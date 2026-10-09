package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/infra/hypervisor"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestCountVMStates(t *testing.T) {
	vms := []*platform.PlatformVM{
		{State: "running"},
		{State: "Running"},
		{State: "error"},
		{State: "starting"},
		{State: "stopped"},
	}
	tally := countVMStates(vms)
	if tally.running != 2 {
		t.Fatalf("running=%d want 2", tally.running)
	}
	if tally.errors != 1 {
		t.Fatalf("errors=%d want 1", tally.errors)
	}
	if tally.transitional != 1 {
		t.Fatalf("transitional=%d want 1", tally.transitional)
	}
}

func TestDashboardHealth(t *testing.T) {
	if dashboardHealth(0, 0) != "ok" {
		t.Fatal("expected ok")
	}
	if dashboardHealth(0, 1) != "warning" {
		t.Fatal("expected warning")
	}
	if dashboardHealth(1, 0) != "critical" {
		t.Fatal("expected critical")
	}
}

func TestMatchesQuery(t *testing.T) {
	if !matchesQuery("web", "web-server", "10.0.0.5") {
		t.Fatal("expected match on name")
	}
	if matchesQuery("x", "abc") {
		t.Fatal("expected no match")
	}
}

func seedDashboardVMs(t *testing.T) (*PlatformService, string) {
	t.Helper()
	st := store.NewMemory()
	tenantID := store.NewID()
	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "acme", Slug: "acme", Namespace: "vf-acme",
		State: "active", CreatedAt: store.Now(),
	})
	st.SaveVM(&platform.PlatformVM{
		ID: store.NewID(), TenantID: tenantID, Name: "secret-vm",
		DisplayName: "Secret", State: "error", ErrorMsg: "boom",
		UpdatedAt: time.Now().UTC(),
	})
	st.SaveVM(&platform.PlatformVM{
		ID: store.NewID(), TenantID: tenantID, Name: "starting-vm",
		State: "starting", UpdatedAt: time.Now().UTC(),
	})
	return NewPlatformService(st, nil, nil, nil), tenantID
}

func TestDashboardSummaryHidesVMNamesWithoutVMsRead(t *testing.T) {
	svc, tenantID := seedDashboardVMs(t)
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVolumesRead}, "")
	if err != nil {
		t.Fatalf("DashboardSummary: %v", err)
	}
	if summary.VMs.Total != 0 || len(summary.RecentActivity) != 0 {
		t.Fatalf("expected no VM data without vms:read, got vms=%+v activity=%d", summary.VMs, len(summary.RecentActivity))
	}
	for _, a := range summary.RecentActivity {
		if a.Name == "secret-vm" {
			t.Fatal("leaked VM name without vms:read")
		}
	}
}

func TestDashboardSummaryIncludesVMsWithVMsRead(t *testing.T) {
	svc, tenantID := seedDashboardVMs(t)
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVMsRead}, "")
	if err != nil {
		t.Fatalf("DashboardSummary: %v", err)
	}
	if summary.VMs.Total != 2 {
		t.Fatalf("vms.total=%d want 2", summary.VMs.Total)
	}
	if len(summary.RecentActivity) == 0 {
		t.Fatal("expected recent activity with vms:read")
	}
}

func TestNotificationsHidesVMNamesWithoutVMsRead(t *testing.T) {
	svc, tenantID := seedDashboardVMs(t)
	items := svc.Notifications(context.Background(), tenantID, nil)
	if len(items) != 0 {
		t.Fatalf("expected empty notifications without vms:read, got %d", len(items))
	}
	items = svc.Notifications(context.Background(), tenantID, []string{auth.PermVMsRead})
	if len(items) == 0 {
		t.Fatal("expected notifications with vms:read")
	}
	found := false
	for _, it := range items {
		if it.Title == "secret-vm" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected secret-vm notification with vms:read")
	}
}

func TestDashboardSummaryHostsNilWithoutDriver(t *testing.T) {
	svc, tenantID := seedDashboardVMs(t)
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVMsRead}, "")
	if err != nil {
		t.Fatalf("DashboardSummary: %v", err)
	}
	if summary.Hosts != nil {
		t.Fatalf("expected hosts=nil when driver is nil, got %+v", summary.Hosts)
	}
}

func TestDashboardSummaryHostsPopulatedWithDriver(t *testing.T) {
	st := store.NewMemory()
	tenantID := store.NewID()
	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "acme", Slug: "acme", Namespace: "vf-acme",
		State: "active", CreatedAt: store.Now(),
	})
	cs := fake.NewSimpleClientset(
		&k8sv1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "n1"},
			Status: k8sv1.NodeStatus{
				Capacity: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("4"),
					k8sv1.ResourceMemory: resource.MustParse("16Gi"),
				},
				Allocatable: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("3500m"),
					k8sv1.ResourceMemory: resource.MustParse("14Gi"),
				},
				NodeInfo: k8sv1.NodeSystemInfo{
					KubeletVersion: "v1.30.0",
					OSImage:        "Ubuntu 22.04",
					Architecture:   "amd64",
				},
				Conditions: []k8sv1.NodeCondition{
					{Type: k8sv1.NodeReady, Status: k8sv1.ConditionTrue},
				},
			},
		},
	)
	kv := hypervisor.NewKubeVirtDriverForTest(nil, cs)
	svc := NewPlatformService(st, nil, kv, nil)
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVolumesRead}, platform.RoleRoot)
	if err != nil {
		t.Fatalf("DashboardSummary: %v", err)
	}
	if summary.Hosts == nil {
		t.Fatal("expected hosts populated when driver returns metrics")
	}
	if summary.Hosts.Nodes != 1 || summary.Hosts.NodesReady != 1 {
		t.Fatalf("nodes=%d ready=%d want 1/1", summary.Hosts.Nodes, summary.Hosts.NodesReady)
	}
	if summary.Hosts.CPUAllocatableMilli != 3500 {
		t.Fatalf("cpu_allocatable_millicores=%d want 3500", summary.Hosts.CPUAllocatableMilli)
	}
	if summary.Hosts.Usage != nil {
		t.Fatalf("expected usage=nil with fake clientset (no RESTClient), got %+v", summary.Hosts.Usage)
	}
}

func TestDashboardSummaryUsagePopulated(t *testing.T) {
	st := store.NewMemory()
	tenantID := store.NewID()
	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "acme", Slug: "acme", Namespace: "vf-acme",
		State: "active", CreatedAt: store.Now(),
	})
	node := &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "n1"},
		Status: k8sv1.NodeStatus{
			Allocatable: k8sv1.ResourceList{
				k8sv1.ResourceCPU:    resource.MustParse("3500m"),
				k8sv1.ResourceMemory: resource.MustParse("14Gi"),
			},
			Conditions: []k8sv1.NodeCondition{
				{Type: k8sv1.NodeReady, Status: k8sv1.ConditionTrue},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/nodes":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items": [` + nodeJSON(node) + `]}`))
		case r.URL.Path == "/apis/metrics.k8s.io/v1beta1/nodes":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items": [{"usage": {"cpu": "1234m", "memory": "7Gi"}}]}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/namespaces/default/persistentvolumeclaims"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items": []}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	cfg := &rest.Config{Host: srv.URL}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("build client: %v", err)
	}
	cs.Discovery() // warm-up: not strictly required, documents intent

	kv := hypervisor.NewKubeVirtDriverForTest(nil, cs)
	svc := NewPlatformService(st, nil, kv, nil)
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVolumesRead}, platform.RoleRoot)
	if err != nil {
		t.Fatalf("DashboardSummary: %v", err)
	}
	if summary.Hosts == nil {
		t.Fatal("expected hosts populated")
	}
	if summary.Hosts.Usage == nil {
		t.Fatal("expected usage populated when metrics.k8s.io is reachable")
	}
	if summary.Hosts.Usage.CPUUsageMilli != 1234 {
		t.Fatalf("cpu_usage_millicores=%d want 1234", summary.Hosts.Usage.CPUUsageMilli)
	}
	if summary.Hosts.Usage.MemoryUsage != 7*1024*1024*1024 {
		t.Fatalf("memory_usage_bytes=%d want 7Gi", summary.Hosts.Usage.MemoryUsage)
	}
}

func nodeJSON(n *k8sv1.Node) string {
	n.TypeMeta = metav1.TypeMeta{Kind: "Node", APIVersion: "v1"}
	b, _ := json.Marshal(n)
	return string(b)
}

func TestDashboardSummaryHidesClusterOverviewForNonRoot(t *testing.T) {
	svc, tenantID := seedDashboardVMs(t)
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVMsRead}, platform.RoleTenantAdmin)
	if err != nil {
		t.Fatalf("DashboardSummary: %v", err)
	}
	if summary.Hosts != nil {
		t.Fatalf("expected hosts=nil for non-root, got %+v", summary.Hosts)
	}
	if summary.Storage != nil {
		t.Fatalf("expected storage=nil for non-root, got %+v", summary.Storage)
	}
	if summary.Addons != nil {
		t.Fatalf("expected addons=nil for non-root, got %+v", summary.Addons)
	}
	if summary.VMs.Total == 0 {
		t.Fatal("non-root with vms:read must still see VM counts")
	}
}

func TestDashboardSummaryShowsClusterOverviewForRoot(t *testing.T) {
	svc, tenantID := seedDashboardVMs(t)
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVMsRead}, platform.RoleRoot)
	if err != nil {
		t.Fatalf("DashboardSummary: %v", err)
	}
	if summary.Hosts != nil {
		t.Fatal("driver is nil in seedDashboardVMs so hosts stays nil, that's fine")
	}
}
