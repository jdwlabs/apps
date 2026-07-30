# Angular UsersUI Util

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

The **Angular UsersUI Util** library provides shared utilities, helper functions, and models used throughout the Users
UI module in the JDW platform. This includes reusable date utilities and strongly-typed user-related models to ensure
consistency and reduce duplication across the codebase.

---

## 📁 Project Structure

```
libs/frontend/usersui/util/
├── src
│   ├── index.ts                    # Public API export for the library
│   ├── lib
│   │   ├── date.util.ts            # Utility functions for date manipulation
│   │   └── date.util.spec.ts       # Unit tests for date utilities
│   └── test-setup.ts               # Test setup configuration
├── vite.config.ts                  # Vitest configuration for unit testing
├── tsconfig.json                   # Base TypeScript configuration
├── tsconfig.lib.json               # Library-specific TypeScript configuration
├── tsconfig.spec.json              # Test-specific TypeScript configuration
└── .eslintrc.json                  # ESLint configuration
```

---

## 🔧 Usage

This library is designed to be imported across multiple libraries and components to ensure centralized utility logic and
typings.

### Using a Utility Function

```ts
import { dateSortComparator } from '@jdw/angular-usersui-util';
```

### Using a Shared Model

```ts
import { User } from '@jdw/angular-shared-util';
```

---

## 🧪 Testing

Run unit tests for the library with:

```bash
nx test angular-usersui-util
```

---

## 🛠 Linting

To check code quality with ESLint:

```bash
nx lint angular-usersui-util
```

---

## 📌 Notes

- This library contains utilities that are framework-agnostic and safe to use across both feature and data-access
  layers.
- All utility functions are unit tested and follow consistent naming conventions.
- Models are strongly typed and align with backend contract definitions where possible.

---

## 📚 Related Packages

- [`@jdw/angular-usersui-data-access`](../data-access): Handles API communication for users and profiles.
- [`@jdw/angular-usersui-feature-core`](../feature/core): Contains core UI components for Users UI.
