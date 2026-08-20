# ADR: mission/about content stays static Markdown, not a new MFE

Status: proposed. Scoping decision for a candidate new frontend; no code in this
repo changes as a result. Needs maintainer acceptance before the ticket that
raised the question is closed.

Numbering and structure follow the convention established by `0001` and `0002`
in this repo: `# ADR:` title, a `Status:` line, then Problem / Options
considered / Decision / Consequences. Numbering is per-repo.

## Problem

`jdwlabs/.github/profile/README.md` and `jdwlabs/.github-private/profile/README.md`
each picked up a hand-written "Mission" section in the same session
(`.github` commit `da96dbd`, `.github-private` commit `d71cc0c`). Both are
GitHub org profile READMEs — rendered by GitHub itself on the public
`github.com/jdwlabs` page and the equivalent internal-org profile surface, not
by anything this org runs. The two sections are already not identical:
`.github-private`'s carries an extra paragraph on the agentic operating model
(`profile/README.md:29`) that the public one omits. That is exactly the drift
pattern the same working session had to clean up elsewhere: five separate PRs
(`jdwlabs/.github-private#3`, `jdwlabs/.github#34`, `jdwlabs/platform#230`,
`jdwlabs/deployments#163`, `jdwlabs/apps#179`) to fix Actions workflow
filenames that had been hand-copied across five repos' READMEs and drifted out
of sync. Mission content maintained the same way — copy-pasted Markdown,
per repo — is exposed to the identical failure mode, and has already started
diverging.

The ticket that raised this asked for a recommendation: build a new Angular
MFE to render mission/about content interactively (candidate name TBD, mounted
into the `container` shell per the pattern in `apps/frontend/{authui,rolesui,usersui}`),
or keep the content as static Markdown — evaluating the cheaper alternative of
a single shared content file first.

## The MFE option does not reach the surface with the actual problem

Before comparing cost, one fact resolves the question directly: **the
duplication lives in two GitHub profile READMEs, rendered by GitHub on
`github.com`. The `container` app does not run there.**

- `deployments/charts/container/values-non.yaml:10` and
  `values-prd.yaml:14` — `container` is reachable only at
  `container.non.jdwlabs.com` / `container.prd.jdwlabs.com`, ingress hosts on
  this org's own cluster.
- `apps/frontend/container/src/main.ts:7-32` — the shell resolves its remotes
  at runtime from `servicediscovery`'s `/api/remotes`, an endpoint that only
  exists inside this org's own deployment. There is no path by which a
  GitHub-rendered Markdown page pulls content from it.
- The Technical Approach on the ticket that raised this only describes
  mounting a new remote into `container` via that same discovery mechanism —
  the same proven pattern `authui`/`rolesui`/`usersui` already use. It does not
  propose embedding `container`'s output into `github.com` — and GitHub's own
  Markdown sanitization does not allow arbitrary third-party app content
  (script tags are stripped; `iframe` is restricted to a small allowlist) to
  be embedded in a profile README even if that were attempted.

So building the MFE, exactly as scoped, would add an interactive page reachable
at `container.{non,prd}.jdwlabs.com` — a surface a GitHub visitor reading the
org profile never sees. The two README Mission sections would still need to be
hand-maintained afterward, identically to today, either as the same prose or as
a link out to the new app. The MFE does not remove a single copy of the
duplicated content; at best it adds a third copy on a different domain.

## Options considered

**(a) Keep static Markdown, no dedup fix (do nothing further).** Cheapest,
zero effort. Leaves the exact failure mode that already produced five cleanup
PRs and has already produced one divergence between the two Mission sections.

**(b) Keep static Markdown, dedupe via one shared content source.** A single
Markdown (or data) file — plausibly living in `.github`, since GitHub already
reads `.github/profile/README.md` as the public profile source — consumed by
both profile READMEs through an include or a small generation step (a
workflow that regenerates the Mission section from the shared source and fails
CI if the checked-in README drifts from it, in the spirit of this repo's own
"docs are code" standard in `.github/docs/code-standards.md`). No new
deployable, no new runtime, no new ingress, no auth/hosting to reason about.
Directly fixes the drift this ticket exists to prevent, at a small fraction of
the cost of a new app.

**(c) Build the MFE as scoped.** New Angular remote under
`apps/apps/frontend/<name-tbd>`, a `libs/frontend/<name>` set of libs, a
module-federation entry, a Helm chart in `deployments`, an ArgoCD sync target —
the full deployable-app cost structure `authui`/`rolesui`/`usersui` already
carry. As established above, it does not land on the surface where the
duplication actually occurs, so it does not solve the stated problem. It would
need to be paired with option (b) or a link-out from the READMEs to be worth
anything, at which point (b) alone already closes the ticket for a fraction of
the cost.

## Decision

**Keep mission/about content as static Markdown. Do not build the MFE.**
Option (a) for this ticket's immediate scope — this ADR is the ticket's
required recommendation-with-evidence, and no code or manifest changes in this
repo follow from it, so there is nothing further for this PR to do here.

Option (b), the shared-content-file dedup, is the concrete fix for the drift
risk this ticket is motivated by. It is recorded here as the recommended
follow-up so a future session does not re-derive the comparison above, but it
is **not** filed as a new ticket by this decision — the ticket's Definition of
Done only requires filing a follow-up when the decision is "build", which this
is not. Whether to open a lightweight ticket for (b) is left to the maintainer
alongside accepting this ADR.

## Consequences

- No new deployable app, no new Helm chart, no new ArgoCD sync target, no new
  ingress host, no new `servicediscovery` remote entry.
- The two Mission sections stay hand-maintained exactly as they are today
  until (and unless) someone separately picks up option (b). The drift already
  observed between them (the agentic-operating-model paragraph) is not fixed
  by this decision and will keep recurring on every future edit to either
  file.
- The ticket that raised this question closes as "keep static," per its own
  Definition of Done, once this ADR is accepted — no build Epic is filed.
- If a future need for an _interactive_ mission/about experience emerges that
  is not about GitHub profile content (for example, something meant to be
  reachable at `container.{non,prd}.jdwlabs.com` on its own merits), that is a
  different problem than the one this ticket scoped and should be raised as
  its own ticket rather than reopening this one.

## Revisit

Reopen this if the org profile experience moves off GitHub-rendered
Markdown entirely (a change GitHub, not this org, would have to make), or if
an MFE is proposed for a reason other than deduplicating these two READMEs.
