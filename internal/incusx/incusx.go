// Package incusx is a thin, fakeable interface over the Incus Go client.
// hostd is the only binary that imports the real implementation; api and
// tests use Fake.
package incusx

import (
	"context"
	"fmt"
	"io"
	"net"
)

// InstanceConfig describes the desired Incus instance (an LXC system
// container, not a KVM VM — see EnsureInstance in real.go for why).
type InstanceConfig struct {
	Name       string // vps-{id}
	Alias      string // image alias, e.g. images:ubuntu/24.04/cloud
	Project    string
	VCPU       int
	MemoryMB   int
	DiskGB     int
	PrivateIP  string // pinned stable address
	Bridge     string // e.g. vpsbr0
	VendorData string
	UserData   string
}

// PortForward maps one public port to an instance port (DNAT network
// forward).
type PortForward struct {
	PublicPort int
	TargetPort int
	Proto      string // tcp|udp
	TargetIP   string
}

// Stats is live data from the Incus state endpoint.
type Stats struct {
	Status   string
	CPUTime  int64
	MemoryKB int64
	PIDs     int
}

// Client is the surface hostd needs. Implementations: Real (Incus API)
// and Fake (tests, seed scripts).
type Client interface {
	EnsureInstance(ctx context.Context, cfg InstanceConfig) (created bool, err error)
	Start(ctx context.Context, name string) error
	Stop(ctx context.Context, name string, force bool) error
	Reboot(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
	ApplyLimits(ctx context.Context, name string, vcpu, memMB, diskGB int) error
	WaitAgent(ctx context.Context, name string) error
	Exec(ctx context.Context, name string, cols, rows int, stdin io.Reader, stdout, stderr io.Writer) error
	Stats(ctx context.Context, name string) (Stats, error)
	SyncACL(ctx context.Context, name string, rules []string) error
	SyncForward(ctx context.Context, name string, fw PortForward) error
	DropForward(ctx context.Context, publicPort int) error
	Exists(ctx context.Context, name string) (bool, error)
}

// ErrNotFound is returned when the Incus instance is absent.
type ErrNotFound struct{ Name string }

func (e *ErrNotFound) Error() string { return fmt.Sprintf("incusx: instance %s not found", e.Name) }

// Fake is an in-memory Client for API integration tests.
type Fake struct {
	Instances map[string]InstanceConfig
	Running   map[string]bool
	Forwards  map[int]PortForward
	ACLs      map[string][]string
	FailOn    map[string]error
}

func NewFake() *Fake {
	return &Fake{
		Instances: map[string]InstanceConfig{},
		Running:   map[string]bool{},
		Forwards:  map[int]PortForward{},
		ACLs:      map[string][]string{},
		FailOn:    map[string]error{},
	}
}

func (f *Fake) fail(op string) error { return f.FailOn[op] }

func (f *Fake) EnsureInstance(_ context.Context, cfg InstanceConfig) (bool, error) {
	if err := f.fail("EnsureInstance"); err != nil {
		return false, err
	}
	_, ok := f.Instances[cfg.Name]
	f.Instances[cfg.Name] = cfg
	return !ok, nil
}

func (f *Fake) Start(_ context.Context, name string) error {
	if err := f.fail("Start"); err != nil {
		return err
	}
	if _, ok := f.Instances[name]; !ok {
		return &ErrNotFound{name}
	}
	f.Running[name] = true
	return nil
}

func (f *Fake) Stop(_ context.Context, name string, _ bool) error {
	if err := f.fail("Stop"); err != nil {
		return err
	}
	if _, ok := f.Instances[name]; !ok {
		return &ErrNotFound{name}
	}
	f.Running[name] = false
	return nil
}

func (f *Fake) Reboot(_ context.Context, name string) error {
	if err := f.fail("Reboot"); err != nil {
		return err
	}
	if _, ok := f.Instances[name]; !ok {
		return &ErrNotFound{name}
	}
	f.Running[name] = true
	return nil
}

func (f *Fake) Delete(_ context.Context, name string) error {
	if err := f.fail("Delete"); err != nil {
		return err
	}
	delete(f.Instances, name)
	delete(f.Running, name)
	return nil
}

func (f *Fake) ApplyLimits(_ context.Context, name string, vcpu, memMB, diskGB int) error {
	if err := f.fail("ApplyLimits"); err != nil {
		return err
	}
	cfg, ok := f.Instances[name]
	if !ok {
		return &ErrNotFound{name}
	}
	cfg.VCPU, cfg.MemoryMB, cfg.DiskGB = vcpu, memMB, diskGB
	f.Instances[name] = cfg
	return nil
}

func (f *Fake) WaitAgent(_ context.Context, name string) error {
	if err := f.fail("WaitAgent"); err != nil {
		return err
	}
	if _, ok := f.Instances[name]; !ok {
		return &ErrNotFound{name}
	}
	return nil
}

func (f *Fake) Exec(_ context.Context, name string, _, _ int, stdin io.Reader, stdout, _ io.Writer) error {
	if _, ok := f.Instances[name]; !ok {
		return &ErrNotFound{name}
	}
	if stdin != nil {
		_, _ = io.Copy(io.Discard, stdin)
	}
	if stdout != nil {
		_, _ = stdout.Write([]byte("$ "))
	}
	return nil
}

func (f *Fake) Stats(_ context.Context, name string) (Stats, error) {
	if _, ok := f.Instances[name]; !ok {
		return Stats{}, &ErrNotFound{name}
	}
	st := "stopped"
	if f.Running[name] {
		st = "running"
	}
	return Stats{Status: st}, nil
}

func (f *Fake) SyncACL(_ context.Context, name string, rules []string) error {
	if err := f.fail("SyncACL"); err != nil {
		return err
	}
	f.ACLs[name] = rules
	return nil
}

func (f *Fake) SyncForward(_ context.Context, name string, fw PortForward) error {
	if err := f.fail("SyncForward"); err != nil {
		return err
	}
	f.Forwards[fw.PublicPort] = fw
	return nil
}

func (f *Fake) DropForward(_ context.Context, publicPort int) error {
	delete(f.Forwards, publicPort)
	return nil
}

func (f *Fake) Exists(_ context.Context, name string) (bool, error) {
	_, ok := f.Instances[name]
	return ok, nil
}

// Dial is a helper for the sshgw jump path: plain TCP to the instance.
func Dial(_ context.Context, ip string, port int) (net.Conn, error) {
	return net.Dial("tcp", net.JoinHostPort(ip, fmt.Sprint(port)))
}
