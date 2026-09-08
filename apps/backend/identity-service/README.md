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
├── cors.go                 # The CorsFilter SecurityConfig installs, reproduced
├── errors.go               # Status and media type per failure the JVM maps
├── handlers.go             # The eighteen operations and the rule each carries
├── main.go                 # Entry point, pool, graceful shutdown
├── metrics.go              # http_server_requests_seconds, as Micrometer names it
├── model.go                # Wire types and the constraints the DTOs declare
├── router.go               # Spring's path specificity and refusal ordering
├── server.go               # Layer order: CORS, logging, metrics, auth, router
├── store.go                # auth.users, auth.roles, auth.users_roles
├── token.go                # Minting and bcrypt, the JVM's JwtService reproduced
└── go.mod                  # Go module dependencies
```

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

The suites are three:

- **Authorization parity** (`parity_test.go`) — every operation against every
  principal its rule admits and refuses, driven from the contract's own
  `x-authorization` values, including the two refusal shapes: an empty 401 and a
  403 carrying Boot's error body. The two public operations are driven the other
  way — they must _not_ answer 401 — because `PUBLIC` and `AUTHENTICATED` both
  carry no predicate and only the filter chain's matcher list separates them.
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

Deliberate departures, each with its reason:

- **`GET /api/roles` is paginated**, where the JVM reads the whole table
  unordered. Specified in the contract as a behaviour change; the deployed
  catalogue holds three rows, so no client sees a difference today.
- **`User` carries `profileId`, not the embedded profile.** Specified in the
  contract; keeping the aggregate would make every user read a synchronous call
  into `profile-service`.
- **A wrong password and an unknown email address answer identically** — an
  empty 401 with the `Access-Denied-Reason` header. The contract's 401 permits
  either shape on this operation, and answering them differently lets an
  anonymous caller enumerate which addresses are registered.
- **An empty grant or revoke list is 400**, matching the contract's `minItems: 1`
  and its 400 response, rather than the 500 the JVM reaches by reading the first
  element of an empty list.
- **The issuer origin is configured, not derived from the request.** The JVM
  builds it from the incoming `Host` and gets away with it because it never
  checks `iss` on the way back in. Here the claim is verified, so a
  caller-controlled header deciding it would let one request mint a token the
  next one refuses.

## 📚 Related Packages

- [`profile-service`](../profile-service): the other Go half of the same split.
- [`backend-shared-auth`](../../../libs/backend/shared/auth): verification and
  the authorization rules both services decide from.
- [`usersrole`](../usersrole): the Spring service whose behaviour this
  reproduces, and the home of the frozen contracts.
