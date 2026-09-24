// Package edgefw holds the edge proxy's pure routing primitives: TLS
// ClientHello SNI peeking (no termination) and inbound firewall-rule
// evaluation shared by the hostname-routed path.
package edgefw

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// PeekSNI parses the ServerName from a TLS ClientHello record without
// terminating TLS. It handles a single record in one read; the edge
// buffers until the full record arrives before calling.
func PeekSNI(record []byte) (string, error) {
	// TLS record header: type(1)=22 handshake, ver(2), len(2).
	if len(record) < 5 {
		return "", fmt.Errorf("edgefw: record too short")
	}
	if record[0] != 22 {
		return "", fmt.Errorf("edgefw: not a handshake record")
	}
	reclen := int(binary.BigEndian.Uint16(record[3:5]))
	if len(record)-5 < reclen {
		return "", fmt.Errorf("edgefw: truncated record")
	}
	hs := record[5 : 5+reclen]
	// Handshake header: type(1)=1 client_hello, len(3).
	if len(hs) < 4 || hs[0] != 1 {
		return "", fmt.Errorf("edgefw: not a client_hello")
	}
	hslen := int(hs[1])<<16 | int(hs[2])<<8 | int(hs[3])
	if len(hs)-4 < hslen {
		return "", fmt.Errorf("edgefw: truncated hello")
	}
	p := hs[4 : 4+hslen]
	// version(2) + random(32)
	if len(p) < 34 {
		return "", fmt.Errorf("edgefw: hello too short")
	}
	p = p[34:]
	// session id
	if len(p) < 1 {
		return "", fmt.Errorf("edgefw: no session id")
	}
	if sl := int(p[0]); len(p)-1 < sl {
		return "", fmt.Errorf("edgefw: truncated session id")
	} else {
		p = p[1+sl:]
	}
	// cipher suites
	if len(p) < 2 {
		return "", fmt.Errorf("edgefw: no ciphers")
	}
	if cl := int(binary.BigEndian.Uint16(p)); len(p)-2 < cl {
		return "", fmt.Errorf("edgefw: truncated ciphers")
	} else {
		p = p[2+cl:]
	}
	// compression methods
	if len(p) < 1 {
		return "", fmt.Errorf("edgefw: no compression")
	}
	if ml := int(p[0]); len(p)-1 < ml {
		return "", fmt.Errorf("edgefw: truncated compression")
	} else {
		p = p[1+ml:]
	}
	// extensions
	if len(p) < 2 {
		return "", fmt.Errorf("edgefw: no extensions")
	}
	extlen := int(binary.BigEndian.Uint16(p))
	p = p[2:]
	if len(p) < extlen {
		return "", fmt.Errorf("edgefw: truncated extensions")
	}
	p = p[:extlen]
	for len(p) >= 4 {
		typ := binary.BigEndian.Uint16(p)
		ln := int(binary.BigEndian.Uint16(p[2:4]))
		p = p[4:]
		if len(p) < ln {
			return "", fmt.Errorf("edgefw: truncated extension")
		}
		body := p[:ln]
		p = p[ln:]
		if typ != 0 { // server_name
			continue
		}
		if len(body) < 2 {
			return "", fmt.Errorf("edgefw: bad server_name ext")
		}
		listlen := int(binary.BigEndian.Uint16(body))
		q := body[2:]
		if len(q) < listlen {
			return "", fmt.Errorf("edgefw: bad server_name list")
		}
		q = q[:listlen]
		for len(q) >= 3 {
			nametype := q[0]
			namelen := int(binary.BigEndian.Uint16(q[1:3]))
			q = q[3:]
			if len(q) < namelen {
				return "", fmt.Errorf("edgefw: bad server_name entry")
			}
			if nametype == 0 {
				return string(q[:namelen]), nil
			}
			q = q[namelen:]
		}
		return "", fmt.Errorf("edgefw: no host_name entry")
	}
	return "", fmt.Errorf("edgefw: no SNI extension")
}

// SplitHost extracts the VPS id from "{id}.vps.{domain}".
// ok=false when host is not a VPS hostname (incl. the bare vps.{domain}
// gateway name, which must never match a VPS route).
func SplitHost(host, vpSuffix string) (id string, ok bool) {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	suffix := "." + strings.ToLower(vpSuffix)
	if !strings.HasSuffix(host, suffix) {
		return "", false
	}
	label := strings.TrimSuffix(host, suffix)
	if label == "" || strings.Contains(label, ".") {
		return "", false
	}
	return label, true
}

// Proto is a firewall protocol selector.
type Proto string

const (
	ProtoTCP Proto = "tcp"
	ProtoUDP Proto = "udp"
)

// Rule is one inbound firewall rule: allow proto ports [from,to] from CIDR.
type Rule struct {
	Proto Proto
	From  int // inclusive
	To    int // inclusive
	CIDR  netip.Prefix
}

// Allow reports whether proto/port from client IP passes the rule set.
// Default-deny: empty rules deny everything.
func Allow(rules []Rule, proto Proto, port int, client netip.Addr) bool {
	for _, r := range rules {
		if r.Proto != proto {
			continue
		}
		if port < r.From || port > r.To {
			continue
		}
		if r.CIDR.Contains(client) {
			return true
		}
	}
	return false
}

// MustCIDR parses a CIDR or panics (seeds/defaults only).
func MustCIDR(s string) netip.Prefix {
	p, err := netip.ParsePrefix(s)
	if err != nil {
		panic(err)
	}
	return p
}

// DefaultRules allow 22/80/443 TCP from anywhere.
func DefaultRules() []Rule {
	any := MustCIDR("0.0.0.0/0")
	any6 := MustCIDR("::/0")
	var out []Rule
	for _, port := range []int{22, 80, 443} {
		out = append(out, Rule{ProtoTCP, port, port, any}, Rule{ProtoTCP, port, port, any6})
	}
	return out
}

var _ = net.IPv4len // keep net import if unused in future edits
