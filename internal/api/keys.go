package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/anshiq/ideserver/internal/db"
	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/crypto/ssh"
)

type sshKeyOutput struct {
	Body db.SshKeys `json:"body"`
}

type sshKeyListOutput struct {
	Body []db.SshKeys `json:"body"`
}

type createKeyInput struct {
	Body struct {
		Name      string `json:"name" maxLength:"64"`
		PublicKey string `json:"publicKey" maxLength:"4096"`
	} `json:"body"`
}

// parseAndFingerprint validates one authorized-key line and returns its
// SHA256 fingerprint (same form as `ssh-keygen -lf`).
func parseAndFingerprint(line string) (string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", fmt.Errorf("empty key")
	}
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return "", fmt.Errorf("bad public key: %w", err)
	}
	return ssh.FingerprintSHA256(pub), nil
}

func (s *Server) registerKeys(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-keys", Method: http.MethodGet, Path: "/api/v1/ssh-keys",
	}, func(ctx context.Context, _ *struct{}) (*sshKeyListOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		keys, err := s.Q.ListSSHKeys(ctx, u.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		out := &sshKeyListOutput{}
		out.Body = orEmpty(keys)
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "create-key", Method: http.MethodPost, Path: "/api/v1/ssh-keys",
	}, func(ctx context.Context, in *createKeyInput) (*sshKeyOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSpace(in.Body.Name)
		if name == "" || len(name) > 64 {
			return nil, huma.Error400BadRequest("name must be 1-64 chars")
		}
		fp, err := parseAndFingerprint(in.Body.PublicKey)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		key, err := s.Q.UpsertSSHKey(ctx, db.UpsertSSHKeyParams{
			UserID: u.ID, Name: name,
			PublicKey: strings.TrimSpace(in.Body.PublicKey), Fingerprint: fp,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		s.audit(ctx, u.ID, "", "keys.create", "", map[string]string{"fingerprint": fp})
		out := &sshKeyOutput{}
		out.Body = key
		return out, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "delete-key", Method: http.MethodDelete, Path: "/api/v1/ssh-keys/{keyId}",
	}, func(ctx context.Context, in *struct {
		KeyID int64 `path:"keyId"`
	}) (*struct{}, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		n, err := s.Q.DeleteSSHKey(ctx, db.DeleteSSHKeyParams{ID: in.KeyID, UserID: u.ID})
		if err != nil || n == 0 {
			return nil, huma.Error404NotFound("not found")
		}
		s.audit(ctx, u.ID, "", "keys.delete", "", nil)
		return nil, nil
	})
	huma.Register(api, huma.Operation{
		OperationID: "import-github-keys", Method: http.MethodPost, Path: "/api/v1/ssh-keys/import-github",
	}, func(ctx context.Context, _ *struct{}) (*sshKeyListOutput, error) {
		u, err := s.requireUser(ctx)
		if err != nil {
			return nil, err
		}
		lines, err := fetchGitHubKeys(ctx, u.Login)
		if err != nil {
			return nil, huma.Error502BadGateway("github keys fetch failed")
		}
		for i, ln := range lines {
			fp, err := parseAndFingerprint(ln)
			if err != nil {
				continue
			}
			_, _ = s.Q.UpsertSSHKey(ctx, db.UpsertSSHKeyParams{
				UserID: u.ID, Name: fmt.Sprintf("github-%d", i+1),
				PublicKey: strings.TrimSpace(ln), Fingerprint: fp,
			})
		}
		s.audit(ctx, u.ID, "", "keys.import-github", "", nil)
		keys, err := s.Q.ListSSHKeys(ctx, u.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("db error")
		}
		out := &sshKeyListOutput{}
		out.Body = orEmpty(keys)
		return out, nil
	})
}

// fetchGitHubKeys pulls the public https://github.com/{login}.keys file.
func fetchGitHubKeys(ctx context.Context, login string) ([]string, error) {
	if login == "" {
		return nil, fmt.Errorf("no login")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://github.com/"+login+".keys", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github .keys: %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ln := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(ln) != "" {
			out = append(out, ln)
		}
	}
	return out, nil
}
