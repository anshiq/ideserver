package cloudinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderUbuntuGolden(t *testing.T) {
	out, err := Render(Input{
		Distro:      DistroUbuntu,
		Hostname:    "a7k2m9xq",
		UserSSHPubs: []string{"ssh-ed25519 AAAA user@laptop"},
		HostKeys: map[string]string{
			"ed25519": "-----BEGIN OPENSSH PRIVATE KEY-----\nXYZ\n-----END OPENSSH PRIVATE KEY-----",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "vendor-ubuntu.yaml.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden missing (run with UPDATE_GOLDEN=1): %v", err)
	}
	if out != string(want) {
		t.Fatalf("mismatch:\n%s", out)
	}
}

func TestRenderFedoraUser(t *testing.T) {
	out, err := Render(Input{Distro: DistroFedora, Hostname: "b8x1p2zz"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "name: fedora") {
		t.Fatalf("expected fedora user:\n%s", out)
	}
	if !strings.Contains(out, "ssh_pwauth: false") || !strings.Contains(out, "disable_root: true") {
		t.Fatalf("missing hardening:\n%s", out)
	}
	if strings.Contains(out, "password") {
		t.Fatalf("must not contain passwords:\n%s", out)
	}
}

func TestRenderErrors(t *testing.T) {
	if _, err := Render(Input{Distro: "arch", Hostname: "x"}); err == nil {
		t.Fatal("expected distro error")
	}
	if _, err := Render(Input{Distro: DistroUbuntu}); err == nil {
		t.Fatal("expected hostname error")
	}
}
