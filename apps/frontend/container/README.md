# Container

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/container)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/container)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwlabs/container?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**Container** is the shell application of the JDW Platform’s micro frontend architecture. Built with Angular and Nx, it serves as the host that dynamically loads remote micro frontends (e.g., AuthUI, UsersUI, RolesUI) using module federation. It also provides common layout elements, navigation, and runtime configuration for all sub-applications.

---

## 📁 Project Structure

```
apps/frontend/container/
├── public/                      # Static assets (VERSION placeholder, favicon)
├── src/                         # Application source code
│   ├── app/                     # Shell layout, application config, and routing
│   ├── bootstrap.ts             # Angular bootstrap logic
│   ├── config.json              # Runtime configuration
│   ├── index.html               # Main HTML entry point
│   ├── main.ts                  # Angular main entry
│   ├── module-federation.manifest.json  # Remote MFE definitions used at runtime
│   └── styles.scss              # Global styles
├── Dockerfile                   # Docker configuration for containerized deployment
├── default.conf                 # Nginx configuration used in container
├── start-nginx.sh               # Entrypoint script for Nginx
├── module-federation.config.ts  # Module federation configuration (host setup)
├── webpack.config.ts            # Webpack base configuration
├── tsconfig*.json               # TypeScript configuration files
└── project.json                 # Nx project configuration
```

---

## 🚀 Getting Started

### Prerequisites

- Node.js (LTS version recommended)
- Angular CLI
- Nx CLI
- Docker (for containerization)

### Development

Install dependencies and serve the shell together with its remotes:

```bash
pnpm install
pnpm exec nx run-many -t serve -p container authui usersui rolesui
```

`nx serve container` starts only the shell (port 4200). It no longer builds or
serves the remotes listed in `src/module-federation.manifest.json`; start each
remote you need with its own `serve` target (`authui` 4201, `usersui` 4202,
`rolesui` 4203). The shell registers remotes at runtime from `servicediscovery`
(`GET /api/remotes`), falling back to `src/module-federation.manifest.json`
when that call fails or returns nothing usable.

### Build

```bash
nx build container
```

### Test

```bash
nx test container
```

---

## 🐳 Docker

To build and run the Docker container:

```bash
# Build Docker image for local development
nx run container:local-build-image

# Run the Docker container
nx run container:serve-container
```

---

## 🌐 Module Federation

This application acts as the **host** in the JDW Platform’s module federation setup. Remote micro frontends are registered at runtime from the `servicediscovery` remote map, with `src/module-federation.manifest.json` as the fallback, so the shell can load and integrate remote applications (such as AuthUI, UsersUI, and RolesUI) without rebuilding. Registration and loading live in `libs/frontend/container/util` (`remote-federation.ts`); the webpack federation config for all four apps is built by `tools/module-federation`.

---

## 📄 Configurations

- **Runtime Config:** `src/config.json` – dynamically loaded environment configuration.
- **Remote MFE Config:** `src/module-federation.manifest.json` – fallback definitions for remote micro frontends, used when `servicediscovery` is unreachable.
- **Web Server Config:** `default.conf` – Nginx configuration used during Docker container startup.

---

## 🔧 Scripts

- **start-nginx.sh** – Entrypoint script used in the Docker container to serve the app via Nginx.
- Additional build and configuration scripts are provided via Nx targets in the project configuration (`project.json`).

---

## 📦 Deployment

Deployment is typically handled via CI/CD pipelines and Docker containers. The project includes targets for building multi-platform Docker images and executing necessary configuration steps (e.g., prepare/restore config) to ensure the application is packaged correctly.

---

## 📌 Notes

- This app uses [Nx](https://nx.dev/) for workspace and monorepo support.
- Container is designed to be the central point of navigation and layout for the JDW Platform’s micro frontend ecosystem.
- All remote micro frontends are dynamically discovered and integrated at runtime using the provided manifest configuration.
