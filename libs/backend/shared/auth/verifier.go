package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SecretKeyEnvVar is the name Spring resolves app.jwt.secret-key from. Both Go
// services read the same variable so a rotation is one change, not three.
const SecretKeyEnvVar = "UR_JWT_SECRET_KEY"

// signingAlgorithm is the only algorithm accepted. Anything else — including
// "none" and an RS256 header pointing at a public key — is a forgery attempt or
// a misconfiguration, never a token this system minted.
const signingAlgorithm = "HS256"

// jjwt picks the HMAC variant from the key length: 256 bits up to 383 gives
// HS256, 384 up to 511 gives HS384, 512 and over gives HS512. A key outside the
// HS256 band therefore makes the JVM sign with an algorithm this verifier
// refuses, which would reject every live token at once. Failing at construction
// turns that outage into a startup error.
const (
	minKeyBytes = 32
	maxKeyBytes = 47
)

var (
	ErrMissingSecretKey    = errors.New("jwt secret key is not set")
	ErrInvalidSecretKey    = errors.New("jwt secret key is unusable")
	ErrMissingBearerToken  = errors.New("no bearer token in the authorization header")
	ErrMalformedToken      = errors.New("token is malformed")
	ErrUnexpectedAlgorithm = errors.New("token is not signed with " + signingAlgorithm)
	ErrInvalidSignature    = errors.New("token signature does not verify")
	ErrTokenExpired        = errors.New("token has expired")
	ErrTokenNotYetValid    = errors.New("token is not yet valid")
	ErrMissingClaim        = errors.New("token is missing a required claim")
	ErrInvalidClaim        = errors.New("token claim has the wrong type")
	ErrInvalidIssuer       = errors.New("token issuer is not the expected one")
	ErrInvalidAudience     = errors.New("token audience is not the expected one")
)

// Config configures a Verifier.
type Config struct {
	// SecretKeyBase64 is the base64 of the HMAC key, byte-identical to the JVM's
	// app.jwt.secret-key.
	SecretKeyBase64 string
	// ExpectedIssuer is the full iss claim, which the JVM builds as the issuer
	// origin with /auth/authenticate appended. Empty accepts any issuer, but
	// never an absent one.
	ExpectedIssuer string
	// ExpectedAudience is the issuer origin. Empty accepts any audience, but
	// never an absent one.
	ExpectedAudience string
	// Leeway absorbs clock skew between the minting and verifying hosts.
	Leeway time.Duration
	Now    func() time.Time
}

// Verifier parses and verifies tokens. It holds no per-request state and is
// safe for concurrent use.
type Verifier struct {
	key              []byte
	parser           *jwt.Parser
	expectedIssuer   string
	expectedAudience string
}

// SecretKeyFromEnv reads the base64 secret from the environment variable Spring
// reads.
func SecretKeyFromEnv() (string, error) {
	secret := os.Getenv(SecretKeyEnvVar)
	if secret == "" {
		return "", fmt.Errorf("%w: %s is empty or unset", ErrMissingSecretKey, SecretKeyEnvVar)
	}
	return secret, nil
}

// NewVerifier validates the configuration and returns a ready Verifier.
func NewVerifier(cfg Config) (*Verifier, error) {
	if cfg.SecretKeyBase64 == "" {
		return nil, fmt.Errorf("%w: SecretKeyBase64 is empty", ErrMissingSecretKey)
	}
	key, err := base64.StdEncoding.DecodeString(cfg.SecretKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: not base64: %w", ErrInvalidSecretKey, err)
	}
	if len(key) < minKeyBytes {
		return nil, fmt.Errorf("%w: %d bytes, %s needs at least %d", ErrInvalidSecretKey, len(key), signingAlgorithm, minKeyBytes)
	}
	if len(key) > maxKeyBytes {
		return nil, fmt.Errorf("%w: %d bytes, which makes the JVM sign with a stronger HMAC variant than %s", ErrInvalidSecretKey, len(key), signingAlgorithm)
	}

	options := []jwt.ParserOption{jwt.WithJSONNumber()}
	if cfg.Leeway > 0 {
		options = append(options, jwt.WithLeeway(cfg.Leeway))
	}
	if cfg.Now != nil {
		options = append(options, jwt.WithTimeFunc(cfg.Now))
	}
	return &Verifier{
		key:              key,
		parser:           jwt.NewParser(options...),
		expectedIssuer:   cfg.ExpectedIssuer,
		expectedAudience: cfg.ExpectedAudience,
	}, nil
}

// VerifyAuthorizationHeader verifies the token carried by an Authorization
// header. The "Bearer " prefix is matched case-sensitively, as the JVM filter
// matches it.
func (v *Verifier) VerifyAuthorizationHeader(header string) (*Principal, error) {
	const scheme = "Bearer "
	if !strings.HasPrefix(header, scheme) {
		return nil, ErrMissingBearerToken
	}
	return v.Verify(header[len(scheme):])
}

// Verify checks the signature, the time window and the required claims, and
// returns the principal the claims describe.
func (v *Verifier) Verify(token string) (*Principal, error) {
	claims := jwt.MapClaims{}
	if _, err := v.parser.ParseWithClaims(token, claims, v.keyFunc); err != nil {
		return nil, translateParseError(err)
	}
	return v.principalFrom(claims)
}

// keyFunc pins the algorithm. The check lives here rather than in
// jwt.WithValidMethods so that a wrong algorithm is reported as one, instead of
// being folded into a generic signature failure.
func (v *Verifier) keyFunc(token *jwt.Token) (any, error) {
	if token.Method.Alg() != signingAlgorithm {
		return nil, fmt.Errorf("%w: header says %q", ErrUnexpectedAlgorithm, token.Method.Alg())
	}
	return v.key, nil
}

