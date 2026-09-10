package auth_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authtest"
)

// The Go half of the cross-implementation parity suite. Its JVM counterpart is
// JwtGoParityTests in the usersrole project, which asserts the mirror image:
// that a token minted here verifies through JwtService.
//
// Signing is the easy half — two HMAC implementations over the same key agree
// or they do not. The half worth a fixture is the claim layout: a token that
// verifies but carries user_id as a string, or omits nbf, would pass every
// signature check and then authorize nobody.
//
// Each fixture lives in the source of the side that consumes it rather than in
// a shared data file, so the directive telling the secrets gate that a token
// signed with a published test key is not a credential sits on the line a
// reviewer reads.

// jvmMintedToken was minted by JwtService.generateToken with paritySecret and
// the deployed two-hour lifetime. Refresh it with the gradle command in the
// README, which writes a replacement to build/parity/jvm-minted-token.json.
const jvmMintedToken = "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJuYmYiOjE3ODg1OTE3NzUsInVzZXJfaWQiOjQyLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvYXV0aC9hdXRoZW50aWNhdGUiLCJqdGkiOiJmZTYwODdkZS04NzE2LTQzYmUtOTYyNC02MGY5M2U1YzIxNzAiLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsImlhdCI6MTc4ODU5MTc3NSwiZXhwIjoxNzg4NTk4OTc1fQ.hOAN1zfivQTS4laaSk899IRfhB1Lg2aijGzV_pB6hqQ" // gitleaks:allow

// The claims that token carries, transcribed from the same dump.
var jvmMintedClaims = struct {
	Subject   string
	Roles     []string
	UserID    int64
	ProfileID int64
	Audience  string
	Issuer    string
	TokenID   string
	IssuedAt  int64
	NotBefore int64
	ExpiresAt int64
}{
	Subject:   "parity@jdw.com",
	Roles:     []string{"ADMIN"},
	UserID:    42,
	ProfileID: 7,
	Audience:  issuerOrigin,
	Issuer:    issuerClaim,
	TokenID:   "fe6087de-8716-43be-9624-60f93e5c2170", // gitleaks:allow
	IssuedAt:  1788591775,
	NotBefore: 1788591775,
	ExpiresAt: 1788598975,
}

func TestATokenMintedByTheJvmVerifiesHere(t *testing.T) {
	// Pinned inside the fixture's own validity window: the token carries the
	// deployed two-hour lifetime, so a wall clock would expire it and the
	// fixture would have to be refreshed on a schedule to stay green.
	at := time.Unix(jvmMintedClaims.IssuedAt, 0).Add(time.Minute)
	v, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  paritySecret,
		ExpectedIssuer:   jvmMintedClaims.Issuer,
		ExpectedAudience: jvmMintedClaims.Audience,
		Now:              func() time.Time { return at },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	p, err := v.Verify(jvmMintedToken)
	if err != nil {
		t.Fatalf("a token minted by JwtService did not verify: %v", err)
	}

	if p.Subject != jvmMintedClaims.Subject {
		t.Errorf("Subject = %q, want %q", p.Subject, jvmMintedClaims.Subject)
	}
	if p.UserID == nil || *p.UserID != jvmMintedClaims.UserID {
		t.Errorf("UserID = %v, want %d", p.UserID, jvmMintedClaims.UserID)
	}
	if p.ProfileID == nil || *p.ProfileID != jvmMintedClaims.ProfileID {
		t.Errorf("ProfileID = %v, want %d", p.ProfileID, jvmMintedClaims.ProfileID)
	}
	if len(p.Roles) != len(jvmMintedClaims.Roles) {
		t.Fatalf("Roles = %v, want %v", p.Roles, jvmMintedClaims.Roles)
	}
	for i, role := range jvmMintedClaims.Roles {
		if p.Roles[i] != role {
			t.Errorf("Roles[%d] = %q, want %q", i, p.Roles[i], role)
		}
	}
	if p.Issuer != jvmMintedClaims.Issuer {
		t.Errorf("Issuer = %q, want %q", p.Issuer, jvmMintedClaims.Issuer)
	}
	if len(p.Audience) != 1 || p.Audience[0] != jvmMintedClaims.Audience {
		t.Errorf("Audience = %v, want [%s]", p.Audience, jvmMintedClaims.Audience)
	}
	if p.TokenID != jvmMintedClaims.TokenID {
		t.Errorf("TokenID = %q, want %q", p.TokenID, jvmMintedClaims.TokenID)
	}
	if p.ExpiresAt.Unix() != jvmMintedClaims.ExpiresAt {
		t.Errorf("ExpiresAt = %d, want %d", p.ExpiresAt.Unix(), jvmMintedClaims.ExpiresAt)
	}
	if p.NotBefore.Unix() != jvmMintedClaims.NotBefore {
		t.Errorf("NotBefore = %d, want %d", p.NotBefore.Unix(), jvmMintedClaims.NotBefore)
	}
}

