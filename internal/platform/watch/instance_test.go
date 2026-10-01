package watch

import (
	"sync"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

type recordedBroadcast struct {
	tenantID  string
	eventType string
	payload   interface{}
}

type recordingHub struct {
	mu   sync.Mutex
	evs  []recordedBroadcast
}

func (h *recordingHub) BroadcastTenant(tenantID, eventType string, payload interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.evs = append(h.evs, recordedBroadcast{tenantID: tenantID, eventType: eventType, payload: payload})
}

func (h *recordingHub) events() []recordedBroadcast {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]recordedBroadcast, len(h.evs))
	copy(out, h.evs)
	return out
}

func testInstance(name, ns, phase, powerState, ip, tenantSlug string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": mapping.Group + "/" + mapping.Version,
		"kind":       "Instance",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": ns,
			"uid":       "uid-" + name,
			"labels": map[string]interface{}{
				mapping.LabelTenant: tenantSlug,
			},
		},
		"spec": map[string]interface{}{
			"displayName": name,
			"powerState":  powerState,
		},
		"status": map[string]interface{}{
			"phase": phase,
			"ip":    ip,
		},
	}}
	obj.SetCreationTimestamp(metav1.Now())
	return obj
}

func TestPayloadFromInstance_FullSemantics(t *testing.T) {
	obj := testInstance("web-01", "virtfoundry-tenant-acme", "Ready", "Running", "10.0.0.5", "acme")
	ev := PayloadFromInstance(obj, "tenant-acme")
	if ev.Name != "web-01" {
		t.Fatalf("name=%q", ev.Name)
	}
	if ev.ID != "uid-web-01" {
		t.Fatalf("id=%q", ev.ID)
	}
	if ev.State != "Running" { // Ready → Running via InstancePhaseToPlatformState
		t.Fatalf("state=%q, want Running (not raw Ready)", ev.State)
	}
	if ev.PowerState != "Running" {
		t.Fatalf("power_state=%q, want Running (desired)", ev.PowerState)
	}
	if ev.IP != "10.0.0.5" {
		t.Fatalf("ip=%q", ev.IP)
	}
	if ev.TenantID != "tenant-acme" {
		t.Fatalf("tenant_id=%q", ev.TenantID)
	}
}

func TestPayloadFromInstance_HaltedDesiredNotStatusOnly(t *testing.T) {
	// Operator lag: status still Ready/Running while desired Halted (Stop in flight).
	obj := testInstance("web-01", "virtfoundry-tenant-acme", "Ready", "Halted", "10.0.0.5", "acme")
	ev := PayloadFromInstance(obj, "tenant-acme")
	if ev.State != "Running" {
		t.Fatalf("observed state=%q, want Running", ev.State)
	}
	if ev.PowerState != "Halted" {
		t.Fatalf("desired power_state=%q, want Halted", ev.PowerState)
	}
}

func TestInstancePublisher_AddUpdateDelete(t *testing.T) {
	hub := &recordingHub{}
	pub := &InstancePublisher{
		Hub: hub,
		Resolve: func(obj *unstructured.Unstructured) string {
			return "tenant-acme"
		},
	}

	obj := testInstance("web-01", "virtfoundry-tenant-acme", "Pending", "Running", "", "acme")
	pub.OnAdd(obj)

	updated := obj.DeepCopy()
	_ = unstructured.SetNestedField(updated.Object, "Ready", "status", "phase")
	_ = unstructured.SetNestedField(updated.Object, "10.0.0.5", "status", "ip")
	pub.OnUpdate(obj, updated)

	pub.OnDelete(updated)

	evs := hub.events()
	if len(evs) != 3 {
		t.Fatalf("events=%d, want 3: %+v", len(evs), evs)
	}
	if evs[0].eventType != eventCreated || evs[0].tenantID != "tenant-acme" {
		t.Fatalf("add: %+v", evs[0])
	}
	created, ok := evs[0].payload.(VMEvent)
	if !ok || created.Name != "web-01" || created.State != "Pending" || created.PowerState != "Running" {
		t.Fatalf("created payload: %#v", evs[0].payload)
	}

	if evs[1].eventType != eventUpdated {
		t.Fatalf("update type: %+v", evs[1])
	}
	upd, ok := evs[1].payload.(VMEvent)
	if !ok || upd.State != "Running" || upd.IP != "10.0.0.5" || upd.PowerState != "Running" {
		t.Fatalf("updated payload: %#v", evs[1].payload)
	}

	if evs[2].eventType != eventDeleted {
		t.Fatalf("delete type: %+v", evs[2])
	}
	del, ok := evs[2].payload.(VMEvent)
	if !ok || del.Name != "web-01" || del.ID == "" || del.TenantID != "tenant-acme" {
		t.Fatalf("deleted payload: %#v", evs[2].payload)
	}
}

