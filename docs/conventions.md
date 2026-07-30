# Coding Conventions

## Angular

### Library Categories

| Type               | What goes here                    | Can import from       |
| ------------------ | --------------------------------- | --------------------- |
| `type:feature`     | Smart components, pages, routes   | ui, util, data-access |
| `type:ui`          | Dumb/presentational components    | util only             |
| `type:data-access` | Services, HTTP, state             | util, data-access     |
| `type:util`        | Pure functions, interfaces, pipes | util only             |

These rules (plus the scope and framework axes) are enforced by lint, not
just convention — see [Module Boundaries](#module-boundaries).

### File Naming

- Components: `my-thing.component.ts` / `.html` / `.scss` / `.spec.ts`
- Services: `my-thing.service.ts` / `.spec.ts`
- Guards: `my-thing.guard.ts`
- Interceptors: `my-thing.interceptor.ts`
- Models/interfaces: `my-thing.model.ts`
- Resolvers: `my-thing.resolver.ts`

No barrel files (`index.ts`) at the app level. Import directly from the source file.

### Component Style

- Standalone components (not NgModules) for new Angular 21 code
- Inject dependencies via `inject()` function, not constructor injection
- Use `signal()` and `computed()` for reactive state where appropriate
- SCSS for all styles, no inline styles

### Testing

- Vitest for unit tests (`.spec.ts` alongside the file)
- Playwright for E2E (all tests in `apps/e2e/platform-e2e/`)
- No `TestBed` bootstrapping shortcuts — configure properly with `TestBed.configureTestingModule`

## Module Boundaries

Every project carries three tag families in its `project.json` (`type:`,
`scope:`, `framework:`). The `@nx/enforce-module-boundaries` rule in
`eslint.config.ts` turns them into hard constraints — a dependency must
satisfy **all** axes that match the importing project, and violations fail
lint in CI.

### Type axis (layering)

| Source             | May depend on                  |
| ------------------ | ------------------------------ |
| `type:app`         | feature, ui, data-access, util |
| `type:feature`     | ui, data-access, util          |
| `type:data-access` | data-access, util              |
| `type:ui`          | util                           |
| `type:util`        | util                           |
| `type:e2e`         | anything (top-level consumer)  |

### Scope axis (app isolation)

A `scope:X` project may only import `scope:X` or `scope:shared` projects.
`scope:shared` may only import `scope:shared`. Cross-app reuse goes through a
shared lib, never a direct import of another app's scope.

### Framework axis (no cross-framework imports)

`framework:angular` code may only import `framework:angular` projects; the
same holds for `go`, `springboot`, `database`, and `node`. ESLint only lints
JS/TS, so in practice this bites on the Angular/Node side; Go and JVM
boundaries are additionally enforced by their own toolchains.

### Grandfathered exceptions (burn-down list)

Existing violations that predate enforcement are marked with
`eslint-disable @nx/enforce-module-boundaries` at the import site. Do not add
new ones — fix the layering instead. Current list:

- `frontend-container-data-access` → `frontend-shared-ui`
  (`dynamic-route-loader.service.ts` renders `FallbackComponent` when a
  remote fails to load; the component reference should move behind a route
  factory or into the feature layer)
- `frontend-shared-data-access` → `frontend-shared-ui`
  (`snackbar.service.ts` opens `SnackbarComponent` via `MatSnackBar`; same
  remedy — inject or relocate the component)

### Adding a new project

Declare all three tag families in `project.json` or the project will not
match the constraints above and its first cross-project import will be
rejected by lint.

## Go

- Standard Go package layout (no internal/ unless needed for encapsulation)
- Package names: lowercase single word, no underscores
- Error handling: always return errors, never swallow them

## Commits

See `CLAUDE.md` → Commit Conventions section. Types and rules are enforced by `.commitlintrc.json` — that file is the source of truth, not this doc.

## JSON / YAML

- `project.json`: 2-space indent, always include `$schema`
- `nx.json`, `package.json`: 2-space indent
- GitHub Actions YAML: 2-space indent

## Nx Targets

When adding a new Nx target to a `project.json`:

1. Use `command` for simple shell one-liners
2. Use `executor: nx:run-commands` for multi-step commands
3. Always add `cache: true` if the target output is deterministic
4. Run `pnpm exec nx reset` after editing `project.json`
