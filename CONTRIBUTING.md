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
| `Refs: KEY-123`                | Links commit to a Jira issue (does not close it)                  |
| `Closes: KEY-123`              | Closes the Jira issue on merge                                    |
| `Closes: #N`                   | Closes a GitHub issue by number                                   |
| `BREAKING CHANGE: <desc>`      | Required when a commit introduces a breaking API/interface change |
| `Co-Authored-By: Name <email>` | Credit a co-author (human or AI)                                  |

**AI contributor trailers** — every AI-assisted commit names the agent and the
model that actually ran (not a copied example):

```
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Assisted-by: Claude Code:claude-opus-5-5
```

Codex: `Co-Authored-By: Codex <codex@openai.com>` and `Assisted-by: Codex:<model-id>`.
Attribution belongs in commit trailers only, never in PR titles, bodies or comments.

**Full examples with footers:**

```
feat(authui): add OIDC token refresh flow

Implements silent refresh using a hidden iframe per the OIDC spec.
Falls back to full re-login if the refresh token is expired.

Refs: KEY-123
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Assisted-by: Claude Code:claude-opus-5-5
```

```
fix!(usersrole): remove deprecated /users/list endpoint

BREAKING CHANGE: /users/list removed; use /users?page=N instead.

Closes: KEY-123
Closes: #17
```

### Rules

- Header ≤100 characters (commitlint enforces it), lowercase subject, no trailing period
- Use imperative mood: "add" not "added" / "adds"
- Scope is optional but encouraged — use the app or lib name
- Breaking changes: add `!` after type/scope and a `BREAKING CHANGE:` footer

## Pull Requests

1. Branch from `main`: `git checkout -b feat/short-description`
2. Keep PRs focused — one logical change per PR
3. PR title must follow conventional commit format: `type(scope): description`
4. Write the body from the PR template, keeping only sections with content (~150 words)
5. Merges are rebase-only — every commit lands on `main` as-is, so tidy the branch into
   one logical commit per change before review

## Development Setup

```bash
pnpm install --frozen-lockfile    # Install dependencies
npx nx run-many -t build          # Verify build passes
npx nx run-many -t test           # Verify tests pass
npx nx run-many -t lint           # Verify lint passes
```
