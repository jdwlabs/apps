# UsersUI E2E

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**UsersUI E2E** is an end-to-end testing project for the UsersUI application, part of the JDW Platform’s micro frontend architecture. This project uses Cypress to perform automated tests against the UsersUI app to verify its functionality and integration with the rest of the platform.

---

## 📁 Project Structure

```
apps/angular/usersui/usersui-e2e/
├── cypress.config.ts           # Cypress configuration file
├── .eslintrc.json              # ESLint configuration for E2E tests
├── project.json                # Nx project configuration
├── tsconfig.json               # TypeScript configuration for E2E tests
└── src
    ├── e2e
    │   └── app.cy.ts          # Cypress tests for UsersUI
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

Install dependencies:

```bash
npm install
```

### Running E2E Tests

To run the end-to-end tests for UsersUI:

```bash
nx e2e usersui-e2e
```

This will trigger Cypress to execute the test suite against your UsersUI application.

---

## 📦 Deployment / CI

UsersUI E2E tests are executed as part of the CI/CD pipeline to ensure that functionality is verified before deployment. Ensure that the UsersUI application is available and accessible in your CI environment for the tests to execute properly.

---

## 📌 Notes

- The project uses [Nx](https://nx.dev/) for workspace management.
- [Cypress](https://www.cypress.io/) is used for writing and running end-to-end tests.
- The folder structure is organized for ease of maintenance, with clear separation between tests, fixtures, and support files.
