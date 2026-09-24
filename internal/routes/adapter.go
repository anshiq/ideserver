package routes

import (
	edgefw "github.com/anshiq/ideserver/internal/edge"
)

// Adapter exposes the Table as an edgefw.Resolver without an import cycle
// (routes imports edge; edge only sees the Resolver interface).
type Adapter struct{ T *Table }

func (a Adapter) ResolveHost(host string) (edgefw.Route, bool) {
	e, ok := a.T.ResolveHost(host)
	if !ok {
		return edgefw.Route{}, false
	}
	return edgefw.Route{ID: e.ID, IP: e.IP, HTTPSMode: e.HTTPSMode, Firewall: e.Firewall}, true
}

func (a Adapter) Suffix() string { return a.T.Suffix() }
