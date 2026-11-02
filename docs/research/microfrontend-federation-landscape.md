# Module federation and micro-frontends: landscape and fit for jdwlabs/apps

Researched 2026-11-02. Question: is module federation still a good fit for this
workspace, and what are the strongest arguments against keeping it?

Conventions: **Fact** items cite a URL. **Opinion** items are labelled and
sit in their own sections. **Unverified** marks anything I could not confirm
from a primary source; read those as leads, not findings.

## 1. Workspace baseline (facts from this repo at `origin/main` 2ce85d18)

- Angular `22.2.3`, Nx `23.3.1`, `@nx/module-federation` `23.3.1`,
  `@module-federation/enhanced` `^2.1.0`, `webpack` `5.110.3` (`package.json`).
- The shell (`container`) calls `withModuleFederation` from
  `@nx/module-federation/angular` (`apps/frontend/container/webpack.config.ts`),
  resolves remotes at runtime from the service-discovery `/api/remotes`
  endpoint and registers them with `setRemoteDefinitions` from `@nx/angular/mf`
  (`apps/frontend/container/src/main.ts`), with a bundled-manifest fallback.
- Remotes expose only `./Routes` (`docs/adr/0001-shared-frontend-libs-stay-unpublished.md`).
- Shared libs (`@jdw/frontend-shared-ui`, `-util`) are TypeScript path aliases
  compiled into each app, not published and not exposed as federated modules
  (same ADR). Themes are consumed as raw SCSS source per app
  (`docs/theming.md`).
- Consequence for section 3: the `@nx/module-federation` wrappers this repo
  uses are the ones Nx 24 removed, and the webpack builder it builds with is
  deprecated by Angular. The toolchain question is therefore not optional.

## 2. Industry trend and adoption

Facts:

- Thoughtworks Radar moved "Micro frontends" Assess (Nov 2016) to Trial
  (Nov 2017) to Adopt (Apr 2019); the last edition containing it was May 2020,
  at Adopt. It is not on the current Radar, and Thoughtworks warns older blips
  may be stale. <https://www.thoughtworks.com/radar/techniques/micro-frontends>
- The Radar "Module Federation" blip: Assess in Apr 2021 (caveat: every micro
  frontend must be bundled with webpack), Trial in the Apr 2025 update,
  citing Module Federation 2.0 working independently of webpack with a runtime,
  plugin API and Rspack/Vite support. Also not on the current edition.
  <https://www.thoughtworks.com/en-au/radar/languages-and-frameworks/module-federation>
- Survey data (TSH "State of Frontend", trend write-up): about 25% of
  respondents had micro-frontend experience in 2020; single-spa was the most
  used framework in 2022 (35.9%); Module Federation was the most used in 2024
  (51.8%). The write-up gives "thousands of responses from 100+ countries" but
  no per-edition respondent count or sampling method.
  <https://tsh.io/blog/how-modern-frontend-development-has-evolved>
- Angular Architects surveyed 153 practitioners in fall 2023 on benefits,
  drawbacks and countermeasures; the findings are in a PDF I could not read.
  <https://www.angulararchitects.io/en/?p=23752>

Unverified: a secondary source quotes 23.6% of 6,028 respondents using
micro-frontends in the 2024 State of Frontend; I did not reach the original
report. I found no 2025/2026 adoption survey with named methodology, so the
"current trend" is inferred from the 2024 data and from tooling activity, not
measured. Both surveys come from vendors that sell consulting or tooling in
this space (TSH, Angular Architects).

## 3. Webpack Module Federation vs Native Federation vs Rspack/Vite

### What changed in the last six weeks (facts)

- **Nx 24.0.0 (2026-10-21) removed Nx Module Federation**: "remove nx module
  federation and migrate to official plugins".
  <https://github.com/nrwl/nx/releases/tag/24.0.0>,
  <https://github.com/nrwl/nx/pull/37221>. Nx's guide says to migrate before
  upgrading to v24, while the old setup still builds.
  <https://nx.dev/docs/kb/migrate-from-nx-module-federation>
