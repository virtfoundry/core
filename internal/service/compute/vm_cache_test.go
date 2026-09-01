package compute

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
)

func TestStoreVMsHaveObservedState(t *testing.T) {
	if storeVMsHaveObservedState(nil) {
		t.Fatal("empty list should be false")
	}
	if storeVMsHaveObservedState([]*platform.PlatformVM{{State: "Pending"}}) {
		t.Fatal("pending should be false")
	}
	if !storeVMsHaveObservedState([]*platform.PlatformVM{{State: "Running"}}) {
		t.Fatal("running should be true")
	}
}
