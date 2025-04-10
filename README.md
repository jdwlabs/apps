# JDW Monorepo

[![CI](https://github.com/jdwillmsen/jdw/actions/workflows/ci.yml/badge.svg)](https://github.com/jdwillmsen/jdw/actions/workflows/ci.yml)
![Java](https://img.shields.io/badge/Java-21-blue)
![Go](https://img.shields.io/badge/Go-1.24-blue)
![Node](https://img.shields.io/badge/Node-22-blue)
![Nx](https://img.shields.io/badge/Nx-20-blue)
![Cypress](https://img.shields.io/badge/Cypress-13-blue)
![Angular](https://img.shields.io/badge/Angular-19-blue)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

JDW Monorepo is a multi-language, multi-project repository that houses all code, configuration, and tooling for the JDW
Platform. This repository is organized into three main directories:

- **apps**: Contains full application code for both frontends and backends.
- **libs**: Contains reusable libraries, grouped by type:
    - **feature**: Components and views specific to a feature or domain.
    - **data-access**: Logic for communicating with backends, APIs, or databases.
    - **util**: Common utilities, helper functions, and shared models.
    - **ui**: Reusable UI components, theming, and styling.
- **tools**: Contains scripts and configuration for versioning, formatting, Docker orchestration, and CI/CD.

## 🔗 Environment Links

Here are the links to the various environments for the JDW Platform:

| Environment | URL                                    | Description                                  |
| ----------- | -------------------------------------- | -------------------------------------------- |
| Development | [container.dev.jdwkube.com](https://container.dev.jdwkube.com/) | For development and testing of new features. |
| UAT         | [container.uat.jdwkube.com](https://container.uat.jdwkube.com/) | For user acceptance testing before release. |
| Production  | [container.prd.jdwkube.com](https://container.prd.jdwkube.com/) | The live, production environment.          |

## Directory Structure

```
.
├── apps/                    # Complete applications
│   ├── angular/             # Grouped by framework / language
│   │   ├── container/       # Specific application
│   │   ├── usersui/
│   │   └── authui/
│   ├── go/
│   │   ├── servicediscovery/
│   │   └── emailsender/
│   └── springboot/
│       └── usersrole/
│
├── libs/                    # Reusable libraries
│   ├── angular/             # Grouped by framework
│   │   ├── container/       # App-specific libraries
│   │   │   ├── feature/
│   │   │   └── util/
│   │   ├── usersui/
│   │   │   ├── feature/
│   │   │   ├── data-access/
│   │   │   └── util/
│   │   ├── shared/          # Framework-wide shared
│   │   │   ├── ui/
│   │   │   ├── util/
│   │   │   └── data-access/
│   ├── go/
│   │   ├── shared/
│   │   └── servicediscovery/
│   │       └── util/
│   └── shared/              # Cross-framework shared
│       └── utils/
│
└── tools/                   # Monorepo tooling
```

### Key Structural Principles:

- **Framework/Language Grouping**: Top-level organization by technology (Angular, Go, Java).
- **App-Specific Isolation**: Libraries scoped to specific applications.
- **Shared Code Hierarchy**:
    - **App-Scoped**: Only used by one application (e.g., `angular/usersui/*`).
    - **Framework-Shared**: Shared within a framework (e.g., `angular/shared/*`).
    - **Cross-Framework**: Shared across technologies (e.g., `shared/*`).
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

- Build the `angular-usersui-data-access` library:

  ```bash
  npx nx build angular-usersui-data-access
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

The deployment configuration for the JDW platform applications is maintained in a separate repository:

- **JDW Apps**: [https://github.com/jdwillmsen/jdw-apps](https://github.com/jdwillmsen/jdw-apps)
    - Uses Helm charts and pipelines for automated deployments via Argo CD.

### Kubernetes Infrastructure

The infrastructure code, including Kubernetes manifests, Helm charts, and Argo CD configurations, is housed in:

- **JDW Kube**: [https://github.com/jdwillmsen/jdw-kube](https://github.com/jdwillmsen/jdw-kube)
    - Manages cluster configurations and automated deployments using ArgoCD.

## 📚 Library Overview

The monorepo organizes libraries by type to encourage reuse and maintainability:

- **Feature Libraries**: Provide UI components and feature-specific logic (e.g., `angular-usersui-feature-core`).
- **Data-Access Libraries**: Encapsulate API communication and business logic (e.g., `angular-usersui-data-access`).
- **Util Libraries**: Offer shared TypeScript utilities, helper functions, and models (e.g., `angular-usersui-util`,
  `angular-shared-util`).
- **UI Libraries**: Supply reusable UI components and theming (e.g., `angular-shared-ui`).

## ✨ Additional Resources

- **NX Documentation**: [https://nx.dev/](https://nx.dev/)
- **JDW Apps Deployment Repository**: [https://github.com/jdwillmsen/jdw-apps](https://github.com/jdwillmsen/jdw-apps)
- **JDW Kubernetes Infrastructure**: [https://github.com/jdwillmsen/jdw-kube](https://github.com/jdwillmsen/jdw-kube)
- **Docker Hub Images**: [https://hub.docker.com/u/jdwillmsen](https://hub.docker.com/u/jdwillmsen)

## 📌 About This Workspace

This monorepo leverages Nx for efficient task management and CI/CD across multiple languages and projects, promoting
code reuse and maintainability.

### **Maintainer:**

- Jake Willmsen
