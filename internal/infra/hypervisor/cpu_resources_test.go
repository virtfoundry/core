package hypervisor

import (
	"testing"

	k8sv1 "k8s.io/api/core/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

func TestGuestCPUCount(t *testing.T) {
	t.Parallel()
	if guestCPUCount(nil) != 0 {
		t.Fatalf("nil cpu")
	}
	if got := guestCPUCount(&kubevirtv1.CPU{Cores: 2}); got != 2 {
		t.Fatalf("cores: got %d", got)
	}
	if got := guestCPUCount(&kubevirtv1.CPU{Sockets: 2, Cores: 2, Threads: 2}); got != 8 {
		t.Fatalf("topology: got %d", got)
	}
}

func TestVMResourceRequirementsSharedOmitsCPURequest(t *testing.T) {
	t.Parallel()
	rr := vmResourceRequirements(1024, 2, false)
	if _, ok := rr.Requests[k8sv1.ResourceCPU]; ok {
		t.Fatal("shared must omit CPU request for cpuAllocationRatio")
	}
	mem := rr.Requests[k8sv1.ResourceMemory]
	if mem.Value() == 0 {
		t.Fatal("memory request required")
	}
	if len(rr.Limits) != 0 {
		t.Fatal("shared must not set CPU limits")
	}
}

func TestVMResourceRequirementsDedicatedGuaranteed(t *testing.T) {
	t.Parallel()
	rr := vmResourceRequirements(2048, 4, true)
	req, ok := rr.Requests[k8sv1.ResourceCPU]
	if !ok || req.Value() != 4 {
		t.Fatalf("dedicated request: %v", rr.Requests)
	}
	lim, ok := rr.Limits[k8sv1.ResourceCPU]
	if !ok || lim.Value() != 4 {
		t.Fatalf("dedicated limit: %v", rr.Limits)
	}
}

func TestGuestCPUSpecAlwaysSetsCores(t *testing.T) {
	t.Parallel()
	cpu := guestCPUSpec(3, false)
	if cpu == nil || cpu.Cores != 3 {
		t.Fatalf("got %+v", cpu)
	}
	if cpu.DedicatedCPUPlacement {
		t.Fatal("must not require CPU Manager DedicatedCPUPlacement")
	}
}
