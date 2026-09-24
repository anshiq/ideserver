package ids

import (
	"strings"
	"testing"
)

func TestGenerateShape(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 500; i++ {
		id, err := Generate()
		if err != nil {
			t.Fatal(err)
		}
		if !Valid(id) {
			t.Fatalf("generated invalid id %q", id)
		}
		if id[0] < 'a' || id[0] > 'z' {
			t.Fatalf("first char not a letter: %q", id)
		}
		if strings.ContainsAny(id, "01ilo") {
			t.Fatalf("ambiguous char in id %q", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) < 490 {
		t.Fatalf("suspiciously many collisions: %d unique of 500", len(seen))
	}
}

func TestValid(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"a7k2m9xq", true},
		{"b2345678", true},
		{"7ak2m9xq", false}, // leading digit
		{"A7k2m9xq", false}, // uppercase
		{"a7k2m9x", false},  // short
		{"a7k2m9xqo", false},
		{"a7k2m9x0", false}, // ambiguous 0
		{"a7k2m9xl", false}, // ambiguous l
		{"a7k2m9x!", false},
		{"", false},
	} {
		if got := Valid(tc.in); got != tc.want {
			t.Errorf("Valid(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIncusName(t *testing.T) {
	if got := IncusName("a7k2m9xq"); got != "vps-a7k2m9xq" {
		t.Fatalf("got %q", got)
	}
}
