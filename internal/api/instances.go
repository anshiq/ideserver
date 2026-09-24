package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/anshiq/ideserver/internal/db"
	"github.com/anshiq/ideserver/internal/ids"
	"github.com/anshiq/ideserver/internal/instances"
	"github.com/anshiq/ideserver/internal/jobs"
	"github.com/anshiq/ideserver/internal/netalloc"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/riverqueue/river"
)

// InstanceOut is the JSON shape for instances (IP as plain string).
type InstanceOut struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PlanID      string `json:"plan_id"`
	ImageID     string `json:"image_id"`
	VCPU        int32  `json:"vcpu"`
	MemoryMB    int32  `json:"memory_mb"`
	DiskGB      int32  `json:"disk_gb"`
	State       string `json:"state"`
	Desired     string `json:"desired_state"`
	PrivateIPv4 string `json:"private_ipv4"`
	HTTPSMode   string `json:"https_mode"`
	HostKeyFPs  string `json:"host_key_fps"`
	LastError   string `json:"last_error"`
}

func toOut(in db.Instances) InstanceOut {
	return InstanceOut{
		ID: in.ID, Name: in.Name, PlanID: in.PlanID, ImageID: in.ImageID,
		VCPU: in.Vcpu, MemoryMB: in.MemoryMb, DiskGB: in.DiskGb,
		State: in.State, Desired: in.DesiredState,
		PrivateIPv4: db.InetString(in.PrivateIpv4),
		HTTPSMode:   in.HttpsMode, HostKeyFPs: in.HostKeyFps, LastError: in.LastError,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Server) registerCatalog(api huma.API) {
	type plansOutput struct {
		Body []db.Plans `json:"body"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "list-plans", Method: http.MethodGet, Path: "/api/v1/plans",
	}, func(ctx context.Context, _ *struct{}) (*plansOutput, error) {
		plans, err := s.Q.ListPlans(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		out := &plansOutput{}
		out.Body = orEmpty(plans)
		return out, nil
	})
	type imagesOutput struct {
		Body []db.Images `json:"body"`
	}
	huma.Register(api, huma.Operation{
		OperationID: "list-images", Method: http.MethodGet, Path: "/api/v1/images",
	}, func(ctx context.Context, _ *struct{}) (*imagesOutput, error) {
		images, err := s.Q.ListImages(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		out := &imagesOutput{}
		out.Body = orEmpty(images)
		return out, nil
	})
}

type createInstanceInput struct {
	Body struct {
		Name     string  `json:"name" maxLength:"64"`
		Plan     string  `json:"plan"`
		Image    string  `json:"image"`
		SSHKeys  []int64 `json:"sshKeys"`
		UserData string  `json:"userData,omitempty" maxLength:"16384"`
	}
}

type instanceOutput struct {
	Body InstanceOut `json:"body"`
}

type listInstancesOutput struct {
	Body []InstanceOut `json:"body"`
}

type actionInput struct {
	ID   string `path:"id"`
	Body struct {
		Action string `json:"action" enum:"start,stop,reboot,resize"`
		Plan   string `json:"plan,omitempty"`
	} `json:"body"`
}

type patchInstanceInput struct {
	ID   string `path:"id"`
	Body struct {
		Name      string `json:"name,omitempty" maxLength:"64"`
		HTTPSMode string `json:"httpsMode,omitempty" enum:"terminate,passthrough"`
	} `json:"body"`
}

func (s *Server) registerInstances(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-instances", Method: http.MethodGet, Path: "/api/v1/instances",
	}, func(ctx context.Context, _ *struct{}) (*listInstancesOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		rows, err := s.Q.ListInstances(ctx, u.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		out := &listInstancesOutput{Body: []InstanceOut{}}
		for _, r := range rows {
			out.Body = append(out.Body, toOut(r))
		}
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "create-instance", Method: http.MethodPost, Path: "/api/v1/instances",
	}, func(ctx context.Context, in *createInstanceInput) (*instanceOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if err := ValidateInstanceName(in.Body.Name); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := s.requireWired(); err != nil {
			return nil, err
		}
		inst, err := s.createInstanceTx(ctx, u, in.Body.Name, in.Body.Plan, in.Body.Image, in.Body.SSHKeys, in.Body.UserData)
		if err != nil {
			var he huma.StatusError
			if errors.As(err, &he) {
				return nil, he
			}
			return nil, huma.Error500InternalServerError("create failed")
		}
		out := &instanceOutput{}
		out.Body = toOut(*inst)
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "get-instance", Method: http.MethodGet, Path: "/api/v1/instances/{id}",
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*instanceOutput, error) {
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
		out := &instanceOutput{}
		out.Body = toOut(*inst)
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "patch-instance", Method: http.MethodPatch, Path: "/api/v1/instances/{id}",
	}, func(ctx context.Context, in *patchInstanceInput) (*instanceOutput, error) {
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
		if in.Body.Name != "" {
			if err := ValidateInstanceName(in.Body.Name); err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			if err := s.Q.RenameInstance(ctx, db.RenameInstanceParams{ID: in.ID, Name: in.Body.Name}); err != nil {
				return nil, huma.Error500InternalServerError("rename failed")
			}
		}
		if in.Body.HTTPSMode != "" {
			if err := s.Q.SetHTTPSMode(ctx, db.SetHTTPSModeParams{ID: in.ID, HttpsMode: in.Body.HTTPSMode}); err != nil {
				return nil, huma.Error500InternalServerError("https-mode failed")
			}
		}
		s.audit(ctx, u.ID, in.ID, "instance.patch", "", nil)
		inst, err := s.Q.GetInstance(ctx, in.ID)
		if err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		out := &instanceOutput{}
		out.Body = toOut(inst)
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "delete-instance", Method: http.MethodDelete, Path: "/api/v1/instances/{id}",
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*struct{}, error) {
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
		if err := s.Q.SetInstanceState(ctx, db.SetInstanceStateParams{ID: in.ID, State: "deleting"}); err != nil {
			return nil, huma.Error500InternalServerError("delete failed")
		}
		if _, err := s.River.Insert(ctx, jobs.DeleteArgs{InstanceID: in.ID}, &river.InsertOpts{Queue: jobs.QueueDefault}); err != nil {
			return nil, huma.Error500InternalServerError("enqueue failed")
		}
		s.audit(ctx, u.ID, in.ID, "instance.delete", "", nil)
		s.Hub.Publish(in.ID + ":deleting")
		return nil, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "instance-action", Method: http.MethodPost, Path: "/api/v1/instances/{id}/actions",
	}, func(ctx context.Context, in *actionInput) (*struct{}, error) {
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
		if err := s.requireWired(); err != nil {
			return nil, err
		}
		act := instances.Action(in.Body.Action)
		if act == instances.ActionResize {
			return nil, s.doResize(ctx, u, inst, in.Body.Plan)
		}
		next, err := instances.Transition(instances.State(inst.State), act)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := s.Q.SetInstanceState(ctx, db.SetInstanceStateParams{ID: in.ID, State: string(next)}); err != nil {
			return nil, huma.Error500InternalServerError("state failed")
		}
		var args river.JobArgs
		switch act {
		case instances.ActionStart:
			args = jobs.StartArgs{InstanceID: in.ID}
		case instances.ActionStop:
			args = jobs.StopArgs{InstanceID: in.ID}
		case instances.ActionReboot:
			args = jobs.RebootArgs{InstanceID: in.ID}
		}
		if _, err := s.River.Insert(ctx, args, &river.InsertOpts{Queue: jobs.QueueDefault}); err != nil {
			return nil, huma.Error500InternalServerError("enqueue failed")
		}
		s.audit(ctx, u.ID, in.ID, "instance."+in.Body.Action, "", nil)
		s.Hub.Publish(in.ID + ":" + string(next))
		return nil, nil
	})
}

// ownedOrAdmin loads the instance enforcing the IDOR gate: owners pass,
// admins pass, everyone else gets 404 (same as missing, no id probing).
func (s *Server) ownedOrAdmin(ctx context.Context, u *db.Users, id string) (*db.Instances, error) {
	inst, err := s.Q.GetInstanceOwned(ctx, db.GetInstanceOwnedParams{ID: id, UserID: u.ID})
	if err == nil {
		return &inst, nil
	}
	if u.Role == "admin" {
		inst, err := s.Q.GetInstance(ctx, id)
		if err != nil {
			return nil, huma.Error404NotFound("not found")
		}
		return &inst, nil
	}
	return nil, huma.Error404NotFound("not found")
}

func (s *Server) doResize(ctx context.Context, u *db.Users, inst *db.Instances, planSlug string) error {
	if planSlug == "" {
		return huma.Error400BadRequest("plan required for resize")
	}
	if _, err := instances.Transition(instances.State(inst.State), instances.ActionResize); err != nil {
		return huma.Error400BadRequest(err.Error())
	}
	plan, err := s.Q.GetPlan(ctx, planSlug)
	if err != nil {
		return huma.Error400BadRequest("unknown plan")
	}
	if plan.DiskGb < inst.DiskGb {
		return huma.Error400BadRequest("disk can only grow")
	}
	if err := s.Q.SetInstanceState(ctx, db.SetInstanceStateParams{ID: inst.ID, State: "resizing"}); err != nil {
		return huma.Error500InternalServerError("state failed")
	}
	if _, err := s.River.Insert(ctx, jobs.ResizeArgs{InstanceID: inst.ID, PlanSlug: planSlug}, &river.InsertOpts{Queue: jobs.QueueDefault}); err != nil {
		return huma.Error500InternalServerError("enqueue failed")
	}
	s.audit(ctx, u.ID, inst.ID, "instance.resize", "", map[string]string{"plan": planSlug})
	s.Hub.Publish(inst.ID + ":resizing")
	return nil
}

// createInstanceTx runs plan §1.5 step 1: quota lock + admission + IP/port
// allocation + pending row + River provision job, all in ONE pgx transaction.
func (s *Server) createInstanceTx(ctx context.Context, u *db.Users, name, planSlug, imageSlug string, keyIDs []int64, userData string) (*db.Instances, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		inst, err := s.createOnce(ctx, u, name, planSlug, imageSlug, keyIDs, userData)
		if err == nil {
			return inst, nil
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	return nil, lastErr
}

func (s *Server) createOnce(ctx context.Context, u *db.Users, name, planSlug, imageSlug string, keyIDs []int64, userData string) (*db.Instances, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var planVcpu, planMem, planDisk int32
	var planExists bool
	if err := tx.QueryRow(ctx, `SELECT vcpu, memory_mb, disk_gb FROM plans WHERE slug=$1 AND active`, planSlug).Scan(&planVcpu, &planMem, &planDisk); err != nil {
		if err == pgx.ErrNoRows {
			return nil, huma.Error400BadRequest("unknown plan")
		}
		return nil, err
	}
	planExists = true
	_ = planExists

	var imgDistro, imgAlias string
	if err := tx.QueryRow(ctx, `SELECT distro, incus_alias FROM images WHERE slug=$1 AND active`, imageSlug).Scan(&imgDistro, &imgAlias); err != nil {
		if err == pgx.ErrNoRows {
			return nil, huma.Error400BadRequest("unknown image")
		}
		return nil, err
	}
	_ = imgDistro
	_ = imgAlias

	if _, err := tx.Exec(ctx, `INSERT INTO quotas (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, u.ID); err != nil {
		return nil, err
	}
	var qMaxInst, qMaxVCPU, qMaxMem, qMaxDisk, qMaxPorts int32
	if err := tx.QueryRow(ctx, `SELECT max_instances, max_vcpu, max_memory_mb, max_disk_gb, max_mapped_ports FROM quotas WHERE user_id=$1 FOR UPDATE`, u.ID).
		Scan(&qMaxInst, &qMaxVCPU, &qMaxMem, &qMaxDisk, &qMaxPorts); err != nil {
		return nil, err
	}
	var useInst, useVCPU, useMem, useDisk, usePorts int32
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(i.id)::int, COALESCE(SUM(i.vcpu),0)::int, COALESCE(SUM(i.memory_mb),0)::int,
		       COALESCE(SUM(i.disk_gb),0)::int,
		       (SELECT COUNT(*)::int FROM port_mappings p JOIN instances i2 ON i2.id=p.vps_id
		         WHERE i2.user_id=$1 AND i2.deleted_at IS NULL)
		FROM instances i WHERE i.user_id=$1 AND i.deleted_at IS NULL`, u.ID).
		Scan(&useInst, &useVCPU, &useMem, &useDisk, &usePorts); err != nil {
		return nil, err
	}
	if err := instances.CheckQuota(
		instances.Quota{MaxInstances: int(qMaxInst), MaxVCPU: int(qMaxVCPU), MaxMemoryMB: int(qMaxMem), MaxDiskGB: int(qMaxDisk), MaxMappedPorts: int(qMaxPorts)},
		instances.Usage{Instances: int(useInst), VCPU: int(useVCPU), MemoryMB: int(useMem), DiskGB: int(useDisk), MappedPort: int(usePorts)},
		instances.Plan{VCPU: int(planVcpu), MemoryMB: int(planMem), DiskGB: int(planDisk)},
		1, // ssh fallback port
	); err != nil {
		return nil, huma.Error403Forbidden(err.Error())
	}
	// Host capacity is advisory: absent row (fresh install) means unchecked.
	var freeVCPU, freeMem, freeDisk int32
	var capPresent bool
	if err := tx.QueryRow(ctx, `SELECT free_vcpu, free_memory_mb, free_disk_gb FROM host_capacity WHERE id=1`).Scan(&freeVCPU, &freeMem, &freeDisk); err == nil {
		capPresent = true
		if err := instances.CheckCapacity(
			instances.Capacity{FreeVCPU: int(freeVCPU), FreeMemoryMB: int(freeMem), FreeDiskGB: int(freeDisk)},
			instances.Plan{VCPU: int(planVcpu), MemoryMB: int(planMem), DiskGB: int(planDisk)},
		); err != nil {
			return nil, huma.Error403Forbidden(err.Error())
		}
	}
	_ = capPresent

	// Verify requested SSH keys belong to the caller.
	if len(keyIDs) > 0 {
		var owned int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ssh_keys WHERE user_id=$1 AND id=ANY($2)`, u.ID, keyIDs).Scan(&owned); err != nil {
			return nil, err
		}
		if owned != len(keyIDs) {
			return nil, huma.Error400BadRequest("unknown ssh key")
		}
	}

	usedIPRows, err := tx.Query(ctx, `SELECT private_ipv4 FROM instances WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	var usedIPs []string
	for usedIPRows.Next() {
		var ip string
		_ = usedIPRows.Scan(&ip)
		usedIPs = append(usedIPs, ip)
	}
	usedIPRows.Close()
	ip, err := netalloc.AllocateIP(usedIPs)
	if err != nil {
		return nil, huma.Error403Forbidden(err.Error())
	}
	usedPortRows, err := tx.Query(ctx, `SELECT public_port FROM port_mappings`)
	if err != nil {
		return nil, err
	}
	var usedPorts []int32
	for usedPortRows.Next() {
		var p int32
		_ = usedPortRows.Scan(&p)
		usedPorts = append(usedPorts, p)
	}
	usedPortRows.Close()
	fallback, err := netalloc.AllocatePort(usedPorts)
	if err != nil {
		return nil, huma.Error403Forbidden(err.Error())
	}

	id, err := ids.Generate()
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instances (id, user_id, name, plan_id, image_id, vcpu, memory_mb, disk_gb,
		  state, desired_state, private_ipv4, https_mode, user_data)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending','running',$9,'terminate',$10)`,
		id, u.ID, name, planSlug, imageSlug, planVcpu, planMem, planDisk, ip.String(), userData); err != nil {
		return nil, err // unique violation → outer retry
	}
	for _, cidr := range []string{"0.0.0.0/0", "::/0"} {
		for _, port := range []int{22, 80, 443} {
			if _, err := tx.Exec(ctx, `INSERT INTO firewall_rules (instance_id, proto, port_from, port_to, source_cidr) VALUES ($1,'tcp',$2,$2,$3)`, id, port, cidr); err != nil {
				return nil, err
			}
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO port_mappings (vps_id, public_port, target_port, proto, kind) VALUES ($1,$2,22,'tcp','ssh-fallback')`, id, fallback); err != nil {
		return nil, err
	}
	for _, kid := range keyIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO instance_ssh_keys (instance_id, ssh_key_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, kid); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_log (actor_id, instance_id, action, meta, ip) VALUES ($1,$2,'instance.create','{}','')`, u.ID, id); err != nil {
		return nil, err
	}
	client, err := river.ClientFromContextSafely[pgx.Tx](ctx)
	if err != nil {
		// Fall back to the server's producer client inside this tx.
		if _, err := s.River.InsertTx(ctx, tx, jobs.ProvisionArgs{InstanceID: id}, &river.InsertOpts{Queue: jobs.QueueDefault}); err != nil {
			return nil, err
		}
	} else {
		if _, err := client.InsertTx(ctx, tx, jobs.ProvisionArgs{InstanceID: id}, &river.InsertOpts{Queue: jobs.QueueDefault}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.Hub.Publish(id + ":pending")
	inst, err := s.Q.GetInstance(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}
