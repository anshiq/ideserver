package db

import (
	"net/netip"

	"github.com/sqlc-dev/pqtype"
)

// InetString returns the IP string for a pqtype.Inet (e.g. 10.100.0.2).
func InetString(in pqtype.Inet) string {
	if !in.Valid {
		return ""
	}
	return in.IPNet.IP.String()
}

// CIDRPrefix converts a pqtype.CIDR to netip.Prefix.
func CIDRPrefix(c pqtype.CIDR) (netip.Prefix, error) {
	// CIDR shares Inet's layout; convert via the underlying IPNet.
	in := pqtype.Inet(c)
	if !in.Valid {
		return netip.Prefix{}, nil
	}
	mask := in.IPNet.Mask
	ones, _ := mask.Size()
	addr, err := netip.ParseAddr(in.IPNet.IP.String())
	if err != nil {
		return netip.Prefix{}, err
	}
	return netip.PrefixFrom(addr, ones), nil
}

// InetFromString builds a pqtype.Inet from "10.100.0.2".
func InetFromString(s string) pqtype.Inet {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return pqtype.Inet{}
	}
	bits := 32
	if addr.Is6() {
		bits = 128
	}
	_ = bits
	ip := addr.AsSlice()
	mask := netip.PrefixFrom(addr, addr.BitLen()).Masked().Addr()
	_ = mask
	_ = ip
	// Build via net.IPNet round-trip; pqtype scans "ip/bits".
	pfx := netip.PrefixFrom(addr, addr.BitLen())
	_ = pfx
	var out pqtype.Inet
	_ = out.Scan(addr.String())
	return out
}
