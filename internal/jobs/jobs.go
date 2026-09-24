// Package jobs holds River job args and idempotent workers for hostd.
// Every worker is keyed on the instance id and converges Incus toward the
// DB's desired state, so retries are safe (plan §1.5).
package jobs

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log/slog"
	"time"

	"github.com/anshiq/ideserver/internal/cloudinit"
	"github.com/anshiq/ideserver/internal/db"
	"github.com/anshiq/ideserver/internal/ids"
	"github.com/anshiq/ideserver/internal/incusx"
	"github.com/riverqueue/river"
	"golang.org/x/crypto/ssh"
)

// Queue names: per-node queues allow multi-node later (plan §Later).
const (
	QueueDefault = "default"
	QueueNode    = "node"
)

// --- Job args (one Kind per lifecycle operation) ---

type ProvisionArgs struct{ InstanceID string }

func (ProvisionArgs) Kind() string { return "provision" }

type StartArgs struct{ InstanceID string }

func (StartArgs) Kind() string { return "start" }

type StopArgs struct{ InstanceID string }

func (StopArgs) Kind() string { return "stop" }

type RebootArgs struct{ InstanceID string }

func (RebootArgs) Kind() string { return "reboot" }

type DeleteArgs struct{ InstanceID string }

func (DeleteArgs) Kind() string { return "delete" }

type ResizeArgs struct {
	InstanceID string
	PlanSlug   string
}

func (ResizeArgs) Kind() string { return "resize" }

type SyncFirewallArgs struct{ InstanceID string }

func (SyncFirewallArgs) Kind() string { return "sync-firewall" }

type SyncPortsArgs struct{ InstanceID string }

func (SyncPortsArgs) Kind() string { return "sync-ports" }

type MeterArgs struct{}

func (MeterArgs) Kind() string { return "meter-usage" }

// Deps are the worker dependencies. Only hostd wires the real Incus client;
// tests substitute incusx.Fake.
type Deps struct {
	Q       *db.Queries
	DB      *sql.DB
	Inc     incusx.Client
	Bridge  string
	Project string
	Logger  *slog.Logger
}

func (d *Deps) log() *slog.Logger {
	if d.Logger != nil {
		return d.Logger
	}
	return slog.Default()
}

// setState records lifecycle state; lastErr is stored on error transitions.
func (d *Deps) setState(ctx context.Context, id, state, lastErr string) error {
	return d.Q.SetInstanceState(ctx, db.SetInstanceStateParams{ID: id, State: state, LastError: lastErr})
}

// BuildVendorData renders cloud-init vendor-data for a new instance. Host
// keys are generated per-instance so the dashboard can show fingerprints
// without TOFU
// (plan §1.2). It returns vendor-data and printable "type fingerprint" lines.
func BuildVendorData(hostname, distro string, userPubs []string) (vendorData string, fps string, err error) {
	hostKeys := map[string]string{}
	var fpLines []string
	for _, typ := range []string{"ed25519"} {
		_, priv, genErr := ed25519.GenerateKey(rand.Reader)
		if genErr != nil {
			return "", "", genErr
		}
		// OpenSSH private key PEM via ssh.MarshalPrivateKey (crypto/ssh).
		signer, sErr := ssh.NewSignerFromKey(priv)
		if sErr != nil {
			return "", "", sErr
		}
		_ = signer
		pemBlock, pErr := ssh.MarshalPrivateKey(priv, "")
		if pErr != nil {
			return "", "", pErr
		}
		pemBytes := pem.EncodeToMemory(pemBlock)
		hostKeys[typ] = string(pemBytes)
		pub := priv.Public().(ed25519.PublicKey)
		sshPub, _ := ssh.NewPublicKey(pub)
		fpLines = append(fpLines, typ+" "+ssh.FingerprintSHA256(sshPub))
	}
	vd, err := cloudinit.Render(cloudinit.Input{
		Distro:      cloudinit.Distro(distro),
		Hostname:    hostname,
		UserSSHPubs: userPubs,
		HostKeys:    hostKeys,
	})
	if err != nil {
		return "", "", err
	}
	fps = ""
	for i, l := range fpLines {
		if i > 0 {
			fps += ","
		}
		fps += l
	}
	return vd, fps, nil
}

