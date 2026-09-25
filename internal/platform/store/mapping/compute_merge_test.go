package mapping

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func TestInstancePowerStateRoundTrip(t *testing.T) {
	vm := &platform.PlatformVM{
		Name:         "web-01",
		DisplayName:  "Web",
		PowerState:   "Halted",
		DedicatedCPU: true,
	}
	obj := InstanceToUnstructured(vm, "default", "small", "cirros", nil)
	got, err := InstanceFromUnstructured(obj, "tenant-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.PowerState != "Halted" {
		t.Fatalf("powerState: got %q", got.PowerState)
	}
	if !got.DedicatedCPU {
		t.Fatal("dedicatedCPU lost")
	}
	if got.TemplateRef != "cirros" {
		t.Fatalf("templateRef: got %q", got.TemplateRef)
	}
}

func TestInstanceToUnstructuredAnnotatesPodNetworkWhenNoMultusNics(t *testing.T) {
	vm := &platform.PlatformVM{
		Name: "teste",
		NICs: []platform.VMNic{{Name: "default", Type: "pod"}},
	}
	obj := InstanceToUnstructured(vm, "default", "small", "ubuntu-2204", nil)
	if got := obj.GetAnnotations()[AnnAllowPodNetwork]; got != "true" {
		t.Fatalf("allow-pod-network annotation: got %q want true", got)
	}
	if _, ok, _ := unstructured.NestedSlice(obj.Object, "spec", "nics"); ok {
		t.Fatal("expected no Multus spec.nics for pod-only NICs")
	}
	if got := obj.GetAnnotations()[AnnLegacyID]; got != "" {
		// SetLegacyID only when vm.ID set — ensure we did not wipe other anns when ID empty
		t.Fatalf("unexpected legacy-id without vm.ID: %q", got)
	}
}

func TestInstanceToUnstructuredKeepsLegacyIDWithPodNetworkAnnotation(t *testing.T) {
	vm := &platform.PlatformVM{ID: "id-1", Name: "web"}
	obj := InstanceToUnstructured(vm, "default", "small", "cirros", nil)
	if obj.GetAnnotations()[AnnLegacyID] != "id-1" {
		t.Fatalf("legacy-id lost: %#v", obj.GetAnnotations())
	}
	if obj.GetAnnotations()[AnnAllowPodNetwork] != "true" {
		t.Fatalf("allow-pod-network missing: %#v", obj.GetAnnotations())
	}
}

func TestInstanceToUnstructuredOmitsPodAnnotationWhenMultusNicsPresent(t *testing.T) {
	vm := &platform.PlatformVM{
		Name: "web",
		NICs: []platform.VMNic{{Name: "eth0", NetworkID: "net-1", Type: "multus"}},
	}
	obj := InstanceToUnstructured(vm, "default", "small", "cirros", map[string]string{"net-1": "default"})
	if _, ok := obj.GetAnnotations()[AnnAllowPodNetwork]; ok {
		t.Fatalf("should not set allow-pod-network when Multus nics written: %#v", obj.GetAnnotations())
	}
	nics, ok, err := unstructured.NestedSlice(obj.Object, "spec", "nics")
	if err != nil || !ok || len(nics) != 1 {
		t.Fatalf("spec.nics: ok=%v len=%d err=%v", ok, len(nics), err)
	}
}

func TestInstanceFromUnstructuredReturnsErrorForInvalidFieldType(t *testing.T) {
	obj := newObject("Instance", "web-01", "ns")
	obj.Object["spec"] = map[string]interface{}{"dedicatedCPU": "true"}

	if _, err := InstanceFromUnstructured(obj, "tenant-1", nil); err == nil {
		t.Fatal("expected invalid dedicatedCPU type to return an error")
	}
}

func TestMergeUnstructuredSpecKeepsExistingKeys(t *testing.T) {
	existing := newObject("Instance", "web-01", "ns")
	_ = unstructured.SetNestedMap(existing.Object, map[string]interface{}{
		"displayName": "Web",
		"offeringRef": map[string]interface{}{"name": "small"},
		"templateRef": map[string]interface{}{"name": "cirros"},
	}, "spec")
	incoming := newObject("Instance", "web-01", "ns")
	_ = unstructured.SetNestedMap(incoming.Object, map[string]interface{}{
		"displayName": "Web",
		"powerState":  "Halted",
	}, "spec")

	MergeUnstructuredSpec(existing, incoming)

	spec, _, _ := unstructured.NestedMap(existing.Object, "spec")
	if spec["powerState"] != "Halted" {
		t.Fatalf("powerState: %#v", spec["powerState"])
	}
	off, _, _ := unstructured.NestedString(existing.Object, "spec", "offeringRef", "name")
	if off != "small" {
		t.Fatalf("offeringRef wiped: %#v", spec["offeringRef"])
	}
	tmpl, _, _ := unstructured.NestedString(existing.Object, "spec", "templateRef", "name")
	if tmpl != "cirros" {
		t.Fatalf("templateRef wiped: %#v", spec["templateRef"])
	}
}
