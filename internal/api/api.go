// Package api implements the REST surface under /api/v1 (plan §1.8).
// Huma generates OpenAPI 3.1; the web dashboard consumes a typed client
// built with openapi-typescript + openapi-fetch.
package api

import (
	"fmt"
	"net/netip"
	"strings"

	edgefw "github.com/anshiq/ideserver/internal/edge"
	"github.com/anshiq/ideserver/internal/ids"
	"github.com/anshiq/ideserver/internal/instances"
)

// orEmpty turns a nil slice into an empty one so list endpoints serialize
// as `[]` rather than `null` — sqlc returns nil on zero rows, and a bare
// `null` breaks any client that calls .map()/.forEach() on the response
// without a defensive null-check (as the dashboard's list pages do).
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// FirewallInput is one inbound rule from the dashboard.
type FirewallInput struct {
	Proto      string `json:"proto"`
	PortFrom   int    `json:"portFrom"`
	PortTo     int    `json:"portTo"`
	SourceCIDR string `json:"sourceCidr"`
}

// ValidateFirewall converts dashboard input into edge rules. Empty input
// means default-deny (the caller seeds DefaultRules on create).
func ValidateFirewall(in []FirewallInput) ([]edgefw.Rule, error) {
	out := make([]edgefw.Rule, 0, len(in))
	for i, r := range in {
		proto := edgefw.Proto(strings.ToLower(r.Proto))
		if proto != edgefw.ProtoTCP && proto != edgefw.ProtoUDP {
			return nil, fmt.Errorf("api: rule %d: bad proto %q", i, r.Proto)
		}
		if r.PortFrom < 1 || r.PortFrom > 65535 || r.PortTo < r.PortFrom || r.PortTo > 65535 {
			return nil, fmt.Errorf("api: rule %d: bad port range %d-%d", i, r.PortFrom, r.PortTo)
		}
		pfx, err := netip.ParsePrefix(r.SourceCIDR)
		if err != nil {
			return nil, fmt.Errorf("api: rule %d: bad CIDR %q", i, r.SourceCIDR)
		}
		out = append(out, edgefw.Rule{Proto: proto, From: r.PortFrom, To: r.PortTo, CIDR: pfx})
	}
	return out, nil
}

// CheckOwner is the IDOR gate for every /instances/{id} route: the row's
// owner must equal the caller, unless the caller is an admin.
func CheckOwner(rowOwner, callerID int64, isAdmin bool) error {
	if isAdmin {
		return nil
	}
	if rowOwner != callerID {
		return fmt.Errorf("api: not found")
	}
	return nil
}

// CheckAction validates a lifecycle action against recorded state.
func CheckAction(state instances.State, act instances.Action) (instances.State, error) {
	return instances.Transition(state, act)
}

// ValidateInstanceName enforces a small, safe display name.
func ValidateInstanceName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return fmt.Errorf("api: name must be 1-64 chars")
	}
	return nil
}

// ValidateVPSID guards the {id} path parameter.
func ValidateVPSID(id string) error {
	if !ids.Valid(id) {
		return fmt.Errorf("api: bad instance id")
	}
	return nil
}

// BuildSSHConfig returns the dashboard snippet + known_hosts hint for a VPS.
func BuildSSHConfig(id, vpsSuffix, gatewayHost, user string, fallbackPort int, hostKeyFPs string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Host *.%s\n", vpsSuffix)
	fmt.Fprintf(&b, "    ProxyJump %s\n", gatewayHost)
	fmt.Fprintf(&b, "    User %s\n\n", user)
	fmt.Fprintf(&b, "# direct:\nssh %s.%s\n", id, vpsSuffix)
	if fallbackPort != 0 {
		fmt.Fprintf(&b, "# fallback:\nssh -p %d %s@%s.%s\n", fallbackPort, user, id, vpsSuffix)
	}
	if hostKeyFPs != "" {
		fmt.Fprintf(&b, "\n# host key fingerprints:\n# %s\n", hostKeyFPs)
	}
	return b.String()
}

// ValidatePortMapping guards POST /instances/{id}/ports.
func ValidatePortMapping(targetPort int, proto string) error {
	if targetPort < 1 || targetPort > 65535 {
		return fmt.Errorf("api: bad target port")
	}
	p := strings.ToLower(proto)
	if p != "tcp" && p != "udp" {
		return fmt.Errorf("api: bad proto %q", proto)
	}
	if targetPort == 22 || targetPort == 80 || targetPort == 443 {
		// 22/80/443 already have first-class paths (jump/hostname routing);
		// mapping them as raw ports would shadow the managed paths.
		return fmt.Errorf("api: port %d is reserved (use the built-in SSH/HTTP paths)", targetPort)
	}
	return nil
}

// IsAdminLogin reports whether login is in the ADMIN_GITHUB_LOGINS set.
func IsAdminLogin(login, adminList string) bool {
	login = strings.ToLower(strings.TrimSpace(login))
	for _, a := range strings.Split(adminList, ",") {
		if strings.ToLower(strings.TrimSpace(a)) == login && login != "" {
			return true
		}
	}
	return false
}

// SignupDecision applies SIGNUP_MODE=open|approval|allowlist.
func SignupDecision(mode, login, allowlist string) (status string, isAdmin bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "open":
		return "active", false
	case "allowlist":
		for _, a := range strings.Split(allowlist, ",") {
			if strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(login)) && login != "" {
				return "active", false
			}
		}
		return "pending", false
	default: // approval
		return "pending", false
	}
}
