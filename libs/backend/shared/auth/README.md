# Go Shared Auth

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

Verifies the HS256 tokens the Spring `usersrole` service mints, and decides the
authorization rules the frozen service contracts name. One implementation, two
consumers: `identity-service` and `profile-service` enforce the same rules from
the same code, so neither can drift from the other or from the contract.

The contracts it reproduces are
[`identity-service.openapi.yaml`](../../../../apps/backend/usersrole/docs/contracts/identity-service.openapi.yaml)
and
[`profile-service.openapi.yaml`](../../../../apps/backend/usersrole/docs/contracts/profile-service.openapi.yaml):
`x-jwt-claims` fixes the claim set, and each operation's `x-authorization` block
names the rule it is decided by.

---

## 📁 Project Structure

```
libs/backend/shared/auth/
├── go.mod                     # module libs/backend/shared/auth
├── principal.go               # claim names and the verified principal
├── verifier.go                # HS256 parsing, signature and claim validation
├── authz/rules.go             # one function per contract rule
├── authhttp/middleware.go     # net/http authentication and refusal shapes
├── authtest/minter.go         # test-only minter reproducing JwtService
├── parity_test.go             # the Go half of the cross-implementation suite
└── project.json               # Nx project configuration
```

---

## 🔐 The secret is shared, and rotating it is one operation

Both services read the base64 HMAC secret from **`UR_JWT_SECRET_KEY`**, the same
variable Spring resolves `app.jwt.secret-key` from. Verification is a symmetric
operation: the key that verifies is the key that signs.

- **The value must be byte-identical everywhere.** A service holding a different
  key rejects every token the others accept.
- **Rotate all consumers together.** There is no key id in the header and no
  second key accepted during a changeover, so a rotation is a simultaneous
  restart of every service that mints or verifies, not a rolling one. Tokens
  minted under the old key stop verifying the moment it is withdrawn — up to a
  full token lifetime of callers forced to re-authenticate.
- **Key length is load-bearing.** jjwt picks the HMAC variant from the key
  length, so a secret of 48 bytes or more makes the JVM sign HS384 or HS512 and
  every Go verification fails at once. `NewVerifier` refuses a key outside the
  HS256 band at construction, turning that outage into a startup error.

---

## 🔧 Usage

### Verifying a token

```go
import "libs/backend/shared/auth"

secret, err := auth.SecretKeyFromEnv()
if err != nil {
  return err
}
verifier, err := auth.NewVerifier(auth.Config{
  SecretKeyBase64:  secret,
  ExpectedIssuer:   issuerOrigin + "/auth/authenticate",
  ExpectedAudience: issuerOrigin,
})
```

`Verify` checks the signature, refuses any algorithm but HS256, enforces the
`nbf`/`exp` window, and requires `sub`, `iss`, `aud`, `jti`, `nbf` and `exp` to
be present. It returns a `*auth.Principal` carrying the subject email, roles,
`user_id` and `profile_id`.

### Authenticating requests

```go
import "libs/backend/shared/auth/authhttp"

mux := http.NewServeMux()
handler := authhttp.Middleware{
  Verifier: verifier,
  Public:   func(r *http.Request) bool { return strings.HasPrefix(r.URL.Path, "/auth/") },
}.Handler(mux)
```

Refusals carry the `Access-Denied-Reason` header and container error body the
contracts pin: `Authentication Required` with 401 from the middleware, and
`Not Authorized` with 403 from `WriteForbidden`.

### Deciding a rule

```go
import "libs/backend/shared/auth/authz"

authorizer := authz.Authorizer{ProfileIDForUser: profiles.IDForUser}

if !authhttp.Authorize(w, r, authorizer, authz.RuleAdminOrSelfByProfileID,
  authz.Subject{ProfileID: &profileID}) {
  return
}
```

Every rule the contracts name has a function, and a test asserts that set
against the contract files themselves — a rule added to a contract fails the
build rather than being discovered by whoever writes the handler.