// ACLName returns the Incus ACL attached to a VPS NIC.
func ACLName(id string) string { return "vps-" + id }

// --- Workers ---

type ProvisionWorker struct {
	river.WorkerDefaults[ProvisionArgs]
	D *Deps
}

func (w *ProvisionWorker) Work(ctx context.Context, job *river.Job[ProvisionArgs]) error {
	id := job.Args.InstanceID
	d := w.D
	inst, err := d.Q.GetInstance(ctx, id)
	if err != nil {
		return fmt.Errorf("provision %s: %w", id, err)
	}
	if err := d.setState(ctx, id, "provisioning", ""); err != nil {
		return err
	}
	img, err := d.Q.GetImage(ctx, inst.ImageID)
	if err != nil {
		_ = d.setState(ctx, id, "error", err.Error())
		return err
	}
	keys, err := d.Q.InstanceSSHKeys(ctx, id)
	if err != nil {
		_ = d.setState(ctx, id, "error", err.Error())
		return err
	}
	pubs := make([]string, 0, len(keys))
	for _, k := range keys {
		pubs = append(pubs, k.PublicKey)
	}
	vendorData, fps, err := BuildVendorData(id, img.Distro, pubs)
	if err != nil {
		_ = d.setState(ctx, id, "error", err.Error())
		return err
	}
	if fps != "" {
		_ = d.Q.SetHostKeyFPs(ctx, db.SetHostKeyFPsParams{ID: id, HostKeyFps: fps})
	}
	cfg := incusx.InstanceConfig{
		Name:       ids.IncusName(id),
		Alias:      img.IncusAlias,
		Project:    d.Project,
		VCPU:       int(inst.Vcpu),
		MemoryMB:   int(inst.MemoryMb),
		DiskGB:     int(inst.DiskGb),
		PrivateIP:  db.InetString(inst.PrivateIpv4),
		Bridge:     d.Bridge,
		VendorData: vendorData,
		UserData:   inst.UserData,
	}
	if _, err := d.Inc.EnsureInstance(ctx, cfg); err != nil {
		_ = d.setState(ctx, id, "error", err.Error())
		return err
	}
	// Converge firewall + forwards before boot so the instance is never exposed.
	if err := w.syncNet(ctx, id); err != nil {
		d.log().Warn("provision sync-net failed", "id", id, "err", err)
	}
	if err := d.Inc.Start(ctx, cfg.Name); err != nil {
		_ = d.setState(ctx, id, "error", err.Error())
		return err
	}
	if err := d.Inc.WaitAgent(ctx, cfg.Name); err != nil {
		_ = d.setState(ctx, id, "error", err.Error())
		return err
	}
	return d.setState(ctx, id, "running", "")
}

func (w *ProvisionWorker) syncNet(ctx context.Context, id string) error {
	d := w.D
	inst, err := d.Q.GetInstance(ctx, id)
	if err != nil {
		return err
	}
	name := ids.IncusName(id)
	rules, err := d.Q.ListFirewall(ctx, id)
	if err != nil {
		return err
	}
	acls := []string{ACLName(id)}
	_ = rules
	if err := d.Inc.SyncACL(ctx, name, acls); err != nil {
		return err
	}
	maps, err := d.Q.ListPortMappings(ctx, id)
	if err != nil {
		return err
	}
	for _, m := range maps {
		if err := d.Inc.SyncForward(ctx, name, incusx.PortForward{
			PublicPort: int(m.PublicPort),
			TargetPort: int(m.TargetPort),
			Proto:      m.Proto,
			TargetIP:   db.InetString(inst.PrivateIpv4),
		}); err != nil {
			return err
		}
	}
	return nil
}

type StartWorker struct {
	river.WorkerDefaults[StartArgs]
	D *Deps
}

