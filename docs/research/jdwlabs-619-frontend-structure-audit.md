# Frontend structure audit (JDWLABS-619)

Research only; no code changed. Snapshot of `main` at `2ce85d18`, Nx 23.3.1, Angular 22.2.3.

## 1. Current map

### Apps (`apps/frontend`)

| Project     | Role                          | Build executor                | Serve executor                             | Port |
| ----------- | ----------------------------- | ----------------------------- | ------------------------------------------ | ---- |
| `container` | MF host/shell                 | `@nx/angular:webpack-browser` | `@nx/angular:module-federation-dev-server` | 4200 |
| `authui`    | MF remote, exposes `./Routes` | `@nx/angular:webpack-browser` | `@nx/angular:dev-server`                   | 4201 |
| `usersui`   | MF remote                     | same                          | same                                       | 4202 |
| `rolesui`   | MF remote                     | same                          | same                                       | 4203 |

Each app carries: `project.json` (about 6 KB, near-identical), `webpack.config.ts`, a `webpack.prod.config.ts` (remotes only), `module-federation.config.ts`, a Dockerfile, an nginx `default.conf`, and four tsconfigs. App source is thin: 38 lines of TS per remote, 114 in the shell (including `remote-definitions.ts`). Almost all logic lives in libs.

Federation wiring:

- Host `module-federation.config.ts` has `remotes: []`. Remotes are resolved at runtime: `main.ts` fetches `${SERVICE_DISCOVERY_BASE_URL}/api/remotes` from `servicediscovery`, falls back to `module-federation.manifest.json` (authui and usersui only; rolesui is absent from the fallback), then calls `setRemoteDefinitions` from `@nx/angular/mf`.
- Remotes expose a single `./Routes` entry (`remote-entry/entry.routes.ts`) re-exporting a feature-lib route array.
- `tsconfig.base.json` maps `authui/Routes`, `usersui/Routes`, `rolesui/Routes` to the remote sources, for type resolution only.

### Libs (`libs/frontend`, 13 path aliases, 12 projects)

| Scope     | Libs (non-spec TS lines)                                                                   |
| --------- | ------------------------------------------------------------------------------------------ |
| shared    | `ui` 478 (packaged, `@nx/angular:package`), `util` 544 (`@nx/js:tsc`), `data-access` 682   |
| container | `feature/core` 214, `data-access` 174, `util` **10**                                       |
| authui    | `feature/core` 310; alias `@jdw/frontend-authui-util` points to a path that does not exist |
| usersui   | `feature/core` 1850, `data-access` 389, `util` **32**                                      |
| rolesui   | `feature/core` 1341, `util` **4**                                                          |

Libs have tests (`@nx/angular:unit-test`, vitest-analog) and lint only; only `shared/ui` and `shared/util` build, and both are released independently (ADR 0001 keeps them unpublished).

### Shared config and rules

- `nx.json`: targetDefaults for six executor names (`webpack-browser`, `package`, `unit-test`, `eslint:lint`, `js:tsc`, plus Go/Gradle), release groups, generators. Plugins: `@nx/eslint/plugin`, `@nx/gradle`, `@nx-go/nx-go`. No `@nx/webpack` or `@nx/angular` inference plugin, so every frontend target is hand-written.
- `eslint.config.ts`: `enforce-module-boundaries` with two axes, `type:` (app > feature > ui/util/data-access) and `scope:` (app scope sees itself + shared).
- `tools/testing/angular-test-host` exists solely so `@nx/angular:unit-test` has a `buildTarget` (uses `@nx/angular:application`).
- `apps/e2e/platform-e2e` (`@nx/playwright`) and `platform-live-e2e`.
- Per-app release (`tag-only-version-actions`), Dockerfile and image per app.

## 2. Simplest viable layout

Premise: one team, one repo, four Angular apps that deploy together, a shell that imports no remote code at build time. The runtime federation via service discovery is the only thing that requires four deployables; everything else is ceremony.

Proposed tiers, each independently shippable:

