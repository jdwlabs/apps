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

### Before cutover: measure the deployed key length

Nothing in this repository knows how long the live `UR_JWT_SECRET_KEY` is, and
a key of 48 decoded bytes or more means the JVM has been signing HS384 all
along — every Go verification would fail on the first request after cutover.
Measure it before switching traffic, without printing it:

```bash
# Byte length only. The secret itself never reaches a terminal, a log or a
# shell history: base64 -d and wc -c both read from the pipe.
kubectl -n <namespace> get secret <secret>   -o jsonpath='{.data.UR_JWT_SECRET_KEY}' | base64 -d | base64 -d | wc -c
```

Expect a number from 32 to 47. Anything else is a cutover blocker rather than a
tuning question: 31 or less and the JVM refuses the key outright, 48 or more and
it signs with an algorithm this library will not accept.

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

middleware, err := authhttp.NewMiddleware(verifier)
if err != nil {
  return err
}
// The operations the contract freezes as PUBLIC, reachable without a token.
middleware.Public = func(r *http.Request) bool {
  return strings.HasPrefix(r.URL.Path, "/auth/")
}

handler := middleware.Handler(mux)
```

This snippet is compiled and run as `Example` in `authhttp/example_test.go`, so
a rename or a signature change fails the build rather than leaving a README that
no longer works.

Refusals carry the `Access-Denied-Reason` header and **no body**:
`Authentication Required` with 401 from the middleware, `Not Authorized` with
403 from `WriteForbidden`. See the note on the error body below.

**Mount the CORS layer outside this middleware**, the ordering Spring uses,
where `CorsFilter` runs ahead of the JWT filter. A browser never attaches
`Authorization` to a preflight, so a preflight that reaches authentication is
refused, and every cross-origin call the frontends make would fail.

If this middleware has to be the outer layer instead, set `Preflight` to the
CORS handler. Preflights are then handed to it and never to the wrapped
handler — a request shaped like a preflight must not reach a business handler
without a principal, which matters on a `net/http` mux whose patterns carry no
method and therefore match `OPTIONS` too. The exemption is off by default, and
matches the three conditions Spring's `CorsUtils.isPreFlightRequest` requires:
`OPTIONS`, an `Origin`, and `Access-Control-Request-Method`.

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

## 📭 The error body: what the contract says, and what the service sends

The frozen contracts document a `ContainerError` JSON body on every 401 and 403.
**The running application does not send one.** Both Spring handlers call
`sendError` with a message, but `server.error.include-message` is set nowhere —
not in `application.yaml`, not in any deployment values file — so Boot's default
of `never` applies, and the deployed service answers with `Content-Length: 0`,
no `Content-Type`, and an empty body. Measured against a booted `usersrole` on a
real port, with `Accept` of `application/json`, `*/*` and `text/html` in turn.

This library reproduces the service, not the document: status, header, empty
body. That is what keeps the cutover invisible, and the contract itself records
that clients key off the status and the header and never the body — the shared
frontend error helper switches on the status code alone.

Two consequences worth knowing:

- **Setting `server.error.include-message` on the JVM would change the wire
  format** of every 401 and 403, and this library would then be the one that is
  wrong. It is a contract change, not a logging tweak.
- **The contract's `ContainerError` schema for 401 and 403 is inaccurate** and
  wants a correction in its own change, since these documents are frozen.

The deployed service also sends `Access-Denied-Reason` twice, because the filter
chain runs again on the error dispatch and the entry point commences a second
time. One copy is sent here: a client reads the first value either way, and
copying an accident is not parity.

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
nx lint backend-shared-auth      # go vet ./...
nx tidy backend-shared-auth
```

`lint` runs `go vet` rather than the executor's default `go fmt`, which rewrites
files and never fails, so it gated nothing. `golangci-lint` is clean on this
package but is not wired in: it is installed neither in CI nor in the
devcontainer, so making it the lint target would fail every run. Installing it
and wiring all four Go projects is worth its own change.

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
  running identity service, and a workspace check
  (`tools/workspace-checks/test-only-go-packages.spec.ts`) fails if any non-test
  Go file in the repository imports it — anything that can sign its own tokens
  can mint itself any principal. It lives there because the offending import
  would sit in another project, and because `nx affected` would not select a
  test in this library to catch it.
- `NewVerifier` requires an expected issuer and audience, and refuses
  `AllowAnyIssuerAndAudience` alongside either — the two state opposite
  intentions. A service that genuinely accepts several origins sets the flag and
  leaves both values empty.
- The only external dependency is `github.com/golang-jwt/jwt/v5`.

---

## 📚 Related Packages

- [`backend-shared-util`](../util): the other Go library shared across services.
- [`usersrole`](../../../../apps/backend/usersrole): the Spring service whose
  behaviour this library reproduces, and the JVM half of the parity suite.
