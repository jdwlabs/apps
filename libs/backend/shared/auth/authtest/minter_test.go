package authtest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Byte-identical to the secret the JVM JwtService unit tests inject, so a
// fixture minted on either side verifies on the other without re-keying.
const paritySecret = "bXl0dGVzdHNlY3JldGtleWZvcmpzb253d2VidG9rZW4xMjM0NTY3ODkwIC1uCg=="

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

func TestMintProducesTheJvmClaimLayout(t *testing.T) {
	mintedAt := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	userID := int64(42)
	profileID := int64(7)
	m := Minter{
		SecretKeyBase64: paritySecret,
		IssuerOrigin:    "http://localhost:8080",
		TTL:             2 * time.Hour,
		Now:             func() time.Time { return mintedAt },
		TokenID:         func() string { return "11111111-2222-4333-8444-555555555555" },
	}

	token, err := m.Mint(Claims{
		Subject:   "user@jdw.com",
		Roles:     []string{"ADMIN"},
		UserID:    &userID,
		ProfileID: &profileID,
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	header, claims := decodeSegments(t, token)
	if header["alg"] != "HS256" {
		t.Errorf("alg = %v, want HS256", header["alg"])
	}
	if _, present := header["typ"]; present {
		t.Errorf("header carries typ=%v; jjwt writes the algorithm alone, and a fixture is only comparable if the header matches", header["typ"])
	}
	if len(header) != 1 {
		t.Errorf("header = %v, want the algorithm alone", header)
	}

	want := map[string]any{
		"sub":        "user@jdw.com",
		"roles":      []any{"ADMIN"},
		"user_id":    float64(42),
		"profile_id": float64(7),
		"aud":        "http://localhost:8080",
		"iss":        "http://localhost:8080/auth/authenticate",
		"jti":        "11111111-2222-4333-8444-555555555555",
		"nbf":        float64(mintedAt.Unix()),
		"iat":        float64(mintedAt.Unix()),
		"exp":        float64(mintedAt.Add(2 * time.Hour).Unix()),
	}
	gotJSON, _ := json.Marshal(claims)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("claims =\n%s\nwant\n%s", gotJSON, wantJSON)
	}
}

func TestMintOmitsNothingWhenTheProfileIsAbsent(t *testing.T) {
	userID := int64(42)
	m := Minter{SecretKeyBase64: paritySecret, IssuerOrigin: "http://localhost:8080"}

	token, err := m.Mint(Claims{Subject: "user@jdw.com", Roles: []string{}, UserID: &userID})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	_, claims := decodeSegments(t, token)
	value, present := claims["profile_id"]
	if !present {
		t.Fatal("profile_id absent; the JVM writes an explicit null so the claim set does not change shape")
	}
	if value != nil {
		t.Errorf("profile_id = %v, want null", value)
	}
}

func TestMintDefaultsMatchTheDeployedConfiguration(t *testing.T) {
	before := time.Now()
	m := Minter{SecretKeyBase64: paritySecret, IssuerOrigin: "http://localhost:8080"}

	token, err := m.Mint(Claims{Subject: "user@jdw.com"})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	_, claims := decodeSegments(t, token)
	iat := int64(claims["iat"].(float64))
	exp := int64(claims["exp"].(float64))
	if exp-iat != int64((2 * time.Hour).Seconds()) {
		t.Errorf("TTL = %ds, want %ds", exp-iat, int64((2 * time.Hour).Seconds()))
	}
	if iat < before.Unix()-2 || iat > time.Now().Unix()+2 {
		t.Errorf("iat = %d, want approximately now", iat)
	}
	if claims["jti"] == "" || claims["jti"] == nil {
		t.Error("jti is empty; every minted token needs a distinct id")
	}
}

func TestMintRawSignsWithTheRequestedAlgorithm(t *testing.T) {
	m := Minter{SecretKeyBase64: paritySecret, IssuerOrigin: "http://localhost:8080"}

	for _, alg := range []string{"HS384", "HS512", "none"} {
		token, err := m.MintRaw(alg, map[string]any{"sub": "user@jdw.com"})
		if err != nil {
			t.Fatalf("MintRaw(%s): %v", alg, err)
		}
		header, _ := decodeSegments(t, token)
		if header["alg"] != alg {
			t.Errorf("alg = %v, want %s", header["alg"], alg)
		}
	}
}

func TestTamperSignatureChangesTheSignatureBytesAndNothingElse(t *testing.T) {
	m := Minter{SecretKeyBase64: paritySecret, IssuerOrigin: "http://localhost:8080"}
	token, err := m.Mint(Claims{Subject: "user@jdw.com"})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	tampered := TamperSignature(token)

	original, altered := strings.Split(token, "."), strings.Split(tampered, ".")
	if strings.Join(altered[:2], ".") != strings.Join(original[:2], ".") {
		t.Errorf("header and payload changed:\n%s\nwant\n%s", strings.Join(altered[:2], "."), strings.Join(original[:2], "."))
	}
	// Comparing the decoded bytes rather than the text: two encodings can
	// differ in their final character and still decode to the same signature.
	before, err := base64.RawURLEncoding.DecodeString(original[2])
	if err != nil {
		t.Fatalf("decode original signature: %v", err)
	}
	after, err := base64.RawURLEncoding.DecodeString(altered[2])
	if err != nil {
		t.Fatalf("decode tampered signature: %v", err)
	}
	if bytes.Equal(before, after) {
		t.Error("the signature bytes are unchanged, so the token still verifies")
	}
}
