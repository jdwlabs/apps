# Agent Tooling Traps

Failure modes specific to this repo that have misled agents (and humans).
`AGENTS.md` points here. Traps in the tools themselves (`rtk`, `gh`, Windows
`curl`, CRLF after a `.gitattributes` change) are box-wide:
`~/.local/share/chezmoi/docs/agent-tooling-traps.md` in the dotfiles repo.

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