func translateParseError(err error) error {
	// Order matters: the algorithm check runs inside the key function, and the
	// parser reports a key function failure as unverifiable, so it has to be
	// recognised before the coarser categories.
	switch {
	case errors.Is(err, ErrUnexpectedAlgorithm):
		return err
	case errors.Is(err, jwt.ErrTokenExpired):
		return fmt.Errorf("%w: %w", ErrTokenExpired, err)
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return fmt.Errorf("%w: %w", ErrTokenNotYetValid, err)
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return fmt.Errorf("%w: %w", ErrInvalidSignature, err)
	default:
		return fmt.Errorf("%w: %w", ErrMalformedToken, err)
	}
}

func (v *Verifier) principalFrom(claims jwt.MapClaims) (*Principal, error) {
	subject, err := requiredString(claims, ClaimSubject)
	if err != nil {
		return nil, err
	}
	issuer, err := requiredString(claims, ClaimIssuer)
	if err != nil {
		return nil, err
	}
	tokenID, err := requiredString(claims, ClaimTokenID)
	if err != nil {
		return nil, err
	}
	audience, err := requiredAudience(claims)
	if err != nil {
		return nil, err
	}
	notBefore, err := requiredTime(claims, ClaimNotBefore)
	if err != nil {
		return nil, err
	}
	expiresAt, err := requiredTime(claims, ClaimExpiresAt)
	if err != nil {
		return nil, err
	}
	issuedAt, err := optionalTime(claims, ClaimIssuedAt)
	if err != nil {
		return nil, err
	}
	roles, err := optionalStrings(claims, ClaimRoles)
	if err != nil {
		return nil, err
	}
	userID, err := optionalInt(claims, ClaimUserID)
	if err != nil {
		return nil, err
	}
	profileID, err := optionalInt(claims, ClaimProfileID)
	if err != nil {
		return nil, err
	}

	if v.expectedIssuer != "" && issuer != v.expectedIssuer {
		return nil, fmt.Errorf("%w: %q", ErrInvalidIssuer, issuer)
	}
	if v.expectedAudience != "" && !contains(audience, v.expectedAudience) {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAudience, audience)
	}

	return &Principal{
		Subject:   subject,
		Roles:     roles,
		UserID:    userID,
		ProfileID: profileID,
		Issuer:    issuer,
		Audience:  audience,
		TokenID:   tokenID,
		IssuedAt:  issuedAt,
		NotBefore: notBefore,
		ExpiresAt: expiresAt,
	}, nil
}

func requiredString(claims jwt.MapClaims, name string) (string, error) {
	raw, present := claims[name]
	if !present || raw == nil {
		return "", fmt.Errorf("%w: %s", ErrMissingClaim, name)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s is not a string", ErrInvalidClaim, name)
	}
	if value == "" {
		return "", fmt.Errorf("%w: %s is empty", ErrMissingClaim, name)
	}
	return value, nil
}

func requiredAudience(claims jwt.MapClaims) ([]string, error) {
	raw, present := claims[ClaimAudience]
	if !present || raw == nil {
		return nil, fmt.Errorf("%w: %s", ErrMissingClaim, ClaimAudience)
	}
	// The JVM writes a bare string; the specification also allows an array, and
	// a token that took a detour through another library may carry one.
	switch value := raw.(type) {
	case string:
		if value == "" {
			return nil, fmt.Errorf("%w: %s is empty", ErrMissingClaim, ClaimAudience)
		}
		return []string{value}, nil
	case []any:
		audience := make([]string, 0, len(value))
		for _, entry := range value {
			text, ok := entry.(string)
			if !ok {
				return nil, fmt.Errorf("%w: %s holds a non-string", ErrInvalidClaim, ClaimAudience)
			}
			audience = append(audience, text)
		}
		if len(audience) == 0 {
			return nil, fmt.Errorf("%w: %s is empty", ErrMissingClaim, ClaimAudience)
		}
		return audience, nil
	default:
		return nil, fmt.Errorf("%w: %s is neither a string nor an array", ErrInvalidClaim, ClaimAudience)
	}
}

func requiredTime(claims jwt.MapClaims, name string) (time.Time, error) {
	if raw, present := claims[name]; !present || raw == nil {
		return time.Time{}, fmt.Errorf("%w: %s", ErrMissingClaim, name)
	}
	return optionalTime(claims, name)
}

func optionalTime(claims jwt.MapClaims, name string) (time.Time, error) {
	raw, present := claims[name]
	if !present || raw == nil {
		return time.Time{}, nil
	}
	seconds, err := toInt64(raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s is not a numeric date: %w", ErrInvalidClaim, name, err)
	}
	return time.Unix(seconds, 0).UTC(), nil
}

func optionalInt(claims jwt.MapClaims, name string) (*int64, error) {
	raw, present := claims[name]
	if !present || raw == nil {
		return nil, nil
	}
	value, err := toInt64(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s is not an integer: %w", ErrInvalidClaim, name, err)
	}
	return &value, nil
}

func optionalStrings(claims jwt.MapClaims, name string) ([]string, error) {
	raw, present := claims[name]
	if !present || raw == nil {
		return nil, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s is not an array", ErrInvalidClaim, name)
	}
	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		text, ok := entry.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s holds a non-string", ErrInvalidClaim, name)
		}
		values = append(values, text)
	}
	return values, nil
}

// toInt64 handles both number representations the parser can produce: a
// json.Number under WithJSONNumber, and a float64 for claims built in memory.
func toInt64(raw any) (int64, error) {
	switch value := raw.(type) {
	case json.Number:
		return value.Int64()
	case float64:
		return int64(value), nil
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	default:
		return 0, fmt.Errorf("unexpected type %T", raw)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
