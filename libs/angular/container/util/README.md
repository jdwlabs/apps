# Angular Container Util

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**Angular Container Util** is a utility library for the container module in the JDW Platform. It provides shared
utilities, models, and helper functions such as route-related types.

This library is part of the **angular-container** module and follows a utility-driven organization under the monorepo
structure.

---

## 📁 Project Structure

```
libs/angular/container/util/
├── src
│   ├── lib
│   │   └── routes.model.ts           # Common route-related models
│   ├── index.ts                      # Library exports
│   └── test-setup.ts
├── jest.config.ts                    # Jest configuration
├── tsconfig.json                     # TypeScript base config
├── tsconfig.lib.json                 # Library-specific TypeScript config
├── tsconfig.spec.json                # Spec-specific TypeScript config
├── .eslintrc.json                    # ESLint config
├── project.json                      # Nx project definition
└── README.md                         # Project documentation
```

---

## 🚀 Getting Started

### Prerequisites

- Node.js (LTS version recommended)
- Nx CLI (for workspace and project management)
- Angular
- Jest (for unit testing)
- ESLint (for linting and code standards)

### Installation

Install project dependencies:

```bash
npm install
```

---

## 🧪 Testing

To run unit tests with Jest:

```bash
nx test angular-container-util
```

---

## ✅ Linting

To lint the codebase using ESLint:

```bash
nx lint angular-container-util
```

---

## 📝 Usage

You can import and use utilities or models in any Angular module or service.

Example of importing a user model:

```ts
import { MicroFrontendRoute } from '@jdw/angular-container-util';
```

---

## 📦 Build

To build this utility library:

```bash
nx build angular-container-util
```

The output will be placed in the `dist/` directory.

---

## 📌 Notes

- This library is designed to be lightweight and reusable across multiple features.
- Models and utility functions are standalone and do not rely on Angular components or services.
- It supports strict typing and unit tests for robust development.
