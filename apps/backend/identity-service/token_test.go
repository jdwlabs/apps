package main

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authtest"
)

func decodeSegment(t *testing.T, segment string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		t.Fatalf("decode %q: %v", segment, err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal %q: %v", raw, err)
	}
	return decoded
}

func TestATokenMintedHereVerifiesThroughTheSharedVerifier(t *testing.T) {
	// The whole cutover rests on this: the tokens this service issues are the
	// tokens profile-service accepts, checked through the production verifier
	// rather than through a reimplementation of it.
	token, err := parityMinter(t).Mint(Credential{
		UserID: selfUserID, EmailAddress: selfEmail,
		Roles: []string{"ADMIN", "USER"}, ProfileID: ptr(fixtureProfileID),
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	principal, err := parityVerifier(t).VerifyAuthorizationHeader("Bearer " + token)

	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if principal.Subject != selfEmail {
		t.Errorf("sub = %q, want %q", principal.Subject, selfEmail)
	}
	if !principal.HasAuthority("ADMIN") || !principal.HasAuthority("USER") {
		t.Errorf("roles = %v, want both the ones minted", principal.Roles)
	}
	if principal.UserID == nil || *principal.UserID != selfUserID {
		t.Errorf("user_id = %v, want %d", principal.UserID, selfUserID)
	}
	if principal.ProfileID == nil || *principal.ProfileID != fixtureProfileID {
		t.Errorf("profile_id = %v, want %d", principal.ProfileID, fixtureProfileID)
	}
}

func TestATokenMintedHereHasTheClaimLayoutTheJvmWrites(t *testing.T) {
	// Compared against the test minter, which is itself held against a
	// JVM-minted fixture in the shared library's own parity suite. That chain is
	// what makes this an assertion about jjwt rather than about two Go files
	// agreeing with each other.
	issuedAt := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	made := parityMinter(t)
	made.now = func() time.Time { return issuedAt }
	made.tokenID = func() string { return "fixed-token-id" }

	mine, err := made.Mint(Credential{
		UserID: selfUserID, EmailAddress: selfEmail, Roles: []string{"USER"}, ProfileID: ptr(fixtureProfileID),
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	theirs, err := authtest.Minter{
		SecretKeyBase64: paritySecret, IssuerOrigin: parityIssuerOrigin, TTL: defaultTokenTTL,
		Now: func() time.Time { return issuedAt }, TokenID: func() string { return "fixed-token-id" },
	}.Mint(authtest.Claims{
		Subject: selfEmail, Roles: []string{"USER"},
		UserID: ptr(selfUserID), ProfileID: ptr(fixtureProfileID),
	})
	if err != nil {
		t.Fatalf("authtest Mint: %v", err)
	}

	mineParts, theirParts := strings.Split(mine, "."), strings.Split(theirs, ".")
	if len(mineParts) != 3 {
		t.Fatalf("a minted token has %d segments, want 3", len(mineParts))
	}
	if mineParts[0] != theirParts[0] {
		t.Errorf("header = %v, want %v; jjwt writes the algorithm and nothing else",
			decodeSegment(t, mineParts[0]), decodeSegment(t, theirParts[0]))
	}
	mineClaims, theirClaims := decodeSegment(t, mineParts[1]), decodeSegment(t, theirParts[1])
	// Two empty maps agree with each other. The set is named rather than
	// counted so a claim that silently stopped being written fails here.
	for _, claim := range []string{"sub", "roles", "iat", "nbf", "exp", "aud", "iss", "jti", "user_id", "profile_id"} {
		if _, present := theirClaims[claim]; !present {
			t.Fatalf("the JVM layout carries no %s claim, so comparing against it asserts nothing", claim)
		}
		if _, present := mineClaims[claim]; !present {
			t.Errorf("%s is in the JVM layout and not minted here", claim)
		}
	}
	for claim, want := range theirClaims {
		if got := mineClaims[claim]; !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, want %v", claim, got, want)
		}
	}
	for claim := range mineClaims {
		if _, expected := theirClaims[claim]; !expected {
			t.Errorf("%s is minted here and not by the JVM layout", claim)
		}
	}
}

func TestAPrincipalWithNoProfileMintsANullClaimRatherThanOmittingIt(t *testing.T) {
	// profile-service falls back to a user_id lookup when it sees a null or
	// absent profile_id. The JVM writes an explicit null, and a caller reading
	// the token by hand should see the same thing.
	token, err := parityMinter(t).Mint(Credential{
		UserID: selfUserID, EmailAddress: selfEmail, Roles: []string{"USER"},
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	claims := decodeSegment(t, strings.Split(token, ".")[1])

	value, present := claims["profile_id"]
	if !present {
		t.Fatal("profile_id is absent; the JVM writes an explicit null")
	}
	if value != nil {
		t.Errorf("profile_id = %v, want null", value)
	}
}

func TestAMinterRefusesAConfigurationItCannotSignWith(t *testing.T) {
	// Failing at startup beats a service that reports itself healthy and then
	// answers every sign-in with a 500.
	for _, tc := range []struct {
		name   string
		secret string
		origin string
	}{
		{name: "no secret", secret: "", origin: parityIssuerOrigin},
		{name: "a secret that is not base64", secret: "not base64 at all", origin: parityIssuerOrigin},
		{name: "no issuer origin to stamp", secret: paritySecret, origin: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := newMinter(tc.secret, tc.origin, defaultTokenTTL); err == nil {
				t.Error("newMinter accepted a configuration it cannot mint a usable token with")
			}
		})
	}
}

func TestTheSeededHashIsOneThisImplementationCanRead(t *testing.T) {
	// The hash the deployed schema seeds its system user with, written by
	// Spring's BCryptPasswordEncoder. Read rather than matched: what has to hold
	// at cutover is that this implementation parses the JVM's output at the cost
	// the JVM chose, and asserting that costs nothing a plaintext in the source
	// tree would.
	const seeded = "$2a$10$v5e3IStWiGNj1tmB8WWoguoVQHFThHyMlqaOJP1vrNyrrcZfwDRBa" // gitleaks:allow

	cost, err := bcrypt.Cost([]byte(seeded))

	if err != nil {
		t.Fatalf("a hash the deployed database holds is unreadable here: %v", err)
	}
	if cost != bcrypt.DefaultCost {
		t.Errorf("cost = %d, want %d; the two encoders disagree on how much work a check is",
			cost, bcrypt.DefaultCost)
	}
	if passwordMatches(seeded, "Password1!") {
		t.Error("an arbitrary password matched a hash it did not produce")
	}
}

func TestEveryPrefixTheJvmEncoderCanEmitStillVerifies(t *testing.T) {
	// BCryptPasswordEncoder writes 2a and accepts 2b and 2y, and rows written by
	// older tooling carry the other two. The prefixes name the same algorithm,
	// so the same salt and cost verify under any of them.
	hash, err := hashPassword(fixturePassword)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}

	for _, prefix := range []string{"$2a$", "$2b$", "$2y$"} {
		t.Run(prefix, func(t *testing.T) {
			if !passwordMatches(prefix+hash[4:], fixturePassword) {
				t.Errorf("a hash carrying the %s prefix did not verify", prefix)
			}
		})
	}
}

func TestAPasswordLongerThanBcryptReadsIsTruncatedRatherThanRefused(t *testing.T) {
	// Spring's BCrypt truncates the key at 72 bytes without saying so; x/crypto
	// refuses it outright. Refusing would turn an account the JVM created into
	// one nobody can sign in to.
	long := strings.Repeat("Aa1!", 30)

	hash, err := hashPassword(long)

	if err != nil {
		t.Fatalf("hashPassword refused a password the JVM would have stored: %v", err)
	}
	if !passwordMatches(hash, long) {
		t.Error("the password that was hashed does not verify against its own hash")
	}
	if !passwordMatches(hash, long[:bcryptKeyBytes]) {
		t.Error("the truncation is not the one bcrypt applies; a JVM-stored hash would not match")
	}
}

func TestHashingTheSamePasswordTwiceProducesDifferentHashes(t *testing.T) {
	// Salted, as it must be: two users who chose the same password must not be
	// visibly the same in the table.
	first, err := hashPassword("Password1!")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	second, err := hashPassword("Password1!")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}

	if first == second {
		t.Error("two encodings of one password are identical; the salt is not being applied")
	}
	if !strings.HasPrefix(first, "$2a$10$") {
		t.Errorf("hash = %q, want the prefix and cost the JVM encoder writes", first[:7])
	}
}

func TestTheVerifierRefusesATokenMintedUnderAnotherKey(t *testing.T) {
	// The secret is symmetric and shared: a service holding a different key
	// rejects every token the others accept, and this is the failure that would
	// show it.
	const otherSecret = "b3RoZXItc2VjcmV0LXdpdGgtdGhpcnR5LXR3by1ieXRlcyE=" // gitleaks:allow
	other, err := newMinter(otherSecret, parityIssuerOrigin, defaultTokenTTL)
	if err != nil {
		t.Fatalf("newMinter: %v", err)
	}
	token, err := other.Mint(Credential{UserID: selfUserID, EmailAddress: selfEmail})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	if _, err := parityVerifier(t).VerifyAuthorizationHeader("Bearer " + token); err == nil {
		t.Error("a token signed with another key verified")
	}
}

func TestTheIssuerTheServiceMintsIsTheIssuerItVerifies(t *testing.T) {
	// Configured once and used for both, because a request-derived origin — the
	// JVM's approach, which never checks iss on the way back in — would let a
	// caller-controlled Host header decide what the next request refuses.
	t.Setenv("UR_JWT_SECRET_KEY", paritySecret)
	t.Setenv("UR_PG_DATASOURCE_URL", "jdbc:postgresql://authdb:5432/jdw")
	t.Setenv("ID_JWT_ISSUER_ORIGIN", "https://auth.example.com/")

	config, err := configFromEnvironment()
	if err != nil {
		t.Fatalf("configFromEnvironment: %v", err)
	}
	made, err := newMinter(config.SecretKeyBase64, config.IssuerOrigin, config.TokenTTL)
	if err != nil {
		t.Fatalf("newMinter: %v", err)
	}
	verifier, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  config.SecretKeyBase64,
		ExpectedIssuer:   config.ExpectedIssuer,
		ExpectedAudience: config.ExpectedAudience,
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	token, err := made.Mint(Credential{UserID: selfUserID, EmailAddress: selfEmail})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	if _, err := verifier.VerifyAuthorizationHeader("Bearer " + token); err != nil {
		t.Errorf("the service cannot verify its own token: %v", err)
	}
}

func TestTheDecoyIsAsExpensiveToCompareAgainstAsAStoredHash(t *testing.T) {
	// What closes the timing oracle: an address with no row has to cost what an
	// address with one costs, or the two identical responses still say which is
	// which. The assertion is on the cost recorded in the hash rather than on a
	// measured duration, which would be a coin flip on a shared runner.
	cost, err := bcrypt.Cost([]byte(decoyHash))

	if err != nil {
		t.Fatalf("the decoy is not a readable hash: %v", err)
	}
	if cost != bcrypt.DefaultCost {
		t.Errorf("decoy cost = %d, want the %d a stored hash carries", cost, bcrypt.DefaultCost)
	}
	if passwordMatches(decoyHash, fixturePassword) {
		t.Error("a password a caller could present matched the decoy")
	}
}

// The minter derives its signing key from the same secret the verifier derives
// its checking key from, so the two have to read an unpadded value the same
// way. They would otherwise disagree silently: this service would sign with one
// key and both services would check with another, and every sign-in would
// succeed while every subsequent request was refused.
func TestAMinterSigningWithAnUnpaddedSecretIsCheckedByThePaddedOne(t *testing.T) {
	if got := len(parityUnpaddedSecret) % 4; got != 2 {
		t.Fatalf("the unpadded fixture's length is %d more than a multiple of 4, want 2; it no longer has the deployed secret's shape", got)
	}

	made, err := newMinter(parityUnpaddedSecret, parityIssuerOrigin, defaultTokenTTL)
	if err != nil {
		t.Fatalf("newMinter refused the shape of the deployed secret: %v", err)
	}
	token, err := made.Mint(Credential{
		UserID: selfUserID, EmailAddress: selfEmail, Roles: []string{"USER"},
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	verifier, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  parityPaddedSecret,
		ExpectedIssuer:   parityIssuerOrigin + "/auth/authenticate",
		ExpectedAudience: parityIssuerOrigin,
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	if _, err := verifier.Verify(token); err != nil {
		t.Errorf("a token signed from the unpadded secret did not verify against the padded form of the same key: %v", err)
	}
}

// This is the service that signs, so it has to sign with the variant both
// verifiers derive from the same key — HS512 for a key the deployed secret's
// length, which is what the JVM it replaces signs with today. Signing HS256
// under that key would mint tokens the shared verifier refuses outright.
func TestAMinterSignsWithTheVariantTheDeployedKeyLengthSelects(t *testing.T) {
	key := make([]byte, 1534)
	for i := range key {
		key[i] = byte(i*7 + 1)
	}
	secret := base64.RawStdEncoding.EncodeToString(key)
	if len(secret) != 2046 {
		t.Fatalf("the fixture is %d characters, not the deployed shape", len(secret))
	}

	made, err := newMinter(secret, parityIssuerOrigin, defaultTokenTTL)
	if err != nil {
		t.Fatalf("newMinter refused the deployed shape: %v", err)
	}
	token, err := made.Mint(Credential{UserID: selfUserID, EmailAddress: selfEmail, Roles: []string{"USER"}})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if header := decodeSegment(t, strings.Split(token, ".")[0]); header["alg"] != "HS512" {
		t.Errorf("minted with %v, want HS512", header["alg"])
	}

	verifier, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  secret,
		ExpectedIssuer:   parityIssuerOrigin + "/auth/authenticate",
		ExpectedAudience: parityIssuerOrigin,
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	if _, err := verifier.Verify(token); err != nil {
		t.Errorf("Verify: %v", err)
	}
}

func TestAMinterRefusesAKeyJjwtWouldRefuse(t *testing.T) {
	short := base64.StdEncoding.EncodeToString(make([]byte, 31))
	if _, err := newMinter(short, parityIssuerOrigin, defaultTokenTTL); err == nil {
		t.Error("newMinter accepted a 31-byte key, which no JVM could have signed a token with")
	}
}
