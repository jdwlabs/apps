# Container

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwillmsen/jdw-container)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwillmsen/jdw-container)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwillmsen/jdw-container?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**Container** is the shell application of the JDW Platform’s micro frontend architecture. Built with Angular and Nx, it serves as the host that dynamically loads remote micro frontends (e.g., AuthUI, UsersUI, RolesUI) using module federation. It also provides common layout elements, navigation, and runtime configuration for all sub-applications.

---

## 📁 Project Structure

```
apps/angular/container/container/
├── public/                      # Static assets (e.g., VERSION file, favicon)
├── src/                         # Application source code
│   ├── app/                     # Shell layout, application config, and routing
│   ├── bootstrap.ts             # Angular bootstrap logic
│   ├── config.json              # Runtime configuration
│   ├── index.html               # Main HTML entry point
│   ├── main.ts                  # Angular main entry
│   ├── module-federation.manifest.json  # Remote MFE definitions used at runtime
│   ├── styles.scss              # Global styles
│   └── test-setup.ts            # Jest test setup
├── Dockerfile                   # Docker configuration for containerized deployment
├── default.conf                 # Nginx configuration used in container
├── start-nginx.sh               # Entrypoint script for Nginx
├── module-federation.config.ts  # Module federation configuration (host setup)
├── webpack.config.ts            # Webpack base configuration
├── jest.config.ts               # Jest configuration
├── tsconfig*.json               # TypeScript configuration files
├── project.json                 # Nx project configuration
└── CHANGELOG.md                 # Release history
```

---

## 🚀 Getting Started

### Prerequisites

- Node.js (LTS version recommended)
- Angular CLI
- Nx CLI
- Docker (for containerization)

### Development

Install dependencies and serve the app locally:

```bash
npm install
nx serve container
```

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

This application acts as the **host** in the JDW Platform’s module federation setup. Remote micro frontends are dynamically configured using the `module-federation.manifest.json` file, allowing the shell to load and integrate remote applications (such as AuthUI, UsersUI, and RolesUI) at runtime.

---

## 📄 Configurations

- **Runtime Config:** `src/config.json` – dynamically loaded environment configuration.
- **Remote MFE Config:** `module-federation.manifest.json` – definitions for remote micro frontends.
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
