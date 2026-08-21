package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// installationTokenSkew is subtracted from GitHub's reported expiry so a
// cached token already close to expiring is refreshed proactively, rather
// than handed to a caller that then gets a 401 mid-request.
const installationTokenSkew = 2 * time.Minute

// appJWTLifetime is the signed JWT's own exp-minus-iat window. GitHub caps it
// at 10 minutes; this stays under that with margin, since the JWT is used
// exactly once, immediately, to mint the installation token below — it is
// never handed to a caller or cached itself.
const appJWTLifetime = 9 * time.Minute

// appJWTClockSkew backdates iat so ordinary clock drift between this host and
// GitHub's never makes the JWT look not-yet-valid.
const appJWTClockSkew = 60 * time.Second

// AppInstallationTokenSource mints and caches short-lived GitHub App
// installation tokens: a JWT signed with the App's own RS256 private key
// (authenticating the App, not the installation) exchanged for a token
// scoped to one installation. This is the same JWT-mint-then-exchange flow
// platform/scripts/open-agent-pr.sh runs in bash/python, reimplemented
// against the standard library here since this module carries no JWT
// dependency of its own.
type AppInstallationTokenSource struct {
	appID          string
	installationID string
	key            *rsa.PrivateKey
	apiBase        string
	hc             *http.Client

	mu      sync.Mutex
	token   string
	expires time.Time
}

// NewAppInstallationTokenSource parses privateKeyPEM once at construction so
// a malformed key fails at startup rather than on the first remediation
// attempt.
func NewAppInstallationTokenSource(appID, installationID string, privateKeyPEM []byte, apiBase string, hc *http.Client) (*AppInstallationTokenSource, error) {
	if appID == "" || installationID == "" {
		return nil, errors.New("github app token source: app id and installation id are both required")
	}
	key, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("github app token source: %w", err)
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	return &AppInstallationTokenSource{
		appID: appID, installationID: installationID, key: key, apiBase: apiBase, hc: hc,
	}, nil
}

func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("private key: no PEM block found")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key: not RSA")
	}
	return key, nil
}

// Token returns a cached installation token, minting a fresh one when none is
// cached yet or the cached one is within installationTokenSkew of expiring.
func (a *AppInstallationTokenSource) Token(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.token != "" && time.Now().Before(a.expires.Add(-installationTokenSkew)) {
		return a.token, nil
	}
	token, expires, err := a.mint(ctx)
	if err != nil {
		return "", err
	}
	a.token, a.expires = token, expires
	return token, nil
}

func (a *AppInstallationTokenSource) mint(ctx context.Context) (string, time.Time, error) {
	jwt, err := signAppJWT(a.appID, a.key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign app jwt: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.apiBase+"/app/installations/"+a.installationID+"/access_tokens", nil)
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := a.hc.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("mint installation token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", time.Time{}, fmt.Errorf("mint installation token: status %d", resp.StatusCode)
	}
	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", time.Time{}, fmt.Errorf("decode installation token: %w", err)
	}
	if out.Token == "" {
		return "", time.Time{}, errors.New("mint installation token: empty token in response")
	}
	return out.Token, out.ExpiresAt, nil
}

// signAppJWT builds and RS256-signs the JWT a GitHub App authenticates its
// own identity with — this token, not the installation token it is
// exchanged for below, is the one whose lifetime GitHub caps at 10 minutes.
func signAppJWT(appID string, key *rsa.PrivateKey) (string, error) {
	now := time.Now()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iat": now.Add(-appJWTClockSkew).Unix(),
		"exp": now.Add(appJWTLifetime).Unix(),
		"iss": appID,
	}
	headerB64, err := base64JSON(header)
	if err != nil {
		return "", err
	}
	claimsB64, err := base64JSON(claims)
	if err != nil {
		return "", err
	}
	signingInput := headerB64 + "." + claimsB64
	hashed := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func base64JSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// newGitHubTokenSource prefers minting App installation tokens so PRs open
// under the App's own identity (GITHUB_APP_ID / GITHUB_APP_INSTALLATION_ID /
// GITHUB_APP_PRIVATE_KEY, all three required together); GITHUB_TOKEN is kept
// as a static fallback so the relay can still run — against a plain PAT,
// with whatever access that PAT carries — before the App identity is wired.
func newGitHubTokenSource(apiBase string, hc *http.Client) (GitHubTokenSource, error) {
	appID := os.Getenv("GITHUB_APP_ID")
	installationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")
	privateKey := os.Getenv("GITHUB_APP_PRIVATE_KEY")

	switch {
	case appID != "" && installationID != "" && privateKey != "":
		src, err := NewAppInstallationTokenSource(appID, installationID, []byte(privateKey), apiBase, hc)
		if err != nil {
			return nil, err
		}
		slog.Info("github: minting installation tokens from app credentials", "app_id", appID)
		return src, nil
	case appID != "" || installationID != "" || privateKey != "":
		return nil, errors.New("github: GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID and GITHUB_APP_PRIVATE_KEY must be set together")
	case os.Getenv("GITHUB_TOKEN") != "":
		slog.Warn("github: no App identity configured (GITHUB_APP_ID/GITHUB_APP_INSTALLATION_ID/GITHUB_APP_PRIVATE_KEY); falling back to static GITHUB_TOKEN")
		return StaticGitHubToken(os.Getenv("GITHUB_TOKEN")), nil
	default:
		return nil, errors.New("github: no credentials configured — set GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID and GITHUB_APP_PRIVATE_KEY, or GITHUB_TOKEN")
	}
}
