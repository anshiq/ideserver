package auth

import (
	"net/http"
	"testing"
)

func TestTokenRoundtrip(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	h := Hash(tok)
	if !Equal(tok, h) {
		t.Fatal("token should match its hash")
	}
	other, _ := NewToken()
	if Equal(other, h) {
		t.Fatal("different token must not match")
	}
}

func TestCheckCSRF(t *testing.T) {
	origin := "https://app.example.com"
	get, _ := http.NewRequest("POST", "/", nil)
	if err := CheckCSRF(get, origin); err == nil {
		t.Fatal("missing header must fail")
	}
	get.Header.Set("X-Requested-With", "fetch")
	if err := CheckCSRF(get, origin); err != nil {
		t.Fatalf("no origin + header should pass: %v", err)
	}
	get.Header.Set("Origin", "https://evil.com")
	if err := CheckCSRF(get, origin); err == nil {
		t.Fatal("wrong origin must fail")
	}
	get.Header.Set("Origin", origin)
	if err := CheckCSRF(get, origin); err != nil {
		t.Fatalf("right origin should pass: %v", err)
	}
	safe, _ := http.NewRequest("GET", "/", nil)
	if err := CheckCSRF(safe, origin); err != nil {
		t.Fatalf("GET should pass: %v", err)
	}
}
