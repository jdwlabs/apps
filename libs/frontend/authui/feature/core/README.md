# Angular AuthUI Feature Core

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**Angular AuthUI Feature Core** is a library containing shared features for the AuthUI component set in the JDW Platform. This library includes common UI components such as `SignIn`, `SignUp`, `Forbidden`, and `EmailSignIn`, designed for the authentication flows of the platform.

This library is part of the **angular-authui** module and follows the feature-driven architecture.

---

## 📁 Project Structure

```
libs/frontend/authui/feature/core/
├── cypress/                    # Cypress files
├── src
│   ├── lib
│   │   ├── email-sign-in
│   │   ├── forbidden
│   │   ├── sign-in
│   │   └── sign-up
├── vite.config.ts              # Vitest configuration for unit tests
├── tsconfig.json               # TypeScript configuration
└── README.md                   # Project documentation
```

---

## 🚀 Getting Started

### Prerequisites

- Node.js (LTS version recommended)
- Nx CLI (for managing the workspace and building projects)
- Angular (for building the components)
- Cypress (for component testing)

### Installation

To install dependencies:

```bash
npm install
```

### Development

This library contains UI components for AuthUI and is part of a larger Angular application. You can build and test individual components as follows:

#### Running Unit Tests

Run the unit tests for the library:

```bash
nx test angular-authui-feature-core
```

#### Running Cypress Component Tests

To run Cypress component tests for specific components like `sign-in`:

```bash
nx run angular-authui-feature-core:component-test
```

#### Running the Library

This library is intended to be used in conjunction with other libraries or applications in the JDW Platform. To build and integrate it into your project:

```bash
nx build angular-authui-feature-core
```

---

## 📦 Deployment / CI

The library is part of the CI pipeline, and tests are run automatically during the build process. Ensure that all components are functioning properly before deploying the platform.

---

## 📌 Notes

- The library is part of a modular architecture using Nx workspaces.
- Components are isolated and can be tested individually using Cypress for component testing.
- Custom components like `SignIn`, `SignUp`, `Forbidden`, and `EmailSignIn` follow a consistent design and functionality pattern, making them reusable across different parts of the platform.
