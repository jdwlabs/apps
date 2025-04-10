# AuthUI E2E

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**AuthUI E2E** is an end-to-end testing application for the AuthUI micro frontend. It leverages Cypress to run automated
tests against the AuthUI application, ensuring its reliability and smooth integration within the JDW Platform’s micro
frontend architecture.

---

## 📁 Project Structure

```
apps/angular/authui/authui-e2e/
├── cypress.config.ts           # Cypress configuration file
├── .eslintrc.json              # ESLint configuration
├── project.json                # Nx project configuration
├── tsconfig.json               # TypeScript configuration for e2e tests
└── src
    ├── e2e
    │   └── app.cy.ts          # Cypress tests for AuthUI
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

Install dependencies with:

```bash
npm install
```

### Running E2E Tests

To execute the end-to-end tests for AuthUI:

```bash
nx e2e authui-e2e
```

This command will launch the Cypress test runner and run the test suite against your AuthUI application.

---

## 📦 Deployment / CI

AuthUI E2E tests are typically executed as part of your CI/CD pipeline to automatically validate application
functionality before deployment. Ensure that the target AuthUI application is available for testing when running these
tests in your CI environment.

---

## 📌 Notes

- This project uses [Nx](https://nx.dev/) for workspace management.
- It leverages [Cypress](https://www.cypress.io/) for writing and executing end-to-end tests.
- The folder structure is organized to clearly separate test files, fixtures, and custom commands, making maintenance
  and scalability easier.

