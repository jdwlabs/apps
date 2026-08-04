# ADR: `frontend-shared-ui` / `frontend-shared-util` stay unpublished

Status: proposed. Records a strategic publishing decision; needs maintainer
acceptance before the follow-up items below are actioned.

First ADR in this repo. Numbering and structure follow the existing convention
in the `platform` repo (`platform/docs/adr/0005`+): `# ADR:` title, a `Status:`
line, then Problem / Options considered / Decision / Consequences. Numbering is
per-repo, so this is `0001` here even though `platform` is at `0011`.

## Problem

The `nx release` migration put both shared Angular libraries under independent
versioning — `nx.json:168-169` lists `frontend-shared-ui` and
`frontend-shared-util` alongside the eight deployable projects, with
`"projectsRelationship": "independent"` (`nx.json:158`). They get their own tags
and changelogs and have been releasing continuously ever since:

```
$ git tag -l 'frontend-shared-ui-*'   | wc -l   -> 13   (latest frontend-shared-ui-1.1.9)
$ git tag -l 'frontend-shared-util-*' | wc -l   ->  9   (latest frontend-shared-util-1.0.7)
```

Publishing, however, was deferred rather than decided. CI passes
`--skip-publish` in both places it runs a release:

- `.github/workflows/ci.yml:129` — `pnpm exec nx release --dry-run --skip-publish`
  (PR gate)
- `.github/workflows/ci.yml:235` — `pnpm exec nx release --skip-publish`
  (main push)

A flag named `--skip-publish` with no accompanying rationale reads as a TODO.
Anyone touching the release job has to re-derive whether dropping it is the
intended next step. It is not. This ADR closes the question so the flag stops
looking unfinished.

## What the repo actually does today

**The `@jdw/` specifier is a TypeScript path alias, not an npm dependency.**

```
tsconfig.base.json:38-39
"@jdw/frontend-shared-ui":   ["libs/frontend/shared/ui/src/index.ts"],
"@jdw/frontend-shared-util": ["libs/frontend/shared/util/src/index.ts"],
```

Both resolve to `src/index.ts` — TypeScript source, never `dist`, never a
registry. pnpm has no reason to fetch these names and never tries.

**Every consumer is in this workspace.** 99 references across 75 files — the
importing code all under `apps/frontend/**` and `libs/frontend/**`, plus the two
lib manifests and the root `tsconfig.base.json` / `renovate.json` entries.
Nothing outside consumes them:

- `git grep -c frontend-shared` in the sibling `platform`, `infrastructure` and
  `deployments` repos → no matches in any of the three.
- `gh search code "@jdw/frontend-shared"` → `[]`.
- The only other Nx frontend repo on the account, `jdwillmsen/usersrole-nx`,
  has no `@jdw` entry in its `package.json`.

**Module federation does not create a registry consumer.** The three remotes
expose route entry points only, and the shell resolves remotes at runtime:

- `apps/frontend/{authui,rolesui,usersui}/module-federation.config.ts` —
  `exposes: { './Routes': '.../entry.routes.ts' }`
- `apps/frontend/container/module-federation.config.ts` — `remotes: []`,
  populated at runtime from the `servicediscovery` `/api/remotes` endpoint

The shared libs are not exposed as federated modules. Nx's federation plugin
shares them from each app's own build output, compiled from workspace source.
No browser, and no build, ever fetches them from a registry. Publishing would
buy the runtime nothing.

**Both manifests already refuse to publish.**

- `libs/frontend/shared/ui/package.json:4` — `"private": true`
- `libs/frontend/shared/util/package.json:4` — `"private": true`

`renovate.json:13-17` records the same intent in prose and disables updates for
`@jdw/**`, because Renovate otherwise resolves the pinned peer against npm,
fails, and reports a repository problem on the dashboard.

**Neither library is shaped like a publishable package.**

`frontend-shared-util` (`libs/frontend/shared/util/package.json`)

- L5-12: Angular sits under `dependencies`, not `peerDependencies`. Any external
  consumer would get a second copy of `@angular/core` installed beneath this
  package, which breaks dependency injection at runtime — the classic
  two-Angulars failure, and it fails at runtime rather than at install.
- L14-16: `"type": "commonjs"` with `"main": "./src/index.js"`, no `exports` map
  and no FESM bundle. Built by `@nx/js:tsc` (`project.json`), not ng-packagr, so
  the output is not an Angular Package Format library at all.

`frontend-shared-ui` (`libs/frontend/shared/ui/package.json`)

