# Angular Container Feature Core

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**Angular Container Feature Core** is a component library providing shared UI elements for the container module in the JDW Platform. This library includes key components like `Dashboard` and `Main`, and supports both unit and Cypress component testing.

This library is part of the **angular-container** module and follows a feature-driven architecture.

---

## 📁 Project Structure

```
libs/angular/container/feature/core/
├── src
│   ├── lib
│   │   ├── dashboard      # Dashboard UI component
│   │   └── main           # Main layout UI component
│   ├── index.ts
│   └── test-setup.ts
├── cypress/               # Cypress component test config & support files
├── cypress.config.ts      # Cypress component test configuration
├── jest.config.ts         # Jest unit test configuration
├── tsconfig*.json         # TypeScript configuration
└── README.md              # Project documentation
```

---

## 🚀 Getting Started

### Prerequisites

- Node.js (LTS version recommended)
- Nx CLI (for managing the workspace and building projects)
- Angular (for building components)
- Cypress (for component testing)
- Jest (for unit testing)

### Installation

To install dependencies:

```bash
npm install
```

---

## 🔧 Development

This library contains core UI components and is part of a larger Angular application. You can build and test individual components as follows:

### Unit Tests

To run unit tests with Jest:

```bash
nx test angular-container-feature-core
```

### Cypress Component Tests

To run Cypress component tests:

```bash
nx run angular-container-feature-core:component-test
```

---

## 🧪 Testing

- **Jest** is used for unit tests.
- **Cypress** is used for testing components in isolation.

Both test runners are configured and can be executed via Nx.

---

## 📝 Usage

To use any of the components from this library, import them into your Angular module or component.

Example usage of `DashboardComponent`:

```ts
import { DashboardComponent } from '@jdw/angular/container/feature/core';

@Component({
  selector: 'app-shell',
  standalone: true,
  imports: [DashboardComponent],
  template: `<app-dashboard></app-dashboard>`,
})
export class ShellComponent {}
```

---

## 📦 Build

To build this library:

```bash
nx build angular-container-feature-core
```

The output will be generated in the `dist/` folder.

---

## 📌 Notes

- This library is part of the modular monorepo managed by Nx.
- All components are written in Angular and tested using Jest and Cypress.
- It adheres to a standalone and feature-first approach using modern Angular.
