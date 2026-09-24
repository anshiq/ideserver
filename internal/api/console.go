package api

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/anshiq/ideserver/internal/auth"
	"github.com/anshiq/ideserver/internal/db"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
)

// handleConsole upgrades to WebSocket and relays to hostd's console bridge
// over the unix socket. Auth + ownership are checked before upgrade; the
// first line sent on the socket is the VPS id, the rest is the PTY stream.
func (s *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := ValidateVPSID(id); err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var userID int64
	var role string
	if u, ok := userFromCtx(r.Context()); ok {
		userID, role = u.ID, u.Role
	} else {
		tok, ok := auth.TokenFromRequest(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		u, err := s.userFromToken(r.Context(), tok)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, role = u.ID, u.Role
	}
	_ = role
	inst, err := s.Q.GetInstanceOwned(r.Context(), db.GetInstanceOwnedParams{ID: id, UserID: userID})
	if err != nil {
		if role == "admin" {
			if _, err2 := s.Q.GetInstance(r.Context(), id); err2 != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
		} else {
			// IDOR-safe: same 404 as missing.
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
	_ = inst
	if s.Cfg.HostdSocket == "" {
		http.Error(w, "console bridge not configured", http.StatusBadGateway)
		return
	}
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{s.Cfg.AppOrigin},
	})
	if err != nil {
		return
	}
	defer ws.CloseNow()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	nc, err := net.DialTimeout("unix", s.Cfg.HostdSocket, 5*time.Second)
	if err != nil {
		_ = ws.Close(websocket.StatusInternalError, "hostd unreachable")
		return
	}
	defer nc.Close()
	if _, err := nc.Write([]byte(id + "\n")); err != nil {
		_ = ws.Close(websocket.StatusInternalError, "bridge write failed")
		return
	}
	go func() {
		defer cancel()
		for {
			mt, rd, err := ws.Reader(ctx)
			if err != nil {
				return
			}
			if mt != websocket.MessageBinary && mt != websocket.MessageText {
				continue
			}
			if _, err := io.Copy(nc, rd); err != nil {
				return
			}
		}
	}()
	go func() {
		defer cancel()
		buf := make([]byte, 4096)
		for {
			n, err := nc.Read(buf)
			if err != nil {
				_ = ws.Close(websocket.StatusGoingAway, "vm closed")
				return
			}
			if err := ws.Write(ctx, websocket.MessageBinary, buf[:n]); err != nil {
				return
			}
		}
	}()
	<-ctx.Done()
}
