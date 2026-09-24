package sshgw

import (
	"net/netip"
	"strings"
	"testing"

	edgefw "github.com/anshiq/ideserver/internal/edge"
)

func TestParseTarget(t *testing.T) {
	tg, err := ParseTarget("a7k2m9xq.vps.example.com", 22, "vps.example.com")
	if err != nil || tg.VPSID != "a7k2m9xq" {
		t.Fatalf("got %+v %v", tg, err)
	}
	if _, err := ParseTarget("a7k2m9xq.vps.example.com", 80, "vps.example.com"); err == nil {
		t.Fatal("non-22 must be rejected")
	}
	if _, err := ParseTarget("vps.example.com", 22, "vps.example.com"); err == nil {
		t.Fatal("gateway name must be rejected")
	}
	if _, err := ParseTarget("a7k2m9xq.evil.com", 22, "vps.example.com"); err == nil {
		t.Fatal("foreign suffix must be rejected")
	}
}

func TestCheckAccess(t *testing.T) {
	rules := edgefw.DefaultRules()
	ip := netip.MustParseAddr("1.2.3.4")
	// owner with bound key passes
	if err := CheckAccess(7, 7, true, rules, ip); err != nil {
		t.Fatalf("owner should pass: %v", err)
	}
	// user B's key on user A's VPS is denied (IDOR at SSH layer)
	if err := CheckAccess(8, 7, true, rules, ip); err == nil {
		t.Fatal("cross-user access must be denied")
	}
	// unregistered key denied even for the owner id
	if err := CheckAccess(7, 7, false, rules, ip); err == nil {
		t.Fatal("unbound key must be denied")
	}
	// firewall deny
	deny := []edgefw.Rule{{Proto: edgefw.ProtoTCP, From: 22, To: 22, CIDR: edgefw.MustCIDR("10.0.0.0/8")}}
	if err := CheckAccess(7, 7, true, deny, ip); err == nil {
		t.Fatal("firewall deny must block")
	}
}

func TestBannerMentionsConfig(t *testing.T) {
	b := Banner("vps.example.com", "app.example.com", "vps.example.com")
	for _, want := range []string{"ProxyJump", "ssh_config", "3xxxx", "app.example.com"} {
		if !strings.Contains(b, want) {
			t.Fatalf("banner missing %q", want)
		}
	}
}
