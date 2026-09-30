package mapping

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
)

func TestNetworkCRName_UniquePerVPC(t *testing.T) {
	t.Parallel()
	a := &platform.Network{Name: "default", NetworkType: platform.NetworkTypeIsolated}
	b := &platform.Network{Name: "default", NetworkType: platform.NetworkTypeIsolated}

	nameA := NetworkCRName(a, "default")
	nameB := NetworkCRName(b, "audit-vpc")
	if nameA == nameB {
		t.Fatalf("expected unique CR names, both got %q", nameA)
	}
	if nameA != "default-default" {
		t.Fatalf("NetworkCRName(default, default) = %q, want default-default", nameA)
	}
	if nameB != "audit-vpc-default" {
		t.Fatalf("NetworkCRName(default, audit-vpc) = %q, want audit-vpc-default", nameB)
	}
}

func TestNetworkCRName_SharedPublic(t *testing.T) {
	t.Parallel()
	n := &platform.Network{Name: "anything", NetworkType: platform.NetworkTypeShared}
	if got := NetworkCRName(n, "ignored"); got != "public" {
		t.Fatalf("shared network CR name = %q, want public", got)
	}
}

func TestSGCRName_UniquePerVPC(t *testing.T) {
	t.Parallel()
	sg := &platform.SecurityGroup{Name: "web"}
	a := SGCRName(sg, "default")
	b := SGCRName(sg, "other")
	if a == b {
		t.Fatalf("expected unique SG CR names, both got %q", a)
	}
	if a != "default-web" || b != "other-web" {
		t.Fatalf("got %q and %q", a, b)
	}
}