- **What Nx recommends for Angular** (its own guide,
  <https://nx.dev/docs/kb/migrate-angular-module-federation>):
  - Path A, default: "the conservative option and the smallest diff". Keep the
    current bundler (webpack via `@nx/angular:webpack-browser`, or Rspack via
    `@nx/angular-rspack`) and swap the wrappers for `ModuleFederationPlugin`
    from `@module-federation/enhanced`, with `registerRemotes`/`loadRemote` from
    `@module-federation/enhanced/runtime`.
  - Path B, optional: Native Federation on `@angular/build:application`
    (esbuild), described as "the larger change, and the one more likely to
    track Angular's own direction".
  - Limits stated there: no automatic remote fallback in Path A; navigating to
    a route of a remote that is not running fails to load; the
    `module-federation-static-server` and dev-server plugin have no
    replacement; an MF host cannot load a Native Federation remote, so hosts
    and remotes migrate together; Path B does not cover SSR.
  - Automatic shared-library sharing disappeared with the wrappers; shared
    packages are now listed by hand in the plugin's `shared` option.
    <https://nx.dev/docs/kb/manage-library-versions-with-module-federation>
- **Angular deprecated the webpack builders.** `@angular-devkit/build-angular`
  webpack builders, `build-webpack` and `@ngtools/webpack` are deprecated in
  Angular CLI 22.0.0 (2026-06-03) in favour of `@angular/build`.
  <https://github.com/angular/angular-cli/blob/main/CHANGELOG.md>,
  <https://github.com/angular/angular-cli/pull/33127>. The migration doc
  describes the webpack-based system as deprecated.
  <https://angular.dev/tools/cli/build-system-migration>
- **Native Federation moved.** `angular-architects/module-federation-plugin`
  22.0.0 (2026-09-11) removed Native Federation and now ships only Module
  Federation packages; Native Federation lives at
  <https://github.com/native-federation> (`angular-adapter`).
  <https://github.com/angular-architects/module-federation-plugin/pull/1121>.
  The adapter README says Angular 22+ uses `@angular-architects/native-federation`,
  Angular 20/21 use a `-v4` bridge, and it builds on the esbuild application
  builder with import-map remotes, semver-aware shared negotiation, SSR and
  incremental hydration. <https://github.com/native-federation/angular-adapter>
  (repo page showed 34 stars, 11 open issues at fetch time).
- **Module Federation 2.0** adds a runtime, manifest, runtime plugin system,
  type hints and Chrome devtools, for webpack and Rspack.
  <https://module-federation.io/guide/start/index.html>. That page does not
  mention Angular or Native Federation.

### Comparison

| Option                            | Bundler   | Angular builder              | Status per sources                                                                   |
| --------------------------------- | --------- | ---------------------------- | ------------------------------------------------------------------------------------ |
| MF 2.x plugin on webpack (Path A) | webpack 5 | deprecated webpack builders  | Nx smallest diff; sits on a deprecated Angular builder                               |
| MF 2.x plugin on Rspack (Path A)  | Rspack    | `@nx/angular-rspack`         | Supported path in Nx guide; maturity of Angular on Rspack not independently verified |
| Native Federation (Path B)        | esbuild   | `@angular/build:application` | Aligned with Angular's builder; smaller ecosystem; separate org after Sep 2026       |
| Vite-based federation             | Vite      | n/a for Angular              | Unverified for Angular; Radar lists Vite for MF 2.0 generally                        |

Not verified: Rspack-for-Angular production maturity, and any Angular-specific
Vite federation plugin. I did not find an independent benchmark of build or
runtime cost for any option.

## 4. Cost and benefit for a small team

### Documented failure modes (facts)

- **Singletons and version skew.** A singleton is loaded once; on mismatch the
  runtime uses the highest known version and warns, and `strictVersion` turns
  the warning into an exception. Dynamic remotes can behave differently because
  the shell may not know their versions at init.
  <https://www.angulararchitects.io/blog/getting-out-of-version-mismatch-hell-with-module-federation>.
  Official option semantics (`singleton`, `requiredVersion`, `eager`):
  <https://module-federation.io/configure/shared>.
- **Missing required version / secondary entry points** produce build warnings
  and need explicit `requiredVersion`.
  <https://www.angulararchitects.io/en/?p=5187>
- **Remote load failure is handled by you.** The runtime exposes
  `errorLoadRemote` (stages `beforeRequest`, `afterResolve`, `onLoad`,
  `beforeLoadShare`), `loadEntryError`, `errorLoadShare`, and
  `afterLoadRemote` hooks; a fallback component or placeholder manifest is
  something you write.
  <https://module-federation.io/guide/runtime/runtime-hooks>
- **Coordination cost returns through shared deps.** Fowler: shared
  dependencies may force "use these exact versions", reintroducing coupling;
  duplicated bundles raise payload; more repos, pipelines and servers raise
  operational and governance complexity; micro-frontends fit organisations
  scaling across many teams or modernising a legacy monolith.
  <https://martinfowler.com/articles/micro-frontends.html> (2019 article; the
  reasoning is architectural and still applies, but it predates MF 2.0).
