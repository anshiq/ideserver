// Package api server wiring: chi router + huma OpenAPI + session auth +
// SSE fan-out + console WebSocket relay (plan §1.8).
//
// The pure validation/IDOR helpers live in api.go (unit-tested). Instance
// creation runs in one pgx transaction that inserts the row AND the River
// job via InsertTx (plan §1.5 step 1 — no orphaned side effects).
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/anshiq/ideserver/internal/auth"
	"github.com/anshiq/ideserver/internal/db"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/riverqueue/river"
	"golang.org/x/time/rate"
)

// Config for the API server.
type Config struct {
	AppDomain   string
	AppOrigin   string // https://app.{domain}, CSRF Origin check
	VPSSuffix   string // vps.{domain}
	AdminLogins string
	SignupMode  string
	Allowlist   string
	HostdSocket string // /run/vps/hostd.sock for console relay
}

// Server bundles dependencies. Pool+River drive the transactional paths;
// Q/DB (database/sql via pgx stdlib) serve the read paths through sqlc.
type Server struct {
	Q       *db.Queries
	DB      *sql.DB
	Pool    *pgxpool.Pool
	River   *river.Client[pgx.Tx]
	OAuth   *auth.OAuth
	Cfg     Config
	Hub     *Hub
	Limiter *rate.Limiter
	Logger  *slog.Logger
}

type ctxKey string

const userCtxKey ctxKey = "vps-user"

// NewServer builds the chi router + huma API. Pool/River may be nil in
// --print-openapi mode (mutating handlers report 503 until wired).
func NewServer(q *db.Queries, dbh *sql.DB, pool *pgxpool.Pool, riv *river.Client[pgx.Tx], oa *auth.OAuth, cfg Config) (chi.Router, huma.API, *Server) {
	s := &Server{Q: q, DB: dbh, Pool: pool, River: riv, OAuth: oa, Cfg: cfg, Hub: NewHub(), Limiter: rate.NewLimiter(rate.Every(time.Second/20), 40), Logger: slog.Default()}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Handle("/metrics", promhttp.Handler())
	// OAuth start/finish are the only truly public, unauthenticated routes
	// (state param protects the callback; there is no session yet to CSRF).
	r.Get("/api/v1/auth/github/login", s.handleOAuthLogin)
	r.Get("/api/v1/auth/github/callback", s.handleOAuthCallback)

	// Logout/events/console do their own token lookup (needed pre-session
	// for the SSE/WS upgrade), so they only need CSRF, applied inline since
	// chi forbids Use() after a route on the same router. The huma API
	// needs both CSRF and a resolved session, so it gets a dedicated
	// sub-router mounted at "/" with its own middleware stack.
	r.Method(http.MethodPost, "/api/v1/auth/logout", s.csrfMiddleware(http.HandlerFunc(s.handleLogout)))
	r.Method(http.MethodGet, "/api/v1/events", s.csrfMiddleware(http.HandlerFunc(s.handleEvents)))
	r.Method(http.MethodGet, "/api/v1/instances/{id}/console", s.csrfMiddleware(http.HandlerFunc(s.handleConsole)))

	protected := chi.NewRouter()
	protected.Use(s.csrfMiddleware, s.authMiddleware)
	humaCfg := huma.DefaultConfig("VPS API", "1.0.0")
	humaCfg.Info.Description = "Persistent Linux machines (plan §1.8)"
	hapi := humachi.New(protected, humaCfg)
	s.register(hapi)
	r.Mount("/", protected)
	return r, hapi, s
}

// --- auth middleware ---

// userFromCtx returns the session user injected by authMiddleware.
func userFromCtx(ctx context.Context) (*db.Users, bool) {
	u, ok := ctx.Value(userCtxKey).(*db.Users)
	return u, ok && u != nil
}

