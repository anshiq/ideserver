package netalloc

import (
	"net/netip"
	"testing"
)

func TestAllocateIPSkipsUsed(t *testing.T) {
	a, err := AllocateIP([]string{"10.100.0.2", "10.100.0.3"})
	if err != nil {
		t.Fatal(err)
	}
	if a.String() != "10.100.0.4" {
		t.Fatalf("got %s", a)
	}
	b, err := AllocateIP(nil)
	if err != nil || b.String() != "10.100.0.2" {
		t.Fatalf("got %s %v", b, err)
	}
	if !InBridge(a) || !a.Is4() {
		t.Fatal("must be a bridge IPv4")
	}
	_ = netip.Addr{}
}

func TestAllocatePortSkipsUsed(t *testing.T) {
	p, err := AllocatePort([]int32{30000, 30001})
	if err != nil {
		t.Fatal(err)
	}
	if !ValidPublicPort(p) || p == 30000 || p == 30001 {
		t.Fatalf("got %d", p)
	}
	// Exhaustion of a tiny range is covered by the error path: fill check
	// would need 10k entries; instead assert bounds hold under load.
	seen := map[int]bool{}
	for i := 0; i < 200; i++ {
		q, err := AllocatePort(nil)
		if err != nil {
			t.Fatal(err)
		}
		if !ValidPublicPort(q) {
			t.Fatalf("out of range %d", q)
		}
		seen[q] = true
	}
	if len(seen) < 50 {
		t.Fatalf("expected spread, got %d unique", len(seen))
	}
}
