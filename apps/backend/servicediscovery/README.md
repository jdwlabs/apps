# ServiceDiscovery

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/servicediscovery)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/servicediscovery)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwlabs/servicediscovery?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**ServiceDiscovery** is a Go-based microservice that manages and serves metadata for dynamically loaded micro frontends
in the JDW Platform. It provides information such as remote configurations, micro frontend details, and service
versioning via RESTful APIs.

This project is managed using [Nx](https://nx.dev) and includes preconfigured Docker workflows for local development and
CI/CD pipelines.

---

## 📁 Project Structure

```
apps/backend/servicediscovery/
├── Dockerfile              # Defines the Go-based container for the service
├── Dockerfile.local        # Local development Dockerfile
├── project.json            # Nx project definition and targets
├── config.json             # Configuration file for remotes and micro frontends
├── main.go                 # Main Go application entry point
├── main_test.go            # Unit tests for the service
└── go.mod                  # Go module dependencies
```

---

## 🚀 Getting Started

### Prerequisites

- Docker
- [Nx CLI](https://nx.dev/)
- Go (1.24+)

### Run Locally (Ephemeral)

```bash
nx run servicediscovery:serve
```

Runs the service in a Docker container and exposes port `9000`.

### Run Locally with Persistent Volume

```bash
nx run servicediscovery:serve-cache
```

Creates and mounts a named Docker volume (`servicediscovery-data`) to persist data across runs.

### Clear Volume Cache

```bash
nx run servicediscovery:clear-cache
```

Removes the persistent Docker volume (`servicediscovery-data`).

---

## 🔨 Build

This project does not require a traditional build step but includes Docker image packaging targets:

### Build for CI/CD

```bash
npx nx run servicediscovery:build-image
```

Builds a multi-platform Docker image (`linux/amd64`, `linux/arm64`) and tags it with both `latest` and semantic version.

### Build for Local Use

```bash
npx nx run servicediscovery:local-build-image
```

Builds a single-platform (host) Docker image for development purposes.

---

## 🗃️ Service Configuration

The service configuration is stored in `config.json` and can be customized to include remote configurations and micro
frontend definitions.

### Example `config.json`:

```json
{
  "remotes": {
    "authui": "http://localhost:4201",
    "usersui": "http://localhost:4202",
    "rolesui": "http://localhost:4203"
  },
  "microFrontends": [
    {
      "name": "authui",
      "path": "auth",
      "remoteName": "authui",
      "moduleName": "./Routes",
      "url": "http://localhost:4201",
      "icon": "login",
      "title": "Auth",
      "description": "Sign in and sign up functionality"
    },
    {
      "name": "usersui",
      "path": "users",
      "remoteName": "usersui",
      "moduleName": "./Routes",
      "url": "http://localhost:4202",
      "icon": "groups",
      "title": "Users",
      "description": "Viewing users and managing profiles"
    },
    {
      "name": "rolesui",
      "path": "roles",
      "remoteName": "rolesui",
      "moduleName": "./Routes",
      "url": "http://localhost:4203",
      "icon": "lock",
      "title": "Roles",
      "description": "Viewing and managing roles"
    }
  ]
}
```

---

## 🌐 API Endpoints

- **GET /**: Returns a status message that the service is up.
- **GET /health**: Returns a health check message indicating the service is healthy.
- **GET /version**: Returns the current version of the service.
- **GET /api/remotes**: Returns the remotes configuration.
- **GET /api/micro-frontends**: Returns the list of micro frontends.

---

## 🧪 Versioning

Versioning is handled via [Conventional Commits](https://www.conventionalcommits.org/)
and [@jscutlery/semver](https://github.com/jscutlery/semver):

```bash
npx nx run servicediscovery:version
```

- Tags the repo
- Pushes a release commit
- Triggers `format` and `build-image`

---

## 📦 Deployment

This container image is used by other services in the JDW Platform during local development and CI pipelines.

---

## 📌 Notes

- This app provides a dynamic configuration for integrating micro frontends into the JDW Platform.
- The service runs in Docker containers and exposes APIs for fetching remotes and micro frontend metadata.
- Consider using environment variables to customize paths, ports, and configurations for different environments.
