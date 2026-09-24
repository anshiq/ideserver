// Package sshgw implements the public SSH jump gateway (plan §1.1/§1.2).
//
// SSH carries no hostname, so "{id}.vps.{domain}:22" cannot be routed by
// name at TCP level. Instead users ProxyJump through vps.{domain}:22; the
// gateway authenticates the user's key against the DB, checks ownership and
// the port-22 firewall rule, then pipes the direct-tcpip channel to
// VM_IP:22. The session stays encrypted end to end with the VM's own sshd.
package sshgw

import (
	"fmt"
	"net/netip"
	"strings"

	edgefw "github.com/anshiq/ideserver/internal/edge"
	"golang.org/x/crypto/ssh"
)

// Target is a parsed direct-tcpip request.
type Target struct {
	VPSID string
	Port  int
}

// ParseTarget validates a direct-tcpip host/port pair. Only
// "{id}.vps.{domain}:22" is accepted; everything else is rejected so the
// gateway can never be used as an open relay.
func ParseTarget(host string, port int, vpsSuffix string) (Target, error) {
	if port != 22 {
		return Target{}, fmt.Errorf("sshgw: only port 22 is forwarded (got %d)", port)
	}
	id, ok := edgefw.SplitHost(strings.ToLower(strings.TrimSuffix(strings.ToLower(host), ".")), strings.ToLower(vpsSuffix))
	if !ok {
		return Target{}, fmt.Errorf("sshgw: invalid target %q", host)
	}
	return Target{VPSID: id, Port: port}, nil
}

// CheckAccess enforces ownership + key binding + firewall for a jump.
// requestingUser is the DB user authenticated by public key; ownerID is the
// VPS owner's user id; keyBound is whether the presented key fingerprint is
// registered to requestingUser; rules are the VPS firewall; clientIP is the
// TCP peer address (port-22 rule evaluated here because DNAT is not
// involved on this path).
func CheckAccess(requestingUser, ownerID int64, keyBound bool, rules []edgefw.Rule, clientIP netip.Addr) error {
	if !keyBound {
		return fmt.Errorf("sshgw: unknown key")
	}
	if requestingUser != ownerID {
		return fmt.Errorf("sshgw: access denied")
	}
	if !edgefw.Allow(rules, edgefw.ProtoTCP, 22, clientIP) {
		return fmt.Errorf("sshgw: firewall denies port 22")
	}
	return nil
}

// Fingerprint returns the OpenSSH SHA256 fingerprint for a public key,
// in the same "SHA256:..." form as `ssh-keygen -lf`.
func Fingerprint(pub ssh.PublicKey) string {
	return ssh.FingerprintSHA256(pub)
}

// ParseAuthorizedKey parses one "ssh-ed25519 AAAA... comment" line.
func ParseAuthorizedKey(line string) (ssh.PublicKey, string, error) {
	pub, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return nil, "", fmt.Errorf("sshgw: bad public key: %w", err)
	}
	return pub, comment, nil
}

// Banner returns the help text for a plain `ssh vps.{domain}` login without
// a ProxyJump target. It prints the exact ssh_config snippet and the
// fallback-port hint (plan §5: users confused by ProxyJump).
func Banner(vpsSuffix, appDomain, gatewayHost string) string {
	return fmt.Sprintf(`Welcome to the VPS jump gateway.

Plain logins do nothing — jump straight to your VPS:

  ssh a7k2m9xq.%s          (with the ssh_config below)
  ssh -J %s a7k2m9xq       (one-off, no config)

One-time ssh_config (or run: vpsctl ssh-config):

  Host *.%s
      ProxyJump %s
      User ubuntu            # or fedora

Zero-config fallback (per-VPS public port, see dashboard):

  ssh -p 3xxxx ubuntu@a7k2m9xq.%s

Dashboard: https://%s
`,
		vpsSuffix, gatewayHost, vpsSuffix, gatewayHost, vpsSuffix, appDomain)
}
