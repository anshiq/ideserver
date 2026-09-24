package api

import (
	"strings"
	"testing"

	"github.com/anshiq/ideserver/internal/instances"
)

func TestValidateFirewall(t *testing.T) {
	rules, err := ValidateFirewall([]FirewallInput{
		{Proto: "tcp", PortFrom: 80, PortTo: 80, SourceCIDR: "0.0.0.0/0"},
		{Proto: "udp", PortFrom: 3000, PortTo: 4000, SourceCIDR: "10.0.0.0/8"},
	})
	if err != nil || len(rules) != 2 {
		t.Fatalf("got %+v %v", rules, err)
	}
	for _, bad := range [][]FirewallInput{
		{{Proto: "icmp", PortFrom: 80, PortTo: 80, SourceCIDR: "0.0.0.0/0"}},
		{{Proto: "tcp", PortFrom: 0, PortTo: 80, SourceCIDR: "0.0.0.0/0"}},
		{{Proto: "tcp", PortFrom: 80, PortTo: 79, SourceCIDR: "0.0.0.0/0"}},
		{{Proto: "tcp", PortFrom: 80, PortTo: 80, SourceCIDR: "not-a-cidr"}},
	} {
		if _, err := ValidateFirewall(bad); err == nil {
			t.Fatalf("expected error for %+v", bad)
		}
	}
}

func TestCheckOwnerIDOR(t *testing.T) {
	// user B must not access user A's instance (same message as not-found
	// so ids cannot be probed).
	if err := CheckOwner(7, 8, false); err == nil {
		t.Fatal("cross-user must be denied")
	}
	if err := CheckOwner(7, 7, false); err != nil {
		t.Fatalf("owner must pass: %v", err)
	}
	if err := CheckOwner(7, 8, true); err != nil {
		t.Fatalf("admin must pass: %v", err)
	}
}

func TestCheckAction(t *testing.T) {
	next, err := CheckAction(instances.StateRunning, instances.ActionStop)
	if err != nil || next != instances.StateStopping {
		t.Fatalf("got %q %v", next, err)
	}
	if _, err := CheckAction(instances.StateRunning, instances.ActionStart); err == nil {
		t.Fatal("start from running must fail")
	}
}

func TestValidateVPSID(t *testing.T) {
	if err := ValidateVPSID("a7k2m9xq"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateVPSID("../etc"); err == nil {
		t.Fatal("path traversal must fail")
	}
}

func TestBuildSSHConfig(t *testing.T) {
	s := BuildSSHConfig("a7k2m9xq", "vps.example.com", "vps.example.com", "ubuntu", 30142, "ed25519 SHA256:abc")
	for _, want := range []string{"ProxyJump", "a7k2m9xq.vps.example.com", "30142", "SHA256:abc"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
}

func TestValidatePortMapping(t *testing.T) {
	if err := ValidatePortMapping(5432, "tcp"); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePortMapping(22, "tcp"); err == nil {
		t.Fatal("reserved port 22 must fail")
	}
	if err := ValidatePortMapping(0, "tcp"); err == nil {
		t.Fatal("port 0 must fail")
	}
}

func TestSignupDecision(t *testing.T) {
	if st, _ := SignupDecision("open", "anyone", ""); st != "active" {
		t.Fatal("open must activate")
	}
	if st, _ := SignupDecision("approval", "anyone", ""); st != "pending" {
		t.Fatal("approval must pend")
	}
	if st, _ := SignupDecision("allowlist", "friend", "friend,other"); st != "active" {
		t.Fatal("allowlisted must activate")
	}
	if st, _ := SignupDecision("allowlist", "stranger", "friend"); st != "pending" {
		t.Fatal("non-allowlisted must pend")
	}
}
