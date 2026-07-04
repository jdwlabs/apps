# jdwlabs/servicediscovery

Micro-frontend **service discovery API** for the jdwlabs platform.

![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/servicediscovery?sort=semver)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/servicediscovery/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/jdwlabs/servicediscovery)

## What it is

A small Go service that tells the `jdwlabs/container` host shell which micro-frontend
remotes exist and where to load them from. It serves the remote map and the
micro-frontend catalog (name, route, icon, title) that drive the platform's runtime
module-federation wiring. Built on distroless for a minimal attack surface.

## Quick start

```bash
docker run -p 9000:9000 jdwlabs/servicediscovery:latest
curl http://localhost:9000/health
```

## Exposed ports

| Port   | Purpose  |
| ------ | -------- |
| `9000` | HTTP API |

## Endpoints

| Method / Path              | Purpose                          |
| -------------------------- | -------------------------------- |
| `GET /health`              | Liveness/readiness check         |
| `GET /version`             | Running build version            |
| `GET /api/remotes`         | Module-federation remote URL map |
| `GET /api/micro-frontends` | Micro-frontend catalog           |

## Environment variables

| Name      | Default | Purpose          |
| --------- | ------- | ---------------- |
| `SD_PORT` | `9000`  | HTTP listen port |

## Tags

| Tag         | Meaning                        |
| ----------- | ------------------------------ |
| `latest`    | most recent release            |
| `{version}` | specific semver (e.g. `1.2.3`) |

## Source & license

Source: <https://github.com/jdwlabs/apps>
License: PolyForm Noncommercial 1.0.0