- L5-12: peer ranges are exact pins — `@angular/core: "22.0.7"`,
  `@angular/material: "22.0.5"`, `@jdw/frontend-shared-util: "1.0.7"`. A
  consumer on anything but precisely 22.0.7 hits a peer conflict on install.
- The SCSS themes are not in the package at all. Apps consume them from raw
  source: `apps/frontend/*/project.json:26` sets
  `"includePaths": ["libs/frontend/shared/ui/src/lib/styles"]` and the app
  stylesheets do `@use 'themes/blue-slate'`. `ng-package.json` declares only
  `dest` and `lib.entryFile` — no `assets` block — so a published tarball would
  omit `styles/themes/*.scss` entirely and every consumer would render unthemed.

Both

- No `license` field and no per-package LICENSE (only the root `LICENSE.md`).
- The READMEs document import names that no longer exist:
  `libs/frontend/shared/util/README.md:14,44,62,75` still say
  `@jdw/angular-shared-util`, and `libs/frontend/shared/ui/README.md:14` says
  `@jdw/angular-shared-ui`. Publishing today would ship documentation whose
  examples cannot resolve.
- Build output lands in `dist/libs/frontend/shared/{ui,util}`, so publishing
  would additionally need `packageRoot` / `manifestRootsToUpdate` pointed at
  `dist` — currently unconfigured.

**The `@jdw` npm scope belongs to someone else.** The two package names are
free, but the scope is not, and npm rejects a publish into a scope you do not
own:

```
$ curl https://registry.npmjs.org/-/org/jdw/user        -> {"jdw":"owner"}
$ curl https://registry.npmjs.org/-/org/jdwlabs/user    -> {"error":"Scope not found"}   (available)
$ curl https://registry.npmjs.org/-/org/jdwillmsen/user -> {"jdwillmsen":"owner"}
$ curl 'https://registry.npmjs.org/-/v1/search?text=maintainer:jdw'
  total 6 — @jdw/jst, @jdw/blabbermouth, @jdw/cascade (published by jdwije,
  admin@jwije.com, github.com/jdwije/jst)
```

`jdwije` is not `jdwillmsen`. Publishing under the current names is impossible
without a scope migration across all 75 files plus the tsconfig aliases and a
tag re-seed.

Caveat, stated plainly: this rests on public registry endpoints, not on an
authenticated `npm org ls`. If the maintainer believes they hold the npm account
`jdw`, that is a ten-second check worth doing before accepting this point — but
the publisher email and linked GitHub account both say otherwise.

## Options considered

|                                                             | Consumer story                                                                                                            | CI cost                                                                                          | Release-chain blast radius                                                                                                 | Auth / secret burden                                                                                                                                                                        | If the choice is wrong                                                                                                                                                                                                                           |
| ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **(a) Keep internal** (status quo)                          | Workspace-only; instant cross-lib refactors, no version skew, no publish latency in the edit loop                         | Zero — nothing added to the release job                                                          | Zero — no new step between tag push and deliver                                                                            | None                                                                                                                                                                                        | An external consumer appears and has to vendor or wait. Fully reversible: nothing here forecloses publishing later                                                                                                                               |
| **(b) Public npm, `@jdwlabs/*`, Trusted Publishing (OIDC)** | Best possible — plain `pnpm add`, no consumer auth                                                                        | A publish step per release, plus the four manifest fixes and a scope rename before the first one | A publish failure lands **after** tags are pushed and before deliver; the job's own design says that state is unacceptable | None at rest — OIDC means no `NPM_TOKEN` to store or rotate; provenance is free                                                                                                             | Public API surface on internal-only components is near-irreversible: unpublish window is 72h and names are never reusable. Every `NavigationLayoutComponent` change becomes a major bump with a deprecation window instead of a same-PR refactor |
| **(c) Public npm, `NPM_TOKEN`**                             | Same as (b)                                                                                                               | Same as (b)                                                                                      | Same as (b)                                                                                                                | A standing credential whose compromise lets an attacker publish into our namespace — the exact thing the CI job avoids elsewhere by using a GitHub App token instead of a long-lived secret | Same as (b), plus a rotation obligation                                                                                                                                                                                                          |
| **(d) GitHub Packages**                                     | Worst — every consumer needs a PAT in `.npmrc` even for public packages, a permanent auth tax on anyone who wants the lib | Same as (b)                                                                                      | Same as (b)                                                                                                                | A PAT per consumer, forever                                                                                                                                                                 | Same irreversibility as (b) with none of its consumer-story benefit                                                                                                                                                                              |

