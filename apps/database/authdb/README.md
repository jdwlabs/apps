# AuthDB

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/authdb)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/authdb)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwlabs/authdb?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**AuthDB** is the PostgreSQL database service for authentication and authorization in the JDW Platform. It provides the
schema and seed data for core identity services like `Usersrole`, `UsersUI`, and `RolesUI`.

This project is managed using [Nx](https://nx.dev) and includes preconfigured Docker workflows for local development and CI/CD pipelines.

---

## 📁 Project Structure

```
database/authdb/
├── Dockerfile           # Defines the Postgres container
├── project.json         # Nx project definition and targets
└── src/
    ├── 00_schema.sql    # SQL schema definitions (tables, relations, constraints)
    └── 01_data.sql      # Initial seed data (e.g., default roles, users)
```

---

## 🚀 Getting Started

### Prerequisites

- Docker
- [Nx CLI](https://nx.dev/)

### Run Locally (Ephemeral)

```bash
nx run authdb:serve
```

Runs the database in a Docker container and exposes port `5432`.

### Run Locally with Persistent Volume

```bash
nx run authdb:serve-cache
```

Creates and mounts a named Docker volume (`authdb-data`) to persist data across runs.

### Clear Volume Cache

```bash
nx run authdb:clear-cache
```

Removes the persistent Docker volume (`authdb-data`).

> **Upgrading across a Postgres major version:** an `authdb-data` volume initialized by an
> older major version will not start under the new image. The contents are pure seed data,
> so run `nx run authdb:clear-cache` and start the container again to re-initialize.

---

## 🔨 Build

This project does not require a traditional build step but includes Docker image packaging targets:

### Build for CI/CD

```bash
npx nx run authdb:build-image
```

Builds a multi-platform Docker image (`linux/amd64`, `linux/arm64`) and tags with both `latest` and semantic version.

### Build for Local Use

```bash
npx nx run authdb:local-build-image
```

Builds a single-platform (host) Docker image for development purposes.

---

## 🗃️ Database Configuration

After starting the container, the following credentials are available by default:

```
Host:     localhost
Port:     5432
Database: auth
User:     postgres
Password: postgres
```

---

## 📄 SQL Files

- `00_schema.sql` – defines all tables, relationships, constraints, etc.
- `01_data.sql` – seed data like default roles, admin users, permissions, etc.

These are executed automatically during container initialization.

---

## 🧪 Versioning

Versioning is handled via [Conventional Commits](https://www.conventionalcommits.org/) and [@jscutlery/semver](https://github.com/jscutlery/semver):

```bash
npx nx run authdb:version
```

- Tags the repo
- Pushes a release commit
- Triggers `format` and `build-image`

---

## 📦 Deployment

This container image is used by other services in the JDW Platform during local dev and CI pipelines.

---

## 📌 Notes

- No runtime application is included; this project provides a clean and reproducible Postgres environment.
- Consider using migration tooling (e.g., [Sqitch](https://sqitch.org/), [Flyway](https://flywaydb.org/)) for production environments if schema versioning grows more complex.
