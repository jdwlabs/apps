# Default theme design

Replaces `black-white` with a chosen default palette, and makes the built
theme bundles reachable at runtime.

Covers JDWLABS-264 (the palette) and JDWLABS-260 (the switcher). They are one
change: both must rewrite the same four `styles.scss` files and the same four
bundle lists, and doing them separately means doing that cutover twice.

## Problem

`black-white` seeds primary and secondary from `#000000` / `#c2c2c2`, so
primary, secondary and tertiary generate the same grey ramp and every
`*-container` role resolves to the same tone. M3 relies on hue to separate a
container role from the surface it sits on — a container is tone 90 in a light
theme and tone 30 in a dark one, close to surface *by design*. With no hue to
spend, only lightness is left.

Measured consequences:

| Element | Measured | Required |
| --- | --- | --- |
| Nav active indicator vs surface | 1.23:1 | 3:1 |
| Version chip vs `surface-container-highest` (dark) | 1.32:1 | 4.5:1 |
| Env chips vs a user-supplied toolbar colour | 1.14:1 | 3:1 |
| Env badge segments vs each other | indistinguishable | distinguishable |

Two component-level workarounds already exist to compensate. The ticket's own
argument stands: accumulating exceptions is the palette telling us it is the
wrong default.

Separately, each app hardcodes one theme (`@use 'themes/black-white'`) as its
only injected stylesheet, while building eight more with `inject: false` that
nothing loads. The hardcoding is also how the four apps drifted apart —
container ran dark `red-teal` while the three remotes ran light `black-white`.

## Decisions

### 1. Seeds: blue-slate

Generated with `ng generate @angular/material:theme-color` from these seeds;
tone values are never hand-edited, matching the existing rule in
`_custom-palettes.scss`.

| Seed | Value | Why |
| --- | --- | --- |
| primary | `#2f5fa8` | Mid blue. Tone 40 lands saturated enough for a toolbar fill and dark enough to clear light surfaces |
| tertiary | `#a8622f` | Warm copper, roughly opposite the primary hue, so `tertiary-container` separates from surface by hue rather than lightness |
| neutral | `#5a6472` | Slate rather than pure grey, so surfaces read as deliberate instead of undertoned |

The seeds are the input to the verification gate below, not the conclusion. If
a generated tone fails a floor, the seed is adjusted and the palette
regenerated — the roles are never patched at the component level.

Blue-slate preserves the tone structure the rest of the system already depends
on: toolbars fill from a tone-40 role in light themes and tone-30 in dark ones,
which is the assumption the environment chips are built against.

### 2. No new colour roles

The semantic layer already exists. `_theming.scss` publishes
`--jdw-sys-success`, `--jdw-sys-info`, `--jdw-sys-warn` and
`--jdw-sys-env-prod` / `-non` / `-local`, parallel to M3's own `error` role and
already reasoned about in the file: the environment colours are fixed rather
than parameterised because the colour is the information, and all three sit in
the tone-80 band because HCT holds luminance near-constant across hues at a
given tone, so one light band clears both the tone-40 and tone-30 toolbars.

Angular Material's theming API accepts primary, secondary, tertiary and neutral
only — it does not surface Material Theme Builder's extended-colour concept.
Adding environment hues as tonal families would mean hand-maintained tone maps
outside `mat.theme`, for information the badge already carries as text.

This work therefore adds no roles. It re-verifies the tone-80 band against
blue-slate's toolbar tones.

### 3. Foregrounds on arbitrary surfaces are computed, not fixed

`user-custom-light` and `user-custom-dark` re-point the toolbar at a
user-supplied colour, so no fixed foreground can be correct for every input.
Today's pass is an accident of one default green.

A pure `on-color` utility derives the foreground from the actual background via
HCT tone math, returning a value that clears 4.5:1 for text and 3:1 for
non-text. The switcher applies it when a theme is selected.

This also fixes the container-role collapse. The user-custom themes currently
set `--mat-sys-primary-container: var(--mat-sys-primary)` because there is no
container-tone variable for a user to set — which overwrites tone 90/30 with a
saturated tone and makes `on-primary-container` equal `on-primary`. Deriving
tones 90 and 30 from the user's colour removes the collapse and the fragility
that follows from it.

### 4. Three themes

`blue-slate`, `user-custom-light`, `user-custom-dark`.

