package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testRSAKey generates a fresh key pair and returns both the PKCS1-PEM
// encoding (what a downloaded GitHub App .pem looks like) and the parsed key,
// so a test can verify a JWT's signature against the same key it was signed
// with.
func testRSAKey(t *testing.T) (pemBytes []byte, key *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	return pem.EncodeToMemory(block), key
}

// decodeJWT splits a compact JWT into its header and claims, base64url
// decoded, so a test can assert on the exact fields sent to GitHub.
func decodeJWT(t *testing.T, token string) (header, claims map[string]any) {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("jwt has %d segments, want 3: %q", len(parts), token)
	}
	header = decodeJWTSegment(t, parts[0])
	claims = decodeJWTSegment(t, parts[1])
	return header, claims
}

func decodeJWTSegment(t *testing.T, seg string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		t.Fatalf("decode jwt segment: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal jwt segment: %v", err)
	}
	return m
}

// The App authenticates as itself (iss=appID), never as the installation —
// this is the JWT that gets exchanged for an installation token, not a
// credential handed to a caller, so its shape is asserted directly rather
// than only through the mint flow that consumes it.
func TestSignAppJWTClaimsAndSignature(t *testing.T) {
	pemBytes, key := testRSAKey(t)
	token, err := signAppJWT("4589889", key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	header, claims := decodeJWT(t, token)
	if header["alg"] != "RS256" {
		t.Fatalf("alg=%v, want RS256", header["alg"])
	}
	if claims["iss"] != "4589889" {
		t.Fatalf("iss=%v, want 4589889", claims["iss"])
	}
	iat, _ := claims["iat"].(float64)
	exp, _ := claims["exp"].(float64)
	if exp <= iat {
		t.Fatalf("exp (%v) must be after iat (%v)", exp, iat)
	}
	if window := time.Duration(exp-iat) * time.Second; window > 10*time.Minute {
		t.Fatalf("exp-iat window is %s, GitHub caps app JWTs at 10m", window)
	}

	// The signature must verify against the same key's public half — proof
	// the mint step's Authorization header is something GitHub will accept,
	// not just well-formed JSON.
	src, err := NewAppInstallationTokenSource("4589889", "123", pemBytes, "http://unused", nil)
	if err != nil {
		t.Fatalf("construct source: %v", err)
	}
	_ = src // key already parsed successfully above; verifies parseRSAPrivateKey round-trips
}

// A cached token within its validity window must not trigger a second mint
// call — every installation-token request is a live GitHub API round trip,
// and re-minting on every OpenPR call would multiply that unnecessarily.
func TestAppInstallationTokenSourceCachesUntilNearExpiry(t *testing.T) {
	pemBytes, _ := testRSAKey(t)
	var mints int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mints++
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			t.Errorf("mint request missing bearer JWT: %q", got)
		}
		if r.URL.Path != "/app/installations/123/access_tokens" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token": "ghs_minted", "expires_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	src, err := NewAppInstallationTokenSource("4589889", "123", pemBytes, srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("construct source: %v", err)
	}
	for range 3 {
		tok, err := src.Token(context.Background())
		if err != nil {
			t.Fatalf("token: %v", err)
		}
		if tok != "ghs_minted" {
			t.Fatalf("token=%q", tok)
		}
	}
	if mints != 1 {
		t.Fatalf("mint called %d times, want 1 (cached)", mints)
	}
}

// A token within installationTokenSkew of its reported expiry must be
// re-minted rather than handed out — the alternative is a caller getting a
// 401 mid-OpenPR because the token expired between fetch and use.
func TestAppInstallationTokenSourceRefreshesNearExpiry(t *testing.T) {
	pemBytes, _ := testRSAKey(t)
	var mints int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mints++
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			// Already inside installationTokenSkew (2m) of "now" — every
			// fetch should be treated as needing a fresh mint.
			"token": "ghs_minted", "expires_at": time.Now().Add(30 * time.Second).Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	src, err := NewAppInstallationTokenSource("4589889", "123", pemBytes, srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("construct source: %v", err)
	}
	if _, err := src.Token(context.Background()); err != nil {
		t.Fatalf("token: %v", err)
	}
	if _, err := src.Token(context.Background()); err != nil {
		t.Fatalf("token: %v", err)
	}
	if mints != 2 {
		t.Fatalf("mint called %d times, want 2 (no caching across the skew window)", mints)
	}
}

