# jdwlabs/container

Angular module-federation **host shell** for the jdwlabs platform.

![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/container?sort=semver)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/container/latest)
![Docker Pulls](https://img.shields.io/docker/pulls/jdwlabs/container)

## What it is

The container is the shell application that hosts the platform's micro-frontends
(`authui`, `usersui`, `rolesui`). It loads the remotes at runtime via module
federation, using the discovery data served by `jdwlabs/servicediscovery`. Served by
nginx from a static build.

## Quick start

```bash
docker run -p 8080:80 jdwlabs/container:latest
```

Then open <http://localhost:8080>.

## Exposed ports

| Port | Purpose      |
| ---- | ------------ |
| `80` | nginx (HTTP) |

## Environment variables

Runtime config is injected at container start: `start-nginx.sh` runs `envsubst` over
the built JS, substituting any `$NAME` placeholder with the value of the matching
environment variable before nginx starts. Set the platform's config values (API and
remote URLs) as environment variables at `docker run` / deployment time; no rebuild is
needed to repoint an environment.

## Tags

| Tag         | Meaning                        |
| ----------- | ------------------------------ |
| `latest`    | most recent release            |
| `{version}` | specific semver (e.g. `1.2.3`) |

## Source & license

Source: <https://github.com/jdwlabs/apps>
License: PolyForm Noncommercial 1.0.0
