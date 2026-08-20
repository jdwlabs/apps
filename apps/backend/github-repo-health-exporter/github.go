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
	"net/http"
	"sync"
	"time"
)

// appJWTLifetime is capped at 10 minutes by GitHub; kept short of that so a
// token is never presented right at its own expiry.
const appJWTLifetime = 9 * time.Minute

// appJWTClockSkew backdates iat so a GitHub App JWT still validates against a
// server clock that runs slightly ahead of this process's — GitHub's own docs
// recommend this exact margin.
const appJWTClockSkew = 60 * time.Second

// installationTokenLifetime is GitHub's fixed lifetime for an installation
// access token; refreshRunner leaves this cached until refreshMargin of it
// remains rather than re-minting one per scrape.
const installationTokenLifetime = time.Hour

// refreshMargin is how long before expiry a cached installation token is
// treated as stale, so a scrape never races a token that expires mid-request.
const refreshMargin = 5 * time.Minute

// parseRSAPrivateKey accepts both PEM encodings GitHub has issued App keys in
// over the product's life: PKCS#1 ("BEGIN RSA PRIVATE KEY", the format the
// App-settings download page still produces) and PKCS#8 ("BEGIN PRIVATE
// KEY"). Guessing wrong on a stored key would fail authentication in a way
// that reads identically to a revoked key, so both are tried rather than
// assuming the newer form.
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("github: no PEM block found in private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("github: private key is neither PKCS1 nor PKCS8: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("github: private key is %T, want *rsa.PrivateKey", key)
	}
	return rsaKey, nil
}

// b64url is JWT's own encoding: base64url with padding stripped, per RFC 7515
// — encoding/base64's RawURLEncoding already produces exactly that.
func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// signAppJWT builds the RS256 App-authentication JWT GitHub's App API
// requires. Signed with crypto/rsa directly rather than a JWT library: three
// stdlib packages (crypto/rsa, crypto/x509, encoding/pem) already cover
// parsing the key and PKCS#1v1.5-signing the token, and this is the only JWT
// this binary ever produces.
func signAppJWT(appID string, key *rsa.PrivateKey, now time.Time) (string, error) {
	iat := now.Add(-appJWTClockSkew)
	header := `{"alg":"RS256","typ":"JWT"}`
	claims := fmt.Sprintf(`{"iat":%d,"exp":%d,"iss":%s}`,
		iat.Unix(), iat.Add(appJWTLifetime).Unix(), jsonString(appID))
	signingInput := b64url([]byte(header)) + "." + b64url([]byte(claims))
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("github: signing app jwt: %w", err)
	}
	return signingInput + "." + b64url(sig), nil
}

// jsonString quotes a Go string as a JSON string. iss is always a numeric
// GitHub App ID read from config, never external input, so a hand-rolled
// quote is safe — this exists only to avoid pulling encoding/json into a
// three-field literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// GitHubClient authenticates as a GitHub App installation and reads Actions
// workflow-run history. It holds no write scope — every call this exporter
// makes is a GET.
type GitHubClient struct {
	apiBase        string // https://api.github.com (override in tests)
	appID          string
	installationID string
	key            *rsa.PrivateKey
	hc             *http.Client

	mu       sync.Mutex
	token    string
	tokenExp time.Time
	nowFn    func() time.Time // overridable in tests; defaults to time.Now
}

func NewGitHubClient(apiBase, appID, installationID string, key *rsa.PrivateKey, hc *http.Client) *GitHubClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &GitHubClient{
		apiBase:        apiBase,
		appID:          appID,
		installationID: installationID,
		key:            key,
		hc:             hc,
		nowFn:          time.Now,
	}
}

func (g *GitHubClient) now() time.Time {
	if g.nowFn != nil {
		return g.nowFn()
	}
	return time.Now()
}

// installationToken returns a cached token when it has more than
// refreshMargin left, and mints a fresh one otherwise. Single-flighted via mu
// rather than per-request: the refresh interval (tens of minutes) is far
// longer than a scrape takes, so the lock is never meaningfully contended.
func (g *GitHubClient) installationToken(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.token != "" && g.now().Before(g.tokenExp.Add(-refreshMargin)) {
		return g.token, nil
	}

	jwt, err := signAppJWT(g.appID, g.key, g.now())
	if err != nil {
		return "", err
	}
	url := g.apiBase + "/app/installations/" + g.installationID + "/access_tokens"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := g.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("github: requesting installation token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("github: installation token request: status %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("github: decoding installation token: %w", err)
	}
	if out.Token == "" {
		return "", errors.New("github: installation token response carried no token")
	}
	g.token = out.Token
	// GitHub's response also carries expires_at, but installationTokenLifetime
	// is documented as fixed and this avoids a second time-parse failure mode.
	g.tokenExp = g.now().Add(installationTokenLifetime)
	return g.token, nil
}

// WorkflowRun is the subset of GitHub's Actions run object this exporter
// reads. Field names/tags mirror the API response verbatim.
type WorkflowRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"created_at"`
}

// runsWindow bounds how many of a repo's most recent main-branch runs are
// read per refresh. GitHub returns runs newest-first, so this is exactly the
// window ciHealth's ratio and streak are computed over; it is also the
// largest page size the Actions runs endpoint accepts in one call, so this
// stays a single request per repo.
const runsWindow = 100

// ListMainRuns returns up to runsWindow most recent Actions runs targeting
// branch main in owner/repo, newest first, across every workflow in the repo.
func (g *GitHubClient) ListMainRuns(ctx context.Context, ownerRepo string) ([]WorkflowRun, error) {
	token, err := g.installationToken(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/repos/%s/actions/runs?branch=main&per_page=%d", g.apiBase, ownerRepo, runsWindow)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := g.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: listing runs for %s: %w", ownerRepo, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: listing runs for %s: status %d", ownerRepo, resp.StatusCode)
	}
	var out struct {
		WorkflowRuns []WorkflowRun `json:"workflow_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("github: decoding runs for %s: %w", ownerRepo, err)
	}
	// Only completed runs (status == "completed") carry a meaningful
	// conclusion — in_progress/queued runs report conclusion "" and would
	// read as neither a pass nor a fail, silently deflating the ratio.
	completed := out.WorkflowRuns[:0]
	for _, r := range out.WorkflowRuns {
		if r.Status == "completed" {
			completed = append(completed, r)
		}
	}
	return completed, nil
}
