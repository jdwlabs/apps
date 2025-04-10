# AuthUI

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwillmsen/jdw-authui)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwillmsen/jdw-authui)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwillmsen/jdw-authui?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**AuthUI** is an Angular-based micro frontend application responsible for handling authentication-related UIs such as
login, registration, password reset, and more. It is designed to operate within a module federation setup as part of the
JDW Platform’s micro frontend architecture.

---

## 📁 Project Structure

```
apps/angular/authui/authui/
├── public/                      # Static assets (e.g., VERSION file, favicon)
├── src/                         # Application source code
│   ├── app/                     # Application-specific configuration and routes
│   ├── bootstrap.ts             # Angular bootstrap logic
│   ├── config.json              # Runtime configuration
│   ├── index.html               # Main HTML entry point
│   ├── main.ts                  # Angular main entry
│   ├── styles.scss              # Global styles
│   └── test-setup.ts            # Jest test setup
├── Dockerfile                   # Docker configuration for containerized deployment
├── default.conf                 # Nginx config used in container
├── start-nginx.sh               # Entrypoint script for Nginx
├── module-federation.config.ts  # Module federation configuration
├── webpack.config.ts            # Webpack base configuration
├── webpack.prod.config.ts       # Webpack production configuration
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
nx serve authui
```

### Build

```bash
nx build authui
```

### Test

```bash
nx test authui
```

---

## 🐳 Docker

To build and run the Docker container:

```bash
# Build Docker image for local development
nx run authui:local-build-image

# Run the Docker container
nx run authui:serve-container
```

---

## 🌐 Module Federation

This application is configured as a remote in a module federation setup. Refer to `module-federation.config.ts` for
details on the exposed modules and configuration.

---

## 📄 Configurations

- **Runtime Config:** `src/config.json` – dynamically loaded environment configuration.
- **Web Server Config:** `default.conf` – Nginx configuration used during Docker container startup.

---

## 🔧 Scripts

- **start-nginx.sh** – Entrypoint script for starting Nginx.
- Additional build and configuration scripts are provided via Nx targets in the project configuration (`project.json`).

---

## 📦 Deployment

Deployment is typically handled via CI/CD pipelines and Docker containers. The project includes targets for building
multi-platform Docker images as well as specific steps (e.g., prepare/restore config) to ensure the application is
packaged correctly.

---

## 📌 Notes

- This app uses [Nx](https://nx.dev/) for workspace and monorepo support.
- AuthUI is designed to seamlessly integrate with other micro frontends within the JDW Platform via dynamic module
  federation.
