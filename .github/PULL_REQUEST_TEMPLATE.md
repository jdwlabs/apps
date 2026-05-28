## Summary

<!-- What does this PR do? Why? 1-3 bullets. -->

-

## Type of Change

- [ ] `feat` — new feature
- [ ] `fix` — bug fix
- [ ] `refactor` — code change with no behavior change
- [ ] `chore` — dependency update, tooling, config
- [ ] `docs` — documentation only
- [ ] `build` — build system or CI change
- [ ] `test` — adding or fixing tests

## Test Plan

<!-- How was this tested? What scenarios were covered? -->

- [ ]

## Checklist

- [ ] Conventional commit messages (`feat(scope): subject`)
- [ ] `pnpm exec nx format:check` passes
- [ ] `pnpm exec nx affected -t lint` passes
- [ ] `pnpm exec nx affected -t test` passes
- [ ] New Angular libs have `type:`, `scope:`, and `framework:` tags in `project.json`
- [ ] No cross-scope imports (e.g., `scope:container` importing `scope:authui` libs)
