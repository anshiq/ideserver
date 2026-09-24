// Package auth implements GitHub OAuth (x/oauth2 + PKCE), opaque
// server-side sessions in Postgres and CSRF checks.
//
// Cookies are host-only `__Host-session` (Secure, HttpOnly, SameSite=Lax)
// scoped to app.{domain} only — never the apex — so VPS subdomains on the
// shared apex cannot toss them (plan §1.6).
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// CookieName is the session cookie. The __Host- prefix forces Secure,
// host-only and Path=/ semantics in browsers.
const CookieName = "__Host-session"

// SessionLifetime bounds an opaque session token.
const SessionLifetime = 30 * 24 * time.Hour

// OAuth wraps the GitHub endpoints with state + PKCE verifiers.
type OAuth struct {
	cfg *oauth2.Config
}

// NewOAuth builds the GitHub client. callback must be the public
// https://app.{domain}/api/v1/auth/github/callback URL.
func NewOAuth(clientID, secret, callback string) *OAuth {
	return &OAuth{cfg: &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secret,
		Endpoint:     github.Endpoint,
		RedirectURL:  callback,
		Scopes:       []string{"read:user", "user:email"},
	}}
}

// LoginURL returns the redirect target plus the state/verifier the caller
// must stash (e.g. in a short-lived signed cookie) until the callback.
func (o *OAuth) LoginURL() (url, state, verifier string, err error) {
	var sb [32]byte
	if _, err := rand.Read(sb[:]); err != nil {
		return "", "", "", fmt.Errorf("auth: rand: %w", err)
	}
	state = base64.RawURLEncoding.EncodeToString(sb[:])
	verifier = oauth2.GenerateVerifier()
	return o.cfg.AuthCodeURL(state,
		oauth2.AccessTypeOnline, oauth2.S256ChallengeOption(verifier)), state, verifier, nil
}

// Exchange trades the callback code for a GitHub access token.
func (o *OAuth) Exchange(r *http.Request, verifier string) (string, error) {
	tok, err := o.cfg.Exchange(r.Context(), r.URL.Query().Get("code"),
		oauth2.VerifierOption(verifier))
	if err != nil {
		return "", fmt.Errorf("auth: exchange: %w", err)
	}
	return tok.AccessToken, nil
}

// NewToken mints a random 32-byte opaque session token (base64url).
func NewToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("auth: rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// Hash returns the stored form of a token: SHA-256, never the token itself.
func Hash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// Equal compares a presented token against a stored hash in constant time.
func Equal(token string, hash []byte) bool {
	sum := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(sum[:], hash) == 1
}

// SetCookie writes the host-only session cookie for appDomain.
func SetCookie(w http.ResponseWriter, appDomain, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Domain:   appDomain,
		Path:     "/",
		Expires:  exp,
		MaxAge:   int(time.Until(exp).Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie expires the session cookie.
func ClearCookie(w http.ResponseWriter, appDomain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Domain:   appDomain,
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// TokenFromRequest extracts the session token, if present.
func TokenFromRequest(r *http.Request) (string, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}

// CheckCSRF enforces Origin matching appOrigin plus a required custom
// header on mutating requests.
func CheckCSRF(r *http.Request, appOrigin string) error {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return nil
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != appOrigin {
		return fmt.Errorf("auth: csrf origin mismatch")
	}
	if strings.TrimSpace(r.Header.Get("X-Requested-With")) == "" {
		return fmt.Errorf("auth: csrf header required")
	}
	return nil
}
