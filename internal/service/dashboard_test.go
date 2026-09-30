package service

import (
	"context"
	"testing"
	"time"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
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
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVolumesRead})
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
	summary, err := svc.DashboardSummary(context.Background(), tenantID, []string{auth.PermVMsRead})
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
