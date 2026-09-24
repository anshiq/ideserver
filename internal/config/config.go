// Package config holds one validated env struct per binary, parsed with
// github.com/caarlos0/env. Internal services never bind the public IP:
// only edge and sshgw take a public listen address.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Shared database / domain settings embedded by every binary.
type Shared struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
	Domain      string `env:"DOMAIN,required"` // e.g. example.com
	AppDomain   string `env:"APP_DOMAIN"`      // defaults to app.{Domain}
	VPSSuffix   string `env:"VPS_SUFFIX"`      // defaults to vps.{Domain}
}

// API serves REST on loopback; the edge proxies /api to it same-origin.
type API struct {
	Shared
	ListenAddr      string `env:"API_LISTEN" envDefault:"127.0.0.1:8081"`
	GitHubClientID  string `env:"GITHUB_CLIENT_ID,required"`
	GitHubSecret    string `env:"GITHUB_CLIENT_SECRET,required"`
	SessionSecret   string `env:"SESSION_SECRET,required"`
	SignupMode      string `env:"SIGNUP_MODE" envDefault:"approval"` // open|approval|allowlist
	AllowlistLogins string `env:"ALLOWLIST_LOGINS"`                  // comma-separated, allowlist mode
	AdminLogins     string `env:"ADMIN_GITHUB_LOGINS"`               // comma-separated
}

// HostD owns the Incus socket and runs River workers.
type HostD struct {
	Shared
	SocketPath    string `env:"HOSTD_SOCKET" envDefault:"/run/vps/hostd.sock"`
	IncusSocket   string `env:"INCUS_SOCKET" envDefault:"/var/lib/incus/unix.socket"`
	IncusProject  string `env:"INCUS_PROJECT" envDefault:"vps"`
	BridgeName    string `env:"BRIDGE_NAME" envDefault:"vpsbr0"`
	Workers       int    `env:"WORKERS" envDefault:"10"`
	CPUOvercommit int    `env:"CPU_OVERCOMMIT" envDefault:"4"`
	RAMOvercommit int    `env:"RAM_OVERCOMMIT" envDefault:"1"`
}

// Edge terminates/proxies public HTTP(S).
type Edge struct {
	Shared
	HTTPAddr    string `env:"EDGE_HTTP" envDefault:":80"`
	HTTPSAddr   string `env:"EDGE_HTTPS" envDefault:":443"`
	ExtraPorts  []int  `env:"EDGE_EXTRA_HTTP_PORTS" envDefault:"3000,5000,8000,8080"`
	HTTPPorts   []int  `env:"EDGE_HTTPS_EXTRA_PORTS" envDefault:"8443"`
	ACMEEmail   string `env:"ACME_EMAIL"`
	DNSProvider string `env:"DNS_PROVIDER"` // libdns provider name; empty = self-signed/off
	WebAddr     string `env:"WEB_ADDR" envDefault:"127.0.0.1:13000"`
	APIAddr     string `env:"API_ADDR" envDefault:"127.0.0.1:8081"`
	CertDir     string `env:"CERT_DIR" envDefault:"/var/lib/vps/certs"`
}

// SSHGW is the public jump host on :22.
type SSHGW struct {
	Shared
	ListenAddr  string `env:"SSHGW_LISTEN" envDefault:":22"`
	HostKeyPath string `env:"SSHGW_HOST_KEY" envDefault:"/var/lib/vps/sshgw_hostkey"`
	APIAddr     string `env:"API_ADDR" envDefault:"127.0.0.1:8081"`
}

func load[T any](v *T) (*T, error) {
	if err := env.Parse(v); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return v, nil
}

// LoadAPI parses the api config and fills domain defaults.
func LoadAPI() (*API, error) {
	c, err := load(&API{})
	if err != nil {
		return nil, err
	}
	c.fill()
	return c, nil
}

// LoadHostD parses the hostd config.
func LoadHostD() (*HostD, error) {
	c, err := load(&HostD{})
	if err != nil {
		return nil, err
	}
	c.fill()
	return c, nil
}

// LoadEdge parses the edge config.
func LoadEdge() (*Edge, error) {
	c, err := load(&Edge{})
	if err != nil {
		return nil, err
	}
	c.fill()
	return c, nil
}

// LoadSSHGW parses the sshgw config.
func LoadSSHGW() (*SSHGW, error) {
	c, err := load(&SSHGW{})
	if err != nil {
		return nil, err
	}
	c.fill()
	return c, nil
}

func (s *Shared) fill() {
	if s.AppDomain == "" {
		s.AppDomain = "app." + s.Domain
	}
	if s.VPSSuffix == "" {
		s.VPSSuffix = "vps." + s.Domain
	}
}
