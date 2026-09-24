package api

import (
	"context"
	"net/http"

	"github.com/anshiq/ideserver/internal/db"
	"github.com/anshiq/ideserver/internal/jobs"
	"github.com/anshiq/ideserver/internal/netalloc"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type firewallOutput struct {
	Body []db.FirewallRules `json:"body"`
}

type firewallPutInput struct {
	ID   string          `path:"id"`
	Body []FirewallInput `json:"body"`
}

type portsOutput struct {
	Body []db.PortMappings `json:"body"`
}

type createPortInput struct {
	ID   string `path:"id"`
	Body struct {
		TargetPort int    `json:"targetPort"`
		Proto      string `json:"proto"`
		PublicPort int    `json:"publicPort,omitempty"`
	} `json:"body"`
}

type deletePortInput struct {
	ID     string `path:"id"`
	PortID int64  `path:"portId"`
}

type sshConfigOutput struct {
	Body struct {
		Config string `json:"config"`
	} `json:"body"`
}

type statsOutput struct {
	Body struct {
		ID       string `json:"id"`
		State    string `json:"state"`
		VCPU     int32  `json:"vcpu"`
		MemoryMB int32  `json:"memory_mb"`
		DiskGB   int32  `json:"disk_gb"`
		Live     bool   `json:"live"`
	} `json:"body"`
}

func (s *Server) registerNetwork(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-firewall", Method: http.MethodGet, Path: "/api/v1/instances/{id}/firewall",
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*firewallOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := ValidateVPSID(in.ID); err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		if _, err := s.ownedOrAdmin(ctx, u, in.ID); err != nil {
			return nil, err
		}
		rules, err := s.Q.ListFirewall(ctx, in.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		out := &firewallOutput{}
		out.Body = orEmpty(rules)
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "put-firewall", Method: http.MethodPut, Path: "/api/v1/instances/{id}/firewall",
	}, func(ctx context.Context, in *firewallPutInput) (*firewallOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := ValidateVPSID(in.ID); err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		if _, err := s.ownedOrAdmin(ctx, u, in.ID); err != nil {
			return nil, err
		}
		rules, err := ValidateFirewall(in.Body)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := s.requireWired(); err != nil {
			return nil, err
		}
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `DELETE FROM firewall_rules WHERE instance_id=$1`, in.ID); err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		for _, r := range rules {
			if _, err := tx.Exec(ctx, `INSERT INTO firewall_rules (instance_id, proto, port_from, port_to, source_cidr) VALUES ($1,$2,$3,$4,$5)`,
				in.ID, string(r.Proto), r.From, r.To, r.CIDR.String()); err != nil {
				return nil, huma.Error500InternalServerError("db error")
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO audit_log (actor_id, instance_id, action, meta, ip) VALUES ($1,$2,'firewall.replace','{}','')`, u.ID, in.ID); err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		if _, err := s.River.InsertTx(ctx, tx, jobs.SyncFirewallArgs{InstanceID: in.ID}, &river.InsertOpts{Queue: jobs.QueueDefault}); err != nil {
			return nil, huma.Error500InternalServerError("enqueue failed")
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		rows, _ := s.Q.ListFirewall(ctx, in.ID)
		out := &firewallOutput{Body: orEmpty(rows)}
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "list-ports", Method: http.MethodGet, Path: "/api/v1/instances/{id}/ports",
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*portsOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := ValidateVPSID(in.ID); err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		if _, err := s.ownedOrAdmin(ctx, u, in.ID); err != nil {
			return nil, err
		}
		maps, err := s.Q.ListPortMappings(ctx, in.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		out := &portsOutput{}
		out.Body = orEmpty(maps)
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "create-port", Method: http.MethodPost, Path: "/api/v1/instances/{id}/ports",
	}, func(ctx context.Context, in *createPortInput) (*portsOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := ValidateVPSID(in.ID); err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		if _, err := s.ownedOrAdmin(ctx, u, in.ID); err != nil {
			return nil, err
		}
		if err := ValidatePortMapping(in.Body.TargetPort, in.Body.Proto); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := s.requireWired(); err != nil {
			return nil, err
		}
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		defer tx.Rollback(ctx)
		var qMax, qUsed int32
		if err := tx.QueryRow(ctx, `SELECT max_mapped_ports FROM quotas WHERE user_id=$1`, u.ID).Scan(&qMax); err != nil {
			return nil, huma.Error500InternalServerError("quota missing")
		}
		if err := tx.QueryRow(ctx, `SELECT COUNT(*)::int FROM port_mappings p JOIN instances i ON i.id=p.vps_id WHERE i.user_id=$1 AND i.deleted_at IS NULL`, u.ID).Scan(&qUsed); err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		if int(qUsed)+1 > int(qMax) {
			return nil, huma.Error403Forbidden("mapped port quota exceeded")
		}
		public := in.Body.PublicPort
		if public == 0 {
			rows, err := tx.Query(ctx, `SELECT public_port FROM port_mappings`)
			if err != nil {
				return nil, huma.Error500InternalServerError("db error")
			}
			var used []int32
			for rows.Next() {
				var p int32
				_ = rows.Scan(&p)
				used = append(used, p)
			}
			rows.Close()
			public, err = netalloc.AllocatePort(used)
			if err != nil {
				return nil, huma.Error403Forbidden(err.Error())
			}
		} else if !netalloc.ValidPublicPort(public) {
			return nil, huma.Error400BadRequest("public port must be 30000-39999")
		}
		proto := in.Body.Proto
		if proto == "" {
			proto = "tcp"
		}
		var newID int64
		if err := tx.QueryRow(ctx, `INSERT INTO port_mappings (vps_id, public_port, target_port, proto, kind) VALUES ($1,$2,$3,$4,'custom') RETURNING id`,
			in.ID, public, in.Body.TargetPort, proto).Scan(&newID); err != nil {
			if isUniqueViolation(err) {
				return nil, huma.Error409Conflict("public port taken")
			}
			return nil, huma.Error500InternalServerError("db error")
		}
		_ = newID
		if _, err := tx.Exec(ctx, `INSERT INTO audit_log (actor_id, instance_id, action, meta, ip) VALUES ($1,$2,'ports.create','{}','')`, u.ID, in.ID); err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		if _, err := s.River.InsertTx(ctx, tx, jobs.SyncPortsArgs{InstanceID: in.ID}, &river.InsertOpts{Queue: jobs.QueueDefault}); err != nil {
			return nil, huma.Error500InternalServerError("enqueue failed")
		}
		if err := tx.Commit(ctx); err != nil {
			if isUniqueViolation(err) {
				return nil, huma.Error409Conflict("public port taken")
			}
			return nil, huma.Error500InternalServerError("db error")
		}
		rows, _ := s.Q.ListPortMappings(ctx, in.ID)
		out := &portsOutput{Body: orEmpty(rows)}
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "delete-port", Method: http.MethodDelete, Path: "/api/v1/instances/{id}/ports/{portId}",
	}, func(ctx context.Context, in *deletePortInput) (*struct{}, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := ValidateVPSID(in.ID); err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		if _, err := s.ownedOrAdmin(ctx, u, in.ID); err != nil {
			return nil, err
		}
		if err := s.requireWired(); err != nil {
			return nil, err
		}
		maps, err := s.Q.ListPortMappings(ctx, in.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		var target *db.PortMappings
		for i, m := range maps {
			if m.ID == in.PortID {
				target = &maps[i]
				break
			}
		}
		if target == nil {
			return nil, huma.Error404NotFound("not found")
		}
		if target.Kind == "ssh-fallback" {
			return nil, huma.Error400BadRequest("cannot delete the SSH fallback port")
		}
		n, err := s.Q.DeletePortMapping(ctx, db.DeletePortMappingParams{ID: in.PortID, VpsID: in.ID})
		if err != nil || n == 0 {
			return nil, huma.Error404NotFound("not found")
		}
		s.audit(ctx, u.ID, in.ID, "ports.delete", "", nil)
		if _, err := s.River.Insert(ctx, jobs.DropForwardArgs{PublicPort: int(target.PublicPort)}, &river.InsertOpts{Queue: jobs.QueueDefault}); err != nil {
			return nil, huma.Error500InternalServerError("enqueue failed")
		}
		return nil, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "get-ssh-config", Method: http.MethodGet, Path: "/api/v1/instances/{id}/ssh-config",
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*sshConfigOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := ValidateVPSID(in.ID); err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		inst, err := s.ownedOrAdmin(ctx, u, in.ID)
		if err != nil {
			return nil, err
		}
		img, err := s.Q.GetImage(ctx, inst.ImageID)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		maps, _ := s.Q.ListPortMappings(ctx, in.ID)
		fallback := 0
		for _, m := range maps {
			if m.Kind == "ssh-fallback" {
				fallback = int(m.PublicPort)
			}
		}
		out := &sshConfigOutput{}
		out.Body.Config = BuildSSHConfig(inst.ID, s.Cfg.VPSSuffix, s.Cfg.VPSSuffix, img.DefaultUser, fallback, inst.HostKeyFps)
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "get-stats", Method: http.MethodGet, Path: "/api/v1/instances/{id}/stats",
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*statsOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := ValidateVPSID(in.ID); err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		inst, err := s.ownedOrAdmin(ctx, u, in.ID)
		if err != nil {
			return nil, err
		}
		// api never touches Incus (plan §1.1): stats served here are the
		// recorded spec. Live per-instance stats arrive via the hostd stats
		// endpoint (Phase 6); `live:false` marks the source honestly.
		out := &statsOutput{}
		out.Body.ID = inst.ID
		out.Body.State = inst.State
		out.Body.VCPU = inst.Vcpu
		out.Body.MemoryMB = inst.MemoryMb
		out.Body.DiskGB = inst.DiskGb
		out.Body.Live = false
		return out, nil
	})
}

var _ = pgx.ErrNoRows
