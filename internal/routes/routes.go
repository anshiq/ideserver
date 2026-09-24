// Package routes holds the shared in-memory route table used by edge and
// sshgw. No lifecycle state lives here: the table is a cache rebuilt from
// Postgres (RoutesSnapshot) and refreshed on LISTEN/NOTIFY instance_events
// with a periodic full reload as backstop (plan §1.5).
package routes

import (
	"context"
	"net/netip"
	"strings"
	"sync"
	"time"

	edgefw "github.com/anshiq/ideserver/internal/edge"
)

// Entry is one routable VPS.
type Entry struct {
	ID        string
	IP        string
	State     string // running required for routing
	HTTPSMode string // terminate|passthrough
	OwnerID   int64
	Firewall  []edgefw.Rule
}

// Running reports whether the entry accepts traffic.
func (e Entry) Running() bool { return e.State == "running" }

// Table is a concurrency-safe VPS route cache.
type Table struct {
	mu        sync.RWMutex
	entries   map[string]Entry
	vpsSuffix string
}

// New returns an empty table for vpSuffix (e.g. vps.example.com).
func New(vpsSuffix string) *Table {
	return &Table{entries: map[string]Entry{}, vpsSuffix: strings.ToLower(vpsSuffix)}
}

// Suffix returns the configured VPS suffix.
func (t *Table) Suffix() string { return t.vpsSuffix }

// Replace swaps the whole table (full reload path).
func (t *Table) Replace(entries []Entry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	m := make(map[string]Entry, len(entries))
	for _, e := range entries {
		m[e.ID] = e
	}
	t.entries = m
}

// Upsert inserts or replaces one entry (NOTIFY fast path).
func (t *Table) Upsert(e Entry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries[e.ID] = e
}

// Remove drops one entry.
func (t *Table) Remove(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, id)
}

// Get returns the entry for id.
func (t *Table) Get(id string) (Entry, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	e, ok := t.entries[id]
	return e, ok
}

// Count returns the number of cached entries.
func (t *Table) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}

// ResolveHost maps "{id}.vps.{domain}" to its entry. ok=false for the bare
// gateway name, unknown ids, or non-running VPSes.
func (t *Table) ResolveHost(host string) (Entry, bool) {
	id, ok := edgefw.SplitHost(host, t.vpsSuffix)
	if !ok {
		return Entry{}, false
	}
	e, ok := t.Get(id)
	if !ok || !e.Running() {
		return Entry{}, false
	}
	return e, true
}

// Allow checks the entry's firewall for proto/port from client IP.
func (e Entry) Allow(proto edgefw.Proto, port int, client netip.Addr) bool {
	return edgefw.Allow(e.Firewall, proto, port, client)
}

// SnapshotFunc loads entries from the authoritative store.
type SnapshotFunc func(ctx context.Context) ([]Entry, error)

// StartRefresher reloads the table every interval until ctx ends, and calls
// reload on every value received on notifyCh (LISTEN/NOTIFY payloads).
// It returns when ctx is done.
func (t *Table) StartRefresher(ctx context.Context, interval time.Duration, notifyCh <-chan string, load SnapshotFunc, onErr func(error)) {
	tkr := time.NewTicker(interval)
	defer tkr.Stop()
	doReload := func() {
		ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		entries, err := load(ctx2)
		if err != nil {
			if onErr != nil {
				onErr(err)
			}
			return
		}
		t.Replace(entries)
	}
	// Initial load (best effort; periodic ticker retries).
	doReload()
	for {
		select {
		case <-ctx.Done():
			return
		case <-notifyCh:
			doReload()
		case <-tkr.C:
			doReload()
		}
	}
}
