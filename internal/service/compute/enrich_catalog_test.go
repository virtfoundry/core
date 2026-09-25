package compute

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

func TestEnrichVMsFromCatalogFillsOfferingAndTemplate(t *testing.T) {
	mem := store.NewMemory()
	mem.SaveServiceOffering(&platform.ServiceOffering{
		ID: "off-1", Name: "small", CPU: 1, MemoryMi: 1024, State: "Active",
	})
	mem.SaveVMTemplate(&platform.VMTemplate{
		ID: "tmpl-1", Name: "ubuntu-2204", DisplayName: "Ubuntu 22.04", State: "Active",
	})

	s := &Service{store: mem}
	vms := []*platform.PlatformVM{{
		Name:              "teste",
		ServiceOfferingID: "small",
		TemplateRef:       "ubuntu-2204",
		CPU:               0,
		MemoryMi:          0,
	}}
	s.enrichVMsFromCatalog(vms)

	if vms[0].CPU != 1 || vms[0].MemoryMi != 1024 {
		t.Fatalf("offering enrich: cpu=%d mem=%d", vms[0].CPU, vms[0].MemoryMi)
	}
	if vms[0].Template != "Ubuntu 22.04" {
		t.Fatalf("template enrich: got %q", vms[0].Template)
	}
}

func TestEnrichVMsFromCatalogPreservesExistingSizing(t *testing.T) {
	mem := store.NewMemory()
	mem.SaveServiceOffering(&platform.ServiceOffering{
		ID: "off-1", Name: "small", CPU: 1, MemoryMi: 1024, State: "Active",
	})
	s := &Service{store: mem}
	vms := []*platform.PlatformVM{{
		ServiceOfferingID: "small",
		CPU:               4,
		MemoryMi:          8192,
	}}
	s.enrichVMsFromCatalog(vms)
	if vms[0].CPU != 4 || vms[0].MemoryMi != 8192 {
		t.Fatalf("should preserve existing sizing: %+v", vms[0])
	}
}
