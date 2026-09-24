// Package netalloc allocates stable private IPv4 addresses on the Incus
// bridge and public mapped ports from 30000-39999.
//
// The DB owns uniqueness (UNIQUE constraints); the helpers here compute
// candidates and validate ranges so jobs stay idempotent.
package netalloc

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
)

const (
	// BridgeCIDR is the vpsbr0 managed bridge.
	BridgeCIDR = "10.100.0.0/16"
	// gateway (.1), network (.0) and broadcast are reserved; allocation
	// starts at .2. .0/24 tail is left alone for future infra use.
	firstHostOffset = 2

	// PortMin/PortMax bound the mapped-port range (inclusive).
	PortMin = 30000
	PortMax = 39999
)

// PrivateIP returns the nth allocatable address on the bridge (n>=0 → .2+n).
func PrivateIP(n int) (netip.Addr, error) {
	_, ipnet, err := net.ParseCIDR(BridgeCIDR)
	if err != nil {
		return netip.Addr{}, err
	}
	base := binary.BigEndian.Uint32(ipnet.IP.To4())
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, base+uint32(firstHostOffset+n))
	addr := netip.AddrFrom4([4]byte(ip.To4()))
	if !addr.Is4() {
		return netip.Addr{}, fmt.Errorf("netalloc: bad address")
	}
	if !ipnet.Contains(ip) {
		return netip.Addr{}, fmt.Errorf("netalloc: bridge exhausted at offset %d", n)
	}
	return addr, nil
}

// InBridge reports whether addr belongs to the bridge network.
func InBridge(addr netip.Addr) bool {
	_, ipnet, _ := net.ParseCIDR(BridgeCIDR)
	return ipnet.Contains(net.ParseIP(addr.String()))
}

// ValidPublicPort reports whether p is inside the mapped-port range.
func ValidPublicPort(p int) bool { return p >= PortMin && p <= PortMax }

// PortRangeSize is the number of allocatable public ports.
func PortRangeSize() int { return PortMax - PortMin + 1 }
