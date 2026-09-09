# IdentityService

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/identity-service)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/identity-service)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwlabs/identity-service?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**IdentityService** is the other Go half of the `usersrole` split: the eighteen
`/auth`, `/api/users` and `/api/roles` operations, served against the same `auth`
schema the JVM service uses.

It is built against the frozen contract at
`apps/backend/usersrole/docs/contracts/identity-service.openapi.yaml` and
authorizes through `libs/backend/shared/auth`, which is the only place either Go
service parses a token. **It is also the only place either Go service mints
one** — see below.

No traffic is routed here yet. This project builds, tests and publishes an
image; the chart and the routing change land separately.

---

## 📁 Project Structure

```
apps/backend/identity-service/
├── Dockerfile              # Distroless runtime consuming the Nx build output
├── Dockerfile.local        # Self-contained build for local iteration
├── project.json            # Nx project definition and targets
├── config.go               # Environment resolution, done once at startup
├── errors.go               # Status and media type per failure the JVM maps
├── handlers.go             # The eighteen operations and the rule each carries
├── main.go                 # Entry point, pool, graceful shutdown
├── model.go                # Wire types and the constraints the DTOs declare
├── server.go               # Layer order: CORS, logging, metrics, auth, router
├── store.go                # auth.users, auth.roles, auth.users_roles
├── token.go                # Minting and bcrypt, the JVM's JwtService reproduced
└── go.mod                  # Go module dependencies
```

The router, the CORS layer and the metrics registry are not here: both Go
services serve through the one copy in
[`libs/backend/shared/servicehttp`](../../../libs/backend/shared/servicehttp),
so neither can resolve a path, refuse a request or label a series the other
would not.

## Minting lives here and nowhere else

`POST /auth/authenticate` is the only operation in the system that issues a
token, so `token.go` is the only production code that can sign one. It is
deliberately **not** in the shared auth library: anything able to sign a token
can sign itself any principal, and a library both services imported would hand
that to the one that must never have it. The shared library's `authtest` minter
stays test-only and a workspace check
(`tools/workspace-checks/test-only-go-packages.spec.ts`) fails if any shipped Go
file imports it.

The claim layout is asserted against `authtest`'s, which is in turn asserted
against a JVM-minted fixture in the shared library's own parity suite — so
`token_test.go` is an assertion about jjwt rather than about two Go files
agreeing with each other.

**Password hashing is bcrypt at cost 10**, the default `BCryptPasswordEncoder`
uses, and the comparison accepts the `$2a$`, `$2b$` and `$2y$` prefixes so hashes
already in `auth.users` verify unchanged. A password longer than 72 bytes is
truncated rather than refused, because Spring's BCrypt truncates silently and
refusing would lock out an account the JVM created.

A sign-in for an address with no row compares against a decoy hash and throws
the result away, so it costs what a real check costs. Without it the two
identical refusals would still be told apart by their durations, which is
`DaoAuthenticationProvider.mitigateAgainstTimingAttack` reproduced.

Neither the stored hash nor a cleartext password can reach a log line: both
types that hold one implement `LogValue`, which is what the JSON handler this
service installs actually consults — it marshals with `encoding/json` and never
looks for a `Stringer`.

## Configuration

The datasource, signing key and token lifetime are read from the variables
`usersrole` reads, so one chart value feeds both services through the cutover.

