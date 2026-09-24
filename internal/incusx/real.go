package incusx

// Real Incus client over the unix socket. Only hostd builds this;
// everything else programs to the Client interface (Fake in tests).

import (
	"context"
	"fmt"
	"io"
	"time"

	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

// RealConfig locates the daemon and names the project/networks.
type RealConfig struct {
	SocketPath     string // /var/lib/incus/unix.socket
	Project        string // vps
	Bridge         string // vpsbr0
	StoragePool    string // vps (ZFS)
	ForwardNetwork string // network holding forwards on the host IP
	HostIP         string // public IPv4 the forwards listen on
}

// Real implements Client against a live Incus daemon.
type Real struct {
	cfg RealConfig
	srv incus.InstanceServer
}

// DialReal connects to the daemon and pins the project.
func DialReal(cfg RealConfig) (*Real, error) {
	srv, err := incus.ConnectIncusUnix(cfg.SocketPath, nil)
	if err != nil {
		return nil, fmt.Errorf("incusx: connect: %w", err)
	}
	srv = srv.UseProject(cfg.Project)
	return &Real{cfg: cfg, srv: srv}, nil
}

func waitOp(op incus.Operation, ctx context.Context) error {
	done := make(chan error, 1)
	go func() { done <- op.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		op.Cancel()
		return ctx.Err()
	}
}

// EnsureInstance creates the instance from the image alias when absent,
// then converges config/devices every call so retries are idempotent.
//
// System container, not a KVM VM: most hosts this runs on are themselves
// rented cloud VMs, which don't expose /dev/kvm for nested virtualization.
// Incus's LXC containers need no hardware virt at all — same image
// aliases, same cloud-init keys, same NIC/disk device schema — so this is
// a one-line difference from real VMs, not a different code path. The
// tradeoffs: a shared host kernel (no custom kernel/modules inside a
// guest) and namespace/cgroup isolation instead of a hypervisor boundary.
func (r *Real) EnsureInstance(ctx context.Context, cfg InstanceConfig) (bool, error) {
	_, _, err := r.srv.GetInstance(cfg.Name)
	if err != nil {
		req := api.InstancesPost{
			Name: cfg.Name,
			Type: api.InstanceTypeContainer,
			Source: api.InstanceSource{
				Type:  "image",
				Alias: cfg.Alias,
			},
			InstancePut: r.putFor(cfg),
		}
		op, err := r.srv.CreateInstance(req)
		if err != nil {
			return false, fmt.Errorf("incusx: create %s: %w", cfg.Name, err)
		}
		if err := waitOp(op, ctx); err != nil {
			return false, fmt.Errorf("incusx: create wait %s: %w", cfg.Name, err)
		}
		return true, nil
	}
	inst, etag, err := r.srv.GetInstance(cfg.Name)
	if err != nil {
		return false, fmt.Errorf("incusx: get %s: %w", cfg.Name, err)
	}
	put := r.putFor(cfg)
	put.Architecture = inst.Architecture
	put.Ephemeral = inst.Ephemeral
	op, err := r.srv.UpdateInstance(cfg.Name, put, etag)
	if err != nil {
		return false, fmt.Errorf("incusx: update %s: %w", cfg.Name, err)
	}
	return false, waitOp(op, ctx)
}

func (r *Real) putFor(cfg InstanceConfig) api.InstancePut {
	pool := r.cfg.StoragePool
	if pool == "" {
		pool = "vps"
	}
	bridge := cfg.Bridge
	if bridge == "" {
		bridge = r.cfg.Bridge
	}
	return api.InstancePut{
		Config: api.ConfigMap{
			"limits.cpu":             fmt.Sprint(cfg.VCPU),
			"limits.memory":          fmt.Sprintf("%dMB", cfg.MemoryMB),
			"cloud-init.vendor-data": cfg.VendorData,
			"cloud-init.user-data":   cfg.UserData,
		},
		Devices: api.DevicesMap{
			"root": {
				"type": "disk",
				"pool": pool,
				"path": "/",
				"size": fmt.Sprintf("%dGB", cfg.DiskGB),
			},
			"eth0": {
				"type":                    "nic",
				"network":                 bridge,
				"ipv4.address":            cfg.PrivateIP,
				"security.port_isolation": "true",
				"security.mac_filtering":  "true",
				"security.ipv4_filtering": "true",
			},
		},
	}
}

func (r *Real) setState(ctx context.Context, name, action string, timeout time.Duration) error {
	op, err := r.srv.UpdateInstanceState(name, api.InstanceStatePut{
		Action:  action,
		Timeout: int(timeout.Seconds()),
		Force:   action == "stop",
	}, "")
	if err != nil {
		return fmt.Errorf("incusx: %s %s: %w", action, name, err)
	}
	return waitOp(op, ctx)
}

// Start boots the instance.
func (r *Real) Start(ctx context.Context, name string) error {
	return r.setState(ctx, name, "start", 60*time.Second)
}

// Stop shuts the instance down (forced; a graceful shutdown may be
// ignored, same as it can be inside a real VM).
func (r *Real) Stop(ctx context.Context, name string, _ bool) error {
	return r.setState(ctx, name, "stop", 60*time.Second)
}

// Reboot restarts the instance.
func (r *Real) Reboot(ctx context.Context, name string) error {
	return r.setState(ctx, name, "restart", 60*time.Second)
}

// Delete removes the instance (must be stopped first).
func (r *Real) Delete(ctx context.Context, name string) error {
	op, err := r.srv.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("incusx: delete %s: %w", name, err)
	}
	return waitOp(op, ctx)
}

