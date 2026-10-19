package mapping

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
)

func TestDiskTagsRoundTrip(t *testing.T) {
	v := &platform.Volume{ID: "vol-1", Name: "data", SizeGi: 20, Tags: []string{"prod", " db ", ""}}
	obj := DiskToUnstructured(v, "acme", "")
	got := DiskFromUnstructured(obj, "tenant-1")
	if len(got.Tags) != 2 || got.Tags[0] != "prod" || got.Tags[1] != "db" {
		t.Fatalf("Volume tags round-trip: %#v", got.Tags)
	}
}

func TestDiskSnapshotTagsRoundTrip(t *testing.T) {
	s := &platform.Snapshot{ID: "snap-1", Name: "snap", VolumeID: "vol-1", Tags: []string{"gold", ""}}
	obj := DiskSnapshotToUnstructured(s, "disk-cr")
	got := DiskSnapshotFromUnstructured(obj, "tenant-1", "vol-1")
	if len(got.Tags) != 1 || got.Tags[0] != "gold" {
		t.Fatalf("Snapshot tags round-trip: %#v", got.Tags)
	}
}

func TestInstanceSnapshotTagsRoundTrip(t *testing.T) {
	s := &platform.VMSnapshot{ID: "vsnap-1", Name: "vsnap", Tags: []string{"pre-upgrade", " prod "}}
	obj := InstanceSnapshotToUnstructured(s, "vm-cr")
	got := InstanceSnapshotFromUnstructured(obj, "tenant-1", "vm-1")
	if len(got.Tags) != 2 || got.Tags[0] != "pre-upgrade" || got.Tags[1] != "prod" {
		t.Fatalf("VMSnapshot tags round-trip: %#v", got.Tags)
	}
}

func TestMergeVMSnapshotPreservesTagsWhenCRDrops(t *testing.T) {
	prior := &platform.VMSnapshot{Phase: "creating", VMName: "web", Tags: []string{"keep"}}
	fromCR := &platform.VMSnapshot{Name: "vsnap", Phase: "", VMName: "", Tags: nil}
	dst := *fromCR
	MergeVMSnapshot(&dst, prior, fromCR)
	if len(dst.Tags) != 1 || dst.Tags[0] != "keep" {
		t.Fatalf("expected tags preserved from prior: %#v", dst.Tags)
	}
	if dst.Phase != "creating" || dst.VMName != "web" {
		t.Fatalf("runtime fields not preserved: %+v", dst)
	}
}

func TestMergeVMSnapshotDoesNotResurrectClearedTags(t *testing.T) {
	prior := &platform.VMSnapshot{Phase: "ready", VMName: "web", Tags: nil}
	fromCR := &platform.VMSnapshot{Name: "vsnap", Tags: nil}
	dst := *fromCR
	MergeVMSnapshot(&dst, prior, fromCR)
	if len(dst.Tags) != 0 {
		t.Fatalf("cleared tags must stay cleared: %#v", dst.Tags)
	}
}

func TestSSHKeyTagsRoundTrip(t *testing.T) {
	k := &platform.SSHKeyPair{
		ID:        "key-1",
		Name:      "lab",
		PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPlaceholder",
		Tags:      []string{"team-a", ""},
	}
	obj := SSHKeyToUnstructured(k, "acme")
	got := SSHKeyFromUnstructured(obj, "tenant-1")
	if len(got.Tags) != 1 || got.Tags[0] != "team-a" {
		t.Fatalf("SSHKey tags round-trip: %#v", got.Tags)
	}
}

func TestNetworkTagsRoundTrip(t *testing.T) {
	n := &platform.Network{
		ID: "net-1", Name: "app", NetworkType: platform.NetworkTypeIsolated,
		CIDR: "10.0.0.0/24", VPCID: "vpc-1", Tags: []string{"frontend", " web "},
	}
	obj := NetworkToUnstructured(n, "acme", "vpc-cr")
	got := NetworkFromUnstructured(obj, "tenant-1", "vpc-1")
	if len(got.Tags) != 2 || got.Tags[0] != "frontend" || got.Tags[1] != "web" {
		t.Fatalf("Network tags round-trip: %#v", got.Tags)
	}
}

func TestVPCTagsRoundTrip(t *testing.T) {
	v := &platform.VPC{ID: "vpc-1", Name: "prod-vpc", CIDR: "10.0.0.0/16", Tags: []string{"prod", " edge ", ""}}
	obj := VPCToUnstructured(v, "acme")
	got := VPCFromUnstructured(obj, "tenant-1")
	if len(got.Tags) != 2 || got.Tags[0] != "prod" || got.Tags[1] != "edge" {
		t.Fatalf("VPC tags round-trip: %#v", got.Tags)
	}
}

func TestSecurityGroupTagsRoundTrip(t *testing.T) {
	sg := &platform.SecurityGroup{
		ID: "sg-1", Name: "web", VPCID: "vpc-1",
		Rules: []platform.SecurityGroupRule{{Protocol: "tcp", PortFrom: 80, PortTo: 80}},
		Tags:  []string{"frontend", " web "},
	}
	obj := SGToUnstructured(sg, "acme", "vpc-cr")
	got := SGFromUnstructured(obj, "tenant-1", "vpc-1")
	if len(got.Tags) != 2 || got.Tags[0] != "frontend" || got.Tags[1] != "web" {
		t.Fatalf("SecurityGroup tags round-trip: %#v", got.Tags)
	}
}

func TestUserTagsRoundTrip(t *testing.T) {
	u := &platform.User{
		ID: "usr-1", Username: "alice", Role: platform.RoleUser,
		Tags: []string{"team-a", " ops "},
	}
	obj := UserToUnstructured(u, "role-cr", "tenant-cr")
	got := UserFromUnstructured(obj, "hash")
	if len(got.Tags) != 2 || got.Tags[0] != "team-a" || got.Tags[1] != "ops" {
		t.Fatalf("User tags round-trip: %#v", got.Tags)
	}
}

func TestRoleTagsRoundTrip(t *testing.T) {
	r := &platform.RoleRecord{ID: "role-1", Name: "operator", Tags: []string{"automation", ""}}
	obj := RoleToUnstructured(r, "ns-1")
	got := RoleFromUnstructured(obj)
	if len(got.Tags) != 1 || got.Tags[0] != "automation" {
		t.Fatalf("Role tags round-trip: %#v", got.Tags)
	}
}

func TestAPIKeyTagsRoundTrip(t *testing.T) {
	k := &platform.APIKey{ID: "key-1", Name: "ci", Prefix: "vf_ci", Tags: []string{"ci", " robot "}}
	obj := APIKeyToUnstructured(k, "user-cr", "key-cr")
	got := APIKeyFromUnstructured(obj)
	if len(got.Tags) != 2 || got.Tags[0] != "ci" || got.Tags[1] != "robot" {
		t.Fatalf("APIKey tags round-trip: %#v", got.Tags)
	}
}
