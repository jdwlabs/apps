# Angular Shared Data Access

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**Angular Shared Data Access** is a centralized library for shared Angular services related to authentication, roles, and UI messaging (snackbars). It promotes reuse and consistency across multiple modules within the JDW platform.

This library lives under the **angular/shared** scope and follows the `data-access` convention for service-oriented logic.

---

## 📁 Project Structure

```
libs/frontend/shared/data-access/
├── src
│   ├── lib
│   │   ├── auth/                         # Auth-related services
│   │   ├── roles/                        # Role management service
│   │   └── snackbar/                     # UI toast/snackbar notifications
│   ├── index.ts                          # Public exports
│   └── test-setup.ts
├── vite.config.ts                        # Vitest config
├── tsconfig.json                         # Base TypeScript config
├── tsconfig.lib.json                     # Library-specific TypeScript config
├── tsconfig.spec.json                    # Spec/test TypeScript config
├── .eslintrc.json                        # ESLint config
├── project.json                          # Nx project metadata
└── README.md                             # Project documentation
```

---

## 🚀 Getting Started

### Prerequisites

- Node.js (LTS)
- Nx CLI
- Angular
- Vitest & ESLint

### Install

Install dependencies:

```bash
npm install
```

---

## 🧪 Testing

Run unit tests using Vitest:

```bash
nx test angular-shared-data-access
```

---

## ✅ Linting

Lint the codebase:

```bash
nx lint angular-shared-data-access
```

---

## ✨ Features

- **AuthService**: Authentication and session-related utilities.
- **RolesService**: Role-based logic for UI or routing.
- **SnackbarService**: Consistent UX for toast/snackbar feedback messages.

---

## 📝 Usage Example

```ts
import { AuthService, SnackbarService } from '@jdw/angular-shared-data-access';

class Example {
  private authService = inject(AuthService);
  private snackbarService = inject(SnackbarService);

  login() {
    this.authService.login('user', 'pass').subscribe({
      next: () => this.snackbar.success('Login successful'),
      error: () => this.snackbar.error('Login failed'),
    });
  }
}
```

---

## 📌 Notes

- This library provides only logic and service layers — no UI components.
- It is intended to be reused in both container and feature libraries.
- Test coverage is expected for all exported services.