func TestInstancePublisher_SkipsNoopUpdate(t *testing.T) {
	hub := &recordingHub{}
	pub := &InstancePublisher{
		Hub:     hub,
		Resolve: func(*unstructured.Unstructured) string { return "tenant-acme" },
	}
	obj := testInstance("web-01", "virtfoundry-tenant-acme", "Ready", "Running", "10.0.0.5", "acme")
	pub.OnUpdate(obj, obj.DeepCopy())
	if n := len(hub.events()); n != 0 {
		t.Fatalf("noop update broadcast %d events", n)
	}
}

func TestInstancePublisher_DropsWithoutTenant(t *testing.T) {
	hub := &recordingHub{}
	pub := &InstancePublisher{
		Hub:     hub,
		Resolve: func(*unstructured.Unstructured) string { return "" },
	}
	pub.OnAdd(testInstance("orphan", "kube-system", "Ready", "Running", "", ""))
	if n := len(hub.events()); n != 0 {
		t.Fatalf("expected drop, got %d", n)
	}
}

func TestInstancePublisher_DeletedFinalStateUnknown(t *testing.T) {
	hub := &recordingHub{}
	pub := &InstancePublisher{
		Hub:     hub,
		Resolve: func(*unstructured.Unstructured) string { return "tenant-acme" },
	}
	obj := testInstance("web-01", "virtfoundry-tenant-acme", "Ready", "Halted", "", "acme")
	pub.OnDelete(cache.DeletedFinalStateUnknown{Key: "ns/web-01", Obj: obj})
	evs := hub.events()
	if len(evs) != 1 || evs[0].eventType != eventDeleted {
		t.Fatalf("events=%+v", evs)
	}
}

func TestTenantResolverFromStore(t *testing.T) {
	mem := store.NewMemory()
	tenant := &platform.Tenant{
		ID: "tid-1", Name: "Acme", Slug: "acme",
		Namespace: "virtfoundry-tenant-acme", State: "active",
	}
	mem.SaveTenant(tenant)

	resolve := TenantResolverFromStore(mem)
	obj := testInstance("web-01", "virtfoundry-tenant-acme", "Ready", "Running", "", "acme")
	if got := resolve(obj); got != "tid-1" {
		t.Fatalf("resolve by ns = %q, want tid-1", got)
	}

	// Namespace miss, slug label hit.
	obj2 := testInstance("web-02", "other-ns", "Ready", "Running", "", "acme")
	if got := resolve(obj2); got != "tid-1" {
		t.Fatalf("resolve by slug label = %q, want tid-1", got)
	}
}

func TestInstanceInformerEnabled(t *testing.T) {
	t.Setenv("VF_INSTANCE_INFORMER", "")
	if !InstanceInformerEnabled(true) {
		t.Fatal("default on for kubernetes store")
	}
	if InstanceInformerEnabled(false) {
		t.Fatal("default off for memory store")
	}
	t.Setenv("VF_INSTANCE_INFORMER", "0")
	if InstanceInformerEnabled(true) {
		t.Fatal("explicit off")
	}
	t.Setenv("VF_INSTANCE_INFORMER", "1")
	if !InstanceInformerEnabled(false) {
		t.Fatal("explicit on")
	}
}
