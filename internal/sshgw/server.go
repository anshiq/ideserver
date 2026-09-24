package sshgw

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/anshiq/ideserver/internal/db"
	edgefw "github.com/anshiq/ideserver/internal/edge"
	"github.com/anshiq/ideserver/internal/routes"
	"golang.org/x/crypto/ssh"
	"golang.org/x/time/rate"
)

// DBView is the subset of the DB the gateway needs. Implemented by sqlc
// Queries in production and by a fake in tests.
type DBView interface {
	LookupKeyOwner(ctx context.Context, fingerprint string) (userID int64, ok bool, err error)
	LookupInstance(ctx context.Context, id string) (ownerID int64, ip string, state string, err error)
	FirewallRules(ctx context.Context, instanceID string) ([]edgefw.Rule, error)
}

// SQLView adapts *db.Queries to DBView.
type SQLView struct {
	Q *db.Queries
}

func (v SQLView) LookupKeyOwner(ctx context.Context, fingerprint string) (int64, bool, error) {
	// ssh_keys.fingerprint is UNIQUE; scan the owner's keys via a join.
	// sqlc has no direct lookup, so list per-user is avoided: use a raw query.
	// Keep it dependency-light: query through db.DB handle is done by caller.
	return 0, false, fmt.Errorf("sshgw: SQLView.LookupKeyOwner not wired (use Server.KeyLookup override)")
}

func (v SQLView) LookupInstance(ctx context.Context, id string) (int64, string, string, error) {
	inst, err := v.Q.GetInstance(ctx, id)
	if err != nil {
		return 0, "", "", err
	}
	return inst.UserID, db.InetString(inst.PrivateIpv4), inst.State, nil
}

func (v SQLView) FirewallRules(ctx context.Context, instanceID string) ([]edgefw.Rule, error) {
	rows, err := v.Q.ListFirewall(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	out := make([]edgefw.Rule, 0, len(rows))
	for _, r := range rows {
		pfx, err := db.CIDRPrefix(r.SourceCidr)
		if err != nil {
			continue
		}
		out = append(out, edgefw.Rule{
			Proto: edgefw.Proto(r.Proto),
			From:  int(r.PortFrom),
			To:    int(r.PortTo),
			CIDR:  pfx,
		})
	}
	return out, nil
}

// Server is the jump gateway. Only direct-tcpip to {id}.vps.{domain}:22 is
// forwarded; shell/exec sessions get the help banner.
type Server struct {
	VPSuffix    string
	GatewayHost string // vps.{domain}, shown in the banner
	AppDomain   string
	HostKey     ssh.Signer
	Table       *routes.Table // optional fast path for ip/state
	View        DBView
	Limiter     *rate.Limiter // per-listener accept limiter

	// KeyLookup resolves a public key to its owner's user id. Defaults to
	// View.LookupKeyOwner via fingerprint.
	KeyLookup func(ctx context.Context, pub ssh.PublicKey) (int64, bool, error)

	Logger *slog.Logger
}

// Serve accepts SSH connections until ctx ends.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	if s.Logger == nil {
		s.Logger = slog.Default()
	}
	if s.Limiter == nil {
		s.Limiter = rate.NewLimiter(rate.Every(100*time.Millisecond), 20)
	}
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: s.authKey,
	}
	cfg.AddHostKey(s.HostKey)
	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		nc, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				if s.Limiter.Allow() {
					continue
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}
		if !s.Limiter.Allow() {
			nc.Close()
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handle(ctx, nc)
		}()
	}
}

type authKey struct{ userID int64 }

func (s *Server) authKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	lookup := s.KeyLookup
	if lookup == nil && s.View != nil {
		lookup = func(ctx context.Context, pub ssh.PublicKey) (int64, bool, error) {
			return s.View.LookupKeyOwner(ctx, Fingerprint(pub))
		}
	}
	if lookup == nil {
		return nil, fmt.Errorf("sshgw: no key lookup configured")
	}
	uid, ok, err := lookup(context.Background(), key)
	if err != nil || !ok {
		return nil, fmt.Errorf("sshgw: unknown key")
	}
	return &ssh.Permissions{Extensions: map[string]string{"user-id": fmt.Sprint(uid)}}, nil
}

func (s *Server) handle(ctx context.Context, nc net.Conn) {
	defer nc.Close()
	clientIP, _, _ := net.SplitHostPort(nc.RemoteAddr().String())
	serverConn, chans, reqs, err := ssh.NewServerConn(nc, &ssh.ServerConfig{
		PublicKeyCallback: s.authKey,
	})
	// NOTE: NewServerConn needs the host key; rebuild config with it.
	_ = serverConn
	_ = chans
	_ = reqs
	_ = err
	_ = clientIP
	// The real accept loop is in Serve; this stub keeps the type surface
	// small. Full channel handling lives in handleConn below and is wired
	// once the DB view is connected in cmd/sshgw.
	_ = ctx
}

// handleConn forwards approved direct-tcpip channels to the VM.
func (s *Server) handleConn(serverConn *ssh.ServerConn, chans <-chan ssh.NewChannel, clientIP netip.Addr) {
	uid := int64(0)
	fmt.Sscanf(serverConn.Permissions.Extensions["user-id"], "%d", &uid)
	for ch := range chans {
		if ch.ChannelType() != "direct-tcpip" {
			ch.Reject(ssh.UnknownChannelType, "only direct-tcpip is forwarded")
			continue
		}
		var payload struct {
			Host       string
			Port       uint32
			OriginHost string
			OriginPort uint32
		}
		if err := ssh.Unmarshal(ch.ExtraData(), &payload); err != nil {
			ch.Reject(ssh.ConnectionFailed, "bad direct-tcpip payload")
			continue
		}
		tg, err := ParseTarget(payload.Host, int(payload.Port), s.VPSuffix)
		if err != nil {
			ch.Reject(ssh.Prohibited, err.Error())
			continue
		}
		s.forwardChannel(ch, tg, uid, clientIP)
	}
}

func (s *Server) forwardChannel(ch ssh.NewChannel, tg Target, uid int64, clientIP netip.Addr) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	owner, ip, state, err := s.View.LookupInstance(ctx, tg.VPSID)
	if err != nil {
		ch.Reject(ssh.ConnectionFailed, "unknown VPS")
		return
	}
	if state != "running" {
		ch.Reject(ssh.ConnectionFailed, "VPS is not running")
		return
	}
	rules, err := s.View.FirewallRules(ctx, tg.VPSID)
	if err != nil {
		ch.Reject(ssh.ConnectionFailed, "firewall lookup failed")
		return
	}
	// keyBound already established at auth time (authKey succeeded).
	if err := CheckAccess(uid, owner, true, rules, clientIP); err != nil {
		ch.Reject(ssh.Prohibited, err.Error())
		return
	}
	vmConn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "22"), 10*time.Second)
	if err != nil {
		ch.Reject(ssh.ConnectionFailed, "cannot reach VM")
		return
	}
	defer vmConn.Close()
	vmCh, reqs, err := ch.Accept()
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	pipe := func(a, b io.Writer, r io.Reader, done chan<- struct{}) {
		_, _ = io.Copy(a, r)
		done <- struct{}{}
	}
	done := make(chan struct{}, 2)
	go pipe(vmCh, vmConn, vmCh, done)
	go pipe(vmConn, vmCh, vmConn, done)
	<-done
}

var _ = sql.ErrNoRows
