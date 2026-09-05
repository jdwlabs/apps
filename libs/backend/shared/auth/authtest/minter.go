// Package authtest mints tokens in the exact claim layout the JVM JwtService
// produces, so services can write parity tests without reaching for a running
// identity service. It is deliberately a separate package: nothing in the
// verification path imports it, so no build can accidentally mint a token in
// production with a test key.
package authtest

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DefaultTTL is the deployed token lifetime, in sync with the JVM's
// app.jwt.expiration-time-ms default of 7200000.
const DefaultTTL = 2 * time.Hour

// Claims are the authorization-carrying claims a caller chooses. Everything
// else in the token is derived, exactly as JwtService.generateToken derives it.
type Claims struct {
	Subject string
	Roles   []string
	// UserID and ProfileID are pointers because the JVM writes an explicit JSON
	// null for a principal with no profile row, and a claim that is null is a
	// different case from a claim that is absent.
	UserID    *int64
	ProfileID *int64
}

// Minter reproduces JwtService.generateToken.
type Minter struct {
	// SecretKeyBase64 is the base64 of the HMAC key, byte-identical to the JVM's
	// app.jwt.secret-key.
	SecretKeyBase64 string
	// IssuerOrigin is the scheme://host:port of the request that mints the
	// token. The JVM stamps it into aud verbatim and into iss with
	// /auth/authenticate appended.
	IssuerOrigin string
	TTL          time.Duration
	Now          func() time.Time
	TokenID      func() string
}

func (m Minter) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m Minter) ttl() time.Duration {
	if m.TTL != 0 {
		return m.TTL
	}
	return DefaultTTL
}

func (m Minter) tokenID() string {
	if m.TokenID != nil {
		return m.TokenID()
	}
	return randomUUIDv4()
}

func (m Minter) key() ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(m.SecretKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode secret key: %w", err)
	}
	return key, nil
}

// Mint signs a token whose claim set is the one JwtService.generateToken
// writes: roles, user_id, profile_id, aud, iss, jti, nbf, sub, iat and exp.
func (m Minter) Mint(c Claims) (string, error) {
	issuedAt := m.now()
	roles := c.Roles
	if roles == nil {
		roles = []string{}
	}
	claims := map[string]any{
		"sub":   c.Subject,
		"roles": roles,
		// Seconds since the epoch as integers, which is how jjwt serializes a
		// Date claim. A float would still parse, but the fixtures checked in
		// here are compared against JVM-minted ones byte for byte.
		"iat":        issuedAt.Unix(),
		"nbf":        issuedAt.Unix(),
		"exp":        issuedAt.Add(m.ttl()).Unix(),
		"aud":        m.IssuerOrigin,
		"iss":        m.IssuerOrigin + "/auth/authenticate",
		"jti":        m.tokenID(),
		"user_id":    nullableInt(c.UserID),
		"profile_id": nullableInt(c.ProfileID),
	}
	return m.MintRaw("HS256", claims)
}

// MintRaw signs an arbitrary claim set with an arbitrary algorithm, for the
// negative fixtures a well-formed mint cannot express. "none" produces an
// unsigned token.
func (m Minter) MintRaw(alg string, claims map[string]any) (string, error) {
	if alg == "none" {
		return unsignedToken(claims)
	}
	method := jwt.GetSigningMethod(alg)
	if method == nil {
		return "", fmt.Errorf("unknown signing algorithm %q", alg)
	}
	key, err := m.key()
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(method, jwt.MapClaims(claims))
	// jjwt writes the algorithm and nothing else. Dropping the typ header that
	// golang-jwt adds by default keeps a minted token byte-identical to a JVM
	// one, which is what makes the checked-in fixtures comparable at all.
	token.Header = map[string]any{"alg": alg}
	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// TamperSignature inverts the first byte of the signature, leaving the header
// and payload intact, so a test can assert that verification fails on the
// signature rather than on a malformed token.
//
// It decodes rather than editing the encoded text because the final base64url
// character of a 32-byte signature carries only two significant bits: four
// distinct characters decode to the same signature, so a naive edit of the last
// character leaves the token valid a quarter of the time.
func TamperSignature(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return token
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) == 0 {
		return token
	}
	signature[0] ^= 0xff
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	return strings.Join(parts, ".")
}

func unsignedToken(claims map[string]any) (string, error) {
	header, err := json.Marshal(map[string]any{"alg": "none"})
	if err != nil {
		return "", fmt.Errorf("encode header: %w", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encode claims: %w", err)
	}
	enc := base64.RawURLEncoding
	return enc.EncodeToString(header) + "." + enc.EncodeToString(payload) + ".", nil
}

func nullableInt(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func randomUUIDv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("authtest: no entropy for a token id: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
