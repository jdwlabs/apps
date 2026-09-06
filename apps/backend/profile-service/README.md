# ProfileService

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/profile-service)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/profile-service)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwlabs/profile-service?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**ProfileService** is the Go half of the `usersrole` split: the fifteen
`/api/profiles` operations, with their address and icon subresources, served
against the same `auth` schema the JVM service uses.

It is built against the frozen contract at
`apps/backend/usersrole/docs/contracts/profile-service.openapi.yaml` and
authorizes through `libs/backend/shared/auth`, which is the only place either Go
service parses a token.

No traffic is routed here yet. This project builds, tests and publishes an
image; the chart and the routing change land separately.

---

## 📁 Project Structure

```
apps/backend/profile-service/
├── Dockerfile              # Distroless runtime consuming the Nx build output
├── Dockerfile.local        # Self-contained build for local iteration
├── project.json            # Nx project definition and targets
├── config.go               # Environment resolution, done once at startup
├── cors.go                 # The CorsFilter SecurityConfig installs, reproduced
├── errors.go               # Status and media type per failure the JVM maps
├── handlers.go             # The fifteen operations and the rule each carries
├── main.go                 # Entry point, pool, graceful shutdown
├── metrics.go              # http_server_requests_seconds, as Micrometer names it
├── model.go                # Wire types and the two date formats Jackson writes
├── router.go               # Spring's path specificity, which ServeMux cannot express
├── server.go               # Layer order: CORS, logging, metrics, auth, router
├── store.go                # auth.profiles, auth.addresses, auth.profile_icons
└── go.mod                  # Go module dependencies
```

## Configuration

The datasource and signing key are read from the variables `usersrole` reads, so
one chart value feeds both services through the cutover.

| Variable                               | Default                          | Purpose                                                           |
| -------------------------------------- | -------------------------------- | ----------------------------------------------------------------- |
| `UR_JWT_SECRET_KEY`                    | _required_                       | Base64 HMAC key, byte-identical to the JVM's `app.jwt.secret-key` |
| `UR_PG_DATASOURCE_URL`                 | _required_                       | The JDBC URL Spring reads; translated to a libpq URL at startup   |
| `UR_PG_USERNAME`                       | —                                | Database user                                                     |
| `UR_PG_PASSWORD`                       | —                                | Database password                                                 |
| `PS_PORT`                              | `8080`                           | Listen port                                                       |
| `PS_JWT_ISSUER_ORIGIN`                 | _required_                       | `scheme://host:port` the tokens are minted from                   |
| `PS_JWT_ALLOW_ANY_ISSUER_AND_AUDIENCE` | unset                            | `true` accepts any issuer; refused alongside the variable above   |
| `PS_DB_MAX_CONNECTIONS`                | `5`                              | Pool ceiling                                                      |
| `PS_DB_MIN_CONNECTIONS`                | `2`                              | Pool floor                                                        |
| `PS_CORS_ALLOWED_ORIGIN_PATTERNS`      | `http://*:[*]`, `https://*:[*]`  | Spring's allowed-origin pattern syntax                            |
| `PS_CORS_ALLOWED_METHODS`              | the seven `SecurityConfig` lists | Preflight `Access-Control-Allow-Methods`                          |
| `PS_CORS_ALLOWED_HEADERS`              | `Authorization`, `Content-Type`  | Preflight `Access-Control-Allow-Headers`                          |
| `PS_SHUTDOWN_TIMEOUT_SECONDS`          | `10`                             | Drain window on `SIGTERM`                                         |

Starting without an expected issuer is a failure rather than a default: an unset
value is far more often an oversight than a decision, and the failure it causes —
accepting a token minted anywhere — is silent.

## Endpoints

Fifteen operations under `/api/profiles`, exactly the set the frozen contract
describes; `contract_test.go` fails if the two sets differ or if an operation is
served under a rule the contract does not name.

Outside that surface, and outside authentication as `SecurityConfig`'s
`permitAll` matchers put them:

| Path                   | Purpose                                                   |
| ---------------------- | --------------------------------------------------------- |
| `/actuator/health`     | `{"status":"UP"}`, the shape the JVM's probe answers with |
| `/actuator/prometheus` | Scrape endpoint                                           |
| `/health`              | The path the sibling Go services expose                   |

## Testing

```bash
nx test profile-service         # go test -race
go test ./... -short            # skips the suites that need Docker
```

The suites are three:

- **Authorization parity** (`parity_test.go`) — every operation against every
  principal its rule admits and refuses, driven from the contract's own
  `x-authorization` values, including the two refusal shapes: an empty 401 and a
  403 carrying Boot's error body.
- **Contract drift** (`contract_test.go`) — the served route set against the
  document, in both directions.
- **Storage and end-to-end** (`*_integration_test.go`) — the deployed
  `00_schema.sql` in a Postgres container, covering the address delete scoping,
  the icon caps and download, the pagination clamps and the JSON shapes.

## Docker

```bash
nx local-build-image profile-service
nx serve-container profile-service
```

## Notes on parity

Three behaviours are reproduced deliberately rather than corrected, because a
client keyed on them would change at cutover:

- Adding an address answers **200**, not 201, and returns the parent profile.
- Replacing an icon on a profile that has none answers **500**, not 404.
- Deleting a profile that does not exist answers **204**.

One is corrected rather than reproduced: replacing an icon stamps
`modified_by_user_id` with the acting user. The JVM writes the icon's original
creator there instead, which is a false audit record in a column this service's
own response exposes.
