// Package ids generates VPS identifiers used as the DNS label, VM
// hostname and Incus name suffix (vps-{id}).
//
// IDs are 8 chars from lowercase [a-z0-9] minus ambiguous characters,
// always starting with a letter.
package ids

import (
	"crypto/rand"
	"fmt"
)

const (
	// Length is the fixed length of a VPS id.
	Length = 8
	// IncusPrefix prefixes every Incus instance name.
	IncusPrefix = "vps-"
)

var (
	// first excludes digits so the id is a valid DNS label start and hostname.
	first = []byte("abcdefghjkmnpqrstuvwxyz")
	// rest excludes ambiguous chars: 0/o, 1/i/l.
	rest = []byte("abcdefghjkmnpqrstuvwxyz23456789")
)

// Generate returns a new random VPS id.
func Generate() (string, error) {
	b := make([]byte, Length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("ids: rand: %w", err)
	}
	b[0] = first[int(b[0])%len(first)]
	for i := 1; i < Length; i++ {
		b[i] = rest[int(b[i])%len(rest)]
	}
	return string(b), nil
}

// MustGenerate panics on failure; for tests and seed scripts only.
func MustGenerate() string {
	id, err := Generate()
	if err != nil {
		panic(err)
	}
	return id
}

// Valid reports whether s is a well-formed VPS id.
func Valid(s string) bool {
	if len(s) != Length {
		return false
	}
	for i := 0; i < Length; i++ {
		c := s[i]
		if i == 0 {
			if !in(c, first) {
				return false
			}
			continue
		}
		if !in(c, rest) {
			return false
		}
	}
	return true
}

// IncusName maps a VPS id to its Incus instance name.
func IncusName(id string) string { return IncusPrefix + id }

func in(c byte, set []byte) bool {
	for _, x := range set {
		if c == x {
			return true
		}
	}
	return false
}
