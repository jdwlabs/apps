package auth_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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

type parityFixture struct {
	Description     string `json:"description"`
	SecretKeyBase64 string `json:"secretKeyBase64"`
	IssuerOrigin    string `json:"issuerOrigin"`
	Token           string `json:"token"`
	Claims          struct {
		Subject   string   `json:"sub"`
		Roles     []string `json:"roles"`
		UserID    int64    `json:"user_id"`
		ProfileID int64    `json:"profile_id"`
		Audience  string   `json:"aud"`
		Issuer    string   `json:"iss"`
		TokenID   string   `json:"jti"`
		IssuedAt  int64    `json:"iat"`
		NotBefore int64    `json:"nbf"`
		ExpiresAt int64    `json:"exp"`
	} `json:"claims"`
}

func readFixture(t *testing.T, name string) parityFixture {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture parityFixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return fixture
}

func TestATokenMintedByTheJvmVerifiesHere(t *testing.T) {
	fixture := readFixture(t, "jvm-minted-token.json")
	// Pinned inside the fixture's own validity window: the token carries the
	// deployed two-hour lifetime, so a wall clock would expire it and the
	// fixture would have to be refreshed on a schedule to stay green.
	at := time.Unix(fixture.Claims.IssuedAt, 0).Add(time.Minute)
	v, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  fixture.SecretKeyBase64,
		ExpectedIssuer:   fixture.Claims.Issuer,
		ExpectedAudience: fixture.IssuerOrigin,
		Now:              func() time.Time { return at },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	p, err := v.Verify(fixture.Token)
	if err != nil {
		t.Fatalf("a token minted by JwtService did not verify: %v", err)
	}

	if p.Subject != fixture.Claims.Subject {
		t.Errorf("Subject = %q, want %q", p.Subject, fixture.Claims.Subject)
	}
	if p.UserID == nil || *p.UserID != fixture.Claims.UserID {
		t.Errorf("UserID = %v, want %d", p.UserID, fixture.Claims.UserID)
	}
	if p.ProfileID == nil || *p.ProfileID != fixture.Claims.ProfileID {
		t.Errorf("ProfileID = %v, want %d", p.ProfileID, fixture.Claims.ProfileID)
	}
	if len(p.Roles) != len(fixture.Claims.Roles) {
		t.Fatalf("Roles = %v, want %v", p.Roles, fixture.Claims.Roles)
	}
	for i, role := range fixture.Claims.Roles {
		if p.Roles[i] != role {
			t.Errorf("Roles[%d] = %q, want %q", i, p.Roles[i], role)
		}
	}
	if p.Issuer != fixture.Claims.Issuer {
		t.Errorf("Issuer = %q, want %q", p.Issuer, fixture.Claims.Issuer)
	}
	if len(p.Audience) != 1 || p.Audience[0] != fixture.Claims.Audience {
		t.Errorf("Audience = %v, want [%s]", p.Audience, fixture.Claims.Audience)
	}
	if p.TokenID != fixture.Claims.TokenID {
		t.Errorf("TokenID = %q, want %q", p.TokenID, fixture.Claims.TokenID)
	}
	if p.ExpiresAt.Unix() != fixture.Claims.ExpiresAt {
		t.Errorf("ExpiresAt = %d, want %d", p.ExpiresAt.Unix(), fixture.Claims.ExpiresAt)
	}
	if p.NotBefore.Unix() != fixture.Claims.NotBefore {
		t.Errorf("NotBefore = %d, want %d", p.NotBefore.Unix(), fixture.Claims.NotBefore)
	}
}

func TestTheJvmFixtureCarriesTheDeployedTokenLifetime(t *testing.T) {
	fixture := readFixture(t, "jvm-minted-token.json")

	if got := time.Duration(fixture.Claims.ExpiresAt-fixture.Claims.IssuedAt) * time.Second; got != authtest.DefaultTTL {
		t.Errorf("fixture lifetime = %v, want %v; the minter's default has drifted from what the JVM issues", got, authtest.DefaultTTL)
	}
}

func TestTheGoFixtureIsTheOneTheJvmSideReads(t *testing.T) {
	// The JVM parity test verifies this fixture through JwtService and compares
	// its claim set against a token it mints itself. Asserting it verifies here
	// too keeps a corrupted or hand-edited fixture from being diagnosed on the
	// wrong side of the boundary.
	fixture := readFixture(t, "go-minted-token.json")
	v, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  fixture.SecretKeyBase64,
		ExpectedIssuer:   fixture.Claims.Issuer,
		ExpectedAudience: fixture.IssuerOrigin,
		Now:              func() time.Time { return time.Unix(fixture.Claims.IssuedAt, 0).Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	if _, err := v.Verify(fixture.Token); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

// The Go fixture outlives any plausible test run because JwtService has no
// clock seam: extractAllClaims reads the wall clock, so the JVM side can only
// exercise its real verification path against a token that has not expired.
const goFixtureLifetime = 50 * 365 * 24 * time.Hour

// TestWriteGoMintedFixture regenerates testdata/go-minted-token.json. It is the
// documented way to refresh the fixture after a claim-layout change, and is
// guarded rather than automatic so an ordinary test run never rewrites a file
// the JVM side asserts against.
//
//	AUTH_PARITY_WRITE_FIXTURE=1 go test ./... -run TestWriteGoMintedFixture
func TestWriteGoMintedFixture(t *testing.T) {
	if os.Getenv("AUTH_PARITY_WRITE_FIXTURE") != "1" {
		t.Skip("set AUTH_PARITY_WRITE_FIXTURE=1 to regenerate testdata/go-minted-token.json")
	}

	// Safely in the past: JwtService reads the wall clock for nbf as well as
	// exp, so a fixture stamped near the moment it was generated would fail on
	// the JVM side for any machine running a little behind.
	issuedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	userID, profileID := int64(42), int64(7)
	m := authtest.Minter{
		SecretKeyBase64: paritySecret,
		IssuerOrigin:    issuerOrigin,
		TTL:             goFixtureLifetime,
		Now:             func() time.Time { return issuedAt },
		TokenID:         func() string { return "6a2f1c34-9b7e-4d51-8f0a-2c6d5e4b3a19" },
	}
	token, err := m.Mint(authtest.Claims{
		Subject: "parity@jdw.com", Roles: []string{"ADMIN"}, UserID: &userID, ProfileID: &profileID,
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	var fixture parityFixture
	fixture.Description = "Minted by the Go library's test minter and verified by JwtService on the JVM side. " +
		"Its lifetime is deliberately far longer than a real token: JwtService reads the wall clock, so a " +
		"realistic expiry would make the JVM assertion time out rather than test anything."
	fixture.SecretKeyBase64 = paritySecret
	fixture.IssuerOrigin = issuerOrigin
	fixture.Token = token
	fixture.Claims.Subject = "parity@jdw.com"
	fixture.Claims.Roles = []string{"ADMIN"}
	fixture.Claims.UserID = userID
	fixture.Claims.ProfileID = profileID
	fixture.Claims.Audience = issuerOrigin
	fixture.Claims.Issuer = issuerClaim
	fixture.Claims.TokenID = "6a2f1c34-9b7e-4d51-8f0a-2c6d5e4b3a19"
	fixture.Claims.IssuedAt = issuedAt.Unix()
	fixture.Claims.NotBefore = issuedAt.Unix()
	fixture.Claims.ExpiresAt = issuedAt.Add(goFixtureLifetime).Unix()

	content, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join("testdata", "go-minted-token.json"), append(content, '\n'), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
