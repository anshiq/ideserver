package api

import (
	"context"
	"net/http"

	"github.com/anshiq/ideserver/internal/db"
	"github.com/danielgtaylor/huma/v2"
)

func (s *Server) requireAdmin(ctx context.Context) (*db.Users, error) {
	u, err := s.requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if u.Role != "admin" {
		return nil, huma.Error403Forbidden("admin only")
	}
	return u, nil
}

type adminUsersOutput struct {
	Body []db.Users `json:"body"`
}

type adminPatchUserInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Status string `json:"status,omitempty" enum:"pending,active,suspended"`
		Role   string `json:"role,omitempty" enum:"user,admin"`
	} `json:"body"`
}

type adminQuotaOutput struct {
	Body db.Quotas `json:"body"`
}

type adminPatchQuotaInput struct {
	ID   int64 `path:"id"`
	Body struct {
		MaxInstances   int `json:"maxInstances"`
		MaxVCPU        int `json:"maxVcpu"`
		MaxMemoryMB    int `json:"maxMemoryMb"`
		MaxDiskGB      int `json:"maxDiskGb"`
		MaxMappedPorts int `json:"maxMappedPorts"`
	} `json:"body"`
}

type adminHostOutput struct {
	Body db.HostCapacity `json:"body"`
}

type adminInstancesOutput struct {
	Body []InstanceOut `json:"body"`
}

func (s *Server) registerAdmin(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "admin-list-users", Method: http.MethodGet, Path: "/api/v1/admin/users",
	}, func(ctx context.Context, _ *struct{}) (*adminUsersOutput, error) {
		if _, err := s.requireAdmin(ctx); err != nil {
			return nil, err
		}
		users, err := s.Q.ListUsers(ctx, db.ListUsersParams{Limit: 100, Offset: 0})
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		out := &adminUsersOutput{}
		out.Body = orEmpty(users)
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "admin-patch-user", Method: http.MethodPatch, Path: "/api/v1/admin/users/{id}",
	}, func(ctx context.Context, in *adminPatchUserInput) (*struct{}, error) {
		a, err := s.requireAdmin(ctx)
		if err != nil {
			return nil, err
		}
		if in.Body.Status != "" {
			if err := s.Q.UpdateUserStatus(ctx, db.UpdateUserStatusParams{ID: in.ID, Status: in.Body.Status}); err != nil {
				return nil, huma.Error500InternalServerError("db error")
			}
		}
		if in.Body.Role != "" {
			if err := s.Q.UpdateUserRole(ctx, db.UpdateUserRoleParams{ID: in.ID, Role: in.Body.Role}); err != nil {
				return nil, huma.Error500InternalServerError("db error")
			}
		}
		s.audit(ctx, a.ID, "", "admin.user.patch", "", nil)
		return nil, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "admin-get-quota", Method: http.MethodGet, Path: "/api/v1/admin/users/{id}/quota",
	}, func(ctx context.Context, in *struct {
		ID int64 `path:"id"`
	}) (*adminQuotaOutput, error) {
		if _, err := s.requireAdmin(ctx); err != nil {
			return nil, err
		}
		q, err := s.Q.GetQuota(ctx, in.ID)
		if err != nil {
			return nil, huma.Error404NotFound("no quota")
		}
		out := &adminQuotaOutput{}
		out.Body = q
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "admin-patch-quota", Method: http.MethodPut, Path: "/api/v1/admin/users/{id}/quota",
	}, func(ctx context.Context, in *adminPatchQuotaInput) (*struct{}, error) {
		a, err := s.requireAdmin(ctx)
		if err != nil {
			return nil, err
		}
		if err := s.Q.UpdateQuota(ctx, db.UpdateQuotaParams{
			UserID: in.ID, MaxInstances: int32(in.Body.MaxInstances), MaxVcpu: int32(in.Body.MaxVCPU),
			MaxMemoryMb: int32(in.Body.MaxMemoryMB), MaxDiskGb: int32(in.Body.MaxDiskGB), MaxMappedPorts: int32(in.Body.MaxMappedPorts),
		}); err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		s.audit(ctx, a.ID, "", "admin.quota.patch", "", nil)
		return nil, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "admin-host", Method: http.MethodGet, Path: "/api/v1/admin/host",
	}, func(ctx context.Context, _ *struct{}) (*adminHostOutput, error) {
		if _, err := s.requireAdmin(ctx); err != nil {
			return nil, err
		}
		cap, err := s.Q.GetCapacity(ctx)
		if err != nil {
			return nil, huma.Error404NotFound("no capacity reported yet")
		}
		out := &adminHostOutput{}
		out.Body = cap
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "admin-instances", Method: http.MethodGet, Path: "/api/v1/admin/instances",
	}, func(ctx context.Context, in *struct {
		State string `query:"state,omitempty"`
	}) (*adminInstancesOutput, error) {
		if _, err := s.requireAdmin(ctx); err != nil {
			return nil, err
		}
		out := &adminInstancesOutput{Body: []InstanceOut{}}
		if in.State != "" {
			rows, err := s.Q.ListInstancesByState(ctx, in.State)
			if err != nil {
				return nil, huma.Error500InternalServerError("db error")
			}
			for _, r := range rows {
				out.Body = append(out.Body, toOut(r))
			}
			return out, nil
		}
		// All non-deleted: fan out per state (no full-table sqlc query).
		for _, st := range []string{"pending", "provisioning", "running", "starting", "stopping", "stopped", "rebooting", "resizing", "deleting", "error"} {
			rows, err := s.Q.ListInstancesByState(ctx, st)
			if err != nil {
				return nil, huma.Error500InternalServerError("db error")
			}
			for _, r := range rows {
				out.Body = append(out.Body, toOut(r))
			}
		}
		return out, nil
	})
}
