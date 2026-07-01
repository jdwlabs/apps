# jdwlabs/usersui

Users management **micro-frontend** for the jdwlabs platform.

![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/usersui?sort=semver)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/usersui/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/jdwlabs/usersui)

## What it is

A module-federation remote exposing the platform's user viewing and profile management
UI. It is loaded at runtime by the `jdwlabs/container` host shell and is not intended
to be browsed on its own. Served by nginx from a static build.

## Quick start

```bash
docker run -p 8080:80 jdwlabs/usersui:latest
```

## Exposed ports

| Port | Purpose      |
| ---- | ------------ |
| `80` | nginx (HTTP) |

## Environment variables

Runtime config is injected at container start: `start-nginx.sh` runs `envsubst` over
the built JS, substituting any `$NAME` placeholder with the value of the matching
environment variable before nginx starts. Set the platform's config values (API URL)
as environment variables at deployment time; no rebuild is needed to repoint an
environment.

## Tags

| Tag         | Meaning                        |
| ----------- | ------------------------------ |
| `latest`    | most recent release            |
| `{version}` | specific semver (e.g. `1.2.3`) |

## Source & license

Source: <https://github.com/jdwlabs/apps>
License: PolyForm Noncommercial 1.0.0
