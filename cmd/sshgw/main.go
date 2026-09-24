// Command sshgw is the public SSH jump host on :22 (plan §1.1).
// It needs only CAP_NET_BIND_SERVICE. Host sshd must move to 2222 key-only
// BEFORE this binds 22 (plan §1.2).
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/anshiq/ideserver/internal/config"
	"github.com/anshiq/ideserver/internal/db"
	"github.com/anshiq/ideserver/internal/routes"
	"github.com/anshiq/ideserver/internal/sshgw"
	"golang.org/x/crypto/ssh"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	cfg, err := config.LoadSSHGW()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	signer, err := loadOrGenerateHostKey(cfg.HostKeyPath)
	if err != nil {
		log.Error("host key", "err", err)
		os.Exit(1)
	}

	dbh, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db open", "err", err)
		os.Exit(1)
	}
	defer dbh.Close()
	q := db.New(dbh)

	table := routes.New(cfg.VPSSuffix)
	srv := &sshgw.Server{
		VPSuffix:    cfg.VPSSuffix,
		GatewayHost: cfg.VPSSuffix, // vps.{domain}; must NOT match *.vps.{domain}
		AppDomain:   cfg.AppDomain,
		HostKey:     signer,
		Table:       table,
		View:        sshgw.SQLView{Q: q},
		Logger:      log,
	}
	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Error("listen", "addr", cfg.ListenAddr, "err", err)
		os.Exit(1)
	}
	log.Info("sshgw listening", "addr", cfg.ListenAddr, "suffix", cfg.VPSSuffix)
	if err := srv.Serve(ctx, ln); err != nil {
		log.Error("sshgw serve", "err", err)
		os.Exit(1)
	}
}

func loadOrGenerateHostKey(path string) (ssh.Signer, error) {
	if raw, err := os.ReadFile(path); err == nil {
		return ssh.ParsePrivateKey(raw)
	}
	key, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return nil, err
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	raw := pem.EncodeToMemory(block)
	if err := os.MkdirAll(dirOf(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(raw)
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
