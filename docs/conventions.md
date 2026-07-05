# Coding Conventions

## Angular

### Library Categories

| Type               | What goes here                    | Can import from       |
| ------------------ | --------------------------------- | --------------------- |
| `type:feature`     | Smart components, pages, routes   | ui, util, data-access |
| `type:ui`          | Dumb/presentational components    | ui, util              |
| `type:data-access` | Services, HTTP, state             | util, data-access     |
| `type:util`        | Pure functions, interfaces, pipes | util only             |

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

- Jest for unit tests (`.spec.ts` alongside the file)
- Playwright for E2E (all tests in `apps/e2e/platform-e2e/`)
- No `TestBed` bootstrapping shortcuts — configure properly with `TestBed.configureTestingModule`

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