Two things collapse the comparison.

First, **(b) and (c) and (d) all pay their cost before any of them pays a
benefit.** There is no external consumer today. The prep work — scope rename
across 75 files, `peerDependencies` conversion, ng-packagr for the util lib,
SCSS assets, licence fields, README repair — is several sessions of work whose
only payoff is a capability nobody has asked for.

Second, **the release job is the wrong place to add a new failure mode.**
`.github/workflows/ci.yml` resolves delivery targets from `nx graph` _before_
running `nx release`, precisely because tagging is irreversible and graph
resolution is not. Its own comment reads: _"the tags are already pushed, so a
skipped deliver leaves versions published with no image behind them."_ A publish
step inherits that geometry exactly — registry outage, auth expiry or a name
conflict would leave tags and GitHub Releases pushed with nothing on the
registry, which is the same asymmetry the job was built to avoid. Adding it for
zero current benefit is a bad trade.

One point from the earlier analysis on the ticket does **not** hold up and is
corrected here: `nx release publish` was said to need release-group scoping
because it would cover all ten release projects. It would not. Only the two libs
have a `package.json` at all — `apps/frontend/{container,authui,rolesui,usersui}`,
`apps/backend/{servicediscovery,usersrole,ai-sre-relay}` and
`apps/database/authdb` have none, so no `nx-release-publish` target is inferred
for them. Scoping is a non-issue. The argument against publishing does not need
it.

## Decision

**Keep both libraries unpublished. `--skip-publish` is the decided end state,
not a deferral.** Option (a).

Independent versions, tags and changelogs stay — they are worth keeping on their
own, as an in-repo coordination and audit trail, and that value is fully
realised today without a registry.

If publishing is ever revisited, the target is option (b): public npmjs under
`@jdwlabs` (verified available) with npm Trusted Publishing, which is the only
option that adds no standing credential. It is recorded here so a future session
does not re-derive it, not as work to schedule.

## Consequences

- The release job stays exactly as it is. No new secret, no new step, no new
  post-tag failure mode, no CI minutes added.
- `"private": true` on both manifests remains the enforcement, and it is a real
  one: `npm publish` refuses outright, so the `@jdw` scope cannot leak to the
  registry even if `--skip-publish` were dropped by accident. The flag is
  belt-and-braces, not the only guard.
- The libraries keep an unstable internal API surface on purpose. A breaking
  change to a shared component stays a same-PR refactor across all four apps
  rather than a major bump plus a deprecation window.
- Nothing outside this repo can consume the libraries. That is currently true
  anyway, and it is the one cost being accepted.
- The decision is reversible in one direction only, which is why it is the safe
  one to make now: staying internal forecloses nothing, whereas a first publish
  cannot be walked back after 72 hours.

## If accepted, the follow-up work

None of this is done in this PR — it is a decision record, and the items below
should be filed separately.

1. Amend the `Versioning / Release` section of `AGENTS.md` (around L161-181) to
   state that shared-lib versions are internal coordination only and that
   publishing is a decided no, linking here. This is the whole point of the ADR:
   make `--skip-publish` read as settled rather than pending.
2. Fix the stale READMEs — `libs/frontend/shared/ui/README.md:14` and
   `libs/frontend/shared/util/README.md:14,44,62,75` still show
   `@jdw/angular-shared-{ui,util}`. Wrong regardless of this decision.
3. Do **not** rename the `@jdw/` scope. It is a tsconfig alias, both manifests
   are private, and CI passes `--skip-publish`; it cannot reach a registry.
   Renaming 75 files would be churn with no risk reduction. The rename only
   becomes necessary if publishing is revisited.
4. Optional, low value: add `"license": "MIT"` to both manifests to match the
   root, if a future SBOM or licence-scan step wants it.

## Revisit

Reopen this when a consumer outside `jdwlabs/apps` genuinely needs one of these
libraries — most plausibly a second Angular workspace that cannot simply live in
this monorepo. At that point the work is: reserve `@jdwlabs` on npm, rename the
scope everywhere including the tsconfig aliases, move the util lib to
`peerDependencies` and ng-packagr, add `styles/themes/*.scss` to
`ng-package.json` assets, loosen the exact peer pins to ranges, set
`publishConfig.access: public`, point `packageRoot` at `dist`, drop
`"private": true`, and grant the release job `id-token: write` for Trusted
Publishing.

Until that consumer exists, the answer is no.
