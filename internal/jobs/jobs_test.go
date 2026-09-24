package jobs

import (
	"strings"
	"testing"
)

func TestACLName(t *testing.T) {
	if got := ACLName("a7k2m9xq"); got != "vps-a7k2m9xq" {
		t.Fatalf("got %q", got)
	}
}

func TestKinds(t *testing.T) {
	kinds := map[string]string{
		ProvisionArgs{}.Kind():    "provision",
		StartArgs{}.Kind():        "start",
		StopArgs{}.Kind():         "stop",
		RebootArgs{}.Kind():       "reboot",
		DeleteArgs{}.Kind():       "delete",
		ResizeArgs{}.Kind():       "resize",
		SyncFirewallArgs{}.Kind(): "sync-firewall",
		SyncPortsArgs{}.Kind():    "sync-ports",
		DropForwardArgs{}.Kind():  "drop-forward",
		MeterArgs{}.Kind():        "meter-usage",
	}
	seen := map[string]bool{}
	for got, want := range kinds {
		if got != want {
			t.Fatalf("kind mismatch: got %q want %q", got, want)
		}
		seen[got] = true
	}
	if len(seen) != 10 {
		t.Fatal("kinds must be unique")
	}
}

func TestBuildVendorData(t *testing.T) {
	vd, fps, err := BuildVendorData("a7k2m9xq", "ubuntu", []string{"ssh-ed25519 AAAA user@laptop"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"hostname: a7k2m9xq", "name: ubuntu", "ssh-ed25519 AAAA", "ssh_pwauth: false"} {
		if !strings.Contains(vd, want) {
			t.Fatalf("vendor-data missing %q:\n%s", want, vd)
		}
	}
	if !strings.Contains(fps, "ed25519 SHA256:") {
		t.Fatalf("fps missing ed25519 fingerprint: %q", fps)
	}
	if _, _, err := BuildVendorData("", "ubuntu", nil); err == nil {
		t.Fatal("empty hostname must fail")
	}
	if _, _, err := BuildVendorData("a7k2m9xq", "arch", nil); err == nil {
		t.Fatal("unknown distro must fail")
	}
}
