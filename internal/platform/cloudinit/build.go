package cloudinit

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNoGuestAuth is returned when Linux user-data would have neither an SSH
// public key nor an explicit one-time password. Callers must not invent a
// default password (historically "ubuntu").
var ErrNoGuestAuth = errors.New("linux guest requires an SSH public key or an explicit one-time password")

// LinuxConfig drives cloud-init user-data for KubeVirt Linux VMs.
type LinuxConfig struct {
	SSHPublicKeys  []string
	Password       string // optional; when set, enables password SSH (ssh_pwauth)
	ExtraUserData  string // optional #cloud-config fragment appended
	FormatDataDisk bool   // mkfs + mount /dev/vdb at /mnt/iops
}

// NetworkInterfaceConfig is one static NIC for cloud-init network-data (Multus public IP).
type NetworkInterfaceConfig struct {
	MACAddress string
	Address    string
	PrefixLen  int
	Gateway    string
	DNS        []string
}

// BuildNetworkData returns cloud-init network v2 YAML for bridge/Multus NICs matched by MAC.
// When Address is empty, uses DHCP (requires a DHCP server on the L2 bridge, e.g. dnsmasq on vf-pub0).
func BuildNetworkData(ifaces []NetworkInterfaceConfig) string {
	if len(ifaces) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("version: 2\n")
	b.WriteString("ethernets:\n")
	for i, nic := range ifaces {
		if nic.MACAddress == "" {
			continue
		}
		key := fmt.Sprintf("nic%d", i)
		fmt.Fprintf(&b, "  %s:\n", key)
		fmt.Fprintf(&b, "    match:\n      macaddress: %q\n", strings.ToLower(nic.MACAddress))
		if nic.Address == "" {
			b.WriteString("    dhcp4: true\n")
			continue
		}
		fmt.Fprintf(&b, "    dhcp4: false\n")
		fmt.Fprintf(&b, "    addresses:\n      - %s/%d\n", nic.Address, nic.prefixLenOrDefault())
		if nic.Gateway != "" {
			fmt.Fprintf(&b, "    gateway4: %s\n", nic.Gateway)
		}
		if len(nic.DNS) > 0 {
			b.WriteString("    nameservers:\n      addresses:\n")
			for _, d := range nic.DNS {
				if d = strings.TrimSpace(d); d != "" {
					fmt.Fprintf(&b, "        - %s\n", d)
				}
			}
		}
	}
	return b.String()
}

func (nic NetworkInterfaceConfig) prefixLenOrDefault() int {
	if nic.PrefixLen > 0 {
		return nic.PrefixLen
	}
	return 24
}

func normalizeSSHKeys(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

// BuildLinuxUserData returns a #cloud-config payload for Ubuntu/Cirros-style images.
//
// Auth policy (fail-closed):
//   - At least one SSH public key OR an explicit Password is required.
//   - Password SSH (ssh_pwauth) is enabled only when Password is non-empty.
//   - Never invents a default password (e.g. "ubuntu").
func BuildLinuxUserData(cfg LinuxConfig) (string, error) {
	keys := normalizeSSHKeys(cfg.SSHPublicKeys)
	pass := strings.TrimSpace(cfg.Password)
	if len(keys) == 0 && pass == "" {
		return "", ErrNoGuestAuth
	}

	var b strings.Builder
	b.WriteString("#cloud-config\n")

	enablePasswordSSH := pass != ""
	if enablePasswordSSH {
		fmt.Fprintf(&b, "password: %s\n", pass)
		// Force password change at first login. PAM enforces the prompt on
		// both TTY and SSH password auth; SSH-key-only logins are unaffected
		// when lock_passwd is set below.
		b.WriteString("chpasswd: { expire: True }\n")
		b.WriteString("ssh_pwauth: true\n")
	} else {
		b.WriteString("ssh_pwauth: false\n")
	}

	b.WriteString("users:\n")
	b.WriteString("  - name: ubuntu\n")
	b.WriteString("    sudo: ALL=(ALL) NOPASSWD:ALL\n")
	b.WriteString("    shell: /bin/bash\n")
	if enablePasswordSSH {
		b.WriteString("    lock_passwd: false\n")
	} else {
		b.WriteString("    lock_passwd: true\n")
	}
	if len(keys) > 0 {
		b.WriteString("    ssh_authorized_keys:\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "      - %s\n", k)
		}
	}

	b.WriteString("runcmd:\n")
	b.WriteString("  - [ systemctl, enable, --now, getty@tty1 ]\n")
	if cfg.FormatDataDisk {
		b.WriteString("  - [ bash, -lc, 'DATA=$(lsblk -dn -o NAME,SIZE,TYPE | awk \\'$3==\"disk\" && $2 ~ /G/ {print \"/dev/\"$1}\\' | tail -1); if [ -n \"$DATA\" ] && ! blkid \"$DATA\"; then mkfs.ext4 -F \"$DATA\"; fi' ]\n")
		b.WriteString("  - [ bash, -lc, 'DATA=$(lsblk -dn -o NAME,SIZE,TYPE | awk \\'$3==\"disk\" && $2 ~ /G/ {print \"/dev/\"$1}\\' | tail -1); mkdir -p /mnt/iops; grep -q /mnt/iops /etc/fstab || echo \"$DATA /mnt/iops ext4 defaults 0 2\" >> /etc/fstab' ]\n")
		b.WriteString("  - [ mount, -a ]\n")
	}
	if extra := strings.TrimSpace(cfg.ExtraUserData); extra != "" {
		b.WriteString("\n")
		b.WriteString(extra)
		if !strings.HasSuffix(extra, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}
