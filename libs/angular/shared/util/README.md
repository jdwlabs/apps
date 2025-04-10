# Angular Shared Util

![Build](https://img.shields.io/github/actions/workflow/status/jdwillmsen/jdw/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

The **Angular Shared Util** library provides common TypeScript models, constants, utility functions, and environment
tokens used across Angular apps and libraries within the JDW platform. It acts as a foundational library to promote
consistency and reuse of logic and types.

---

## 📦 Package

- **Name:** `@jdw/angular-shared-util`
- **Version:** `0.0.1`
- **Type:** Angular Utility Library
- **Dependencies:** None (pure TypeScript)

---

## 📁 Project Structure

```
libs/angular/shared/util/
├── src
│   ├── lib/
│   │   ├── *.model.ts                # Data models (role, user, alert, etc.)
│   │   ├── *.constant.ts             # Constants (messages, validators)
│   │   ├── *.util.ts                 # Utility functions (error handling, etc.)
│   │   └── environment.token.ts      # DI token for runtime environment config
│   └── index.ts
├── jest.config.ts
├── tsconfig.*.json
└── README.md
```

---

## 🔧 Usage

### Importing a Model

```ts
import { Role } from '@jdw/angular-shared-util';

const adminRole: Role = {
  id: '1',
  name: 'Admin',
  description: 'Full access',
  status: 'ACTIVE',
  users: [],
  createdByUserId: 'system',
  createdTime: new Date().toISOString(),
  modifiedByUserId: 'system',
  modifiedTime: new Date().toISOString(),
};
```

### Using a Utility Function

```ts
import { getErrorMessage } from '@jdw/angular-shared-util';

catchError((error) => {
  const message = getErrorMessage(error);
  this.snackbarService.show(message);
  return of(null);
});
```

### Injecting Environment Token

```ts
import { inject } from '@angular/core';
import { ENVIRONMENT } from '@jdw/angular-shared-util';

const env = inject(ENVIRONMENT);
console.log('Current API:', env.apiUrl);
```

---

## ✨ Features

- ✅ Shared TypeScript models for roles, users, navigation, alerts, and more
- ✅ Global environment token for runtime config
- ✅ Constants for common messages and form validation
- ✅ Reusable utility functions like HTTP error message parsing

---

## 🧪 Testing

Run unit tests with:

```bash
nx test angular-shared-util
```

---

## 📌 Related Packages

- [`@jdw/angular-shared-ui`](../ui): Shared UI components
- [`@jdw/angular-shared-data-access`](../data-access): Shared data services

---

## 📝 Notes

- This library is framework-agnostic and does not depend on Angular modules.
- You can use this library in both Angular apps and Node-based tooling if needed.