func TestTheJvmFixtureCarriesTheDeployedTokenLifetime(t *testing.T) {
	got := time.Duration(jvmMintedClaims.ExpiresAt-jvmMintedClaims.NotBefore) * time.Second

	if got != authtest.DefaultTTL {
		t.Errorf("fixture lifetime = %v, want %v; the minter's default has drifted from what the JVM issues", got, authtest.DefaultTTL)
	}
}

// The token the JVM side holds outlives any plausible test run because
// JwtService has no clock seam: extractAllClaims reads the wall clock, so the
// JVM can only exercise its real verification path against one that has not
// expired.
const goFixtureLifetime = 50 * 365 * 24 * time.Hour

// goFixtureIssuedAt is safely in the past because JwtService reads the wall
// clock for nbf as well as exp, and a fixture stamped near the moment it was
// generated would fail on the JVM side for any machine running a little behind.
var goFixtureIssuedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

const goFixtureTokenID = "6a2f1c34-9b7e-4d51-8f0a-2c6d5e4b3a19" // gitleaks:allow

// goMintedToken is the token JwtGoParityTests holds, duplicated here so a
// minter change fails on this side too. The two copies are kept identical by
// TestTheMinterStillProducesTheFixtureLayout, which regenerates and compares.
const goMintedToken = "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.ko6vRk3iGxy_NsN2HQcZxr8CPhRll-8yFm85mh_gC7E" // gitleaks:allow

