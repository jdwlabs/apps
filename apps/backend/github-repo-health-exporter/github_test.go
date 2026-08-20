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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating test key: %v", err)
	}
	return key
}

func pemEncodePKCS1(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}

func pemEncodePKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshalling PKCS8: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b})
}

func TestParseRSAPrivateKey_BothEncodings(t *testing.T) {
	key := testKey(t)

	pkcs1, err := parseRSAPrivateKey(pemEncodePKCS1(t, key))
	if err != nil {
		t.Fatalf("PKCS1: %v", err)
	}
	if !pkcs1.Equal(key) {
		t.Error("PKCS1: parsed key does not match original")
	}

	pkcs8, err := parseRSAPrivateKey(pemEncodePKCS8(t, key))
	if err != nil {
		t.Fatalf("PKCS8: %v", err)
	}
	if !pkcs8.Equal(key) {
		t.Error("PKCS8: parsed key does not match original")
	}
}

func TestParseRSAPrivateKey_Invalid(t *testing.T) {
	if _, err := parseRSAPrivateKey([]byte("not a pem block")); err == nil {
		t.Error("want error for non-PEM input, got nil")
	}
}

// TestSignAppJWT_VerifiesAndCarriesExpectedClaims signs a JWT with the
// production code path and then verifies it independently with only the
// public key and stdlib primitives — the same check GitHub's own JWT
// verifier performs — so a padding or encoding mistake in signAppJWT fails
// this test rather than surfacing for the first time against the live API.
func TestSignAppJWT_VerifiesAndCarriesExpectedClaims(t *testing.T) {
	key := testKey(t)
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

	token, err := signAppJWT("123456", key, now)
	if err != nil {
		t.Fatalf("signAppJWT: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("want 3 dot-separated JWT segments, got %d (%q)", len(parts), token)
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decoding header: %v", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		t.Fatalf("unmarshalling header: %v", err)
	}
	if header.Alg != "RS256" || header.Typ != "JWT" {
		t.Errorf("header = %+v, want alg=RS256 typ=JWT", header)
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decoding claims: %v", err)
	}
	var claims struct {
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
		Iss string `json:"iss"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		t.Fatalf("unmarshalling claims: %v", err)
	}
	if claims.Iss != "123456" {
		t.Errorf("iss = %q, want 123456", claims.Iss)
	}
	wantIat := now.Add(-appJWTClockSkew).Unix()
	if claims.Iat != wantIat {
		t.Errorf("iat = %d, want %d (now - clock skew)", claims.Iat, wantIat)
	}
	if got, want := claims.Exp-claims.Iat, int64(appJWTLifetime/time.Second); got != want {
		t.Errorf("exp-iat = %ds, want %ds", got, want)
	}
	if claims.Exp-now.Unix() > int64(10*time.Minute/time.Second) {
		t.Errorf("exp is %ds past now, exceeds GitHub's 10-minute cap", claims.Exp-now.Unix())
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decoding signature: %v", err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], sig); err != nil {
		t.Errorf("signature does not verify against the signer's own public key: %v", err)
	}
}

func TestInstallationToken_CachesUntilNearExpiry(t *testing.T) {
	key := testKey(t)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/installations/999/access_tokens" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		calls++
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token":      "tok-" + string(rune('a'+calls-1)),
			"expires_at": "2026-08-20T13:00:00Z",
		})
	}))
	defer srv.Close()

	gh := NewGitHubClient(srv.URL, "1", "999", key, srv.Client())
	base := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	cur := base
	gh.nowFn = func() time.Time { return cur }

	tok1, err := gh.installationToken(context.Background())
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if calls != 1 {
		t.Fatalf("want 1 HTTP call after first installationToken, got %d", calls)
	}

	// Still well inside the hour-long lifetime minus the refresh margin: must
	// reuse the cached token, not mint a second one.
	cur = base.Add(10 * time.Minute)
	tok2, err := gh.installationToken(context.Background())
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if calls != 1 {
		t.Fatalf("want 1 HTTP call after cached reuse, got %d", calls)
	}
	if tok1 != tok2 {
		t.Errorf("token changed on a cached call: %q -> %q", tok1, tok2)
	}

	// Inside refreshMargin of the cached expiry: must mint a new one rather
	// than risk presenting a token that expires mid-request.
	cur = base.Add(installationTokenLifetime - refreshMargin + time.Second)
	tok3, err := gh.installationToken(context.Background())
	if err != nil {
		t.Fatalf("third call: %v", err)
	}
	if calls != 2 {
		t.Fatalf("want 2 HTTP calls after crossing the refresh margin, got %d", calls)
	}
	if tok3 == tok2 {
		t.Error("token did not change after crossing the refresh margin")
	}
}

func TestListMainRuns_FiltersToCompletedAndUsesBranchMain(t *testing.T) {
	key := testKey(t)
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/access_tokens"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok", "expires_at": "2026-08-20T13:00:00Z"})
		case strings.Contains(r.URL.Path, "/actions/runs"):
			gotQuery = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workflow_runs": []map[string]any{
					{"name": "prd Drift", "status": "completed", "conclusion": "failure", "created_at": "2026-08-19T00:00:00Z"},
					{"name": "prd Drift", "status": "in_progress", "conclusion": "", "created_at": "2026-08-20T00:00:00Z"},
					{"name": "CI", "status": "completed", "conclusion": "success", "created_at": "2026-08-19T00:00:00Z"},
				},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	gh := NewGitHubClient(srv.URL, "1", "999", key, srv.Client())
	runs, err := gh.ListMainRuns(context.Background(), "jdwlabs/deployments")
	if err != nil {
		t.Fatalf("ListMainRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("want 2 completed runs (in_progress excluded), got %d: %+v", len(runs), runs)
	}
	if !strings.Contains(gotQuery, "branch=main") {
		t.Errorf("query %q does not scope to branch=main", gotQuery)
	}
}
