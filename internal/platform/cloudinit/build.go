package cloudinit

import (
	"fmt"
	"strings"
)

// LinuxConfig drives cloud-init user-data for KubeVirt Linux VMs.
type LinuxConfig struct {
	SSHPublicKeys []string
	Password      string // optional; ignored when SSH keys are set
	FormatDataDisk bool   // mkfs + mount /dev/vdb at /mnt/iops
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
	return b.String()
}