| Variable                               | Default                          | Purpose                                                           |
| -------------------------------------- | -------------------------------- | ----------------------------------------------------------------- |
| `UR_JWT_SECRET_KEY`                    | _required_                       | Base64 HMAC key, byte-identical to the JVM's `app.jwt.secret-key` |
| `UR_JWT_EXPIRATION_TIME_MS`            | `7200000`                        | Token lifetime                                                    |
| `UR_PG_DATASOURCE_URL`                 | _required_                       | The JDBC URL Spring reads; translated to a libpq URL at startup   |
| `UR_PG_USERNAME`                       | —                                | Database user                                                     |
| `UR_PG_PASSWORD`                       | —                                | Database password                                                 |
| `ID_PORT`                              | `8080`                           | Listen port                                                       |
| `ID_JWT_ISSUER_ORIGIN`                 | _required_                       | `scheme://host:port` stamped into every token this service mints  |
| `ID_JWT_ALLOW_ANY_ISSUER_AND_AUDIENCE` | unset                            | `true` accepts a token from any issuer; relaxes verification only |
| `ID_DB_MAX_CONNECTIONS`                | `5`                              | Pool ceiling                                                      |
| `ID_DB_MIN_CONNECTIONS`                | `2`                              | Pool floor                                                        |
| `ID_CORS_ALLOWED_ORIGIN_PATTERNS`      | `http://*:[*]`, `https://*:[*]`  | Spring's allowed-origin pattern syntax                            |
| `ID_CORS_ALLOWED_METHODS`              | the seven `SecurityConfig` lists | Preflight `Access-Control-Allow-Methods`                          |
| `ID_CORS_ALLOWED_HEADERS`              | `Authorization`, `Content-Type`  | Preflight `Access-Control-Allow-Headers`                          |
| `ID_SHUTDOWN_TIMEOUT_SECONDS`          | `10`                             | Drain window on `SIGTERM`                                         |

**`ID_JWT_ISSUER_ORIGIN` must match `PS_JWT_ISSUER_ORIGIN` on `profile-service`.**
It is what this service stamps into `aud` and `iss`, and what both services check
on the way back in. Setting the flag above accepts a token from any issuer but
does not supply one to mint with, so the origin stays required either way.

## Endpoints

Eighteen operations — `/auth` (2), `/api/users` (8), `/api/roles` (8) — exactly
the set the frozen contract describes; `contract_test.go` fails if the two sets
differ or if an operation is served under a rule the contract does not name.

Outside that surface, and outside authentication as `SecurityConfig`'s
`permitAll` matchers put them:

| Path                   | Purpose                                                   |
| ---------------------- | --------------------------------------------------------- |
| `/actuator/health`     | `{"status":"UP"}`, the shape the JVM's probe answers with |
| `/actuator/prometheus` | Scrape endpoint                                           |
| `/health`              | The path the sibling Go services expose                   |

## Testing

```bash
nx test identity-service        # go test -race
go test ./... -short            # skips the suites that need Docker
```

The suites are four:

- **Authorization parity** (`parity_test.go`) — every operation against every
  principal its rule admits and refuses, driven from the contract's own
  `x-authorization` values, including the two refusal shapes: an empty 401 and a
  403 carrying Boot's error body. The two public operations are driven with no
  token at all and asserted on their own success status, which is what separates
  `PUBLIC` from `AUTHENTICATED`: both carry no predicate, and only the filter
  chain's matcher list tells them apart.
- **Credential disclosure** (`logging_test.go`) — the default logger redirected
  into a buffer while the whole surface is driven over every outcome it has,
  asserting no line carries a cleartext password or a bcrypt prefix. The type
  redactions do not cover a handler that dereferences the password pointer into
  a log call, and this does.
- **Contract drift** (`contract_test.go`) — the served route set against the
  document, in both directions, plus the rule each operation is served under.
- **Storage and end-to-end** (`*_integration_test.go`) — the deployed
  `00_schema.sql` and `01_data.sql` in a Postgres container, covering the
  register-then-sign-in round trip, the delete cascade, the elevated-role guard,
  the pagination clamps and the two frozen 500s.

The seed data is loaded as well as the schema, because it is what populates the
role catalogue and the elevated-role guard names a role by id.

## Docker

```bash
nx local-build-image identity-service
nx serve-container identity-service
```

## Notes on parity

Behaviours reproduced deliberately rather than corrected, because a client keyed
on them would change at cutover:

