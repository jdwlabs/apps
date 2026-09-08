package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// defaultTokenTTL is the deployed lifetime, matching the JVM's
// app.jwt.expiration-time-ms default of 7200000.
const defaultTokenTTL = 2 * time.Hour

// bcryptKeyBytes is the key length bcrypt actually consumes. Spring's BCrypt
// truncates silently at this length; x/crypto refuses anything longer outright.
// Truncating here keeps both halves working: a password over the limit is
// accepted as the JVM accepts it, and a hash the JVM stored for such a password
// still verifies.
const bcryptKeyBytes = 72

// ErrNoSigningKey is returned by newMinter rather than left to surface on the
// first sign-in, which would take the service down after it had already
// reported itself healthy.
var ErrNoSigningKey = errors.New("the minter needs a signing key")

// minter issues the tokens both Go services verify. It reproduces
// JwtService.generateToken claim for claim.
//
// It lives here rather than in the shared auth library on purpose. Anything
// able to sign a token can sign itself any principal, so the capability belongs
// only to the service that has already checked a password — a library both
// services import would hand it to the one that must never have it.
type minter struct {
	key          []byte
	issuerOrigin string
	ttl          time.Duration
	now          func() time.Time
	tokenID      func() string
}

func newMinter(secretKeyBase64, issuerOrigin string, ttl time.Duration) (*minter, error) {
	if secretKeyBase64 == "" {
		return nil, fmt.Errorf("%w: the secret is empty", ErrNoSigningKey)
	}
	key, err := base64.StdEncoding.DecodeString(secretKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: not base64: %w", ErrNoSigningKey, err)
	}
	if issuerOrigin == "" {
		return nil, fmt.Errorf("%w: no issuer origin to stamp", ErrNoSigningKey)
	}
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	return &minter{key: key, issuerOrigin: issuerOrigin, ttl: ttl}, nil
}

func (m *minter) clock() time.Time {
	if m.now != nil {
		return m.now()
	}
	return time.Now()
}

func (m *minter) nextTokenID() string {
	if m.tokenID != nil {
		return m.tokenID()
	}
	return randomUUIDv4()
}

// Mint signs the claim set JwtService.generateToken writes: sub, roles, iat,
// nbf, exp, aud, iss, jti, user_id and profile_id.
func (m *minter) Mint(credential Credential) (string, error) {
	issuedAt := m.clock()
	roles := credential.Roles
	if roles == nil {
		roles = []string{}
	}
	claims := jwt.MapClaims{
		"sub":   credential.EmailAddress,
		"roles": roles,
		// Seconds since the epoch as integers, which is how jjwt serializes a
		// Date claim. A float parses too, but the cross-implementation fixtures
		// are compared byte for byte.
		"iat":        issuedAt.Unix(),
		"nbf":        issuedAt.Unix(),
		"exp":        issuedAt.Add(m.ttl).Unix(),
		"aud":        m.issuerOrigin,
		"iss":        m.issuerOrigin + "/auth/authenticate",
		"jti":        m.nextTokenID(),
		"user_id":    credential.UserID,
		"profile_id": nullableInt(credential.ProfileID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// jjwt writes the algorithm and nothing else. Dropping the typ header
	// golang-jwt adds by default keeps a token minted here byte-identical to a
	// JVM one, which is what lets either side verify the other's fixtures.
	token.Header = map[string]any{"alg": "HS256"}
	signed, err := token.SignedString(m.key)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// hashPassword encodes a password for storage, at the cost BCryptPasswordEncoder
// defaults to so that hashes written by either implementation cost the same to
// verify.
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(bcryptKey(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// passwordMatches compares a presented password against a stored hash. It
// accepts every 2-family prefix Spring's encoder emits, so hashes already in
// auth.users verify unchanged.
func passwordMatches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), bcryptKey(password)) == nil
}

// decoyHash is compared against when there is no stored hash to compare
// against. Answering an unknown address in the microseconds a lookup takes,
// where a known one costs a full bcrypt round, tells an anonymous caller which
// addresses are registered however identical the two responses look.
// DaoAuthenticationProvider does the same thing under the name
// mitigateAgainstTimingAttack.
//
// Encoded once at startup rather than held as a literal, so the cost tracks
// whatever cost this build encodes at.
var decoyHash = mustDecoyHash()

func mustDecoyHash() string {
	// The value is never a password anyone can present: it is discarded here and
	// the comparison's result is thrown away by the only caller.
	hash, err := bcrypt.GenerateFromPassword([]byte("no such account"), bcrypt.DefaultCost)
	if err != nil {
		panic("bcrypt could not encode the decoy: " + err.Error())
	}
	return string(hash)
}

// spendAComparison does the work a real comparison would have done, so that an
// address with no row costs what an address with one costs.
func spendAComparison(password string) {
	_ = passwordMatches(decoyHash, password)
}

func bcryptKey(password string) []byte {
	key := []byte(password)
	if len(key) > bcryptKeyBytes {
		return key[:bcryptKeyBytes]
	}
	return key
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
		// crypto/rand does not fail on any supported platform; a service that
		// could not produce a token id has no safe way to continue.
		panic("no entropy for a token id: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
