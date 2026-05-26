# ESLint v9/v10 Upgrade — Flat Config Migration

**Jira:** JDWLABS-4
**Date:** 2026-05-25
**Branch:** `feat/eslint-v10-upgrade`
**Status:** Approved for implementation

---

## Context

The monorepo currently runs ESLint 8.57.1 with 18 legacy `.eslintrc.json` files using cascading inheritance. ESLint v9 dropped `.eslintrc.*` support entirely in favour of flat config (`eslint.config.ts`). This ticket upgrades ESLint to v9.x (latest stable) and migrates all configs to flat format.

Go (`@nx-go/nx-go:lint` → golangci-lint) and Spring Boot (lint is a no-op) are unaffected.

---

## Package changes

| Package | Current | Action |
|---------|---------|--------|
| `eslint` | 8.57.1 | bump → `^9.x` latest |
| `globals` | — | add (replaces `env: { jest: true }`) |
| `@eslint/js` | — | add (provides `recommended` base) |
| `@nx/eslint` / `@nx/eslint-plugin` | 22.7.3 | no change — already flat-config-ready |
| `@typescript-eslint/*` | 8.40.0 | no change — v8 supports flat config |
| `@angular-eslint/*` | 21.2.0 | no change — v21 supports flat config |
| `jsonc-eslint-parser` | ^2.4.0 | no change |

---

## File inventory

18 `.eslintrc.json` files deleted. 18 `eslint.config.ts` files created in their place.

| Template | Projects | Count |
|----------|---------|-------|
| Root | `.` | 1 |
| Angular app (prefix: `app`) | authui, container, rolesui, usersui apps | 4 |
| Angular lib (prefix: `jdw`) | all non-shared libs | 10 |
| Angular lib + JSON dep-check | `shared/ui`, `shared/util` | 2 |
| E2E / TS-only | `platform-e2e` | 1 |

---

## Config designs

### Root `eslint.config.ts`

```ts
import nx from '@nx/eslint-plugin';
import globals from 'globals';

export default [
  ...nx.configs['flat/base'],
  {
    files: ['**/*.ts', '**/*.tsx', '**/*.js', '**/*.jsx'],
    rules: {
      '@nx/enforce-module-boundaries': [
        'error',
        {
          enforceBuildableLibDependency: true,
          allow: [],
          depConstraints: [
            { sourceTag: 'scope:shared', onlyDependOnLibsWithTags: ['scope:shared'] },
            { sourceTag: 'type:app', onlyDependOnLibsWithTags: ['type:feature', 'type:ui', 'type:util', 'type:data-access'] },
            { sourceTag: 'type:feature', onlyDependOnLibsWithTags: ['type:ui', 'type:util', 'type:data-access'] },
            { sourceTag: 'type:data-access', onlyDependOnLibsWithTags: ['type:util', 'type:data-access'] },
            { sourceTag: 'type:ui', onlyDependOnLibsWithTags: ['type:ui', 'type:util'] },
            { sourceTag: 'type:util', onlyDependOnLibsWithTags: ['type:util'] },
          ],
        },
      ],
    },
  },
  {
    files: ['**/*.ts', '**/*.tsx'],
    ...nx.configs['flat/typescript'][0],
    rules: { 'no-extra-semi': 'off' },
  },
  {
    files: ['**/*.js', '**/*.jsx'],
    ...nx.configs['flat/javascript'][0],
    rules: { 'no-extra-semi': 'off' },
  },
  {
    files: ['**/*.spec.ts', '**/*.spec.tsx', '**/*.spec.js', '**/*.spec.jsx'],
    languageOptions: { globals: { ...globals.jest } },
  },
];
```

---

### Template A — Angular app (`prefix: 'app'`)

Applies to: `apps/angular/{authui,container,rolesui,usersui}/<name>/eslint.config.ts`

```ts
import baseConfig from '../../../../eslint.config';
import nx from '@nx/eslint-plugin';

export default [
  ...baseConfig,
  ...nx.configs['flat/angular'],
  ...nx.configs['flat/angular-template'],
  {
    files: ['**/*.ts'],
    rules: {
      '@angular-eslint/directive-selector': ['error', { type: 'attribute', prefix: 'app', style: 'camelCase' }],
      '@angular-eslint/component-selector': ['error', { type: 'element', prefix: 'app', style: 'kebab-case' }],
      '@angular-eslint/prefer-standalone': 'off',
    },
  },
  {
    files: ['**/*.html'],
    rules: {},
  },
];
```

---

### Template B — Angular lib (`prefix: 'jdw'`)

Applies to: all `libs/angular/**` except `shared/ui` and `shared/util`

Same as Template A with `prefix: 'jdw'`.

---

### Template C — Angular lib + JSON dep-check

Applies to: `libs/angular/shared/ui`, `libs/angular/shared/util`

Template B + additional block:

```ts
  {
    files: ['**/*.json'],
    plugins: { '@nx': nx },
    languageOptions: { parser: await import('jsonc-eslint-parser') },
    rules: { '@nx/dependency-checks': 'error' },
  },
```

---

### Template D — E2E / TS-only

Applies to: `apps/angular/platform-e2e`

```ts
import baseConfig from '../../../eslint.config';
import nx from '@nx/eslint-plugin';

export default [
  ...baseConfig,
  ...nx.configs['flat/typescript'],
  { files: ['**/*.ts'] },
];
```

---

## Execution order

1. `pnpm add -D eslint@^9 globals @eslint/js`
2. Write root `eslint.config.ts`
3. Convert project configs: apps → libs → e2e
4. Delete all 18 `.eslintrc.json` files
5. Update `nx.json` `targetDefaults` inputs if referencing `.eslintrc` paths
6. `pnpm exec nx run-many -t lint` — all 24 projects pass
7. `pnpm exec nx run-many -t lint --skip-nx-cache` — confirm no cache false-positives

---

## nx.json changes required

Two places in `nx.json` reference ESLint config filenames — both need updating:

**`namedInputs.production`** — exclude `eslint.config.ts` from production hash (currently excludes `.eslintrc.json` and `eslint.config.js`):
```json
"!{projectRoot}/eslint.config.ts"
```
Keep the `.eslintrc.json` and `eslint.config.js` exclusions too (defensive, avoids re-hashing if any remain).

**`targetDefaults["@nx/eslint:lint"].inputs`** — swap `.eslintrc.json` for `eslint.config.ts`:
```json
"{workspaceRoot}/eslint.config.ts"
```
Remove `{workspaceRoot}/.eslintrc.json`. Keep `{workspaceRoot}/eslint.config.js` exclusion for safety.

---

## Edge cases

| Issue | Fix |
|-------|-----|
| Type-aware TS rules need `parserOptions.project` | Copy existing tsconfig refs into `languageOptions.parserOptions` per project |
| `process-inline-templates` must stay on `*.ts` block | Keep on `files: ['**/*.ts']` config object |
| `jsonc-eslint-parser` needs explicit `languageOptions.parser` | Handled in Template C |
| `nx.json` `namedInputs.production` excludes `.eslintrc.json` | Add `eslint.config.ts` exclusion |
| `nx.json` `targetDefaults["@nx/eslint:lint"].inputs` references `.eslintrc.json` | Swap to `eslint.config.ts` |

---

## Verification gate

- `nx run-many -t lint` passes for all 24 projects
- No lint rule count regression (same warnings/errors before and after on a clean run)
- CI pipeline lint step passes on PR
