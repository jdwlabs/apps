# Scripts Directory

Non-Nx shell scripts and Docker helpers for the jdwlabs monorepo.
These are **not Nx projects** and are excluded from Nx project detection via `.nxignore`.

## Shell Scripts

All scripts are executable and accept arguments described in the usage comment at the top of each file.

| Script                  | Purpose                                                                                                                               |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `build-image.sh`        | Build a `jdwlabs/*` image with the full OCI label set and hand off to `docker buildx` (caller passes `--platform`, `--push`/`--load`) |
| `prepare-config.sh`     | Swap `config.json` to its production values before a Docker image build                                                               |
| `prepare-version.sh`    | Stamp the released version into an Angular app's `public/VERSION` before an image build                                               |
| `resolve-version.sh`    | Print a project's released version, from `RELEASE_VERSION` or the matching git tag                                                    |
| `restore-config.sh`     | Restore `config.json` to its development defaults after a build                                                                       |
| `restore-version.sh`    | Put the `public/VERSION` development placeholder back after an image build                                                            |
| `update-app.sh`         | Open a pull request that bumps the `appVersion` in a Helm `Chart.yaml` to match the released semver tag, then merge it                |
| `update-description.sh` | Sync a DockerHub repo's Overview from that image's `README.docker.md` (needs `DOCKERHUB_USERNAME`/`DOCKERHUB_PASSWORD`)               |

### resolve-version.sh

`resolve-version.sh <project>`

The single source of a released version. Resolution order:

1. `RELEASE_VERSION` — set by the deliver matrix, which reads it off the tag the
   release job just pushed. Preferred in CI, where the checkout is one commit
   with no tags.
2. A `{project}-{version}` tag pointing at `HEAD`.
3. The newest reachable `{project}-*` tag.

Exits non-zero when none resolves. Nothing in the working tree records a
version, so there is no manifest to fall back to and no placeholder worth
publishing.

Tests run offline against throwaway git repos: `bash scripts/tests/resolve-version.test.sh`

### update-app.sh

`update-app.sh <chart_file_path> <version> <project_name>`

Opens an auto-merging pull request on `jdwlabs/deployments` that sets the chart's
`appVersion`. Requires `gh` and `jq`, and a token from either
`GH_APP_ID` + `GH_APP_PRIVATE_KEY` or `DEPLOYMENTS_TOKEN`.

Every mutation goes through the GitHub API rather than a clone, so the commit is
minted server-side under the App's identity. Reruns are idempotent: the branch is
rebuilt from the current `main` tip, and a chart that already pins the requested
version is left untouched.

Tests run offline with a stubbed `gh`: `bash scripts/tests/update-app.test.sh`

## Docker

| File                  | Purpose                                                                     |
| --------------------- | --------------------------------------------------------------------------- |
| `docker/compose.yaml` | Docker Compose config for the local development stack (databases, services) |
| `docker/config.json`  | Docker-related configuration shared across services                         |

## Usage

```bash
# Start the local stack
docker compose -f scripts/docker/compose.yaml up -d

# Prepare an app config before building manually
bash scripts/prepare-config.sh apps/frontend/container/src/config.json

# Restore config after manual build
bash scripts/restore-config.sh apps/frontend/container/src/config.json

# Read the version a project would ship
bash scripts/resolve-version.sh container
```

> These scripts are typically invoked by Nx targets (via `project.json` `command` entries), not directly by developers.
