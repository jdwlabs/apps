# Agent Tooling Traps

Failure modes that have misled agents (and humans) working in this repo. Each
one produced output that looked fine and was not. `AGENTS.md` points here.

## Windows checkouts

This repo's Windows clone lives on the **F: drive**
(`F:\Dev\projects\personal\jdwlabs\apps`), and its worktrees must be on the
same drive.

pnpm uses hard links for its content-addressable store. Hard links cannot cross
NTFS volume boundaries. If a worktree is on C: and the repo is on F:,
`pnpm install` silently succeeds but produces an empty `node_modules` (only a
`.pnpm` dir) — no binaries, no hoisted packages. Git hooks that call
`npx --no-install <tool>` then fail.

```bash
# same drive as the repo
git worktree add F:/Dev/worktrees/apps/feat/my-feature -b feat/my-feature
```

If a cross-drive worktree already left `node_modules` with only `.pnpm` and no
`.bin`, replace it with a junction to the main repo's `node_modules`:

```powershell
Remove-Item -Recurse -Force C:\path\to\worktree\node_modules
New-Item -ItemType Junction -Path C:\path\to\worktree\node_modules -Target F:\Dev\projects\personal\jdwlabs\apps\node_modules
```

## CLI output that lies

RTK's filtered output is **not** the tool's output — it summarises, truncates,
and prints its own status lines. Every `rtk` row below is that one root cause.
Run anything you intend to act on through `rtk proxy <cmd>` and read the raw
result.

| Symptom                                                                                                    | Cause                                                                                                                                                                                                                                                                                                                                 | Fix                                                                                                                                                                                                                                          |
| ---------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `rtk go build -o <path>` reports `Go build: Success` with no binary written                                | RTK's success line doesn't reflect the actual Go toolchain result. Reproduced from a worktree: `Go build: Success` printed, exit code **1**, no binary — the VCS-stamping failure (`error obtaining VCS status: exit status 128`) was swallowed. Compile errors _are_ printed, so a clean-looking run is not the same as a silent one | Trust the exit code, never the success line. `rtk proxy go build ...` for suppressed output; add `-buildvcs=false` when building from a worktree                                                                                             |
| `gh pr view <n>` reports `OPEN` for a PR that has already been merged                                      | RTK caches the `gh` response, and the cached body is well-formed — unlike a truncation marker or a bogus success line, a stale answer gives you nothing to notice. Observed on three PRs at once: `gh pr view` said `OPEN` while all three were already merged                                                                        | `rtk proxy gh pr view <n>` (or `rtk proxy gh pr list`) returns live state. Via the API read `.merged`, not `.state` — REST only reports `open`/`closed`, so a merged PR reads `closed`: `gh api repos/<owner>/<repo>/pulls/<n> --jq .merged` |
| `gh pr edit` fails on every PR in this org                                                                 | `gh` resolves the org through a GraphQL **query** that requires the `read:org` scope, and the active `GITHUB_TOKEN` (`ghp_...`) lacks it — it fails before any mutation is attempted (`the 'login' field requires ... ['read:org']`)                                                                                                  | `unset GITHUB_TOKEN` so `gh` falls back to the keyring `gho_` OAuth token, which already carries `read:org`. Fallback if that token is unavailable: `gh api -X PATCH repos/<owner>/<repo>/pulls/<n> --input payload.json`                    |
| `gh run watch <n>` errors or watches nothing                                                               | It takes the run's **databaseId**, not the run number shown in the UI or in a `gh run list` number column                                                                                                                                                                                                                             | Resolve it first — `gh run list --json databaseId,number,headBranch` — and pass the `databaseId`                                                                                                                                             |
| `curl --cacert <ca>.pem https://host` returns HTTP 000 on Windows                                          | HTTP 000 just means no response was ever parsed — it accompanies every failure mode and distinguishes none of them. Windows curl is built against **Schannel**, which does honour `--cacert`; the bundle is not being ignored                                                                                                         | Read the **exit code**, not the HTTP status: `60` = cert verify failed (wrong/untrusted CA), `77` = CA bundle unreadable or malformed, `7` = connection refused. Confirm the control case resolves with the system store                     |
| `nx format:check --all` still reports EOL diffs on Windows right after pulling the `.gitattributes` LF pin | `.gitattributes` only governs new checkouts/re-adds — it does not retroactively rewrite files already sitting CRLF in an existing working tree                                                                                                                                                                                        | One-time per existing Windows clone: `git rm --cached -r . && git reset --hard` to force every tracked file through the new attribute                                                                                                        |