| Rule                            | Predicate it transcribes                                       |
| ------------------------------- | -------------------------------------------------------------- |
| `PUBLIC`                        | no predicate; reachable through the `permitAll` matchers       |
| `AUTHENTICATED`                 | no predicate; any verified principal                           |
| `ADMIN`                         | `hasAuthority('ADMIN')`                                        |
| `ADMIN_OR_MANAGER`              | `hasAnyAuthority('ADMIN', 'MANAGER')`                          |
| `ADMIN_OR_SELF_BY_USER_ID`      | `... or #userId == principal.getUserId()`                      |
| `ADMIN_OR_SELF_BY_EMAIL`        | `... or #emailAddress == principal.getUsername()`              |
| `ADMIN_OR_SELF_BY_PROFILE_ID`   | `... or #profileId == principal.getProfileId()`, with fallback |
| `ADMIN_OR_SELF_BY_BODY_USER_ID` | `... or #profile.userId() == principal.getUserId()`            |

---

## ⚠️ Where this is deliberately not the JVM

Spring rebuilds the principal from the database on every request. This library
reads it from the token, which is the point — `profile-service` owns none of the
tables the JVM re-reads. Two consequences are recorded in the contracts rather
than papered over, and both are pinned by tests:

- **Role revocation is bounded by the token lifetime.** A revoked ADMIN keeps
  ADMIN access here until their token expires, up to two hours, where the JVM
  refuses them on the next request.
- **A stale `profile_id` claim still authorizes.** A caller who deletes their own
  profile keeps a claim naming it, and profile-scoped rules pass. An _absent_
  claim is the case the fallback covers: `ProfileIDForUser` resolves the profile
  from the `user_id` claim, so a profile created mid-session is not locked out.
  The lookup keys on the claim and never on a path variable, so falling back
  cannot widen a caller's own authorization.

---

## 🧪 Testing

```bash
nx test backend-shared-auth      # go test -race ./...
nx lint backend-shared-auth
nx tidy backend-shared-auth
```

### Cross-implementation parity

One token minted by each implementation is checked in, and each side asserts the
other's. Each fixture lives in the source of the side that consumes it rather
than in a shared data file, so the directive telling the secrets gate that a
token signed with a published test key is not a credential sits on the line a
reviewer reads.

- `parity_test.go` holds a token minted by `JwtService.generateToken` and
  verifies it in `TestATokenMintedByTheJvmVerifiesHere`, with the clock pinned
  inside its two-hour window.
- `JwtGoParityTests` holds a token minted by `authtest.Minter` and verifies it
  through the production `extractAllClaims`, then compares the JOSE header and
  the claim set against a token it mints itself. That token's lifetime is
  deliberately long: `JwtService` reads the wall clock and offers no seam, so a
  realistic expiry would make the assertion untestable rather than strict.

Refreshing them after a claim-layout change — each command produces a
replacement to paste into the other side:

```bash
# JVM side, writes build/parity/jvm-minted-token.json
cd apps/backend/usersrole
bash gradlew test --tests com.jdw.usersrole.services.JwtGoParityTests

# Go side, prints a replacement for JwtGoParityTests.GO_MINTED_TOKEN
cd libs/backend/shared/auth
AUTH_PARITY_PRINT_TOKEN=1 go test . -run TestPrintGoMintedToken -v
```

---

## 📌 Notes

- Token issuance stays on the JVM. `authtest` mints only so that tests need no
  running identity service, and it is a separate package precisely so nothing on
  the verification path can import it.
- The only external dependency is `github.com/golang-jwt/jwt/v5`.

---

## 📚 Related Packages

- [`backend-shared-util`](../util): the other Go library shared across services.
- [`usersrole`](../../../../apps/backend/usersrole): the Spring service whose
  behaviour this library reproduces, and the JVM half of the parity suite.