The five legacy M2 palettes (`pink-bluegrey`, `deeppurple-amber`, `indigo-pink`,
`purple-green`, `red-teal`) are deleted. They were inherited defaults rather
than chosen ones, and were unreachable for their entire existence, so no user
loses a theme they had. Keeping them would mean contrast-verifying five
unchosen palettes against the gates below, or shipping a switcher that offers
themes which fail them. The schematic regenerates any palette from a seed if
one is wanted later.

### 5. Themes switch by attribute

Each theme's tokens are emitted under `html[data-theme='<id>']` in a single
stylesheet. Switching writes one attribute:

```ts
document.documentElement.dataset.theme = next;
```

No network round-trip, no flash, and no load-order race between the container
and its remotes — the race being the mechanism that produced the drift.

`_all-themes.scss` warns that moving the component mixin inside an `html` block
gives every component selector an extra ancestor and changes which rules win.
Scoping under `[data-theme]` does exactly that. The bump is safe only if it is
uniform: when every theme's component block gains the same ancestor, relative
precedence is preserved. Mixing scoped and unscoped emission is what breaks it,
which is what a partial migration produces — so all four apps cut over in one
change as a correctness requirement.

`toolbar/_variants.scss` branches on light/dark at compile time, so component
blocks are emitted per theme rather than once.

### 6. The container owns the theme

The four frontends are federated into one DOM. If remotes keep shipping theme
CSS they compete for `--mat-sys-*` on `html`, resolved by load order.

The container emits the theme stylesheet. `authui`, `usersui` and `rolesui`
ship none, and consume the tokens from the host document.

## Architecture

| Piece | Location | Responsibility |
| --- | --- | --- |
| `on-color` | `libs/frontend/shared/ui/…/theming` | Given a background, return a foreground meeting the contrast floor. Pure, framework-free, unit-tested |
| Theme registry | `libs/frontend/shared/ui/…/theming` | The three themes as data (id, label, type). One source for both the switcher and the `[data-theme]` blocks |
| Switcher service | `libs/frontend/shared/ui/…/theming` | Writes `data-theme`, persists the choice, recomputes user-custom tones through `on-color` |
| Theme stylesheet | container | All three themes under `html[data-theme=…]`, component blocks scoped uniformly |
| Remote styles | authui, usersui, rolesui | Layout and font only |

The registry is the seam that keeps the switcher from knowing about Sass and
the stylesheet from knowing about Angular. Adding a theme means adding a
registry entry and a `[data-theme]` block; nothing else changes.

## Migration

One PR:

1. Generate the blue-slate palettes with the schematic; add them to
   `_custom-palettes.scss`.
2. Add the theming lib (`on-color`, registry, switcher) with unit tests.
3. Rewrite the container stylesheet to emit all three themes under
   `[data-theme]`, component blocks included.
4. Strip the theme `@use` from the three remotes' `styles.scss`.
5. Delete the five legacy palette files and the twenty corresponding
   `inject: false` bundle entries across the four `project.json` files.
6. Add the switcher control to the container's header.

## Verification

Measured against the roles that actually failed, not eyeballed:

| Check | Floor |
| --- | --- |
| `tertiary-container` vs `surface` | 3:1 |
| Env chips vs toolbar, light and dark | 3:1 |
| Version chip vs `surface-container-highest` | 4.5:1 |
| Env chips vs six sampled user-supplied colours | 3:1 |
| Nav active indicator vs surface | 3:1 |

The first check decides whether the nav-indicator override is removed. It
exists because a shared grey ramp left `tertiary-container` at 1.23:1 against
the drawer; blue-slate gives tertiary a distinct hue, which should restore the
separation the role was designed to have. If the measurement clears 3:1 the
override is deleted; if it does not, the override stays and its comment is
updated to say why it survived a palette that was supposed to remove it.

The env-badge treatment is retained either way. It is not a workaround: the
badge carries `PROD` / `NON` / `LOCAL` as text plus an `aria-label`, so colour
is redundant reinforcement rather than the sole carrier, and it is there for
error prevention — a glanceable production signal.

## Risks

- **Blue-slate may not clear 3:1 on `tertiary-container` vs `surface`.** The
  gate catches it. Outcome is a retained override with honest reasoning, not a
  silent failure.
- **The specificity bump is all-or-nothing.** Partial migration produces mixed
  scoped and unscoped rules with no obvious symptom until a component renders
  wrong in one theme. Mitigated by cutting over in one change and by rendering
  every theme in the e2e shell check.
- **`on-color` runs at theme-apply time, not per paint.** A user colour changed
  outside the switcher would not recompute. Acceptable: the switcher is the
  only path that sets one.

## Out of scope

Typography, density, and the ag-grid theme, which already tracks `--mat-sys-*`.
