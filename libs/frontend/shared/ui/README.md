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
│   ├── _custom-palettes.scss          # Generated M3 tonal palettes (schematic output)
│   ├── _nav-geometry.scss             # Shared rail/drawer widths (see invariant within)
│   └── _theming.scss                  # Theme mixin (mat.theme + extra semantic roles)
├── vite.config.ts
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

This library uses the Angular Material **M3 token-based theming system**
(`mat.theme()`). Each predefined theme lives in:

```
libs/frontend/shared/ui/src/lib/styles/themes/
```

A theme file exposes `@mixin emit($selector)`, which applies `theming.theme()`
and the component themes under the selector it is given. That emits the
Material system tokens (`--mat-sys-*`) plus this design system's extra semantic
roles (`--jdw-sys-success`, `--jdw-sys-info`, `--jdw-sys-warn` and their `on-`
counterparts).

Only the container includes these mixins, once per theme, each under
`html[data-theme='<id>']`. The remotes are federated into the container's
document and define no theme tokens of their own — a second stylesheet
defining `--mat-sys-*` would compete for the same custom properties on the
root element and be resolved by load order. Switching themes at runtime is
therefore a single attribute write on the document element.

The selector has to wrap the component themes as well as the token block. If
some rules were scoped and others were not, the specificity bump would apply
unevenly and change which rules win against Material's own.

Component styles should consume tokens (`var(--mat-sys-primary)`,
`var(--jdw-sys-success)`, …) instead of Sass theme maps. The
`user-custom-light` / `user-custom-dark` themes additionally re-point the key
system tokens at runtime CSS variables (`--primary-500`, `--accent-500`,
`--error-500`, `--success-500`, `--info-500`, `--warn-500`, the matching
`--*-contrast-500`, and the `--*-container-500` / `--*-container-contrast-500`
pairs), so user-supplied colors keep working without a rebuild.

Custom tonal palettes (blue-slate, user-custom defaults) are generated with
`ng generate @angular/material:theme-color` from the seed colors recorded
above each map — see `styles/_custom-palettes.scss`. Change the seed and
regenerate rather than hand-editing tone values.

---

## 🧪 Testing

### Unit Tests (Vitest)

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