| Step | Change                                                                                                                                                                                | Result                                              |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| A    | Delete the dead `@jdw/frontend-authui-util` alias; fix `rolesui` missing from the fallback manifest (or document that it is intended)                                                 | Config matches reality                              |
| B    | Fold the three tiny `util` libs (10, 32, 4 lines) into the neighbouring `feature/core` (or `data-access`) lib of the same scope                                                       | -3 projects, -3 aliases                             |
| C    | Collapse per-scope `feature/core`, `data-access` into one lib per scope (`libs/frontend/<scope>`) with internal folders; keep `shared/*`                                              | 12 -> about 6 libs                                  |
| D    | Share one app `project.json` shape: move duplicated options and the budgets to `targetDefaults` or a small generator-free helper, delete `webpack.prod.config.ts` copies if identical | -about 4 KB x 4 duplicated config                   |
| E    | Replace hand-written `project.json` build/serve with the plugin that owns the build, once chosen (see section 3)                                                                      | Inferred targets, fewer executor names in `nx.json` |
| F    | Optional and larger: drop federation, ship one Angular app with lazy-loaded routes per scope                                                                                          | One deployable, one Dockerfile, no discovery fetch  |

Merging `shared/ui`, `shared/util`, `shared/data-access` is not recommended: they have separate build semantics (`package` vs `tsc`) and separate release streams today.

## 3. Migration risk per step

| Step | Risk        | Why / mitigation                                                                                                                                                                                                                                                                      |
| ---- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A    | Very low    | Alias is unreferenced; fallback manifest only matters when discovery is down. Check with `grep` and one local run with discovery stopped.                                                                                                                                             |
| B    | Low         | Mechanical move plus alias edits; rename in `tsconfig.base.json` and `release` list unaffected (not released). `nx graph` diff proves no new cycles.                                                                                                                                  |
| C    | Medium      | Loses the `type:` layering inside a scope unless folder-level lint rules replace it; `nx.json` release/CI project lists and cached test targets change names. Do after B, one scope per PR, `rolesui` first (no data-access).                                                         |
| D    | Low-medium  | Pure config, but a wrong `targetDefaults` merge silently changes budgets; compare `nx show project <p> --json` before and after.                                                                                                                                                      |
| E    | Medium-high | Depends on the Nx 24 outcome below.                                                                                                                                                                                                                                                   |
| F    | High        | Removes independent deploy, the service-discovery contract (`servicediscovery`, `/api/remotes`), four release streams, and per-remote nginx images. Bundle size budgets (500 KB warn) must be re-set. This is an architecture decision, not a cleanup; it needs an ADR and the owner. |

## 4. Effect on the Nx v24 upgrade (PR #271)

Nx 24 release notes (as quoted in the Renovate PR) list: "remove nx module federation and migrate to official plugins", "remove deprecated executors", "migrate removed config helpers with loadable stubs", and "stop generators from accepting targets on removed executors". This workspace is exposed on exactly those surfaces:

- `@nx/module-federation` (`ModuleFederationConfig`, `withModuleFederation` from `@nx/module-federation/angular`) is used in 4 `module-federation.config.ts` and 4 `webpack.config.ts` files.
- `@nx/angular/mf` `setRemoteDefinitions` is used in the shell `main.ts`.
- `@nx/angular:module-federation-dev-server` (container serve) is an MF executor.
- `@nx/angular:webpack-browser` (all four builds) and `@nx/angular:dev-server` sit on the Angular-webpack path that module federation requires; `@nx/webpack` and `@angular-devkit/build-angular` are direct dependencies for it.
- Not affected: `@nx/angular:unit-test`, `@nx/angular:package`, `@nx/js:tsc`, `@nx/eslint:lint`, `@nx/playwright:playwright`, `@nx/web:file-server`.

How structure interacts:

1. The MF-specific code is concentrated in four near-identical files per app and one shell `main.ts`. That is a small, uniform surface: the replacement (the official `@module-federation/*` Angular integration, already present as `@module-federation/enhanced`) can be proven on `authui` plus `container` and then copied. Doing step D first makes that copy a one-file change instead of four.
2. The runtime-manifest contract (`/api/remotes` returning name to URL) is independent of Nx; keep it unchanged and the migration stays a build-tool swap.
3. Steps B and C are orthogonal to the upgrade and should land before it only if they reduce churn; they do not unblock it.
4. Step F makes the blockers disappear (no MF, no dev-server executor pair, Angular `application` builder only). It is the only option where Nx 24 needs no replacement plugin, but its cost is the highest in section 3. If the owner has no concrete need for independent remote deploys, F is the cheapest path through the upgrade; otherwise the order is A, B, D, then E on the official MF plugin, then C.

Not verified here: the exact replacement API of the official plugin and whether it supports Angular 22 with the current `webpack-browser` target. That should be a time-boxed spike (one remote + shell on a branch with Nx 24) before E is estimated.

## 5. Recommendation

Do A, B, D now (low risk, shrinks the surface the Nx 24 work must touch). Run the E spike. Raise F as an explicit decision with the owner rather than slipping it in as cleanup.