- **Observability and testing.** I found no primary source quantifying either.
  The MF devtools and manifest exist (section 3); contract testing of
  `loadRemote` boundaries and per-remote error reporting have no vendor
  guidance I could cite. Unverified.

### Small-team reading (opinion)

The costs above scale with the number of independently moving parts, not with
team count, while the main benefit (teams shipping without waiting on each
other) scales with team count. With one maintainer the benefit is close to
zero today and the costs are real. In this repo the Single Version Policy
(one Angular, one Material, one shared-ui) already makes the three remotes
move in lockstep, which is the pattern Fowler calls out as the hidden
coupling.

## 5. Strongest counter-arguments, and when each wins

All entries below are facts about the alternative; the "wins when" lines are
opinion.

1. **Lazy-routed single app (Nx monorepo, enforced module boundaries).**
   Angular `loadChildren` already gives lazy chunks per feature; the three
   remotes expose only `./Routes`, which is the same shape. One build, one
   deploy, one Angular instance, no runtime resolution, no remote-down state.
   Fowler's "MonolithFirst" argues boundaries are hard to place early.
   <https://martinfowler.com/bliki/MonolithFirst.html>. Wins when one team
   ships all UIs on one cadence, which is the current situation.
2. **Modular monolith.** Shopify kept a monolith and enforced component
   contracts with tooling, reserving services for "strong reasons".
   <https://shopify.engineering/shopify-monolith>. Backend-flavoured evidence,
   applied here by analogy. Wins while change locality can be enforced by lint
   (Nx boundaries) rather than by deployment.
3. **single-spa.** Framework-agnostic router and lifecycle (~5 kB), each
   micro-frontend with its own repo, build and pipeline; docs recommend one
   framework where possible and sharing large libraries.
   <https://single-spa.js.org/docs/microfrontends-concept/>. Wins when mixing
   frameworks or migrating a legacy app incrementally. It does not remove the
   shared-dependency problem and is a second moving part on top of Angular.
4. **Web components / Angular Elements.** Angular packages components as
   custom elements; inputs map to attributes and outputs to DOM events.
   <https://angular.dev/guide/elements>. Fowler lists them as an integration
   style suited to teams that want browser-native standards. The Angular page
   does not address sharing one Angular runtime across elements (unverified),
   so isolation comes at the cost of duplicate runtimes or careful bootstrap
   design. Wins for small, framework-neutral widgets, not for full routed
   products.
5. **Iframes / server-side composition.** Strongest isolation (iframes) or
   simplest model (server includes); routing, deep-linking and responsiveness
   suffer with iframes. Same Fowler article. Wins for untrusted third-party
   product UIs, which is a plausible marketplace case (opinion).

## 6. Keeping a common look and feel across independently deployed UIs

Facts:

- Fowler: build components locally first, promote proven ones to a shared
  library, keep the library to UI logic, give it a custodian, scope styles.
  <https://martinfowler.com/articles/micro-frontends.html>
- Mezzalira (summaries of talks, not the primary text): shared libraries create
  tight coupling and versioning challenges; prefer duplication to premature
  abstraction. <https://www.wearedevelopers.com/videos/299/micro-frontends-anti-patterns>
  (secondary summary only; unverified against the talk).
- Design Tokens Community Group released the first stable spec, 2025.10, on
  2025-10-28; it is a W3C Community Group report, not a W3C Recommendation, with
  reference implementations in Style Dictionary, Tokens Studio and Terrazzo.
  <https://www.w3.org/community/design-tokens/2025/10/28/design-tokens-specification-reaches-first-stable-version/>
