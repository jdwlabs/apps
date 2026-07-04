# jdwlabs/usersrole

Users & roles **backend service** for the jdwlabs platform.

![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/usersrole?sort=semver)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/usersrole/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/jdwlabs/usersrole)

## What it is

A Spring Boot (Java 21) REST service backing the platform's user and role management.
It exposes secured endpoints consumed by the `usersui` and `rolesui` micro-frontends
and persists to the `jdwlabs/authdb` PostgreSQL database. Built with Jib on an
`eclipse-temurin:21` base.

## Quick start

```bash
docker run -p 8080:8080 \
  -e SPRING_DATASOURCE_URL=jdbc:postgresql://authdb:5432/jdw \
  -e SPRING_DATASOURCE_USERNAME=jdw \
  -e SPRING_DATASOURCE_PASSWORD=change-me \
  jdwlabs/usersrole:latest
```

## Exposed ports

| Port   | Purpose                          |
| ------ | -------------------------------- |
| `8080` | HTTP API (incl. actuator health) |

## Environment variables

Standard Spring Boot externalized config. Common overrides:

| Name                         | Purpose                        |
| ---------------------------- | ------------------------------ |
| `SPRING_DATASOURCE_URL`      | JDBC URL for the auth database |
| `SPRING_DATASOURCE_USERNAME` | database user                  |
| `SPRING_DATASOURCE_PASSWORD` | database password              |

## Tags

| Tag         | Meaning                        |
| ----------- | ------------------------------ |
| `latest`    | most recent release            |
| `{version}` | specific semver (e.g. `1.2.3`) |

## Source & license

Source: <https://github.com/jdwlabs/apps>
License: PolyForm Noncommercial 1.0.0
