# jdwlabs/profile-service

**Profiles, addresses and icons API** for the jdwlabs platform.

![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/profile-service?sort=semver)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/profile-service/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/jdwlabs/profile-service)

## What it is

A Go service serving the fifteen `/api/profiles` operations of the jdwlabs auth
surface: profile records, their addresses and a single PNG icon each. It reads
and writes the same `auth` schema the JVM `usersrole` service uses, and
authorizes from the verified JWT alone — no call to another service on the
request path. Built on distroless for a minimal attack surface, running as a
numeric non-root uid.

## Quick start

```bash
docker run -p 8080:8080 \
  -e UR_JWT_SECRET_KEY="$(base64 < /dev/urandom | head -c 44)" \
  -e UR_PG_DATASOURCE_URL="jdbc:postgresql://authdb:5432/jdw" \
  -e UR_PG_USERNAME=jdw \
  -e UR_PG_PASSWORD=jdw \
  -e PS_JWT_ISSUER_ORIGIN="http://localhost:8080" \
  jdwlabs/profile-service:latest

curl http://localhost:8080/actuator/health
```

The service needs a reachable Postgres carrying the `auth` schema — the
`jdwlabs/authdb` image ships it — and refuses to start without one.

## Exposed ports

| Port   | Purpose  |
| ------ | -------- |
| `8080` | HTTP API |

## Environment

| Variable                               | Default    | Purpose                                      |
| -------------------------------------- | ---------- | -------------------------------------------- |
| `UR_JWT_SECRET_KEY`                    | _required_ | Base64 HMAC key shared with the token issuer |
| `UR_PG_DATASOURCE_URL`                 | _required_ | JDBC URL of the auth database                |
| `UR_PG_USERNAME`                       | —          | Database user                                |
| `UR_PG_PASSWORD`                       | —          | Database password                            |
| `PS_PORT`                              | `8080`     | Listen port                                  |
| `PS_JWT_ISSUER_ORIGIN`                 | _required_ | Origin the tokens are minted from            |
| `PS_JWT_ALLOW_ANY_ISSUER_AND_AUDIENCE` | unset      | `true` accepts a token from any issuer       |
| `PS_DB_MAX_CONNECTIONS`                | `5`        | Connection pool ceiling                      |
| `PS_DB_MIN_CONNECTIONS`                | `2`        | Connection pool floor                        |
| `PS_CORS_ALLOWED_ORIGIN_PATTERNS`      | any origin | Comma-separated allowed-origin patterns      |
| `PS_CORS_ALLOWED_METHODS`              | seven      | Comma-separated preflight methods            |
| `PS_CORS_ALLOWED_HEADERS`              | two        | Comma-separated preflight headers            |
| `PS_SHUTDOWN_TIMEOUT_SECONDS`          | `10`       | Drain window on `SIGTERM`                    |

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
  `apps/backend/profile-service`

## License

PolyForm Noncommercial 1.0.0
