package auth_test

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authtest"
)

// Byte-identical to the secret the JVM JwtService unit tests inject.
const paritySecret = "bXl0dGVzdHNlY3JldGtleWZvcmpzb253d2VidG9rZW4xMjM0NTY3ODkwIC1uCg=="

const (
	issuerOrigin = "http://localhost:8080"
	issuerClaim  = issuerOrigin + "/auth/authenticate"
)

var mintedAt = time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

func minter() authtest.Minter {
	return authtest.Minter{
		SecretKeyBase64: paritySecret,
		IssuerOrigin:    issuerOrigin,
		Now:             func() time.Time { return mintedAt },
	}
}

func verifier(t *testing.T) *auth.Verifier {
	t.Helper()
	v, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  paritySecret,
		ExpectedIssuer:   issuerClaim,
		ExpectedAudience: issuerOrigin,
		Now:              func() time.Time { return mintedAt.Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return v
}

func mint(t *testing.T, c authtest.Claims) string {
	t.Helper()
	token, err := minter().Mint(c)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	return token
}

func ptr(v int64) *int64 { return &v }

func TestVerifyReturnsThePrincipalTheClaimsDescribe(t *testing.T) {
	token := mint(t, authtest.Claims{
		Subject:   "user@jdw.com",
		Roles:     []string{"ADMIN", "MANAGER"},
		UserID:    ptr(42),
		ProfileID: ptr(7),
	})

	p, err := verifier(t).Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if p.Subject != "user@jdw.com" {
		t.Errorf("Subject = %q, want user@jdw.com", p.Subject)
	}
	if p.EmailAddress() != "user@jdw.com" {
		t.Errorf("EmailAddress() = %q, want user@jdw.com", p.EmailAddress())
	}
	if got := p.UserID; got == nil || *got != 42 {
		t.Errorf("UserID = %v, want 42", got)
	}
	if got := p.ProfileID; got == nil || *got != 7 {
		t.Errorf("ProfileID = %v, want 7", got)
	}
	if !p.HasAuthority("ADMIN") || !p.HasAuthority("MANAGER") {
		t.Errorf("Roles = %v, want ADMIN and MANAGER granted", p.Roles)
	}
	if p.HasAuthority("admin") {
		t.Error("HasAuthority is case-insensitive; the JVM compares authority names exactly")
	}
	if !p.HasAnyAuthority("ADMIN", "MANAGER") {
		t.Error("HasAnyAuthority(ADMIN, MANAGER) = false, want true")
	}
	if p.HasAnyAuthority("USER") {
		t.Error("HasAnyAuthority(USER) = true, want false")
	}
	if p.Issuer != issuerClaim {
		t.Errorf("Issuer = %q, want %q", p.Issuer, issuerClaim)
	}
	if len(p.Audience) != 1 || p.Audience[0] != issuerOrigin {
		t.Errorf("Audience = %v, want [%s]", p.Audience, issuerOrigin)
	}
	if p.TokenID == "" {
		t.Error("TokenID is empty")
	}
	if !p.ExpiresAt.Equal(mintedAt.Add(authtest.DefaultTTL)) {
		t.Errorf("ExpiresAt = %v, want %v", p.ExpiresAt, mintedAt.Add(authtest.DefaultTTL))
	}
	if !p.NotBefore.Equal(mintedAt) {
		t.Errorf("NotBefore = %v, want %v", p.NotBefore, mintedAt)
	}
	if !p.IssuedAt.Equal(mintedAt) {
		t.Errorf("IssuedAt = %v, want %v", p.IssuedAt, mintedAt)
	}
}

func TestVerifyTreatsANullProfileClaimAsAbsent(t *testing.T) {
	token := mint(t, authtest.Claims{Subject: "user@jdw.com", UserID: ptr(42)})

	p, err := verifier(t).Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if p.ProfileID != nil {
		t.Errorf("ProfileID = %v, want nil so the profile fallback engages", *p.ProfileID)
	}
}

func TestVerifyTreatsAMissingProfileClaimAsAbsent(t *testing.T) {
	token, err := minter().MintRaw("HS256", map[string]any{
		"sub": "user@jdw.com", "roles": []string{}, "user_id": 42,
		"iss": issuerClaim, "aud": issuerOrigin, "jti": "id",
		"nbf": mintedAt.Unix(), "exp": mintedAt.Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("MintRaw: %v", err)
	}

	p, err := verifier(t).Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if p.ProfileID != nil {
		t.Errorf("ProfileID = %v, want nil", *p.ProfileID)
	}
}

func TestVerifyRejectsBadTokens(t *testing.T) {
	baseClaims := func() map[string]any {
		return map[string]any{
			"sub": "user@jdw.com", "roles": []string{"ADMIN"}, "user_id": 42, "profile_id": 7,
			"iss": issuerClaim, "aud": issuerOrigin, "jti": "a-token-id",
			"iat": mintedAt.Unix(), "nbf": mintedAt.Unix(),
			"exp": mintedAt.Add(authtest.DefaultTTL).Unix(),
		}
	}
	without := func(key string) map[string]any {
		c := baseClaims()
		delete(c, key)
		return c
	}
	with := func(key string, value any) map[string]any {
		c := baseClaims()
		c[key] = value
		return c
	}

	tests := []struct {
		name  string
		token func(t *testing.T) string
		want  error
	}{
		{"expired", func(t *testing.T) string {
			return mintRaw(t, "HS256", with("exp", mintedAt.Add(-time.Second).Unix()))
		}, auth.ErrTokenExpired},
		{"not yet valid", func(t *testing.T) string {
			return mintRaw(t, "HS256", with("nbf", mintedAt.Add(time.Hour).Unix()))
		}, auth.ErrTokenNotYetValid},
		{"wrong issuer", func(t *testing.T) string {
			return mintRaw(t, "HS256", with("iss", "http://evil.example/auth/authenticate"))
		}, auth.ErrInvalidIssuer},
		{"wrong audience", func(t *testing.T) string {
			return mintRaw(t, "HS256", with("aud", "http://evil.example"))
		}, auth.ErrInvalidAudience},
		{"missing issuer", func(t *testing.T) string { return mintRaw(t, "HS256", without("iss")) }, auth.ErrMissingClaim},
		{"missing audience", func(t *testing.T) string { return mintRaw(t, "HS256", without("aud")) }, auth.ErrMissingClaim},
		{"missing token id", func(t *testing.T) string { return mintRaw(t, "HS256", without("jti")) }, auth.ErrMissingClaim},
		{"missing not before", func(t *testing.T) string { return mintRaw(t, "HS256", without("nbf")) }, auth.ErrMissingClaim},
		{"missing expiry", func(t *testing.T) string { return mintRaw(t, "HS256", without("exp")) }, auth.ErrMissingClaim},
		{"missing subject", func(t *testing.T) string { return mintRaw(t, "HS256", without("sub")) }, auth.ErrMissingClaim},
		{"empty subject", func(t *testing.T) string { return mintRaw(t, "HS256", with("sub", "")) }, auth.ErrMissingClaim},
		{"algorithm none", func(t *testing.T) string { return mintRaw(t, "none", baseClaims()) }, auth.ErrUnexpectedAlgorithm},
		{"algorithm HS384", func(t *testing.T) string { return mintRaw(t, "HS384", baseClaims()) }, auth.ErrUnexpectedAlgorithm},
		{"algorithm HS512", func(t *testing.T) string { return mintRaw(t, "HS512", baseClaims()) }, auth.ErrUnexpectedAlgorithm},
		{"tampered signature", func(t *testing.T) string {
			return authtest.TamperSignature(mintRaw(t, "HS256", baseClaims()))
		}, auth.ErrInvalidSignature},
		{"signed with another key", func(t *testing.T) string {
			// A second published test key, so the failure is a genuine
			// signature mismatch rather than a malformed token.
			const otherSecret = "b3RoZXJzZWNyZXRrZXlmb3Jqc29ud2VidG9rZW4xMjM0NTY3ODkwYWJjZGU=" // gitleaks:allow
			other := authtest.Minter{SecretKeyBase64: otherSecret, IssuerOrigin: issuerOrigin, Now: func() time.Time { return mintedAt }}
			token, err := other.MintRaw("HS256", baseClaims())
			if err != nil {
				t.Fatalf("MintRaw: %v", err)
			}
			return token
		}, auth.ErrInvalidSignature},
		{"not a token", func(_ *testing.T) string { return "not-a-token" }, auth.ErrMalformedToken},
		{"empty", func(_ *testing.T) string { return "" }, auth.ErrMalformedToken},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := verifier(t).Verify(tc.token(t))
			if !errors.Is(err, tc.want) {
				t.Errorf("Verify error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestVerifyRejectsClaimsOfTheWrongType(t *testing.T) {
	// A claim that verifies but carries the wrong JSON type is the failure a
	// signature check cannot see, and the one a drifting minter would produce.
	base := map[string]any{
		"sub": "user@jdw.com", "roles": []string{"ADMIN"}, "user_id": 42, "profile_id": 7,
		"iss": issuerClaim, "aud": issuerOrigin, "jti": "a-token-id",
		"nbf": mintedAt.Unix(), "exp": mintedAt.Add(authtest.DefaultTTL).Unix(),
	}
	tests := []struct {
		name  string
		claim string
		value any
	}{
		{"user id as a string", "user_id", "42"},
		{"profile id as a string", "profile_id", "7"},
		{"roles as a bare string", "roles", "ADMIN"},
		{"roles holding a number", "roles", []any{1}},
		{"subject as a number", "sub", 42},
		{"audience as a number", "aud", 42},
		{"expiry as a string", "exp", "later"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims := map[string]any{}
			for k, v := range base {
				claims[k] = v
			}
			claims[tc.claim] = tc.value

			_, err := verifier(t).Verify(mintRaw(t, "HS256", claims))

			if !errors.Is(err, auth.ErrInvalidClaim) && !errors.Is(err, auth.ErrMalformedToken) {
				t.Errorf("Verify error = %v, want it to reject the claim", err)
			}
		})
	}
}

func TestVerifyAcceptsAnyIssuerAndAudienceOnlyWhenAskedTo(t *testing.T) {
	v, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:           paritySecret,
		AllowAnyIssuerAndAudience: true,
		Now:                       func() time.Time { return mintedAt.Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	// A foreign origin, so the test fails if the value checks still run. Minting
	// with the expected origin would pass either way and prove nothing.
	foreign := authtest.Minter{
		SecretKeyBase64: paritySecret,
		IssuerOrigin:    "http://other.example",
		Now:             func() time.Time { return mintedAt },
	}
	token, err := foreign.Mint(authtest.Claims{Subject: "user@jdw.com", UserID: ptr(42)})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	p, err := v.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if p.Issuer != "http://other.example/auth/authenticate" {
		t.Errorf("Issuer = %q, want the foreign issuer to have been accepted", p.Issuer)
	}
}

// The flag and an expected value state opposite intentions. Honouring one
// silently would leave the configuration reading as though the other applied.
func TestNewVerifierRejectsTheAnyFlagAlongsideAnExpectedValue(t *testing.T) {
	tests := []struct {
		name     string
		issuer   string
		audience string
	}{
		{"with an issuer", issuerClaim, ""},
		{"with an audience", "", issuerOrigin},
		{"with both", issuerClaim, issuerOrigin},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := auth.NewVerifier(auth.Config{
				SecretKeyBase64:           paritySecret,
				ExpectedIssuer:            tc.issuer,
				ExpectedAudience:          tc.audience,
				AllowAnyIssuerAndAudience: true,
			})
			if !errors.Is(err, auth.ErrConflictingExpectedClaims) {
				t.Errorf("error = %v, want %v", err, auth.ErrConflictingExpectedClaims)
			}
		})
	}
}

func TestVerifyAppliesClockLeeway(t *testing.T) {
	v, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  paritySecret,
		ExpectedIssuer:   issuerClaim,
		ExpectedAudience: issuerOrigin,
		Leeway:           30 * time.Second,
		Now:              func() time.Time { return mintedAt.Add(authtest.DefaultTTL + 10*time.Second) },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	token := mint(t, authtest.Claims{Subject: "user@jdw.com", UserID: ptr(42)})

	if _, err := v.Verify(token); err != nil {
		t.Fatalf("Verify inside the leeway window: %v", err)
	}
}

// An unset ExpectedIssuer or ExpectedAudience used to disable the check
// silently, which is the failure mode with no symptom: tokens from any issuer
// verify and nothing says so.
func TestNewVerifierRequiresAnExpectedIssuerAndAudience(t *testing.T) {
	tests := []struct {
		name     string
		issuer   string
		audience string
	}{
		{"neither", "", ""},
		{"issuer only", issuerClaim, ""},
		{"audience only", "", issuerOrigin},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := auth.NewVerifier(auth.Config{
				SecretKeyBase64:  paritySecret,
				ExpectedIssuer:   tc.issuer,
				ExpectedAudience: tc.audience,
			})
			if !errors.Is(err, auth.ErrMissingExpectedClaims) {
				t.Errorf("error = %v, want %v", err, auth.ErrMissingExpectedClaims)
			}
		})
	}
}

// Algorithm confusion: a token whose header claims an asymmetric algorithm must
// be refused on the header alone, before anything treats the HMAC secret as a
// public key. The signature bytes are deliberately meaningless — reaching the
// point where they matter would already be the bug.
func TestVerifyRejectsAnAsymmetricAlgorithmHeader(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user@jdw.com","iss":"` +
		issuerClaim + `","aud":"` + issuerOrigin + `","jti":"id","nbf":1,"exp":99999999999}`))
	token := header + "." + payload + "." + base64.RawURLEncoding.EncodeToString([]byte("not-a-signature"))

	_, err := verifier(t).Verify(token)

	if !errors.Is(err, auth.ErrUnexpectedAlgorithm) {
		t.Errorf("Verify error = %v, want %v", err, auth.ErrUnexpectedAlgorithm)
	}
}

func TestNewVerifierRejectsAnUnusableSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   error
	}{
		{"empty", "", auth.ErrMissingSecretKey},
		{"not base64", "!!!not base64!!!", auth.ErrInvalidSecretKey},
		{"too short for HS256", "c2hvcnQ=", auth.ErrInvalidSecretKey},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := auth.NewVerifier(auth.Config{
				SecretKeyBase64:  tc.secret,
				ExpectedIssuer:   issuerClaim,
				ExpectedAudience: issuerOrigin,
			})
			if !errors.Is(err, tc.want) {
				t.Errorf("NewVerifier error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestVerifyAuthorizationHeader(t *testing.T) {
	token := mint(t, authtest.Claims{Subject: "user@jdw.com", UserID: ptr(42)})
	tests := []struct {
		name   string
		header string
		want   error
	}{
		{"bearer token", "Bearer " + token, nil},
		{"no header", "", auth.ErrMissingBearerToken},
		{"wrong scheme", "Basic " + token, auth.ErrMissingBearerToken},
		{"lowercase scheme", "bearer " + token, auth.ErrMissingBearerToken},
		{"scheme only", "Bearer ", auth.ErrMalformedToken},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := verifier(t).VerifyAuthorizationHeader(tc.header)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("VerifyAuthorizationHeader: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestSecretKeyFromEnvReadsTheNameSpringReads(t *testing.T) {
	t.Setenv(auth.SecretKeyEnvVar, paritySecret)

	got, err := auth.SecretKeyFromEnv()
	if err != nil {
		t.Fatalf("SecretKeyFromEnv: %v", err)
	}

	if got != paritySecret {
		t.Errorf("SecretKeyFromEnv() = %q, want the value of %s", got, auth.SecretKeyEnvVar)
	}
	if auth.SecretKeyEnvVar != "UR_JWT_SECRET_KEY" {
		t.Errorf("SecretKeyEnvVar = %q; both services must read the same name Spring reads", auth.SecretKeyEnvVar)
	}
}

func TestSecretKeyFromEnvFailsWhenUnset(t *testing.T) {
	t.Setenv(auth.SecretKeyEnvVar, "")

	if _, err := auth.SecretKeyFromEnv(); !errors.Is(err, auth.ErrMissingSecretKey) {
		t.Errorf("error = %v, want %v", err, auth.ErrMissingSecretKey)
	}
}

func mintRaw(t *testing.T, alg string, claims map[string]any) string {
	t.Helper()
	token, err := minter().MintRaw(alg, claims)
	if err != nil {
		t.Fatalf("MintRaw: %v", err)
	}
	return token
}
