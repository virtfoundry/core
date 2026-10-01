package mapping

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
)

func TestOfferingToUnstructured_PlatformOwnedLabel(t *testing.T) {
	o := &platform.ServiceOffering{ID: "id", Name: "small-dedicated", DedicatedCPU: true}
	obj := OfferingToUnstructured(o)
	if got := obj.GetLabels()[LabelPlatformOwned]; got != "true" {
		t.Fatalf("want platform-owned=true, got %q", got)
	}
	if got := obj.GetLabels()[LabelPartOf]; got != PartOfValue {
		t.Fatalf("want part-of=%s, got %q", PartOfValue, got)
	}
}
