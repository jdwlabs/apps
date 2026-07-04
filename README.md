# Apps

[![CI](https://github.com/jdwlabs/apps/actions/workflows/ci.yml/badge.svg)](https://github.com/jdwlabs/apps/actions/workflows/ci.yml)
![Java](https://img.shields.io/badge/Java-21-blue)
![Go](https://img.shields.io/badge/Go-1.23-blue)
![Node](https://img.shields.io/badge/Node-24-blue)
![Nx](https://img.shields.io/badge/Nx-22-blue)
![Playwright](https://img.shields.io/badge/Playwright-1.60-blue)
![Angular](https://img.shields.io/badge/Angular-21-blue)
[![License](https://img.shields.io/badge/License-PolyForm%20NonCommercial%201.0-blue)](https://polyformproject.org/licenses/noncommercial/1.0.0/)

This is a multi-language, multi-project repository that houses all code, configuration, and tooling. This repository is
organized into three main directories:

- **apps**: Contains full application code for both frontends and backends.
- **libs**: Contains reusable libraries, grouped by type:
  - **feature**: Components and views specific to a feature or domain.
  - **data-access**: Logic for communicating with backends, APIs, or databases.
  - **util**: Common utilities, helper functions, and shared models.
  - **ui**: Reusable UI components, theming, and styling.
- **tools**: Contains scripts and configuration for versioning, formatting, Docker orchestration, and CI/CD.

## Directory Structure

```
.
├── apps/                    # Complete applications
│   ├── frontend/            # Angular micro-frontends
│   │   ├── container/
│   │   ├── authui/
│   │   ├── usersui/
│   │   └── rolesui/
│   ├── backend/             # Go + Spring Boot services
│   │   ├── servicediscovery/
│   │   └── usersrole/
│   ├── database/            # DB migrations
│   │   └── authdb/
│   └── e2e/                 # Playwright end-to-end tests
│       └── platform-e2e/
│
├── libs/                    # Reusable libraries
│   ├── frontend/            # Angular shared libs
│   │   ├── authui/
│   │   │   ├── feature/
│   │   │   └── util/
│   │   ├── container/
│   │   │   ├── feature/
│   │   │   └── util/
│   │   ├── rolesui/
│   │   │   ├── feature/
│   │   │   └── util/
│   │   ├── usersui/
│   │   │   ├── feature/
│   │   │   ├── data-access/
│   │   │   └── util/
│   │   └── shared/          # Cross-app frontend shared
│   │       ├── ui/
│   │       ├── util/
│   │       └── data-access/
│   └── backend/             # Go shared libs
│       └── shared/
│           └── util/
│
└── tools/                   # Monorepo tooling
```

### Key Structural Principles:

- **Role-Based Grouping**: Top-level organization by role (`frontend`, `backend`, `e2e`, `database`).
- **App-Specific Isolation**: Libraries scoped to specific applications.
- **Shared Code Hierarchy**:
  - **App-Scoped**: Only used by one application (e.g., `frontend/usersui/*`).
  - **Frontend-Shared**: Shared across Angular apps (e.g., `frontend/shared/*`).
- **Library Types**:
  - `feature/`: Domain-specific components and logic.
  - `data-access/`: API/backend communication.
  - `util/`: Helper functions and utilities.
  - `ui/`: Reusable UI components.

## 🚀 Running Tasks

Execute tasks with Nx using the following syntax:

```bash
npx nx <target> <project> [options]
```

**Examples:**

- Build the `frontend-usersui-data-access` library:

  ```bash
  npx nx build frontend-usersui-data-access
  ```

- Run multiple targets:

  ```bash
  npx nx run-many -t <target1> <target2>
  ```

- Filter specific projects:

  ```bash
  npx nx run-many -t <target1> <target2> -p <proj1> <proj2>
  ```

Learn more at [Nx Documentation](https://nx.dev/features/run-tasks).

## 🌐 Explore the Project Graph

Generate an interactive visualization of the workspace dependencies:

```bash
npx nx graph
```

This graph helps you understand how projects are connected and see which tasks can be executed. See more
at [NX Explore Graph](https://nx.dev/core-features/explore-graph).

## 📦 Deployment & Infrastructure

### App Deployment

Helm charts and Argo CD configurations for the applications live in:

- **Deployments**: [https://github.com/jdwlabs/deployments](https://github.com/jdwlabs/deployments)

### Platform & Infrastructure

Cluster configuration, Terraform, and tenant definitions live in:

- **Infrastructure**: [https://github.com/jdwlabs/infrastructure](https://github.com/jdwlabs/infrastructure)
- **Platform**: [https://github.com/jdwlabs/platform](https://github.com/jdwlabs/platform)

## 📚 Library Overview

The monorepo organizes libraries by type to encourage reuse and maintainability:

- **Feature Libraries**: Provide UI components and feature-specific logic (e.g., `frontend-usersui-feature-core`).
- **Data-Access Libraries**: Encapsulate API communication and business logic (e.g., `frontend-usersui-data-access`).
- **Util Libraries**: Offer shared TypeScript utilities, helper functions, and models (e.g., `frontend-usersui-util`,
  `frontend-shared-util`).
- **UI Libraries**: Supply reusable UI components and theming (e.g., `frontend-shared-ui`).

## ✨ Additional Resources

- **NX Documentation**: [https://nx.dev/](https://nx.dev/)
- **Deployments**: [https://github.com/jdwlabs/deployments](https://github.com/jdwlabs/deployments)
- **Infrastructure**: [https://github.com/jdwlabs/infrastructure](https://github.com/jdwlabs/infrastructure)
- **Platform**: [https://github.com/jdwlabs/platform](https://github.com/jdwlabs/platform)
- **Docker Hub Images**: [https://hub.docker.com/u/jdwlabs](https://hub.docker.com/u/jdwlabs)

## 📌 About This Workspace

This monorepo leverages Nx for efficient task management and CI/CD across multiple languages and projects, promoting
code reuse and maintainability.

### **Maintainer:**

- Jake Willmsen
