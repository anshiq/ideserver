package netalloc

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"sort"
)

// AllocateIP returns the lowest free bridge address given the used set.
// used holds string IPs as stored (e.g. "10.100.0.2"). The scan starts at
// .2 and walks upward; exhaustion returns an error so callers surface
// capacity failure instead of colliding (UNIQUE constraint is the backstop).
func AllocateIP(used []string) (netip.Addr, error) {
	taken := make(map[netip.Addr]bool, len(used))
	for _, s := range used {
		if a, err := netip.ParseAddr(s); err == nil {
			taken[a] = true
		}
	}
	// Bridge is a /16: at most 65533 candidates; walk from .2.
	for n := 0; n < 65533; n++ {
		a, err := PrivateIP(n)
		if err != nil {
			return netip.Addr{}, err
		}
		if !taken[a] {
			return a, nil
		}
	}
	return netip.Addr{}, fmt.Errorf("netalloc: no free private IP")
}

// AllocatePort returns a free public port from 30000-39999 given the used
// set. It tries a random start offset (spread across restarts) then scans
// linearly, so concurrent creators rarely collide; the DB UNIQUE constraint
// is the final arbiter and callers must retry on conflict.
func AllocatePort(used []int32) (int, error) {
	taken := make(map[int]bool, len(used))
	for _, p := range used {
		taken[int(p)] = true
	}
	size := PortRangeSize()
	start := 0
	if size > 0 {
		start = rand.IntN(size)
	}
	for i := 0; i < size; i++ {
		p := PortMin + (start+i)%size
		if !taken[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("netalloc: no free public port")
}

// SortedPorts is a test helper: sorted copy of a port list.
func SortedPorts(in []int32) []int32 {
	out := append([]int32(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
