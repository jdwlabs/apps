# Contributing

## Commit Convention

This repository follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

### Types

| Type | When to use |
|------|-------------|
| `feat` | New feature or user-visible capability |
| `fix` | Bug fix |
| `chore` | Maintenance: dependency upgrades, config, tooling |
| `docs` | Documentation only (no code changes) |
| `ci` | CI/CD pipeline changes |
| `refactor` | Code restructure with no behavior change |
| `test` | Adding or updating tests |
| `perf` | Performance improvement |
| `revert` | Reverting a previous commit |

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Examples

```
feat(authui): add OIDC token refresh flow
fix(usersrole): return 404 when user not found
ci: add pnpm cache to build workflow
docs: document module federation remote config
chore: upgrade angular to 19.2
refactor(rolesui): extract role list to shared component
test(usersrole): add integration test for role assignment
```

### Rules

- Subject line ≤72 characters, lowercase, no trailing period
- Use imperative mood: "add" not "added" / "adds"
- Scope is optional but encouraged — use the app or lib name
- Breaking changes: add `!` after type/scope and a `BREAKING CHANGE:` footer

## Pull Requests

1. Branch from `main`: `git checkout -b feat/short-description`
2. Keep PRs focused — one logical change per PR
3. PR title must follow conventional commit format: `type(scope): description`
4. Fill the PR template completely
5. Squash-merge to main to keep history clean

## Development Setup

```bash
pnpm install --frozen-lockfile    # Install dependencies
npx nx run-many -t build          # Verify build passes
npx nx run-many -t test           # Verify tests pass
npx nx run-many -t lint           # Verify lint passes
```
