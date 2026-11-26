package compute

import (
	"context"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

type countingStore struct {
	store.Repository
	offeringLookups int
}

func (c *countingStore) GetServiceOffering(id string) (*platform.ServiceOffering, bool) {
	c.offeringLookups++
	return c.Repository.GetServiceOffering(id)
}

// A change made by the operator or another replica must show on the next read.
func TestOperatorReconcileReadsAreNotCached(t *testing.T) {
	mem := &countingStore{Repository: store.NewMemory()}
	mem.SaveTenant(&platform.Tenant{ID: "t1", Slug: "t1", Namespace: "virtfoundry-tenant-t1"})
	for _, name := range []string{"a", "b"} {
		if err := mem.SaveVM(&platform.PlatformVM{ID: name, TenantID: "t1", Name: name, State: "Running", ServiceOfferingID: "small"}); err != nil {
			t.Fatal(err)
		}
	}
	s := &Service{store: mem, operatorReconcile: true}
	if _, err := s.ListVMs(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	if mem.offeringLookups != 1 {
		t.Fatalf("offering lookups = %d, want 1", mem.offeringLookups)
	}
	if err := mem.SaveVM(&platform.PlatformVM{ID: "a", TenantID: "t1", Name: "a", State: "Stopped"}); err != nil {
		t.Fatal(err)
	}
	vms, err := s.ListVMs(context.Background(), "t1")
	if err != nil {
		t.Fatal(err)
	}
	for _, vm := range vms {
		if vm.Name == "a" && vm.State != "Stopped" {
			t.Fatalf("state = %s, want Stopped", vm.State)
		}
	}
	got, err := s.GetVM(context.Background(), "t1", "a")
	if err != nil || got.State != "Stopped" {
		t.Fatalf("GetVM = %+v, %v", got, err)
	}
}
