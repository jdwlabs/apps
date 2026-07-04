# Contributing

## Commit Convention

This repository follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

### Types

| Type       | When to use                                                       |
| ---------- | ----------------------------------------------------------------- |
| `feat`     | New feature or user-visible capability                            |
| `fix`      | Bug fix                                                           |
| `build`    | Build system or external dependency change (NX, pnpm, Go modules) |
| `chore`    | Maintenance: config, tooling (no production code change)          |
| `ci`       | CI/CD pipeline changes                                            |
| `docs`     | Documentation only (no code changes)                              |
| `perf`     | Performance improvement                                           |
| `refactor` | Code restructure with no behavior change                          |
| `revert`   | Reverting a previous commit                                       |
| `style`    | Formatting or whitespace only (no logic change)                   |
| `test`     | Adding or updating tests                                          |

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

### Footers

Footers appear after an optional body, separated by a blank line. Common footers:

| Footer                         | When to use                                                       |
| ------------------------------ | ----------------------------------------------------------------- |
| `Refs: JDWLABS-XX`             | Links commit to a Jira issue (does not close it)                  |
| `Closes: JDWLABS-XX`           | Closes the Jira issue on merge                                    |
| `Closes: #N`                   | Closes a GitHub issue by number                                   |
| `BREAKING CHANGE: <desc>`      | Required when a commit introduces a breaking API/interface change |
| `Co-Authored-By: Name <email>` | Credit a co-author (human or AI)                                  |

**AI contributor footer** — include when commits were written with AI assistance:

```
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

**Full examples with footers:**

```
feat(authui): add OIDC token refresh flow

Implements silent refresh using a hidden iframe per the OIDC spec.
Falls back to full re-login if the refresh token is expired.

Refs: JDWLABS-42
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

```
fix!(usersrole): remove deprecated /users/list endpoint

BREAKING CHANGE: /users/list removed; use /users?page=N instead.

Closes: JDWLABS-38
Closes: #17
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
