package routes

import (
	"context"
	"testing"

	edgefw "github.com/anshiq/ideserver/internal/edge"
)

func TestResolveHost(t *testing.T) {
	tbl := New("vps.example.com")
	tbl.Replace([]Entry{
		{ID: "a7k2m9xq", IP: "10.100.0.2", State: "running", HTTPSMode: "terminate", OwnerID: 7, Firewall: edgefw.DefaultRules()},
		{ID: "b8x1p2zz", IP: "10.100.0.3", State: "stopped"},
	})
	if _, ok := tbl.ResolveHost("vps.example.com"); ok {
		t.Fatal("gateway name must not resolve")
	}
	e, ok := tbl.ResolveHost("a7k2m9xq.vps.example.com")
	if !ok || e.IP != "10.100.0.2" {
		t.Fatalf("expected running entry, got %+v %v", e, ok)
	}
	if _, ok := tbl.ResolveHost("b8x1p2zz.vps.example.com"); ok {
		t.Fatal("stopped VPS must not resolve")
	}
	if _, ok := tbl.ResolveHost("zzzzzzzz.vps.example.com"); ok {
		t.Fatal("unknown id must not resolve")
	}
	if _, ok := tbl.ResolveHost("A7K2M9XQ.vps.example.com"); !ok {
		t.Fatal("host matching must be case-insensitive")
	}
}

func TestUpsertRemove(t *testing.T) {
	tbl := New("vps.example.com")
	tbl.Upsert(Entry{ID: "a7k2m9xq", State: "running"})
	if tbl.Count() != 1 {
		t.Fatal("expected 1")
	}
	tbl.Remove("a7k2m9xq")
	if tbl.Count() != 0 {
		t.Fatal("expected 0")
	}
}

func TestStartRefresherReloads(t *testing.T) {
	tbl := New("vps.example.com")
	ch := make(chan string, 1)
	calls := 0
	load := func(ctx context.Context) ([]Entry, error) {
		calls++
		return []Entry{{ID: "a7k2m9xq", State: "running"}}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { tbl.StartRefresher(ctx, 1000000000000, ch, load, nil); close(done) }()
	ch <- "a7k2m9xq:running"
	// poll briefly
	for i := 0; i < 100 && tbl.Count() == 0; i++ {
		select {
		case <-done:
		default:
		}
		if tbl.Count() > 0 {
			break
		}
		select {
		case <-ctx.Done():
		default:
		}
		// tiny spin
	}
	cancel()
	<-done
	if calls < 1 {
		t.Fatal("expected at least one load")
	}
}
