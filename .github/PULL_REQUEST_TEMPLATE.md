## Summary

<!-- What does this PR do? Why? 1-3 bullets. -->

-

## Type of Change

- [ ] `feat` — new feature
- [ ] `fix` — bug fix
- [ ] `refactor` — code change with no behavior change
- [ ] `perf` — performance improvement
- [ ] `chore` — dependency update, tooling, config
- [ ] `ci` — CI/CD pipeline change
- [ ] `docs` — documentation only
- [ ] `build` — build system change
- [ ] `test` — adding or fixing tests
- [ ] `style` — formatting / whitespace (no logic change)
- [ ] `revert` — revert a previous commit

## Test Plan

<!-- How was this tested? What scenarios were covered? -->

- [ ]

## Checklist

- [ ] Conventional commit messages (`feat(scope): subject`)
- [ ] No secrets or credentials in the diff
- [ ] `pnpm exec nx format:check` passes
- [ ] `pnpm exec nx affected -t lint` passes
- [ ] `pnpm exec nx affected -t test` passes
- [ ] New Angular libs have `type:`, `scope:`, and `framework:` tags in `project.json`
- [ ] No cross-scope imports (e.g., `scope:container` importing `scope:authui` libs)
