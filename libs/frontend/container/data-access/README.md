# Angular Container Data Access

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

**Angular Container Data Access** is a library providing core data access services for the container module in the JDW Platform. This library includes services related to dynamic route loading, micro frontend management, and versioning.

This library is part of the **angular-container** module and follows a feature-driven architecture.

---

## 📁 Project Structure

```
libs/frontend/container/data-access/
├── src
│   ├── lib
│   │   ├── dynamic-route-loader       # Service for dynamic route loading
│   │   ├── micro-frontend             # Service for managing micro frontends
│   │   └── version                    # Service for version management
├── vite.config.ts                     # Vitest configuration for unit testing
├── tsconfig.json                      # TypeScript configuration
└── README.md                          # Project documentation
```

---

## 🚀 Getting Started

### Prerequisites

- Node.js (LTS version recommended)
- Nx CLI (for managing the workspace and building projects)
- Angular (for building the services)
- Vitest (for unit testing)

### Installation

To install dependencies:

```bash
npm install
```

---

## 🔧 Development

To contribute to this library, you can use the following commands:

- Run unit tests:

  ```bash
  nx test angular-container-data-access
  ```

- Lint the project:

  ```bash
  nx lint angular-container-data-access
  ```

---

## 🧪 Testing

This library uses **Vitest** for unit testing. You can run the tests using the following command:

```bash
nx test angular-container-data-access
```

---

## 📝 Usage

To use any of the services from this library, import the service into your Angular module or component and inject it as a dependency.

Example of using `DynamicRouteLoaderService`:

```typescript
import { DynamicRouteLoaderService } from '@jdw/angular/container/data-access';
import { inject } from '@angular/core';

// constructor(private routeLoader: DynamicRouteLoaderService) {}
const routeLoader = inject(DynamicRouteLoaderService);

ngOnInit();
{
  this.routeLoader.loadRoutes();
}
```

---

## 📦 Build

To build this library, run:

```bash
nx build angular-container-data-access
```

This will build the library and output it to the `dist/` folder.
