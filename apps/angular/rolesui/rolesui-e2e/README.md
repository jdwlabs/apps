# RolesUI E2E

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**RolesUI E2E** is an end-to-end testing project for the RolesUI application, which is part of the JDW Platform’s micro
frontend architecture. It leverages Cypress to run automated tests against the RolesUI app to ensure its functionality
and smooth integration within the platform.

---

## 📁 Project Structure

```
apps/angular/rolesui/rolesui-e2e/
├── cypress.config.ts           # Cypress configuration file
├── .eslintrc.json              # ESLint configuration for E2E tests
├── project.json                # Nx project configuration
├── tsconfig.json               # TypeScript configuration for E2E tests
└── src
    ├── e2e
    │   └── app.cy.ts          # Cypress tests for RolesUI
    ├── fixtures
    │   └── example.json       # Sample fixture data
    └── support
        ├── app.po.ts          # Page object model
        ├── commands.ts        # Custom Cypress commands
        └── e2e.ts             # Cypress support configuration
```

---

## 🚀 Getting Started

### Prerequisites

- Node.js (LTS version recommended)
- Nx CLI (for running workspace commands)
- Cypress (included with the project)

### Development

Install dependencies using:

```bash
npm install
```

### Running E2E Tests

To execute the end-to-end tests for RolesUI:

```bash
nx e2e rolesui-e2e
```

This command launches the Cypress test runner and runs the test suite against your RolesUI application.

---

## 📦 Deployment / CI

RolesUI E2E tests are typically executed as part of your CI/CD pipeline to automatically validate application
functionality before deployment. Ensure that the RolesUI application is available and accessible during the test
execution in your CI environment.

---

## 📌 Notes

- This project uses [Nx](https://nx.dev/) for workspace management.
- It leverages [Cypress](https://www.cypress.io/) for writing and executing end-to-end tests.
- The folder structure is organized to clearly separate tests, fixtures, and support files, facilitating easy
  maintenance and scalability.