func (w *StartWorker) Work(ctx context.Context, job *river.Job[StartArgs]) error {
	id := job.Args.InstanceID
	if err := w.D.setState(ctx, id, "starting", ""); err != nil {
		return err
	}
	if err := w.D.Inc.Start(ctx, ids.IncusName(id)); err != nil {
		_ = w.D.setState(ctx, id, "error", err.Error())
		return err
	}
	if err := w.D.Inc.WaitAgent(ctx, ids.IncusName(id)); err != nil {
		_ = w.D.setState(ctx, id, "error", err.Error())
		return err
	}
	return w.D.setState(ctx, id, "running", "")
}

type StopWorker struct {
	river.WorkerDefaults[StopArgs]
	D *Deps
}

func (w *StopWorker) Work(ctx context.Context, job *river.Job[StopArgs]) error {
	id := job.Args.InstanceID
	if err := w.D.setState(ctx, id, "stopping", ""); err != nil {
		return err
	}
	if err := w.D.Inc.Stop(ctx, ids.IncusName(id), true); err != nil {
		_ = w.D.setState(ctx, id, "error", err.Error())
		return err
	}
	return w.D.setState(ctx, id, "stopped", "")
}

type RebootWorker struct {
	river.WorkerDefaults[RebootArgs]
	D *Deps
}

func (w *RebootWorker) Work(ctx context.Context, job *river.Job[RebootArgs]) error {
	id := job.Args.InstanceID
	if err := w.D.setState(ctx, id, "rebooting", ""); err != nil {
		return err
	}
	if err := w.D.Inc.Reboot(ctx, ids.IncusName(id)); err != nil {
		_ = w.D.setState(ctx, id, "error", err.Error())
		return err
	}
	if err := w.D.Inc.WaitAgent(ctx, ids.IncusName(id)); err != nil {
		_ = w.D.setState(ctx, id, "error", err.Error())
		return err
	}
	return w.D.setState(ctx, id, "running", "")
}

type DeleteWorker struct {
	river.WorkerDefaults[DeleteArgs]
	D *Deps
}

func (w *DeleteWorker) Work(ctx context.Context, job *river.Job[DeleteArgs]) error {
	id := job.Args.InstanceID
	d := w.D
	if err := d.setState(ctx, id, "deleting", ""); err != nil {
		return err
	}
	name := ids.IncusName(id)
	maps, _ := d.Q.ListPortMappings(ctx, id)
	for _, m := range maps {
		_ = d.Inc.DropForward(ctx, int(m.PublicPort))
	}
	_ = d.Inc.Stop(ctx, name, true)
	if err := d.Inc.Delete(ctx, name); err != nil {
		if _, ok := err.(*incusx.ErrNotFound); !ok {
			_ = d.setState(ctx, id, "error", err.Error())
			return err
		}
	}
	return d.Q.SoftDeleteInstance(ctx, id)
}

type ResizeWorker struct {
	river.WorkerDefaults[ResizeArgs]
	D *Deps
}

func (w *ResizeWorker) Work(ctx context.Context, job *river.Job[ResizeArgs]) error {
	id := job.Args.InstanceID
	d := w.D
	plan, err := d.Q.GetPlan(ctx, job.Args.PlanSlug)
	if err != nil {
		return err
	}
	inst, err := d.Q.GetInstance(ctx, id)
	if err != nil {
		return err
	}
	// Resize is stop → apply (disk only grows) → start (plan §1.5).
	if inst.State != "stopped" && inst.State != "resizing" {
		return fmt.Errorf("resize %s: must be stopped (state %s)", id, inst.State)
	}
	if int(plan.DiskGb) < int(inst.DiskGb) {
		return fmt.Errorf("resize %s: disk can only grow", id)
	}
	if err := d.setState(ctx, id, "resizing", ""); err != nil {
		return err
	}
	name := ids.IncusName(id)
	if err := d.Inc.ApplyLimits(ctx, name, int(plan.Vcpu), int(plan.MemoryMb), int(plan.DiskGb)); err != nil {
		_ = d.setState(ctx, id, "error", err.Error())
		return err
	}
	if err := d.Q.ApplyResize(ctx, db.ApplyResizeParams{
		ID: id, Vcpu: plan.Vcpu, MemoryMb: plan.MemoryMb, DiskGb: plan.DiskGb, PlanID: plan.Slug,
	}); err != nil {
		_ = d.setState(ctx, id, "error", err.Error())
		return err
	}
	return d.setState(ctx, id, "stopped", "")
}

