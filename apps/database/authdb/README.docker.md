# jdwlabs/authdb

PostgreSQL image with **jdwlabs auth schema** initialization scripts.

![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/authdb?sort=semver)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/authdb/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/jdwlabs/authdb)

## What it is

A `postgres:16.4` image that bundles the jdwlabs auth database bootstrap. The SQL in
`/docker-entrypoint-initdb.d/` runs on first startup against an empty data directory,
creating the schema and seed data the `jdwlabs/usersrole` backend expects.

## Quick start

```bash
docker run -p 5432:5432 \
  -e POSTGRES_USER=jdw \
  -e POSTGRES_PASSWORD=change-me \
  -e POSTGRES_DB=jdw \
  jdwlabs/authdb:latest
```

The init scripts only run when the data volume is empty (standard Postgres behavior).

## Exposed ports

| Port   | Purpose    |
| ------ | ---------- |
| `5432` | PostgreSQL |

## Environment variables

| Name                | Default            | Purpose                  |
| ------------------- | ------------------ | ------------------------ |
| `POSTGRES_USER`     | `default_user`     | superuser name           |
| `POSTGRES_PASSWORD` | `default_password` | superuser password       |
| `POSTGRES_DB`       | `jdw`              | database created on init |

Override the defaults in any non-local environment.

## Tags

| Tag         | Meaning                        |
| ----------- | ------------------------------ |
| `latest`    | most recent release            |
| `{version}` | specific semver (e.g. `1.2.3`) |

## Source & license

Source: <https://github.com/jdwlabs/apps>
License: PolyForm Noncommercial 1.0.0
