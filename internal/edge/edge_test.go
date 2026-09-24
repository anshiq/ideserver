package edgefw

import (
	"bytes"
	"encoding/binary"
	"net/netip"
	"testing"
)

// buildHello crafts a minimal TLS ClientHello record advertising sni.
func buildHello(sni string) []byte {
	var exts bytes.Buffer
	if sni != "" {
		name := []byte(sni)
		var list bytes.Buffer
		list.WriteByte(0) // host_name
		binary.Write(&list, binary.BigEndian, uint16(len(name)))
		list.Write(name)
		var body bytes.Buffer
		binary.Write(&body, binary.BigEndian, uint16(list.Len()))
		body.Write(list.Bytes())
		binary.Write(&exts, binary.BigEndian, uint16(0)) // server_name
		binary.Write(&exts, binary.BigEndian, uint16(body.Len()))
		exts.Write(body.Bytes())
	}
	// also add a dummy extension to prove skipping works
	binary.Write(&exts, binary.BigEndian, uint16(0x0017))
	binary.Write(&exts, binary.BigEndian, uint16(0))

	var hs bytes.Buffer
	hs.Write([]byte{0x03, 0x03}) // version
	hs.Write(make([]byte, 32))   // random
	hs.WriteByte(0)              // session id len
	hs.Write([]byte{0x00, 0x02, 0x13, 0x01})
	hs.WriteByte(1) // compression methods
	hs.WriteByte(0)
	binary.Write(&hs, binary.BigEndian, uint16(exts.Len()))
	hs.Write(exts.Bytes())

	var hello bytes.Buffer
	hello.Write(hs.Bytes())
	var rec bytes.Buffer
	rec.WriteByte(22)
	rec.Write([]byte{0x03, 0x01})
	binary.Write(&rec, binary.BigEndian, uint16(hello.Len()))
	rec.Write(hello.Bytes())

	var out bytes.Buffer
	out.WriteByte(1) // client_hello
	l := hello.Len()
	out.Write([]byte{byte(l >> 16), byte(l >> 8), byte(l)})
	out.Write(hello.Bytes())
	_ = out
	// rebuild: record payload must be handshake message(s)
	payload := out.Bytes()
	var final bytes.Buffer
	final.WriteByte(22)
	final.Write([]byte{0x03, 0x01})
	binary.Write(&final, binary.BigEndian, uint16(len(payload)))
	final.Write(payload)
	return final.Bytes()
}

func TestPeekSNI(t *testing.T) {
	rec := buildHello("a7k2m9xq.vps.example.com")
	sni, err := PeekSNI(rec)
	if err != nil {
		t.Fatal(err)
	}
	if sni != "a7k2m9xq.vps.example.com" {
		t.Fatalf("got %q", sni)
	}
}

func TestPeekSNI_Errors(t *testing.T) {
	if _, err := PeekSNI([]byte{1, 2}); err == nil {
		t.Fatal("expected short error")
	}
	rec := buildHello("")
	if _, err := PeekSNI(rec); err == nil {
		t.Fatal("expected no-SNI error")
	}
	rec[0] = 21
	if _, err := PeekSNI(rec); err == nil {
		t.Fatal("expected non-handshake error")
	}
}

func TestSplitHost(t *testing.T) {
	id, ok := SplitHost("a7k2m9xq.vps.example.com", "vps.example.com")
	if !ok || id != "a7k2m9xq" {
		t.Fatalf("got %q,%v", id, ok)
	}
	if _, ok := SplitHost("vps.example.com", "vps.example.com"); ok {
		t.Fatal("gateway name must not match a VPS route")
	}
	if _, ok := SplitHost("a.b.vps.example.com", "vps.example.com"); ok {
		t.Fatal("nested label must not match")
	}
	if _, ok := SplitHost("a7k2m9xq.evil.com", "vps.example.com"); ok {
		t.Fatal("foreign suffix must not match")
	}
}

func TestAllow(t *testing.T) {
	rules := DefaultRules()
	ip := netip.MustParseAddr("1.2.3.4")
	for _, port := range []int{22, 80, 443} {
		if !Allow(rules, ProtoTCP, port, ip) {
			t.Fatalf("port %d should be allowed", port)
		}
	}
	if Allow(rules, ProtoTCP, 5432, ip) {
		t.Fatal("5432 should be denied by default")
	}
	if Allow(rules, ProtoUDP, 80, ip) {
		t.Fatal("udp 80 should be denied")
	}
	if Allow(nil, ProtoTCP, 80, ip) {
		t.Fatal("empty rules must default-deny")
	}
	// source-scoped rule
	rules = append(rules, Rule{ProtoTCP, 5432, 5432, MustCIDR("10.0.0.0/8")})
	if !Allow(rules, ProtoTCP, 5432, netip.MustParseAddr("10.1.2.3")) {
		t.Fatal("10/8 should reach 5432")
	}
	if Allow(rules, ProtoTCP, 5432, ip) {
		t.Fatal("public IP must not reach 5432")
	}
}
