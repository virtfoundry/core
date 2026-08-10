package cloudinit

import (
	"fmt"
	"strings"
)

// LinuxConfig drives cloud-init user-data for KubeVirt Linux VMs.
type LinuxConfig struct {
	SSHPublicKeys  []string
	Password       string // optional; ignored when SSH keys are set
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

// BuildLinuxUserData returns a #cloud-config payload for Ubuntu/Cirros-style images.
func BuildLinuxUserData(cfg LinuxConfig) string {
	var b strings.Builder
	b.WriteString("#cloud-config\n")
	if len(cfg.SSHPublicKeys) > 0 {
		b.WriteString("ssh_pwauth: false\n")
		b.WriteString("users:\n")
		b.WriteString("  - name: ubuntu\n")
		b.WriteString("    sudo: ALL=(ALL) NOPASSWD:ALL\n")
		b.WriteString("    shell: /bin/bash\n")
		b.WriteString("    lock_passwd: true\n")
		b.WriteString("    ssh_authorized_keys:\n")
		for _, k := range cfg.SSHPublicKeys {
			k = strings.TrimSpace(k)
			if k != "" {
				fmt.Fprintf(&b, "      - %s\n", k)
			}
		}
	} else {
		pass := cfg.Password
		if pass == "" {
			pass = "ubuntu"
		}
		fmt.Fprintf(&b, "password: %s\n", pass)
		b.WriteString("chpasswd: { expire: False }\n")
		b.WriteString("ssh_pwauth: true\n")
		b.WriteString("users:\n")
		b.WriteString("  - name: ubuntu\n")
		b.WriteString("    sudo: ALL=(ALL) NOPASSWD:ALL\n")
		b.WriteString("    shell: /bin/bash\n")
		b.WriteString("    lock_passwd: false\n")
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
	return b.String()
}
