# jdwlabs/identity-service

**Authentication, users and roles API** for the jdwlabs platform.

![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/identity-service?sort=semver)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/identity-service/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/jdwlabs/identity-service)

## What it is

A Go service serving the eighteen `/auth`, `/api/users` and `/api/roles`
operations of the jdwlabs auth surface: sign-in, self-registration, user records
and the role catalogue with its grants. It reads and writes the same `auth`
schema the JVM `usersrole` service uses, issues the HS256 tokens the rest of the
platform verifies, and stores passwords bcrypt-encoded. Built on distroless for a
minimal attack surface, running as a numeric non-root uid.

## Quick start

```bash
docker run -p 8080:8080 \
  -e UR_JWT_SECRET_KEY="$(base64 < /dev/urandom | head -c 44)" \
  -e UR_PG_DATASOURCE_URL="jdbc:postgresql://authdb:5432/jdw" \
  -e UR_PG_USERNAME=jdw \
  -e UR_PG_PASSWORD=jdw \
  -e ID_JWT_ISSUER_ORIGIN="http://localhost:8080" \
  jdwlabs/identity-service:latest

curl http://localhost:8080/actuator/health
```

The service needs a reachable Postgres carrying the `auth` schema — the
`jdwlabs/authdb` image ships it — and refuses to start without one.

The HMAC key must be byte-identical everywhere and 32 to 47 bytes once decoded;
a longer key makes the JVM sign with a stronger HMAC variant that the Go
verifiers refuse.

## Exposed ports

| Port   | Purpose  |
| ------ | -------- |
| `8080` | HTTP API |

## Environment

| Variable                               | Default    | Purpose                                            |
| -------------------------------------- | ---------- | -------------------------------------------------- |
| `UR_JWT_SECRET_KEY`                    | _required_ | Base64 HMAC key shared with every verifier         |
| `UR_JWT_EXPIRATION_TIME_MS`            | `7200000`  | Token lifetime                                     |
| `UR_PG_DATASOURCE_URL`                 | _required_ | JDBC URL of the auth database                      |
| `UR_PG_USERNAME`                       | —          | Database user                                      |
| `UR_PG_PASSWORD`                       | —          | Database password                                  |
| `ID_PORT`                              | `8080`     | Listen port                                        |
| `ID_JWT_ISSUER_ORIGIN`                 | _required_ | Origin stamped into every token this service mints |
| `ID_JWT_ALLOW_ANY_ISSUER_AND_AUDIENCE` | unset      | `true` accepts a token from any issuer             |
| `ID_DB_MAX_CONNECTIONS`                | `5`        | Connection pool ceiling                            |
| `ID_DB_MIN_CONNECTIONS`                | `2`        | Connection pool floor                              |
| `ID_CORS_ALLOWED_ORIGIN_PATTERNS`      | any origin | Comma-separated allowed-origin patterns            |
| `ID_CORS_ALLOWED_METHODS`              | seven      | Comma-separated preflight methods                  |
| `ID_CORS_ALLOWED_HEADERS`              | two        | Comma-separated preflight headers                  |
| `ID_SHUTDOWN_TIMEOUT_SECONDS`          | `10`       | Drain window on `SIGTERM`                          |

## Observability

| Path                   | Purpose                                        |
| ---------------------- | ---------------------------------------------- |
| `/actuator/health`     | Liveness and readiness, `{"status":"UP"}`      |
| `/actuator/prometheus` | Scrape endpoint                                |
| `/health`              | Alias, for parity with the sibling Go services |

Request timings are published as `http_server_requests_seconds`, labelled
`method`, `uri`, `status` and `outcome` — the series name and labels Micrometer
publishes for the JVM service, so existing dashboards keep working.

## Tags

- `latest` — most recent release
- `X.Y.Z` — immutable semver release

## Source

- [jdwlabs/apps](https://github.com/jdwlabs/apps) —
  `apps/backend/identity-service`

## License

PolyForm Noncommercial 1.0.0
