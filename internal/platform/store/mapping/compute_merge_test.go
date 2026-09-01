package mapping

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
)

func TestInstancePhaseToPlatformState(t *testing.T) {
	tests := map[string]string{
		"Ready":    "Running",
		"Failed":   "Error",
		"Running":  "Running",
		"Pending":  "Pending",
		"Starting": "Starting",
	}
	for in, want := range tests {
		if got := InstancePhaseToPlatformState(in); got != want {
			t.Fatalf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestMergePlatformVMPreservesHypervisorFields(t *testing.T) {
	prior := &platform.PlatformVM{
		State: "Running", CPU: 2, MemoryMi: 2048, IP: "10.0.0.2", Image: "fedora",
	}
	fromCR := &platform.PlatformVM{
		State: "Pending", CPU: 0, MemoryMi: 0, Name: "vm1", ID: "id1",
	}
	dst := *fromCR
	MergePlatformVM(&dst, prior, fromCR)
	if dst.State != "Running" {
		t.Fatalf("state: got %q", dst.State)
	}
	if dst.CPU != 2 || dst.MemoryMi != 2048 || dst.IP != "10.0.0.2" {
		t.Fatalf("runtime fields not preserved: %+v", dst)
	}
	if dst.Name != "vm1" {
		t.Fatalf("cr fields lost: %+v", dst)
	}
}

func TestMergePlatformVMUsesCRStatusWhenSet(t *testing.T) {
	prior := &platform.PlatformVM{State: "Pending", IP: ""}
	fromCR := &platform.PlatformVM{State: "Running", IP: "10.0.0.5", Name: "vm1"}
	dst := *fromCR
	MergePlatformVM(&dst, prior, fromCR)
	if dst.State != "Running" || dst.IP != "10.0.0.5" {
		t.Fatalf("expected CR status, got %+v", dst)
	}
}
