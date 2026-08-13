# Angular UsersUI Feature Core

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

The **Angular UsersUI Feature Core** library provides key standalone components and layout features for the Users UI
module in the JDW platform. It includes foundational UI elements such as `user`, `profile`, `dashboard`, and `account`
views, supporting consistent and modular frontend architecture.

---

## 📁 Project Structure

```
libs/frontend/usersui/feature/core/
├── src
│   ├── index.ts                    # Public API export for the library
│   ├── lib
│   │   ├── user/                   # Standalone component for user view
│   │   ├── profile/                # Standalone component for profile view
│   │   ├── dashboard/              # Dashboard layout and features
│   │   ├── account/                # Account management UI
│   │   └── lib.routes.ts           # Route definitions for the feature
│   └── test-setup.ts               # Test setup configuration
├── vite.config.ts                  # Vitest configuration for unit testing
├── cypress.config.ts               # Cypress configuration for component testing
├── tsconfig.json                   # Base TypeScript configuration
├── tsconfig.lib.json               # Library-specific TypeScript configuration
└── tsconfig.spec.json              # Test-specific TypeScript configuration
```

---

## 🔧 Usage

This library provides Angular standalone components that can be used directly in route configurations or imported in
other feature modules.

### Example Route Registration

```ts
export const routes: Route[] = [
  {
    path: 'user',
    loadComponent: () =>
      import('@jdw/angular-usersui-feature-core/user').then(
        (m) => m.UserComponent,
      ),
  },
  {
    path: 'profile',
    loadComponent: () =>
      import('@jdw/angular-usersui-feature-core/profile').then(
        (m) => m.ProfileComponent,
      ),
  },
];
```

### Direct Component Import

```ts
import { DashboardComponent } from '@jdw/angular-usersui-feature-core/dashboard';

@Component({
  selector: 'app-wrapper',
  standalone: true,
  imports: [DashboardComponent],
  template: `<app-dashboard />`,
})
export class WrapperComponent {}
```

---

## 🧪 Testing

Run unit tests for the library with:

```bash
nx test angular-usersui-feature-core
```

Run Cypress component tests with:

```bash
nx component-test angular-usersui-feature-core
```

---

## 🛠 Linting

To check code quality with ESLint:

```bash
nx lint angular-usersui-feature-core
```

---

## 📌 Notes

- This library serves as the foundation for the Users UI feature layer.
- All components follow Angular’s standalone API and best practices.
- Designed for reusability in micro frontend contexts and scalable application structure.

---

## 📚 Related Packages

- [`@jdw/angular-usersui-data-access`](../../data-access): Handles user and profile API communication.
- [`@jdw/angular-usersui-util`](../../util): Shared utilities and models used across Users UI.