- Updating a user onto an email address another user holds answers **500**, not
  409 — a unique-constraint violation no handler maps. Renaming a role onto a
  taken name is the same, and is asymmetric with the create, which pre-checks
  and answers 409.
- Deleting a user or a role that does not exist answers **204**.
- Every update to a user rewrites the bcrypt hash: both body fields are
  required, so a client changing only the address resends the password.
- The public registration audits its row as created by user id 1, hard-coded.
- Granting or revoking role id 1 requires the caller to already hold it, checked
  against `auth.users_roles` rather than against the token's roles claim,
  because the guard is written in terms of an id and the claim carries names.
- `@Email` is two checks rather than one regexp: Hibernate Validator's own
  address checks run before the pattern the DTO declares, so a 65-character
  local part, `a..b@x.co` and `a@-b.co` are all 400. Reproducing the pattern
  alone would let this service create rows the JVM refuses.

Deliberate departures, each with its reason:

- **`GET /api/roles` is paginated**, where the JVM reads the whole table
  unordered. Specified in the contract as a behaviour change; the deployed
  catalogue holds three rows, so no client sees a difference today. Declaring
  the parameters brings a 400 for a non-numeric `page` or `size` with them,
  where the JVM handler declares no arguments and never looks at the query
  string.
- **`User` carries `profileId`, not the embedded profile.** Specified in the
  contract; keeping the aggregate would make every user read a synchronous call
  into `profile-service`.
- **The issuer origin is configured, not derived from the request.** The JVM
  builds it from the incoming `Host` and gets away with it because it never
  checks `iss` on the way back in. Here the claim is verified, so a
  caller-controlled header deciding it would let one request mint a token the
  next one refuses.
- **The public registration checks the address before it encodes.** The JVM does
  too; the order is called out because reversing it makes every attempt at an
  address already registered cost a bcrypt round on an endpoint that takes no
  token. `PUT /api/users/{userId}` reads the row before it encodes for the same
  reason — a rule gates it, so it is waste rather than a flood, but it is the
  same asymmetry and `UserService.updateUser` does not have it.
- **A status the container sets carries the container's body.** An argument that
  will not convert, a path nothing routes, a method a path does not map, an
  `Accept` a `produces` cannot satisfy, a storage failure, a rule that could not
  be decided: none of these is a response a handler composed, so all of them
  answer the way the JVM answers — Boot's error JSON for a caller holding a
  verified token, and the empty 401 for one holding none, whatever status the
  first dispatch set. The rule, and what was measured to establish it, is
  `x-container-error` in the contract and the error-body section of
  `libs/backend/shared/auth/README.md`.

Two behaviours here were once recorded as departures and are not: both were the
contract being wrong about the JVM rather than this service diverging from it,
and both corrections are in `docs/contracts/README.md`.

- **A wrong password and an unknown email address answer identically** — an
  empty 401 with the `Access-Denied-Reason` header, after the same amount of
  work. The JVM has no second shape to offer: `SecurityConfig` builds its
  `DaoAuthenticationProvider` without `setHideUserNotFoundExceptions(false)`, so
  the default converts the unknown-user exception into a bad-credentials one
  inside the provider and the entry point answers both. Answering them
  differently, in shape or in timing, lets an anonymous caller enumerate which
  addresses are registered.
- **An empty grant or revoke list is 400**, matching the contract's
  `minItems: 1`. So is the JVM's: the `@NotEmpty` sits directly on the
  controller parameter, so built-in method validation refuses the call before
  the handler body runs.

## 📚 Related Packages

- [`profile-service`](../profile-service): the other Go half of the same split.
- [`backend-shared-auth`](../../../libs/backend/shared/auth): verification and
  the authorization rules both services decide from.
- [`backend-shared-servicehttp`](../../../libs/backend/shared/servicehttp): the
  router, CORS layer and metrics both services serve through.
- [`usersrole`](../usersrole): the Spring service whose behaviour this
  reproduces, and the home of the frozen contracts.
