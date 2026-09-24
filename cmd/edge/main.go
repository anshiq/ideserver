// Command edge is the public HTTP(S) proxy (plan §1.1). It needs only
// CAP_NET_BIND_SERVICE: it never touches Incus or Postgres writes, only the
// route snapshot + LISTEN/NOTIFY refresh.
package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anshiq/ideserver/internal/config"
	"github.com/anshiq/ideserver/internal/db"
	edgefw "github.com/anshiq/ideserver/internal/edge"
	"github.com/anshiq/ideserver/internal/routes"
	"github.com/caddyserver/certmagic"
	"github.com/jackc/pgx/v5"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	cfg, err := config.LoadEdge()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	table := routes.New(cfg.VPSSuffix)
	srv := edgefw.New(edgefw.Config{
		Domain: cfg.Domain, AppDomain: cfg.AppDomain, VPSuffix: cfg.VPSSuffix,
		WebAddr: cfg.WebAddr, APIAddr: cfg.APIAddr, CertDir: cfg.CertDir, ACMEEmail: cfg.ACMEEmail,
	}, routes.Adapter{T: table})

	go refreshRoutes(ctx, log, cfg.DatabaseURL, table, cfg.VPSSuffix)

	// Wildcard TLS via certmagic DNS-01 (libdns provider configured in
	// deploy; empty DNS_PROVIDER logs and serves HTTP only + self-signed).
	if cfg.DNSProvider != "" {
		certmagic.Default.Storage = &certmagic.FileStorage{Path: cfg.CertDir}
		if cfg.ACMEEmail != "" {
			certmagic.DefaultACME.Agreed = true
			certmagic.DefaultACME.Email = cfg.ACMEEmail
			certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
		}
		names := []string{"*." + cfg.VPSSuffix, cfg.AppDomain}
		go func() {
			if err := certmagic.ManageSync(ctx, names); err != nil {
				log.Warn("certmagic manage", "err", err)
			}
		}()
		https := &http.Server{
			Addr: cfg.HTTPSAddr, Handler: srv.Handler(),
			TLSConfig:         certmagic.Default.TLSConfig(),
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			log.Info("edge https", "addr", cfg.HTTPSAddr)
			if err := https.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Error("edge https", "err", err)
			}
		}()
		defer https.Shutdown(context.Background())
		_ = tls.VersionTLS12
	} else {
		log.Warn("edge: DNS_PROVIDER empty, TLS disabled (dev mode)")
	}

	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shut, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel2()
		_ = httpSrv.Shutdown(shut)
	}()
	// Extra HTTP ports (3000,5000,8000,8080) share the same handler.
	for _, p := range cfg.ExtraPorts {
		go func(port int) {
			addr := net.JoinHostPort("", itoa(port))
			_ = http.ListenAndServe(addr, srv.Handler())
		}(p)
	}
	log.Info("edge http", "addr", cfg.HTTPAddr, "suffix", cfg.VPSSuffix)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("edge http", "err", err)
		os.Exit(1)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := []byte{}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// refreshRoutes rebuilds the table from RoutesSnapshot every 30s and on
// LISTEN/NOTIFY instance_events (plan §1.5 step 3).
func refreshRoutes(ctx context.Context, log *slog.Logger, databaseURL string, table *routes.Table, suffix string) {
	load := func(ctx context.Context) ([]routes.Entry, error) {
		conn, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return nil, err
		}
		defer conn.Close(ctx)
		rows, err := conn.Query(ctx, `SELECT i.id, i.private_ipv4::text, i.state, i.https_mode, i.user_id FROM instances i WHERE i.deleted_at IS NULL`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []routes.Entry
		for rows.Next() {
			var e routes.Entry
			if err := rows.Scan(&e.ID, &e.IP, &e.State, &e.HTTPSMode, &e.OwnerID); err != nil {
				continue
			}
			// Firewall rules are loaded lazily per-request in v1 skeleton;
			// default-allow 22/80/443 keeps dev working until sync lands.
			e.Firewall = edgefw.DefaultRules()
			out = append(out, e)
		}
		return out, rows.Err()
	}
	notifyCh := make(chan string, 16)
	go func() {
		conn, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, `LISTEN "instance_events"`); err != nil {
			return
		}
		for {
			n, err := conn.WaitForNotification(ctx)
			if err != nil {
				return
			}
			select {
			case notifyCh <- n.Payload:
			default:
			}
		}
	}()
	table.StartRefresher(ctx, 30*time.Second, notifyCh, load, func(err error) {
		log.Warn("routes refresh", "err", err)
	})
	_ = db.New
}
