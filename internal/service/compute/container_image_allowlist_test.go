package compute

import (
	"testing"
)

func TestValidateContainerDiskImage_AllowsDefaults(t *testing.T) {
	t.Parallel()
	for _, img := range []string{
		"quay.io/kubevirt/cirros-container-disk-demo",
		"quay.io/containerdisks/ubuntu:22.04",
		"quay.io/containerdisks/fedora:40",
		"ghcr.io/virtfoundry/node-ubuntu:1.36.5@sha256:deadbeef",
	} {
		if err := ValidateContainerDiskImage(img, nil); err != nil {
			t.Fatalf("%q: %v", img, err)
		}
	}
}

func TestValidateContainerDiskImage_RejectsUnlisted(t *testing.T) {
	t.Parallel()
	for _, img := range []string{
		"evil.example.com/pwn:latest",
		"docker.io/library/nginx:latest",
		"ghcr.io/attacker/malware:1",
		"http://evil.example.com/disk.qcow2",
		"https://mirror.example.com/ubuntu.iso",
		"",
		"   ",
	} {
		if err := ValidateContainerDiskImage(img, nil); err == nil {
			t.Fatalf("expected reject for %q", img)
		}
	}
}

func TestValidateContainerDiskImage_CustomAllowlist(t *testing.T) {
	t.Parallel()
	allowed := []string{"registry.homelab/vf/", "quay.io/containerdisks/"}
	if err := ValidateContainerDiskImage("registry.homelab/vf/cirros:1", allowed); err != nil {
		t.Fatal(err)
	}
	if err := ValidateContainerDiskImage("quay.io/kubevirt/cirros-container-disk-demo", allowed); err == nil {
		t.Fatal("expected reject when kubevirt prefix not in custom list")
	}
}

func TestParseContainerImageAllowlistEnv(t *testing.T) {
	t.Parallel()
	got := parseContainerImageAllowlist(" quay.io/a/ , ,quay.io/b/ ")
	want := []string{"quay.io/a/", "quay.io/b/"}
	if len(got) != len(want) {
		t.Fatalf("got %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestEffectiveContainerImagePrefixes_EnvOverridesDefaults(t *testing.T) {
	t.Setenv(EnvAllowedContainerImagePrefixes, " registry.homelab/vf/ , quay.io/mirror/ ")
	got := EffectiveContainerImagePrefixes(nil)
	want := []string{"registry.homelab/vf/", "quay.io/mirror/"}
	if len(got) != len(want) {
		t.Fatalf("got %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestEffectiveContainerImagePrefixes_OverrideBeatsEnv(t *testing.T) {
	t.Setenv(EnvAllowedContainerImagePrefixes, "env.example/")
	got := EffectiveContainerImagePrefixes([]string{"override.example/"})
	if len(got) != 1 || got[0] != "override.example/" {
		t.Fatalf("got %#v", got)
	}
}

func TestBootstrapContainerImageAllowlist_Defaults(t *testing.T) {
	t.Setenv(EnvAllowedContainerImagePrefixes, "")
	got := BootstrapContainerImageAllowlist(nil)
	if len(got) != 3 {
		t.Fatalf("got %#v", got)
	}
}
