// Command api serves REST under /api/v1 on loopback (plan §1.1).
// It never touches Incus: it writes desired state + enqueues River jobs in
// one Postgres transaction (plan §1.5 step 1).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anshiq/ideserver/internal/api"
	"github.com/anshiq/ideserver/internal/auth"
	"github.com/anshiq/ideserver/internal/config"
	"github.com/anshiq/ideserver/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	printOpenAPI := flag.Bool("print-openapi", false, "print OpenAPI 3.1 JSON and exit (for `just gen`)")
	flag.Parse()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	if *printOpenAPI {
		// No DB needed: handlers that need rows return 401/503 until wired.
		_, humaAPI, _ := api.NewServer(db.New(nil), nil, nil, nil, nil, api.Config{})
		raw, err := json.MarshalIndent(humaAPI.OpenAPI(), "", "  ")
		if err != nil {
			log.Error("openapi marshal", "err", err)
			os.Exit(1)
		}
		os.Stdout.Write(raw)
		os.Stdout.Write([]byte("\n"))
		return
	}
	cfg, err := config.LoadAPI()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dbh, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db open", "err", err)
		os.Exit(1)
	}
	defer dbh.Close()
	q := db.New(dbh)

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("pgxpool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.MigrateRiver(ctx, pool); err != nil {
		log.Error("river migrate", "err", err)
		os.Exit(1)
	}
	producer, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		log.Error("river producer", "err", err)
		os.Exit(1)
	}

	callback := "https://" + cfg.AppDomain + "/api/v1/auth/github/callback"
	oa := auth.NewOAuth(cfg.GitHubClientID, cfg.SessionSecret, callback)

	apiCfg := api.Config{
		AppDomain:   cfg.AppDomain,
		AppOrigin:   "https://" + cfg.AppDomain,
		VPSSuffix:   cfg.VPSSuffix,
		AdminLogins: cfg.AdminLogins,
		SignupMode:  cfg.SignupMode,
		Allowlist:   cfg.AllowlistLogins,
		HostdSocket: "/run/vps/hostd.sock",
	}
	r, _, srvState := api.NewServer(q, dbh, pool, producer, oa, apiCfg)
	go api.StartNotifyBridge(ctx, pool, srvState.Hub, log)

	srv := &http.Server{Addr: cfg.ListenAddr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shut, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel2()
		_ = srv.Shutdown(shut)
	}()
	log.Info("api listening", "addr", cfg.ListenAddr, "app", cfg.AppDomain)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("api serve", "err", err)
		os.Exit(1)
	}
}
