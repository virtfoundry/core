package cloudinit

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildLinuxUserData_RequiresSSHKeyOrPassword(t *testing.T) {
	t.Parallel()
	_, err := BuildLinuxUserData(LinuxConfig{})
	if !errors.Is(err, ErrNoGuestAuth) {
		t.Fatalf("got err %v, want ErrNoGuestAuth", err)
	}
	_, err = BuildLinuxUserData(LinuxConfig{SSHPublicKeys: []string{"", "  "}})
	if !errors.Is(err, ErrNoGuestAuth) {
		t.Fatalf("blank keys: got err %v, want ErrNoGuestAuth", err)
	}
}

func TestBuildLinuxUserData_SSHKeysDisablesPasswordAuth(t *testing.T) {
	t.Parallel()
	out, err := BuildLinuxUserData(LinuxConfig{
		SSHPublicKeys: []string{"ssh-ed25519 AAAA test@host"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "password:") {
		t.Fatalf("must not set a password when only SSH keys are provided:\n%s", out)
	}
	if !strings.Contains(out, "ssh_pwauth: false") {
		t.Fatalf("expected ssh_pwauth: false:\n%s", out)
	}
	if !strings.Contains(out, "lock_passwd: true") {
		t.Fatalf("expected lock_passwd: true:\n%s", out)
	}
	if !strings.Contains(out, "ssh-ed25519 AAAA test@host") {
		t.Fatalf("expected authorized key:\n%s", out)
	}
	if strings.Contains(out, "password: ubuntu") {
		t.Fatalf("must never default password ubuntu:\n%s", out)
	}
}

func TestBuildLinuxUserData_ExplicitPasswordEnablesPasswordAuth(t *testing.T) {
	t.Parallel()
	out, err := BuildLinuxUserData(LinuxConfig{Password: "one-time-lab"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "password: one-time-lab") {
		t.Fatalf("expected explicit password:\n%s", out)
	}
	if !strings.Contains(out, "ssh_pwauth: true") {
		t.Fatalf("expected ssh_pwauth: true:\n%s", out)
	}
	if !strings.Contains(out, "chpasswd: { expire: True }") {
		t.Fatalf("expected password expire:\n%s", out)
	}
	if !strings.Contains(out, "lock_passwd: false") {
		t.Fatalf("expected lock_passwd: false:\n%s", out)
	}
}

func TestBuildLinuxUserData_KeysPlusPassword(t *testing.T) {
	t.Parallel()
	out, err := BuildLinuxUserData(LinuxConfig{
		SSHPublicKeys: []string{"ssh-ed25519 AAAA test@host"},
		Password:      "explicit",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ssh_pwauth: true") {
		t.Fatalf("explicit password must enable password SSH:\n%s", out)
	}
	if !strings.Contains(out, "ssh-ed25519 AAAA test@host") {
		t.Fatalf("expected authorized key:\n%s", out)
	}
}

func TestBuildLinuxUserData_NeverDefaultsUbuntu(t *testing.T) {
	t.Parallel()
	// Empty password with keys must not invent "ubuntu".
	out, err := BuildLinuxUserData(LinuxConfig{
		SSHPublicKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "password: ubuntu") {
		t.Fatalf("defaulted password ubuntu:\n%s", out)
	}
}
