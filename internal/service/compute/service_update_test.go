package compute

import (
	"context"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

// TestUpdateVMUsesGetVMWhenStoreLookupMisses guards against the nil dereference
// where GetVMByName missed but GetVM (via the list cache) resolved the VM. The
// old code discarded the GetVM result and re-queried the store, leaving vm nil.
func TestUpdateVMUsesGetVMWhenStoreLookupMisses(t *testing.T) {
	mem := store.NewMemory()
	const tenantID = "t-update"

	s := New(mem, nil, nil, nil)
	s.operatorReconcile = true

	vm := &platform.PlatformVM{
		ID: "id-1", TenantID: tenantID, Name: "web", DisplayName: "web",
		State: "Running", CPU: 1, MemoryMi: 512, Tags: []string{"prod"},
	}
	mem.SaveVM(vm)

	ctx := context.Background()
	// Populate the list cache while the VM still resolves in the store.
	if vms, _ := s.ListVMs(ctx, tenantID); len(vms) != 1 {
		t.Fatalf("expected 1 cached VM, got %d", len(vms))
	}
	// Simulate the store no longer resolving the VM by name while the list
	// cache still has it: the condition that used to nil-deref.
	mem.DeleteVM(vm.ID)

	newTags := []string{"prod", "staging"}
	got, err := s.UpdateVM(ctx, tenantID, "web", UpdateVMInput{DisplayName: "Web", Tags: &newTags})
	if err != nil {
		t.Fatalf("UpdateVM returned error: %v", err)
	}
	if got == nil {
		t.Fatal("UpdateVM returned nil vm")
	}
	if got.DisplayName != "Web" {
		t.Fatalf("display name not applied: %q", got.DisplayName)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "prod" || got.Tags[1] != "staging" {
		t.Fatalf("tags not applied: %#v", got.Tags)
	}
}
