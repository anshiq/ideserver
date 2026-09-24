// Package cloudinit renders Incus vendor-data for new instances.
//
// Our vendor-data creates the distro default user, injects the user's SSH
// keys plus per-instance generated host keys, sets the hostname, hardens
// sshd and lets growpart resize the disk. Caller-supplied user-data rides
// alongside.
package cloudinit

import (
	"fmt"
	"strings"
)

// Distro selects the default user and package behaviour.
type Distro string

const (
	DistroUbuntu Distro = "ubuntu"
	DistroFedora Distro = "fedora"
)

// DefaultUser returns the cloud-image login user for a distro.
func DefaultUser(d Distro) (string, error) {
	switch d {
	case DistroUbuntu:
		return "ubuntu", nil
	case DistroFedora:
		return "fedora", nil
	default:
		return "", fmt.Errorf("cloudinit: unknown distro %q", d)
	}
}

// Input to the vendor-data renderer.
type Input struct {
	Distro      Distro
	Hostname    string // VPS id, e.g. a7k2m9xq
	UserSSHPubs []string
	// HostKeys maps key type ("rsa","ecdsa","ed25519") to PEM private key.
	HostKeys map[string]string
}

// Render returns vendor-data YAML. It never includes a password.
func Render(in Input) (string, error) {
	user, err := DefaultUser(in.Distro)
	if err != nil {
		return "", err
	}
	if in.Hostname == "" {
		return "", fmt.Errorf("cloudinit: empty hostname")
	}
	var b strings.Builder
	b.WriteString("#cloud-config\n")
	fmt.Fprintf(&b, "hostname: %s\n", in.Hostname)
	fmt.Fprintf(&b, "manage_etc_hosts: true\n")
	b.WriteString("users:\n")
	fmt.Fprintf(&b, "  - name: %s\n", user)
	b.WriteString("    sudo: ALL=(ALL) NOPASSWD:ALL\n")
	b.WriteString("    shell: /bin/bash\n")
	b.WriteString("    lock_passwd: true\n")
	if len(in.UserSSHPubs) > 0 {
		b.WriteString("    ssh_authorized_keys:\n")
		for _, k := range in.UserSSHPubs {
			fmt.Fprintf(&b, "      - %s\n", strings.TrimSpace(k))
		}
	}
	if len(in.HostKeys) > 0 {
		b.WriteString("ssh_keys:\n")
		for _, typ := range []string{"rsa", "ecdsa", "ed25519"} {
			if pem, ok := in.HostKeys[typ]; ok {
				fmt.Fprintf(&b, "  %s_private: |\n", typ)
				for _, line := range strings.Split(strings.TrimSpace(pem), "\n") {
					fmt.Fprintf(&b, "    %s\n", line)
				}
			}
		}
	}
	b.WriteString("ssh_pwauth: false\n")
	b.WriteString("disable_root: true\n")
	b.WriteString("growpart:\n  mode: auto\n")
	return b.String(), nil
}