// public reports whether path needs no session.
func public(path string) bool {
	for _, p := range []string{
		"/api/v1/auth/github/login", "/api/v1/auth/github/callback",
		"/api/v1/plans", "/api/v1/images",
		"/metrics", "/openapi.json", "/docs",
	} {
		if path == p || strings.HasPrefix(path, p+"?") ||
			strings.HasPrefix(path, "/docs/") {
			return true
		}
	}
	return false
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if public(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		tok, ok := auth.TokenFromRequest(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := s.userFromToken(r.Context(), tok)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Status != "active" && user.Role != "admin" {
			http.Error(w, "account pending approval", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, user)))
	})
}

func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if err := auth.CheckCSRF(r, s.Cfg.AppOrigin); err != nil {
			http.Error(w, "csrf check failed", http.StatusForbidden)
			return
		}
		if !s.Limiter.Allow() {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- SSE hub ---

// Hub fans Postgres NOTIFY instance_events out to browsers over SSE.
type Hub struct {
	mu   sync.RWMutex
	subs map[chan string]struct{}
}

func NewHub() *Hub { return &Hub{subs: map[chan string]struct{}{}} }

func (h *Hub) Subscribe() (chan string, func()) {
	ch := make(chan string, 16)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
		close(ch)
	}
}

func (h *Hub) Publish(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	tok, ok := auth.TokenFromRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := s.userFromToken(r.Context(), tok)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_ = user
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fl, _ := w.(http.Flusher)
	ch, unsub := s.Hub.Subscribe()
	defer unsub()
	fmt.Fprintf(w, ": connected\n\n")
	if fl != nil {
		fl.Flush()
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if fl != nil {
				fl.Flush()
			}
		}
	}
}

// userFromToken resolves the opaque session cookie via its SHA-256 hash.
func (s *Server) userFromToken(ctx context.Context, tok string) (*db.Users, error) {
	row, err := s.Q.GetSession(ctx, auth.Hash(tok))
	if err != nil {
		return nil, err
	}
	return &db.Users{
		ID: row.UID, GithubID: row.GithubID, Login: row.Login, Name: row.Name,
		Email: row.Email, AvatarUrl: row.AvatarUrl, Role: row.Role, Status: row.Status,
	}, nil
}

// requireUser is the huma-path auth gate (chi middleware covers raw paths;
// huma handlers call this because humachi bypasses chi route middleware for
// the API prefix only when mounted — belt and suspenders).
func (s *Server) requireUser(ctx context.Context) (*db.Users, error) {
	if u, ok := userFromCtx(ctx); ok {
		return u, nil
	}
	return nil, huma.Error401Unauthorized("unauthorized")
}

// requireWired fails mutating handlers when Pool/River are absent
// (--print-openapi mode).
func (s *Server) requireWired() error {
	if s.Pool == nil || s.River == nil {
		return huma.Error503ServiceUnavailable("job queue not connected")
	}
	return nil
}

// audit writes one audit_log row; failures are logged, never fatal.
func (s *Server) audit(ctx context.Context, actor int64, instance string, action, ip string, meta map[string]string) {
	raw, _ := json.Marshal(meta)
	_ = s.Q.Audit(ctx, db.AuditParams{
		ActorID:    sql.NullInt64{Int64: actor, Valid: true},
		InstanceID: sql.NullString{String: instance, Valid: instance != ""},
		Action:     action,
		Meta:       raw,
		Ip:         ip,
	})
}

// --- Huma operation types shared + registration dispatch ---

type meOutput struct {
	Body struct {
		ID     int64  `json:"id"`
		Login  string `json:"login"`
		Role   string `json:"role"`
		Status string `json:"status"`
	} `json:"body"`
}

func (s *Server) register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-me", Method: http.MethodGet, Path: "/api/v1/me",
	}, func(ctx context.Context, _ *struct{}) (*meOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		out := &meOutput{}
		out.Body.ID = u.ID
		out.Body.Login = u.Login
		out.Body.Role = u.Role
		out.Body.Status = u.Status
		return out, nil
	})
	s.registerCatalog(api)
	s.registerInstances(api)
	s.registerNetwork(api)
	s.registerKeys(api)
	s.registerAdmin(api)
}
