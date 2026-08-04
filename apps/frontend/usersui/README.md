# UsersUI

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Docker Image Version](https://img.shields.io/docker/v/jdwlabs/usersui)
![Docker Image Size](https://img.shields.io/docker/image-size/jdwlabs/usersui)
![Docker Downloads](https://img.shields.io/docker/pulls/jdwlabs/usersui?label=downloads)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**UsersUI** is an Angular-based micro frontend application for managing users within the JDW Platform. It supports
functionality such as listing users, editing user details, assigning roles, and managing user status. This app is part
of the platform’s micro frontend architecture using module federation.

---

## 📁 Project Structure

```
apps/frontend/usersui/
├── public/                      # Static assets (VERSION placeholder, favicon)
├── src/                         # Application source code
│   ├── app/                     # App-specific configuration and routes
│   ├── bootstrap.ts             # Angular bootstrap logic
│   ├── config.json              # Runtime configuration
│   ├── index.html               # Main HTML entry point
│   ├── main.ts                  # Angular main entry
│   ├── styles.scss              # Global styles
│   └── test-setup.ts            # Vitest test setup
├── Dockerfile                   # Docker configuration for containerized deployment
├── default.conf                 # Nginx config used in container
├── start-nginx.sh               # Entrypoint script for Nginx
├── module-federation.config.ts  # Module federation configuration
├── webpack.config.ts            # Webpack base configuration
├── webpack.prod.config.ts       # Webpack production configuration
├── vite.config.ts               # Vite build and Vitest configuration
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

Install dependencies and serve the app locally:

```bash
npm install
nx serve usersui
```

### Build

```bash
nx build usersui
```

### Test

```bash
nx test usersui
```

---

## 🐳 Docker

To build and run the Docker container:

```bash
# Build Docker image for local development
nx run usersui:local-build-image

# Run the Docker container
nx run usersui:serve-container
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
multi-platform Docker images and executing necessary configuration steps (e.g., prepare/restore config) to ensure the
application is packaged correctly.

---

## 📌 Notes

- This app uses [Nx](https://nx.dev/) for workspace and monorepo support.
- UsersUI is designed to integrate seamlessly within the JDW Platform’s micro frontend architecture.