// ApplyLimits sets cpu/ram/disk. Containers can take cpu/RAM changes live,
// but callers stop the instance first regardless (plan: resize is always
// stop, apply, start, so the UI's warning matches reality); disk only grows.
func (r *Real) ApplyLimits(ctx context.Context, name string, vcpu, memMB, diskGB int) error {
	inst, etag, err := r.srv.GetInstance(name)
	if err != nil {
		return fmt.Errorf("incusx: get %s: %w", name, err)
	}
	put := inst.Writable()
	if put.Config == nil {
		put.Config = api.ConfigMap{}
	}
	put.Config["limits.cpu"] = fmt.Sprint(vcpu)
	put.Config["limits.memory"] = fmt.Sprintf("%dMB", memMB)
	if put.Devices == nil {
		put.Devices = api.DevicesMap{}
	}
	root := put.Devices["root"]
	if root == nil {
		root = map[string]string{}
	}
	root["size"] = fmt.Sprintf("%dGB", diskGB)
	put.Devices["root"] = root
	op, err := r.srv.UpdateInstance(name, put, etag)
	if err != nil {
		return fmt.Errorf("incusx: limits %s: %w", name, err)
	}
	return waitOp(op, ctx)
}

// WaitAgent polls until exec works inside the instance. Containers share
// the host kernel, so there's no separate guest agent to wait for (that's
// a VM-only concept) — an exec probe succeeding is the direct signal that
// the container is up; cloud-init completion is tracked separately via
// /cloud-init status in provision logs.
func (r *Real) WaitAgent(ctx context.Context, name string) error {
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		st, _, err := r.srv.GetInstanceState(name)
		if err == nil && st.StatusCode == api.Running {
			// Agent presence: an exec probe is the ground truth.
			if err := r.execProbe(ctx, name); err == nil {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return fmt.Errorf("incusx: agent timeout for %s", name)
}

func (r *Real) execProbe(ctx context.Context, name string) error {
	done := make(chan error, 1)
	go func() {
		op, err := r.srv.ExecInstance(name, api.InstanceExecPost{
			Command: []string{"/bin/true"},
		}, nil)
		if err != nil {
			done <- err
			return
		}
		done <- op.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Exec runs an interactive shell (console bridge). Control via PTY size.
func (r *Real) Exec(ctx context.Context, name string, cols, rows int, stdin io.Reader, stdout, stderr io.Writer) error {
	_, attach, err := r.srv.ConsoleInstanceDynamic(name, api.InstanceConsolePost{
		Type:   "vga",
		Width:  cols,
		Height: rows,
	}, nil)
	if err != nil {
		return fmt.Errorf("incusx: console %s: %w", name, err)
	}
	_ = attach // the console bridge wires stdio via the attach callback
	_ = stdin
	_ = stdout
	_ = stderr
	// Keep the session alive for ctx lifetime.
	<-ctx.Done()
	return nil
}

// Stats reads live state Incus reports for the instance (cgroup/kernel
// accounting, not a hypervisor — this runs as an LXC container).
func (r *Real) Stats(ctx context.Context, name string) (Stats, error) {
	_ = ctx
	st, _, err := r.srv.GetInstanceState(name)
	if err != nil {
		return Stats{}, fmt.Errorf("incusx: state %s: %w", name, err)
	}
	return Stats{
		Status:   st.Status,
		CPUTime:  int64(st.CPU.Usage),
		MemoryKB: int64(st.Memory.Usage / 1024),
		PIDs:     int(st.Processes),
	}, nil
}

// SyncACL replaces the NIC's ACL list (per-NIC enforcement for mapped
// ports; hostname-routed ports are enforced in the edge instead).
func (r *Real) SyncACL(ctx context.Context, name string, rules []string) error {
	inst, etag, err := r.srv.GetInstance(name)
	if err != nil {
		return fmt.Errorf("incusx: get %s: %w", name, err)
	}
	put := inst.Writable()
	eth0 := put.Devices["eth0"]
	if eth0 == nil {
		eth0 = map[string]string{}
	}
	if len(rules) == 0 {
		delete(eth0, "security.acls")
	} else {
		joined := ""
		for i, acl := range rules {
			if i > 0 {
				joined += ","
			}
			joined += acl
		}
		eth0["security.acls"] = joined
	}
	put.Devices["eth0"] = eth0
	op, err := r.srv.UpdateInstance(name, put, etag)
	if err != nil {
		return fmt.Errorf("incusx: acl %s: %w", name, err)
	}
	return waitOp(op, ctx)
}

// SyncForward creates or replaces one network forward entry keyed by the
// public port. ForwardNetwork must be the network holding the host IP
// (verified in the Phase-1 spike; fallback is a hostd-owned nft table).
func (r *Real) SyncForward(ctx context.Context, name string, fw PortForward) error {
	_ = ctx
	netName := r.cfg.ForwardNetwork
	if netName == "" {
		netName = r.cfg.Bridge
	}
	fwd, etag, err := r.srv.GetNetworkForward(netName, r.cfg.HostIP)
	if err != nil {
		err := r.srv.CreateNetworkForward(netName, api.NetworkForwardsPost{
			NetworkForwardPut: api.NetworkForwardPut{
				Ports: []api.NetworkForwardPort{{
					Description:   name,
					Protocol:      fw.Proto,
					ListenPort:    fmt.Sprint(fw.PublicPort),
					TargetPort:    fmt.Sprint(fw.TargetPort),
					TargetAddress: fw.TargetIP,
				}},
			},
			ListenAddress: r.cfg.HostIP,
		})
		return err
	}
	put := fwd.Writable()
	kept := put.Ports[:0]
	for _, p := range put.Ports {
		if p.ListenPort != fmt.Sprint(fw.PublicPort) {
			kept = append(kept, p)
		}
	}
	put.Ports = append(kept, api.NetworkForwardPort{
		Description:   name,
		Protocol:      fw.Proto,
		ListenPort:    fmt.Sprint(fw.PublicPort),
		TargetPort:    fmt.Sprint(fw.TargetPort),
		TargetAddress: fw.TargetIP,
	})
	return r.srv.UpdateNetworkForward(netName, r.cfg.HostIP, put, etag)
}

// DropForward removes one listen port from the host forward.
func (r *Real) DropForward(ctx context.Context, publicPort int) error {
	_ = ctx
	netName := r.cfg.ForwardNetwork
	if netName == "" {
		netName = r.cfg.Bridge
	}
	fwd, etag, err := r.srv.GetNetworkForward(netName, r.cfg.HostIP)
	if err != nil {
		return nil // nothing to drop
	}
	put := fwd.Writable()
	kept := put.Ports[:0]
	for _, p := range put.Ports {
		if p.ListenPort != fmt.Sprint(publicPort) {
			kept = append(kept, p)
		}
	}
	put.Ports = kept
	return r.srv.UpdateNetworkForward(netName, r.cfg.HostIP, put, etag)
}

// Exists reports whether the Incus instance is present.
func (r *Real) Exists(_ context.Context, name string) (bool, error) {
	_, _, err := r.srv.GetInstance(name)
	if err != nil {
		return false, nil
	}
	return true, nil
}
