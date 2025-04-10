# Container E2E

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**Container E2E** is an end-to-end testing project for the Container application, which serves as the host in the JDW
Platform’s micro frontend architecture. It leverages Cypress to run automated tests against the Container app, verifying
its functionality and integration with remote micro frontends.

---

## 📁 Project Structure

```
apps/angular/container/container-e2e/
├── cypress.config.ts           # Cypress configuration file
├── .eslintrc.json              # ESLint configuration for E2E tests
├── project.json                # Nx project configuration
├── tsconfig.json               # TypeScript configuration for E2E tests
└── src
    ├── e2e
    │   └── app.cy.ts          # Cypress tests for the Container app
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
- Nx CLI (to run workspace commands)
- Cypress (included with the project)

### Development

Install dependencies:

```bash
npm install
```

### Running E2E Tests

To execute the end-to-end tests for the Container app:

```bash
nx e2e container-e2e
```

This command launches the Cypress test runner and runs the test suite against your Container application.

---

## 📦 Deployment / CI

Container E2E tests are typically executed as part of your CI/CD pipeline to automatically validate application
functionality before deployment. Ensure that the target Container application is up and accessible when these tests run
in your CI environment.

---

## 📌 Notes

- This project uses [Nx](https://nx.dev/) for workspace management.
- It leverages [Cypress](https://www.cypress.io/) for writing and executing end-to-end tests.
- The folder structure is organized to separate test files, fixtures, and support files, allowing for easy maintenance
  and scalability.