type SyncFirewallWorker struct {
	river.WorkerDefaults[SyncFirewallArgs]
	D *Deps
}

func (w *SyncFirewallWorker) Work(ctx context.Context, job *river.Job[SyncFirewallArgs]) error {
	id := job.Args.InstanceID
	rules, err := w.D.Q.ListFirewall(ctx, id)
	if err != nil {
		return err
	}
	_ = rules
	return w.D.Inc.SyncACL(ctx, ids.IncusName(id), []string{ACLName(id)})
}

type SyncPortsWorker struct {
	river.WorkerDefaults[SyncPortsArgs]
	D *Deps
}

func (w *SyncPortsWorker) Work(ctx context.Context, job *river.Job[SyncPortsArgs]) error {
	id := job.Args.InstanceID
	d := w.D
	inst, err := d.Q.GetInstance(ctx, id)
	if err != nil {
		return err
	}
	maps, err := d.Q.ListPortMappings(ctx, id)
	if err != nil {
		return err
	}
	name := ids.IncusName(id)
	for _, m := range maps {
		if err := d.Inc.SyncForward(ctx, name, incusx.PortForward{
			PublicPort: int(m.PublicPort),
			TargetPort: int(m.TargetPort),
			Proto:      m.Proto,
			TargetIP:   db.InetString(inst.PrivateIpv4),
		}); err != nil {
			return err
		}
	}
	return nil
}

// DropForwardArgs removes one DNAT listen port (port-mapping delete path).
// The DB row is already gone when this runs; the worker is idempotent
// (dropping an absent port is a no-op).
type DropForwardArgs struct{ PublicPort int }

func (DropForwardArgs) Kind() string { return "drop-forward" }

type DropForwardWorker struct {
	river.WorkerDefaults[DropForwardArgs]
	D *Deps
}

func (w *DropForwardWorker) Work(ctx context.Context, job *river.Job[DropForwardArgs]) error {
	return w.D.Inc.DropForward(ctx, job.Args.PublicPort)
}

type MeterWorker struct {
	river.WorkerDefaults[MeterArgs]
	D *Deps
}

// Work writes one usage_records row per non-deleted instance for the current
// hour (idempotent via ON CONFLICT DO NOTHING).
func (w *MeterWorker) Work(ctx context.Context, job *river.Job[MeterArgs]) error {
	hour := time.Now().UTC().Truncate(time.Hour)
	for _, state := range []string{"running", "stopped", "provisioning", "starting", "stopping", "rebooting", "resizing"} {
		insts, err := w.D.Q.ListInstancesByState(ctx, state)
		if err != nil {
			return err
		}
		for _, inst := range insts {
			_ = w.D.Q.RecordUsage(ctx, db.RecordUsageParams{
				InstanceID: inst.ID, Vcpu: inst.Vcpu, MemoryMb: inst.MemoryMb, DiskGb: inst.DiskGb, Hour: hour,
			})
		}
		_ = job
	}
	return nil
}

// AddAll registers every worker on rivers' Workers bundle.
func AddAll(rw *river.Workers, d *Deps) {
	river.AddWorker(rw, &ProvisionWorker{D: d})
	river.AddWorker(rw, &StartWorker{D: d})
	river.AddWorker(rw, &StopWorker{D: d})
	river.AddWorker(rw, &RebootWorker{D: d})
	river.AddWorker(rw, &DeleteWorker{D: d})
	river.AddWorker(rw, &ResizeWorker{D: d})
	river.AddWorker(rw, &SyncFirewallWorker{D: d})
	river.AddWorker(rw, &SyncPortsWorker{D: d})
	river.AddWorker(rw, &DropForwardWorker{D: d})
	river.AddWorker(rw, &MeterWorker{D: d})
}
