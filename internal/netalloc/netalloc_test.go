package netalloc

import "testing"

func TestPrivateIP(t *testing.T) {
	a, err := PrivateIP(0)
	if err != nil {
		t.Fatal(err)
	}
	if a.String() != "10.100.0.2" {
		t.Fatalf("got %s", a)
	}
	b, _ := PrivateIP(254)
	if b.String() != "10.100.1.0" {
		t.Fatalf("got %s", b)
	}
	if !InBridge(a) {
		t.Fatal("expected in bridge")
	}
	if _, err := PrivateIP(1 << 20); err == nil {
		t.Fatal("expected exhaustion error")
	}
}

func TestValidPublicPort(t *testing.T) {
	for p, want := range map[int]bool{29999: false, 30000: true, 35000: true, 39999: true, 40000: false, 22: false} {
		if got := ValidPublicPort(p); got != want {
			t.Errorf("ValidPublicPort(%d) = %v, want %v", p, got, want)
		}
	}
}
