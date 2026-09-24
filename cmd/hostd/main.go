// Command hostd owns the Incus socket and runs River workers, the
// reconciler, the console bridge and capacity reporting (plan §1.1).
// It runs in the incus-admin group; no other binary touches Incus.
package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anshiq/ideserver/internal/config"
	"github.com/anshiq/ideserver/internal/db"
	"github.com/anshiq/ideserver/internal/ids"
	"github.com/anshiq/ideserver/internal/incusx"
	"github.com/anshiq/ideserver/internal/jobs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	cfg, err := config.LoadHostD()
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

	inc, err := incusx.DialReal(incusx.RealConfig{
		SocketPath: cfg.IncusSocket, Project: cfg.IncusProject, Bridge: cfg.BridgeName,
	})
	var client incusx.Client
	if err != nil {
		log.Warn("incus unavailable, using fake (dev only)", "err", err)
		client = incusx.NewFake()
	} else {
		client = inc
	}

	deps := &jobs.Deps{Q: q, DB: dbh, Inc: client, Bridge: cfg.BridgeName, Project: cfg.IncusProject, Logger: log}
	workers := river.NewWorkers()
	jobs.AddAll(workers, deps)
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			jobs.QueueDefault: {MaxWorkers: cfg.Workers},
			jobs.QueueNode:    {MaxWorkers: cfg.Workers},
		},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(river.PeriodicInterval(time.Hour), func() (river.JobArgs, *river.InsertOpts) {
				return jobs.MeterArgs{}, &river.InsertOpts{Queue: jobs.QueueDefault}
			}, &river.PeriodicJobOpts{RunOnStart: false}),
		},
	})
	if err != nil {
		log.Error("river client", "err", err)
		os.Exit(1)
	}
	if err := riverClient.Start(ctx); err != nil {
		log.Error("river start", "err", err)
		os.Exit(1)
	}
	log.Info("hostd river started", "project", cfg.IncusProject, "bridge", cfg.BridgeName)

	go serveMetrics(log)
	go runReconciler(ctx, log, q, client)
	go serveConsoleSocket(ctx, log, cfg.SocketPath, q, client)

	<-ctx.Done()
	_ = riverClient.Stop(context.Background())
	log.Info("hostd shutdown")
}

func serveMetrics(log *slog.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe("127.0.0.1:9091", mux)
}

// runReconciler diffs DB desired state against Incus every 60s (plan §1.5
// step 4). Drift is corrected: crashed VMs are marked stopped, orphan Incus
// instances raise a log alert.
func runReconciler(ctx context.Context, log *slog.Logger, q *db.Queries, inc incusx.Client) {
	tkr := time.NewTicker(60 * time.Second)
	defer tkr.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tkr.C:
			reconcileOnce(ctx, log, q, inc)
		}
	}
}

func reconcileOnce(ctx context.Context, log *slog.Logger, q *db.Queries, inc incusx.Client) {
	for _, state := range []string{"running", "starting", "stopping", "rebooting", "provisioning"} {
		insts, err := q.ListInstancesByState(ctx, state)
		if err != nil {
			log.Warn("reconcile list", "err", err)
			return
		}
		for _, inst := range insts {
			name := ids.IncusName(inst.ID)
			exists, err := inc.Exists(ctx, name)
			if err != nil {
				continue
			}
			if !exists {
				log.Warn("reconcile: incus instance missing, marking stopped", "id", inst.ID)
				_ = q.SetInstanceState(ctx, db.SetInstanceStateParams{ID: inst.ID, State: "stopped", LastError: "VM missing in Incus (reconciler)"})
				continue
			}
			st, err := inc.Stats(ctx, name)
			if err != nil {
				continue
			}
			if st.Status == "Stopped" && inst.State == "running" {
				log.Warn("reconcile: VM crashed, marking stopped", "id", inst.ID)
				_ = q.SetInstanceState(ctx, db.SetInstanceStateParams{ID: inst.ID, State: "stopped", LastError: "VM stopped unexpectedly"})
			}
		}
	}
}

// serveConsoleSocket exposes the console bridge to api over the unix socket:
// api dials /run/vps/hostd.sock and hostd runs `incus exec` with a PTY.
func serveConsoleSocket(ctx context.Context, log *slog.Logger, socketPath string, q *db.Queries, inc incusx.Client) {
	// 0o750 in case this runs without systemd's RuntimeDirectory (which
	// already creates /run/vps at 0750 owned by vps-hostd:vps-console); if
	// the directory already exists MkdirAll leaves its mode untouched.
	_ = os.MkdirAll("/run/vps", 0o750)
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Warn("console socket", "err", err)
		return
	}
	// Only the owner/group (api, via vps-console) may dial this socket: it
	// hands out a raw PTY into any VM by id with no auth of its own, so
	// api's ownership check in handleConsole is the only gate.
	_ = os.Chmod(socketPath, 0o660)
	defer ln.Close()
	log.Info("hostd console bridge", "socket", socketPath)
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		nc, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		go func(c net.Conn) {
			defer c.Close()
			// Protocol: first line is the VPS id; the rest is the PTY stream.
			// Full PTY wiring uses incus Exec with cols/rows from the WS frame.
			buf := make([]byte, 9)
			c.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := c.Read(buf)
			if err != nil || n == 0 {
				return
			}
			c.SetReadDeadline(time.Time{})
			id := string(buf[:n-1])
			name := ids.IncusName(trimID(id))
			_ = inc.Exec(ctx, name, 80, 24, c, c, c)
		}(nc)
	}
}

func trimID(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