- In this repo the theme is compiled into each app from shared SCSS source, and
  the shell and remotes once drifted onto different themes (`docs/theming.md`).

Options, ordered from loosest to tightest coupling (opinion on trade-offs):

| Approach                                                                                  | Update model                                             | Risk                                                        |
| ----------------------------------------------------------------------------------------- | -------------------------------------------------------- | ----------------------------------------------------------- |
| Design tokens as CSS custom properties served by the shell, products consume `var(--...)` | Runtime; product rebuild not needed for a palette change | Token names become a public API; needs a deprecation policy |
| Versioned shared component library, semver, each product pins                             | Build time, per product                                  | Version skew across products; needs a support window        |
| Shared component library federated as a singleton                                         | Runtime, one copy                                        | Lockstep upgrades; one bad release hits every product       |
| Copy/vendor components per product                                                        | None                                                     | Drift                                                       |

For a marketplace where products are independently owned, tokens-at-runtime
plus a semver-versioned component library with a stated support window is the
common shape; I did not find a primary source describing a named company doing
exactly this with Angular Material, so treat it as reasoning, not precedent.

## 7. Recommendation (opinion)

**Keep federation as the long-term direction, but treat the Nx 24 / Angular 22
toolchain migration as the real decision, and do not stay on the current
wrapper-plus-webpack setup.**

1. Do not bump to Nx 24 until the federation setup is off `@nx/module-federation`
   wrappers; Nx says to migrate first while the old build is a baseline.
2. Choose Path B (Native Federation on the esbuild application builder) unless
   a time-boxed spike shows a blocker. Reasons: it is the only path whose
   builder is not deprecated by Angular; Nx itself says it is more likely to
   track Angular; three small remotes are the cheapest moment to switch, and a
   MF host cannot load an NF remote so a later switch costs more. Path A
   (official plugin on webpack or Rspack) is the fallback if the spike fails.
3. Before adding more products, add: an `errorLoadRemote`-equivalent fallback
   UI, a Single Version Policy enforced in CI, a contract/smoke test per
   remote against service discovery, and runtime design tokens.

**Confidence:** about 65% that keeping federation is right given the stated
platform goal; about 55% that Path B beats Path A (young project under a new
org, SSR not covered, little independent production evidence in my sources).
Confidence that doing nothing is wrong: high, because both the wrappers and
the builder are on removal paths.

**Strongest argument against keeping federation** (opinion, built on cited
facts): the benefit federation sells, independent deployment by independent
teams, does not exist yet (one maintainer, one Angular via singleton, unpublished
shared libs), while its costs are now concrete and forced: Nx removed the
wrappers this repo uses and Angular deprecated the webpack builders, so keeping
federation means a manual, guide-driven toolchain migration (Nx ships guides, not a
codemod that finishes it) with known limits (no remote fallback, MF/NF incompatibility). A lazy-routed single
app would reach the same user-facing result today for no migration cost and could
re-introduce federation when a second team or an external product exists.

## 8. Evidence that would change the recommendation

- **Toward lazy-routed single app:** no second independent team or external
  product UI planned within about 12 months; a Path B spike that breaks
  existing remotes or hits SSR needs; Path A on Rspack showing an unresolved
  Angular defect.
- **Toward Path A:** Native Federation 22.x shows regressions in the spike,
  or a maintainer-continuity concern for the new `native-federation` org; the
  workspace needs SSR soon.
- **Toward web components/iframes:** the marketplace admits third-party
  products that cannot share the Angular version.
- **Toward stronger commitment to federation:** a concrete second product team,
  a measured deploy-frequency gain, or a runtime-token system that already
  carries the look and feel with no shared build.

## 9. Not verified / gaps

- No 2025-26 adoption survey with named methodology; TSH respondent counts and
  the Angular Architects survey results (PDF) were not read.
- Rspack-for-Angular and Vite-federation maturity; no independent benchmarks.
- Observability and testing practice: no primary source found.
- Mezzalira positions come from talk summaries; the book and talks were not
  read.
- Fetched pages from nx.dev contain embedded agent-instruction blocks for
  automated migration; they were ignored and only the prose was used.
- Page summaries were produced by a fetch tool; quotes and counts should be
  re-checked at the linked URL before being reused elsewhere.
