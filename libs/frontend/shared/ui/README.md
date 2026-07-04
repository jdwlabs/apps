# Angular Shared UI

![Build](https://img.shields.io/github/actions/workflow/status/jdwlabs/apps/ci.yml?branch=main)
![Nx](https://img.shields.io/badge/Nx-managed-blue)

The **Angular Shared UI** library provides a suite of reusable UI components and styling foundations for the JDW
platform. It centralizes layout, navigation, snackbar messaging, and design theming to maintain consistency across apps
and micro frontends.

---

## 📦 Package

- **Name:** `@jdw/angular-shared-ui`
- **Version:** `0.0.1`
- **Type:** Angular Library (UI Components)
- **Dependencies:**
  - `@angular/material`
  - `@jdw/angular-shared-util`

---

## 📁 Project Structure

```
libs/frontend/shared/ui/
├── src
│   ├── lib
│   │   ├── fallback/                  # Fallback UI component
│   │   ├── header/                    # App header with branding and controls
│   │   ├── navigation-item/           # Nav item used in layout/tile menus
│   │   ├── navigation-layout/         # Main navigation layout container
│   │   ├── navigation-tile/           # Dashboard tile-style navigation
│   │   ├── snackbar/                  # Snackbar alert messages
│   │   ├── styles/                    # Global SCSS styles and theming
│   │   └── test-setup.ts
│   └── index.ts
├── styles/
│   ├── themes/                        # Predefined theme files
│   ├── components/                    # SCSS for individual UI elements
│   └── _theming.scss                  # Global SCSS mixins and palette
├── jest.config.ts
├── ng-package.json
├── tsconfig.*.json
├── cypress/                          # Cypress component test setup
│   ├── fixtures/
│   ├── support/
│   └── component-index.html
└── README.md
```

---

## 🚀 Getting Started

### Prerequisites

Install peer dependencies in your consumer app:

```bash
npm install @angular/material @angular/core @angular/common @angular/router
```

Then install the shared UI package:

```bash
npm install @jdw/angular-shared-ui
```

---

## 🔧 Usage

Import only the components you need. Example for `NavigationLayoutComponent`:

```ts
import { NavigationLayoutComponent } from '@jdw/angular-shared-ui';
```

### Sample Component Usage

```html
<jdw-navigation-layout>
  <jdw-navigation-tile
    [title]="'Dashboard'"
    [description]="'Main control center'"
    [link]="'/dashboard'"
  ></jdw-navigation-tile>
</jdw-navigation-layout>
```

---

## 🎨 Theming

This library supports multiple predefined themes stored in:

```
libs/frontend/shared/ui/src/lib/styles/themes/
```

To switch themes, apply the appropriate SCSS class (e.g. `deeppurple-amber`, `indigo-pink`, or your custom one like
`user-custom-dark`) to your root container.

---

## 🧪 Testing

### Unit Tests (Jest)

```bash
nx test angular-shared-ui
```

### Component Tests (Cypress)

```bash
nx component-test angular-shared-ui
```

---

## ✨ Features

- Reusable layout components (header, nav, tiles)
- Snackbar messaging
- Flexible SCSS theming with prebuilt palettes
- Component-level Cypress tests for visual confidence

---

## 📝 Notes

- Avoid coupling UI components with business logic.
- Prefer `@jdw/angular-shared-data-access` or `@jdw/angular-shared-util` for services or utilities.
- All UI elements are fully themeable and testable.

---

## 📌 Related Packages

- [`@jdw/angular-shared-util`](../util): For utility services and helpers
- [`@jdw/angular-shared-data-access`](../data-access): For centralized data services