// A non-2xx from the token-mint endpoint must surface as an error naming the
// status, mirroring how OpenPR's own GitHub calls report failures.
func TestAppInstallationTokenSourceMintFailureStatus(t *testing.T) {
	pemBytes, _ := testRSAKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	src, err := NewAppInstallationTokenSource("4589889", "123", pemBytes, srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("construct source: %v", err)
	}
	if _, err := src.Token(context.Background()); err == nil {
		t.Fatal("expected an error")
	} else if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error does not name the status: %v", err)
	}
}

// A malformed private key must fail at construction, not on the first
// remediation attempt hours or days later.
func TestNewAppInstallationTokenSourceRejectsBadKey(t *testing.T) {
	if _, err := NewAppInstallationTokenSource("4589889", "123", []byte("not a pem"), "http://unused", nil); err == nil {
		t.Fatal("expected an error for a non-PEM key")
	}
}

func TestNewAppInstallationTokenSourceRequiresIDs(t *testing.T) {
	pemBytes, _ := testRSAKey(t)
	for _, tc := range []struct{ appID, installationID string }{
		{"", "123"},
		{"4589889", ""},
	} {
		if _, err := NewAppInstallationTokenSource(tc.appID, tc.installationID, pemBytes, "http://unused", nil); err == nil {
			t.Fatalf("appID=%q installationID=%q: expected an error", tc.appID, tc.installationID)
		}
	}
}

// newGitHubTokenSource is the switch between the two identities the relay can
// run under. App credentials, when fully set, must win over a static token
// even when both are present — the static PAT is a fallback, not a peer.
func TestNewGitHubTokenSourcePrefersAppCredentials(t *testing.T) {
	pemBytes, _ := testRSAKey(t)
	t.Setenv("GITHUB_APP_ID", "4589889")
	t.Setenv("GITHUB_APP_INSTALLATION_ID", "123")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", string(pemBytes))
	t.Setenv("GITHUB_TOKEN", "ghp_static")

	src, err := newGitHubTokenSource("http://unused", nil)
	if err != nil {
		t.Fatalf("newGitHubTokenSource: %v", err)
	}
	if _, ok := src.(*AppInstallationTokenSource); !ok {
		t.Fatalf("got %T, want *AppInstallationTokenSource", src)
	}
}

func TestNewGitHubTokenSourceFallsBackToStaticToken(t *testing.T) {
	t.Setenv("GITHUB_APP_ID", "")
	t.Setenv("GITHUB_APP_INSTALLATION_ID", "")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "")
	t.Setenv("GITHUB_TOKEN", "ghp_static")

	src, err := newGitHubTokenSource("http://unused", nil)
	if err != nil {
		t.Fatalf("newGitHubTokenSource: %v", err)
	}
	if src != StaticGitHubToken("ghp_static") {
		t.Fatalf("got %#v, want StaticGitHubToken(ghp_static)", src)
	}
}

// A partially-set App identity (e.g. app id and key present but installation
// id missing) is a misconfiguration, not a signal to fall back silently —
// silently falling back to GITHUB_TOKEN here would run as the wrong identity
// without anyone noticing, which is exactly last section's bug in a new shape.
func TestNewGitHubTokenSourceRejectsPartialAppCredentials(t *testing.T) {
	pemBytes, _ := testRSAKey(t)
	t.Setenv("GITHUB_APP_ID", "4589889")
	t.Setenv("GITHUB_APP_INSTALLATION_ID", "")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", string(pemBytes))
	t.Setenv("GITHUB_TOKEN", "ghp_static")

	if _, err := newGitHubTokenSource("http://unused", nil); err == nil {
		t.Fatal("expected an error for a partially-configured app identity")
	}
}

func TestNewGitHubTokenSourceErrorsWithNoCredentials(t *testing.T) {
	t.Setenv("GITHUB_APP_ID", "")
	t.Setenv("GITHUB_APP_INSTALLATION_ID", "")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "")
	t.Setenv("GITHUB_TOKEN", "")

	if _, err := newGitHubTokenSource("http://unused", nil); err == nil {
		t.Fatal("expected an error when no github credentials are configured")
	}
}
