package service

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
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