// TestPrintGoMintedToken regenerates the token JwtGoParityTests holds. It is
// the documented way to refresh that fixture after a claim-layout change, and
// is guarded rather than automatic because its output is pasted into a Java
// source file by hand.
//
//	AUTH_PARITY_PRINT_TOKEN=1 go test . -run TestPrintGoMintedToken -v
func TestPrintGoMintedToken(t *testing.T) {
	if os.Getenv("AUTH_PARITY_PRINT_TOKEN") != "1" {
		t.Skip("set AUTH_PARITY_PRINT_TOKEN=1 to print a replacement for the JVM side's fixture")
	}

	userID, profileID := int64(42), int64(7)
	m := authtest.Minter{
		SecretKeyBase64: paritySecret,
		IssuerOrigin:    issuerOrigin,
		TTL:             goFixtureLifetime,
		Now:             func() time.Time { return goFixtureIssuedAt },
		TokenID:         func() string { return goFixtureTokenID },
	}

	token, err := m.Mint(authtest.Claims{
		Subject: "parity@jdw.com", Roles: []string{"ADMIN"}, UserID: &userID, ProfileID: &profileID,
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	t.Logf("paste into JwtGoParityTests.GO_MINTED_TOKEN:\n%s", token)
	t.Logf("nbf=%d iat=%d exp=%d", goFixtureIssuedAt.Unix(), goFixtureIssuedAt.Unix(),
		goFixtureIssuedAt.Add(goFixtureLifetime).Unix())
}

// The JVM side compares its live mint against the frozen Go token, which pins
// JVM drift. This pins the other direction: a change to the minter's claim
// names, JSON types or header fails here rather than passing unnoticed because
// the regenerator only runs behind an environment variable.
func TestTheMinterStillProducesTheFixtureLayout(t *testing.T) {
	userID, profileID := int64(42), int64(7)
	token, err := authtest.Minter{
		SecretKeyBase64: paritySecret,
		IssuerOrigin:    issuerOrigin,
		TTL:             goFixtureLifetime,
		Now:             func() time.Time { return goFixtureIssuedAt },
		TokenID:         func() string { return goFixtureTokenID },
	}.Mint(authtest.Claims{
		Subject: "parity@jdw.com", Roles: []string{"ADMIN"}, UserID: &userID, ProfileID: &profileID,
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	// Same inputs and the same fixed clock and token id, so a live mint must be
	// byte-identical to the token the JVM side holds.
	if token != goMintedToken {
		t.Errorf("the minter no longer reproduces the token JwtGoParityTests holds.\ngot  %s\nwant %s\nRegenerate both sides: see the README.", token, goMintedToken)
	}

	header, claims := decodeSegments(t, token)
	if len(header) != 1 || header["alg"] != "HS256" {
		t.Errorf("header = %v, want the algorithm alone, as jjwt writes it", header)
	}

	// Claim names and JSON types, which is what a signature check cannot see.
	wantTypes := map[string]string{
		"sub": "string", "roles": "[]interface {}", "user_id": "float64",
		"profile_id": "float64", "aud": "string", "iss": "string",
		"jti": "string", "iat": "float64", "nbf": "float64", "exp": "float64",
	}
	for name, want := range wantTypes {
		value, present := claims[name]
		if !present {
			t.Errorf("claim %s is missing", name)
			continue
		}
		if got := fmt.Sprintf("%T", value); got != want {
			t.Errorf("claim %s has type %s, want %s", name, got, want)
		}
	}
	for name := range claims {
		if _, want := wantTypes[name]; !want {
			t.Errorf("claim %s is not in the layout the JVM writes", name)
		}
	}
}

func decodeSegments(t *testing.T, token string) (header, claims map[string]any) {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d segments, want 3", len(parts))
	}
	for i, into := range []*map[string]any{&header, &claims} {
		raw, err := base64.RawURLEncoding.DecodeString(parts[i])
		if err != nil {
			t.Fatalf("segment %d is not base64url: %v", i, err)
		}
		if err := json.Unmarshal(raw, into); err != nil {
			t.Fatalf("segment %d is not JSON: %v", i, err)
		}
	}
	return header, claims
}

// The padding half of the parity suite. The deployed secret carries no final
// padding character and the JVM has always accepted it, so these assert the
// same two directions as the fixtures above with the padding stripped from the
// secret on this side.
//
// Stripping padding cannot change the key bytes, so a signature check is the
// proof: HMAC over a different key produces a different signature, and the
// tokens below were signed by the other implementation over the padded form.

func TestATokenMintedByTheJvmVerifiesUnderTheUnpaddedSecret(t *testing.T) {
	at := time.Unix(jvmMintedClaims.IssuedAt, 0).Add(time.Minute)
	v, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  paritySecretUnpadded,
		ExpectedIssuer:   jvmMintedClaims.Issuer,
		ExpectedAudience: jvmMintedClaims.Audience,
		Now:              func() time.Time { return at },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	p, err := v.Verify(jvmMintedToken)
	if err != nil {
		t.Fatalf("a token minted by JwtService did not verify against the unpadded form of the secret it was signed with: %v", err)
	}
	if p.Subject != jvmMintedClaims.Subject {
		t.Errorf("Subject = %q, want %q", p.Subject, jvmMintedClaims.Subject)
	}
}

// The token JwtGoParityTests verifies, minted here from the unpadded secret.
// Byte-identical output means the key bytes are identical, which is the whole
// claim: the JVM half of this suite verifies the same string against the padded
// form and against the unpadded one.
func TestTheMinterProducesTheJvmSideFixtureFromTheUnpaddedSecret(t *testing.T) {
	userID, profileID := int64(42), int64(7)
	token, err := authtest.Minter{
		SecretKeyBase64: paritySecretUnpadded,
		IssuerOrigin:    issuerOrigin,
		TTL:             goFixtureLifetime,
		Now:             func() time.Time { return goFixtureIssuedAt },
		TokenID:         func() string { return goFixtureTokenID },
	}.Mint(authtest.Claims{
		Subject: "parity@jdw.com", Roles: []string{"ADMIN"}, UserID: &userID, ProfileID: &profileID,
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	if token != goMintedToken {
		t.Errorf("minting from the unpadded secret produced a different token from the padded one.\ngot  %s\nwant %s", token, goMintedToken)
	}
}

// The algorithm half of the parity suite: one key per HMAC variant jjwt can
// select, plus the deployed shape, each round-tripped in both directions.
//
// parityBandKey builds an n-byte key from a fixed pattern. JwtGoParityTests
// builds the same bytes with the same arithmetic, so neither side carries a
// key literal and the two cannot drift into different keys. The secrets are
// encoded without padding, as the deployed one is.
func parityBandKey(n int) []byte {
	key := make([]byte, n)
	for i := range key {
		key[i] = byte(i*7 + n)
	}
	return key
}

func parityBandSecret(n int) string {
	return base64.RawStdEncoding.EncodeToString(parityBandKey(n))
}

var parityBands = []struct {
	name     string
	keyBytes int
	alg      string
}{
	{"HS256", 32, "HS256"},
	{"HS384", 48, "HS384"},
	{"HS512", 64, "HS512"},
	{"deployed", deployedKeyBytes, "HS512"},
}

func goBandMinter(keyBytes int) authtest.Minter {
	return authtest.Minter{
		SecretKeyBase64: parityBandSecret(keyBytes),
		IssuerOrigin:    issuerOrigin,
		TTL:             goFixtureLifetime,
		Now:             func() time.Time { return goFixtureIssuedAt },
		TokenID:         func() string { return goFixtureTokenID },
	}
}

func mintGoBandToken(t *testing.T, keyBytes int) string {
	t.Helper()
	userID, profileID := int64(42), int64(7)
	token, err := goBandMinter(keyBytes).Mint(authtest.Claims{
		Subject: "parity@jdw.com", Roles: []string{"ADMIN"}, UserID: &userID, ProfileID: &profileID,
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	return token
}

// TestPrintGoMintedBandTokens regenerates JwtGoParityTests.GO_MINTED_BAND_TOKENS.
//
//	AUTH_PARITY_PRINT_TOKEN=1 go test . -run TestPrintGoMintedBandTokens -v
func TestPrintGoMintedBandTokens(t *testing.T) {
	if os.Getenv("AUTH_PARITY_PRINT_TOKEN") != "1" {
		t.Skip("set AUTH_PARITY_PRINT_TOKEN=1 to print replacements for the JVM side's band fixtures")
	}
	for _, band := range parityBands {
		t.Logf("%s (%d-byte key): %s", band.name, band.keyBytes, mintGoBandToken(t, band.keyBytes))
	}
}

// jvmMintedBandTokens were minted by JwtService.generateToken under each band's
// key, and are refreshed from build/parity/jvm-minted-band-*.json by the gradle
// command in the README. The JVM stamps the wall clock, so each carries the
// moment it was minted and is verified with the clock pinned just after it.
var jvmMintedBandTokens = map[string]struct {
	token    string
	issuedAt int64
}{
	"HS256":    {token: "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJuYmYiOjE3ODkwNjQ4MDMsInVzZXJfaWQiOjQyLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvYXV0aC9hdXRoZW50aWNhdGUiLCJqdGkiOiJlMzQ0YWNmMi03MWZlLTQwOGYtYTkyMy1hMzI4ZWU1ZTVkZTQiLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsImlhdCI6MTc4OTA2NDgwMywiZXhwIjoxNzg5MDcyMDAzfQ.G9ksyRohAgs5eIpj_HUl3KJt8eYKMUsYqS5MSMpiE_k", issuedAt: 1789064803},                                            // gitleaks:allow
	"HS384":    {token: "eyJhbGciOiJIUzM4NCJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJuYmYiOjE3ODkwNjQ4MDMsInVzZXJfaWQiOjQyLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvYXV0aC9hdXRoZW50aWNhdGUiLCJqdGkiOiJmYTI3YzZjZS03OTljLTQ1ZDEtYTMxZi1iOGIyZTUwMTE5ODUiLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsImlhdCI6MTc4OTA2NDgwMywiZXhwIjoxNzg5MDcyMDAzfQ.iELxkb3oYHXbCKK0OWBHeJ6MFP9ypX_MZ_sEg51m0bYDnLpNoFuXJMYRaQoH3xJ1", issuedAt: 1789064803},                       // gitleaks:allow
	"HS512":    {token: "eyJhbGciOiJIUzUxMiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJuYmYiOjE3ODkwNjQ4MDMsInVzZXJfaWQiOjQyLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvYXV0aC9hdXRoZW50aWNhdGUiLCJqdGkiOiI5YzlkOTRmNS04YjkyLTRiOGYtYjcyZC1lOTQzMzI1OTJmZTkiLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsImlhdCI6MTc4OTA2NDgwMywiZXhwIjoxNzg5MDcyMDAzfQ.0xUYbaka4oYiaKOivyn1qRrB9c-hdBcXOvSALZNba5L3KcU6tIDpv1rsn1MZv9QOL4tj0MoKby6hFV9OIoENLA", issuedAt: 1789064803}, // gitleaks:allow
	"deployed": {token: "eyJhbGciOiJIUzUxMiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJuYmYiOjE3ODkwNjQ4MDMsInVzZXJfaWQiOjQyLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwODAvYXV0aC9hdXRoZW50aWNhdGUiLCJqdGkiOiIyM2Y3Yjk1Ny0wMTU5LTQxOGYtYjY2NC04ZmQzNmFhNzQyYTAiLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsImlhdCI6MTc4OTA2NDgwMywiZXhwIjoxNzg5MDcyMDAzfQ.Mqr66psPVr_zntXGflxCp7DORUU1t9zfMmQ6Q3SXM8qgfI2iH2Nb4zZYdD-l01d38XvHI5JUJ4JX9q03o6XaWQ", issuedAt: 1789064803}, // gitleaks:allow
}

// goMintedBandTokens are the tokens JwtGoParityTests holds, duplicated here so a
// minter change fails on this side too.
var goMintedBandTokens = map[string]string{
	"HS256":    "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.oURFk6r6xXfZpGWxhCn8f_QYaRmWi6sEBvlNB9tzpew",                                            // gitleaks:allow
	"HS384":    "eyJhbGciOiJIUzM4NCJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.F2LTul5M2rwV48OH0FDSb27NOIDKneKu80CWlPPi8KbBmwgckmHvUr1PVF5o9b9L",                       // gitleaks:allow
	"HS512":    "eyJhbGciOiJIUzUxMiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.7arF5v8mQ33OOYZi_lLcc9-VMmSZkLddYm7VjX2BjHvrrow_cri8HCW7huKDRQRx7gHs_r4smwcmVSuyNIZoKg", // gitleaks:allow
	"deployed": "eyJhbGciOiJIUzUxMiJ9.eyJhdWQiOiJodHRwOi8vbG9jYWxob3N0OjgwODAiLCJleHAiOjMzMTI0ODk2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL2F1dGgvYXV0aGVudGljYXRlIiwianRpIjoiNmEyZjFjMzQtOWI3ZS00ZDUxLThmMGEtMmM2ZDVlNGIzYTE5IiwibmJmIjoxNzM1Njg5NjAwLCJwcm9maWxlX2lkIjo3LCJyb2xlcyI6WyJBRE1JTiJdLCJzdWIiOiJwYXJpdHlAamR3LmNvbSIsInVzZXJfaWQiOjQyfQ.VVo34tIkRkzj8Iu5qAtQHizjv8EGQQU1djVmtZDu48uVApfn7o-DSxsiCTFtraEu40kRKHAvc9jIcmaez6TdvQ", // gitleaks:allow
}

func TestATokenMintedByTheJvmVerifiesHereInEveryKeyLengthBand(t *testing.T) {
	for _, band := range parityBands {
		t.Run(band.name, func(t *testing.T) {
			fixture, ok := jvmMintedBandTokens[band.name]
			if !ok {
				t.Fatalf("no JVM fixture for the %s band", band.name)
			}
			if header, _ := decodeSegments(t, fixture.token); header["alg"] != band.alg {
				t.Fatalf("the JVM signed with %v, want %s; the fixture no longer shows what jjwt derives", header["alg"], band.alg)
			}

			at := time.Unix(fixture.issuedAt, 0).Add(time.Minute)
			v, err := auth.NewVerifier(auth.Config{
				SecretKeyBase64:  parityBandSecret(band.keyBytes),
				ExpectedIssuer:   issuerClaim,
				ExpectedAudience: issuerOrigin,
				Now:              func() time.Time { return at },
			})
			if err != nil {
				t.Fatalf("NewVerifier: %v", err)
			}

			p, err := v.Verify(fixture.token)
			if err != nil {
				t.Fatalf("a %s token minted by JwtService did not verify: %v", band.name, err)
			}
			if p.Subject != "parity@jdw.com" || p.UserID == nil || *p.UserID != 42 {
				t.Errorf("principal = %s / %v, want parity@jdw.com / 42", p.Subject, p.UserID)
			}
		})
	}
}

// Same inputs, same fixed clock and token id: a live mint must be byte-identical
// to the token the JVM side holds for the band, which pins both the variant the
// minter derives and the key bytes it derives it from.
func TestTheMinterStillProducesTheBandFixtures(t *testing.T) {
	for _, band := range parityBands {
		t.Run(band.name, func(t *testing.T) {
			if got, want := mintGoBandToken(t, band.keyBytes), goMintedBandTokens[band.name]; got != want {
				t.Errorf("the minter no longer reproduces the %s token JwtGoParityTests holds.\ngot  %s\nwant %s\nRegenerate both sides: see the README.", band.name, got, want)
			}
		})
	}
}
