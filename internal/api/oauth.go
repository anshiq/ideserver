package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anshiq/ideserver/internal/auth"
	"github.com/anshiq/ideserver/internal/db"
)

const oauthStateCookie = "oauth_state"

// handleOAuthLogin redirects to GitHub with state + PKCE.
func (s *Server) handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	if s.OAuth == nil {
		http.Error(w, "oauth not configured", http.StatusServiceUnavailable)
		return
	}
	url, state, verifier, err := s.OAuth.LoginURL()
	if err != nil {
		http.Error(w, "oauth failed", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: oauthStateCookie, Value: state + "." + verifier,
		Path: "/", MaxAge: 300, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, url, http.StatusFound)
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// handleOAuthCallback verifies state, exchanges the code, upserts the user,
// ensures quota, mints an opaque session and sets the __Host- cookie.
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if s.OAuth == nil {
		http.Error(w, "oauth not configured", http.StatusServiceUnavailable)
		return
	}
	st, err := r.Cookie(oauthStateCookie)
	if err != nil {
		http.Error(w, "missing oauth state", http.StatusBadRequest)
		return
	}
	var wantState, verifier string
	if n, _ := fmt.Sscanf(st.Value, "%[^.].%s", &wantState, &verifier); n != 2 {
		http.Error(w, "bad oauth state", http.StatusBadRequest)
		return
	}
	_ = wantState
	if r.URL.Query().Get("state") == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Path: "/", MaxAge: -1})

	token, err := s.OAuth.Exchange(r, verifier)
	if err != nil {
		http.Error(w, "github exchange failed", http.StatusBadGateway)
		return
	}
	gh, err := fetchGitHubUser(r, token)
	if err != nil {
		http.Error(w, "github user fetch failed", http.StatusBadGateway)
		return
	}
	ctx := r.Context()
	user, err := s.Q.GetUserByGithubID(ctx, gh.ID)
	if err != nil {
		status, _ := SignupDecision(s.Cfg.SignupMode, gh.Login, s.Cfg.Allowlist)
		role := "user"
		if IsAdminLogin(gh.Login, s.Cfg.AdminLogins) {
			role = "admin"
			status = "active"
		}
		user, err = s.Q.CreateUser(ctx, db.CreateUserParams{
			GithubID: gh.ID, Login: gh.Login, Name: gh.Name, Email: gh.Email,
			AvatarUrl: gh.AvatarURL, Role: role, Status: status,
		})
		if err != nil {
			http.Error(w, "user create failed", http.StatusInternalServerError)
			return
		}
		if _, err := s.Q.EnsureQuota(ctx, user.ID); err != nil {
			http.Error(w, "quota init failed", http.StatusInternalServerError)
			return
		}
	}
	tok, err := auth.NewToken()
	if err != nil {
		http.Error(w, "session failed", http.StatusInternalServerError)
		return
	}
	exp := time.Now().Add(auth.SessionLifetime)
	if _, err := s.Q.CreateSession(ctx, db.CreateSessionParams{
		UserID: user.ID, TokenHash: auth.Hash(tok), ExpiresAt: exp,
		Ip: r.RemoteAddr, UserAgent: r.UserAgent(),
	}); err != nil {
		http.Error(w, "session create failed", http.StatusInternalServerError)
		return
	}
	auth.SetCookie(w, s.Cfg.AppDomain, tok, exp)
	http.Redirect(w, r, "/", http.StatusFound)
}

func fetchGitHubUser(r *http.Request, token string) (*githubUser, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(r.Context(), "GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github /user: %d", resp.StatusCode)
	}
	var gh githubUser
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, err
	}
	// Primary email is often private: try /user/emails best-effort.
	if gh.Email == "" {
		if email := fetchPrimaryEmail(r, client, token); email != "" {
			gh.Email = email
		}
	}
	return &gh, nil
}

func fetchPrimaryEmail(r *http.Request, client *http.Client, token string) string {
	req, _ := http.NewRequestWithContext(r.Context(), "GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()
	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email
		}
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}

// handleLogout deletes the session and clears the cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if tok, ok := auth.TokenFromRequest(r); ok {
		_ = s.Q.DeleteSession(r.Context(), auth.Hash(tok))
	}
	auth.ClearCookie(w, s.Cfg.AppDomain)
	w.WriteHeader(http.StatusNoContent)
}
