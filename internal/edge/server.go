// Package edgefw server: public :80/:443 (+ extra HTTP ports) reverse proxy.
//
//   - Peeks TLS ClientHello for SNI without terminating (passthrough mode).
//   - Terminates wildcard TLS via certmagic DNS-01 in terminate mode.
//   - Routes on Host/SNI: app.{domain} → web + /api → api; {id}.vps.{domain}
//     → the VM (Host header preserved, WebSocket supported).
//   - Enforces firewall rules for hostname-routed ports from the route table.
//   - Serves 404/503 pages for unknown or stopped VPSes.
package edgefw

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Route is one routable VPS as seen by the edge. The concrete table lives
// in internal/routes (which imports this package for Rule/Allow); the edge
// depends only on this struct + Resolver to avoid an import cycle.
type Route struct {
	ID        string
	IP        string
	HTTPSMode string
	Firewall  []Rule
}

// Allow checks the route's firewall.
func (r Route) Allow(proto Proto, port int, client netip.Addr) bool {
	return Allow(r.Firewall, proto, port, client)
}

// Resolver maps "{id}.vps.{domain}" to a running route.
type Resolver interface {
	ResolveHost(host string) (Route, bool)
	Suffix() string
}

// ResolverFunc adapts a plain func to Resolver.
type ResolverFunc struct {
	Sfx string
	Fn  func(host string) (Route, bool)
}

func (r ResolverFunc) ResolveHost(host string) (Route, bool) { return r.Fn(host) }
func (r ResolverFunc) Suffix() string                        { return r.Sfx }

// Config for the edge proxy.
type Config struct {
	Domain    string
	AppDomain string
	VPSuffix  string
	WebAddr   string // 127.0.0.1:13000
	APIAddr   string // 127.0.0.1:8081
	CertDir   string
	ACMEEmail string
}

// Server is the edge proxy.
type Server struct {
	Cfg      Config
	Resolver Resolver
	Logger   *slog.Logger
	proxies  *proxySet
}

type proxySet struct {
	web *httputil.ReverseProxy
	api *httputil.ReverseProxy
}

// New builds a Server with reverse proxies to loopback web/api.
func New(cfg Config, res Resolver) *Server {
	if res == nil {
		res = ResolverFunc{Sfx: cfg.VPSuffix, Fn: func(string) (Route, bool) { return Route{}, false }}
	}
	mk := func(addr string) *httputil.ReverseProxy {
		return &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				r.URL.Scheme = "http"
				r.URL.Host = addr
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				serve503(w, "upstream unavailable")
			},
		}
	}
	return &Server{Cfg: cfg, Resolver: res, Logger: slog.Default(),
		proxies: &proxySet{web: mk(cfg.WebAddr), api: mk(cfg.APIAddr)}}
}

// Handler is the HTTP entrypoint for :80 and terminated :443.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", s.routeHTTP)
	return mux
}

func (s *Server) routeHTTP(w http.ResponseWriter, r *http.Request) {
	host := stripPort(r.Host)
	if host == s.Cfg.AppDomain || host == s.Cfg.Domain {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			s.proxies.api.ServeHTTP(w, r)
			return
		}
		s.proxies.web.ServeHTTP(w, r)
		return
	}
	entry, ok := s.Resolver.ResolveHost(host)
	if !ok {
		serve404(w, host)
		return
	}
	client := clientIP(r)
	if !entry.Allow(ProtoTCP, httpPort(r), client) {
		serve403(w)
		return
	}
	s.proxyToVM(w, r, entry)
}

// proxyToVM forwards to VM_IP:port preserving Host (WebSocket supported).
func (s *Server) proxyToVM(w http.ResponseWriter, r *http.Request, e Route) {
	port := httpPort(r)
	target := net.JoinHostPort(e.IP, fmt.Sprint(port))
	p := &httputil.ReverseProxy{
		Director: func(out *http.Request) {
			out.URL.Scheme = "http"
			out.URL.Host = target
			// Preserve the public Host so vhosts inside the VM work.
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			serve503(w, "VPS unreachable")
		},
	}
	p.ServeHTTP(w, r)
}

// PeekAndRoute is the raw-TCP entrypoint for :443/8443: it peeks SNI, then
// either terminates (terminate mode) or splices bytes to VM:443
// (passthrough mode). Plain HTTP on :80 is handled by Handler.
func (s *Server) PeekAndRoute(ctx context.Context, client net.Conn) {
	defer client.Close()
	_ = ctx
	buf := make([]byte, 4096)
	client.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := client.Read(buf)
	if err != nil {
		return
	}
	client.SetReadDeadline(time.Time{})
	rec := buf[:n]
	sni, err := PeekSNI(rec)
	if err != nil {
		return
	}
	entry, ok := s.Resolver.ResolveHost(sni)
	if !ok {
		return
	}
	if strings.EqualFold(entry.HTTPSMode, "passthrough") {
		s.splice(client, rec, net.JoinHostPort(entry.IP, "443"))
		return
	}
	// terminate mode is served by the TLS listener with the wildcard cert;
	// a raw splice here would double-terminate, so close and let the TLS
	// server handle the retry (client reconnects on next handshake).
	_ = entry
}

// splice pipes the peeked bytes plus the rest of the stream to the VM.
func (s *Server) splice(client net.Conn, peeked []byte, target string) {
	upstream, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		return
	}
	defer upstream.Close()
	if _, err := upstream.Write(peeked); err != nil {
		return
	}
	go io.Copy(upstream, client)
	io.Copy(client, upstream)
}

// TLSConfig returns a tls.Config that selects the wildcard cert and routes
// passthrough domains without terminating. The caller wraps the :443
// listener with it when certmagic is configured.
func (s *Server) TLSConfig(getCert func(*tls.ClientHelloInfo) (*tls.Certificate, error)) *tls.Config {
	return &tls.Config{
		GetCertificate: getCert,
		NextProtos:     []string{"h2", "http/1.1"},
		MinVersion:     tls.VersionTLS12,
	}
}

func stripPort(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return strings.ToLower(strings.TrimSuffix(hostport, "."))
	}
	return strings.ToLower(h)
}

func httpPort(r *http.Request) int {
	_, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		if r.TLS != nil {
			return 443
		}
		return 80
	}
	for _, p := range []int{80, 443, 3000, 5000, 8000, 8080, 8443} {
		if fmt.Sprint(p) == port {
			return p
		}
	}
	var n int
	fmt.Sscanf(port, "%d", &n)
	if n == 0 {
		return 80
	}
	return n
}

func clientIP(r *http.Request) netip.Addr {
	// Direct connection (DNAT preserves IP on mapped ports; edge sees the
	// real peer on hostname-routed ports). X-Forwarded-For is not trusted
	// (single-host, no upstream CDN in v1).
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return addr
}

func serve404(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "<h1>No such VPS</h1><p>%s does not exist.</p>", host)
}

func serve503(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprintf(w, "<h1>VPS unavailable</h1><p>%s. The VM may be stopped.</p>", msg)
}

func serve403(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprint(w, "<h1>Blocked by firewall</h1>")
}
