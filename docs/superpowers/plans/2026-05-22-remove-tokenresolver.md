# Plan: remove the tokenresolver package and Keychain dependency

**Date:** 2026-05-22
**Branch:** main (direct — single-commit cleanup, no worktree)

## Why

`internal/tokenresolver/` resolves an Anthropic API token from `ENGRAM_API_TOKEN` then falls back to the macOS Keychain (`security find-generic-password -s "Claude Code-credentials" -w`). It has full implementation and test coverage but **no production importer** on `main`:

```
$ grep -rn "tokenresolver" --include="*.go" . | grep -v worktrees | grep -v _test.go
internal/tokenresolver/tokenresolver.go:1: // Package tokenresolver ...
internal/tokenresolver/tokenresolver.go:3: package tokenresolver
```

Only the package itself and its tests reference it. The earlier worktrees that wired `resolveToken()` into `cli.go` never merged, and the user has confirmed nothing will use it going forward.

The C4 L1 diagram (`docs/architecture/c1-system-context.md`) names S6 macOS Keychain and edge R5 as live integrations, which is drift — the diagram describes intent, not code. Per `MOCs/58.c4-and-diagram-authoring`, diagrams render the actual code; code wins.

## Inventory of references on `main`

1. `internal/tokenresolver/tokenresolver.go` — package source (delete)
2. `internal/tokenresolver/tokenresolver_test.go` — tests (delete)
3. `CLAUDE.md` line 20 — `│   ├── tokenresolver/ # API token resolution` (delete)
4. `README.md` line 101 — `tokenresolver/     API token resolution` (delete)
5. `docs/architecture/c1-system-context.md` — five touchpoints:
   - L8 line 19: `keychain(S6 · macOS Keychain)` node declaration
   - L8 line 26: `engram -->|"R5: ..."| keychain` edge
   - L8 line 30: `keychain` included in `external` class list
   - L8 line 38: `click keychain href "#s6-macos-keychain"` handler
   - Catalog row for S6 (line 51)
   - Relationships row for R5 (line 62)

## Steps

1. **Delete the package directory.**
   `rm -rf internal/tokenresolver/`
2. **Strip CLAUDE.md** — remove the `tokenresolver/` line from the directory tree.
3. **Strip README.md** — remove the `tokenresolver/` line.
4. **Edit C4 L1 diagram** — remove S6 node, R5 edge, S6 from class list, click handler, S6 catalog row, R5 relationships row.
5. **Verify removal**:
   - `grep -rn "tokenresolver\|TokenResolver\|find-generic-password\|Claude Code-credentials\|claudeAiOauth" --include="*.md" --include="*.go" . | grep -v worktrees | grep -v .git/` should return empty.
   - `targ check-full` PASS:N / FAIL:0.
6. **Delete the plan file** (this file) — it served its purpose.
7. **Commit** via `/commit` skill — single conventional-commit `refactor:` covering both the code delete and doc updates, or split if /commit insists on atomic separation.

## TDD framing

For deletion, RED is the package's presence; GREEN is its absence verified by:
- `grep` returns empty for all tokens above
- `targ check-full` passes (nothing imported it, so removal cannot break a build)

REFACTOR: confirm CLAUDE.md and README.md directory trees still read cleanly with the line gone (no orphan separators, alignment preserved).

## Non-goals

- Worktrees under `.claude/worktrees/agent-{a002e559,a74e551f}/` still contain the package and `resolveToken()` wiring. They are out of scope — those branches were never merged and are not affected by this cleanup.
- No replacement is being designed. The user has stated nothing will use a token resolver.
