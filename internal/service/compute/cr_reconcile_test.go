package compute

import (
	"context"
	"strings"
	"sync"
	"testing"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

type recordingHub struct {
	mu     sync.Mutex
	events []recordedEvent
}

type recordedEvent struct {
	tenantID  string
	eventType string
	payload   interface{}
}

func (h *recordingHub) BroadcastTenant(tenantID, eventType string, payload interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, recordedEvent{tenantID: tenantID, eventType: eventType, payload: payload})
}

func TestCanDeployViaOperator(t *testing.T) {
	s := &Service{operatorReconcile: true}
	linuxTmpl := &platform.VMTemplate{Name: "cirros", SourceType: "container"}
	isoTmpl := &platform.VMTemplate{Name: "win", SourceType: "iso"}

	if !s.canDeployViaOperator(linuxTmpl, DeployVMInput{}, nil) {
		t.Fatal("expected container deploy via operator")
	}
	if s.canDeployViaOperator(isoTmpl, DeployVMInput{}, nil) {
		t.Fatal("iso must not use CR-first path")
	}
	if s.canDeployViaOperator(linuxTmpl, DeployVMInput{PublicIP: true}, nil) {
		t.Fatal("public IP must not use CR-first path")
	}
	if s.canDeployViaOperator(linuxTmpl, DeployVMInput{}, []string{"net-1"}) {
		t.Fatal("extra networks must not use CR-first path yet")
	}
	if !s.canDeployViaOperator(linuxTmpl, DeployVMInput{SSHKeyID: "key-1"}, nil) {
		t.Fatal("SSH key should deploy via operator once sshKeyRefs is wired")
	}
	if s.canDeployViaOperator(linuxTmpl, DeployVMInput{CloudInitPassword: "x"}, nil) {
		t.Fatal("cloud_init_password must not use CR-first path")
	}
}

func TestOperatorDeployUnsupportedReason(t *testing.T) {
	msg := operatorDeployUnsupportedReason(
		&platform.VMTemplate{SourceType: "iso"},
		DeployVMInput{PublicIP: true, CloudInitPassword: "x"},
		[]string{"net-1"},
	)
	for _, want := range []string{"iso template", "public_ip", "extra networks", "cloud_init_password", "refuse hypervisor dual-write"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("reason missing %q: %s", want, msg)
		}
	}
}

func TestStartStopVM_OperatorReconcileUsesPowerStateOnly(t *testing.T) {
	mem := store.NewMemory()
	tenantID := "t-dual"
	mem.SaveTenant(&platform.Tenant{ID: tenantID, Slug: "acme", Namespace: "virtfoundry-tenant-acme"})
	vm := &platform.PlatformVM{
		ID:         "vm-1",
		TenantID:   tenantID,
		Name:       "demo",
		Namespace:  "virtfoundry-tenant-acme",
		State:      "Running",
		PowerState: instancePowerHalted,
	}
	mem.SaveVM(vm)

	// kvBase is nil: Start/Stop must not require a KubeVirt patch when
	// operatorReconcile is on (core#131 — CR powerState is the only actuator).
	s := &Service{store: mem, operatorReconcile: true}
	ctx := context.Background()

	started, err := s.StartVM(ctx, tenantID, "demo")
	if err != nil {
		t.Fatalf("StartVM: %v", err)
	}
	if started.PowerState != instancePowerRunning {
		t.Fatalf("StartVM powerState=%q, want %q", started.PowerState, instancePowerRunning)
	}
	stored, ok := mem.GetVMByName(tenantID, "demo")
	if !ok || stored.PowerState != instancePowerRunning {
		t.Fatalf("store powerState after Start=%q", stored.PowerState)
	}

	stopped, err := s.StopVM(ctx, tenantID, "demo")
	if err != nil {
		t.Fatalf("StopVM: %v", err)
	}
	if stopped.PowerState != instancePowerHalted {
		t.Fatalf("StopVM powerState=%q, want %q", stopped.PowerState, instancePowerHalted)
	}
}

func TestDeployVM_OperatorReconcileSingleActuatorInvariant(t *testing.T) {
	// core#131: with operatorReconcile, DeployVM either uses deployVMViaOperator
	// (SaveVM/Instance only) or returns BadRequest via operatorDeployUnsupportedReason.
	// Never CreateVM + SaveVM in the same flow.
	s := &Service{operatorReconcile: true}
	linux := &platform.VMTemplate{Name: "ubuntu", SourceType: "container"}

	if !s.canDeployViaOperator(linux, DeployVMInput{SSHKeyID: "key-1"}, nil) {
		t.Fatal("CR-first path must accept SSH-key deploys")
	}

	blocked := DeployVMInput{CloudInitPassword: "x", PublicIP: true}
	if s.canDeployViaOperator(linux, blocked, []string{"net-1"}) {
		t.Fatal("blocked shapes must not take CR-first path")
	}
	msg := operatorDeployUnsupportedReason(linux, blocked, []string{"net-1"})
	if !strings.Contains(msg, "refuse hypervisor dual-write") {
		t.Fatalf("expected dual-write refuse message, got %q", msg)
	}
	_ = iaerrors.NewBadRequestError(msg) // DeployVM wraps this exact helper
}

func TestSyncAllVMStates_OperatorReconcileBroadcastsPhaseChange(t *testing.T) {
	// core#132: under operatorReconcile, Sync must detect CR phase changes and
	// broadcast — not cache-only short-circuit.
	mem := store.NewMemory()
	tenantID := "t-sync"
	mem.SaveTenant(&platform.Tenant{ID: tenantID, Slug: "acme", Namespace: "virtfoundry-tenant-acme"})
	mem.SaveVM(&platform.PlatformVM{
		ID:        "vm-1",
		TenantID:  tenantID,
		Name:      "demo",
		Namespace: "virtfoundry-tenant-acme",
		State:     "Running",
	})

	hub := &recordingHub{}
	s := &Service{
		store:             mem,
		hub:               hub,
		operatorReconcile: true,
		vmStates:          make(map[vmStateKey]string),
	}
	ctx := context.Background()

	s.SyncAllVMStates(ctx)
	if len(hub.events) != 0 {
		t.Fatalf("first sync seeds signatures only, got %d events", len(hub.events))
	}

	stored, ok := mem.GetVMByName(tenantID, "demo")
	if !ok {
		t.Fatal("vm missing")
	}
	stored.State = "Pending"
	mem.SaveVM(stored)

	s.SyncAllVMStates(ctx)
	if len(hub.events) != 1 {
		t.Fatalf("phase change should broadcast once, got %d events", len(hub.events))
	}
	ev := hub.events[0]
	if ev.tenantID != tenantID || ev.eventType != "vm.updated" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	payload, ok := ev.payload.(vmEvent)
	if !ok {
		t.Fatalf("payload type %T", ev.payload)
	}
	if payload.Name != "demo" || payload.State != "Pending" {
		t.Fatalf("broadcast payload: %+v", payload)
	}

	// Intermediate Starting must also broadcast (no masking).
	stored.State = "Starting"
	mem.SaveVM(stored)
	s.SyncAllVMStates(ctx)
	if len(hub.events) != 2 {
		t.Fatalf("Starting change should broadcast, got %d events", len(hub.events))
	}
	payload, ok = hub.events[1].payload.(vmEvent)
	if !ok || payload.State != "Starting" {
		t.Fatalf("expected Starting broadcast, got %+v", hub.events[1].payload)
	}
}
