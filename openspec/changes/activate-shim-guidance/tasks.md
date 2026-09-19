## 1. Verify the deployment prerequisite (no code expected to change)

- [x] 1.1 Re-confirm `~/.claude/engram/guidance/shim.md` (and its `~/.claude/engram/shim.md` compat
      symlink) are current and byte-identical to `agent-instructions/guidance/shim.md`. If not,
      run `engram update --with-guidance` first and re-diff. — **confirmed identical**, symlink present.
- [x] 1.2 Confirm no code change is actually needed in `internal/update/` for this (per design.md
      D2) — grep `internal/update/` for any per-file guidance allowlist that would need `shim.md`
      added explicitly; if one exists and doesn't already include it, that's a real gap this task
      group must then fix (a genuine finding, not assumed away) before proceeding to task 2. —
      **no per-file allowlist exists**: `internal/update/update.go`'s `--with-guidance` deploys all
      of `agent-instructions/guidance/*.md` generically (confirmed by reading the `WithGuidance`/
      `GuidanceFiles`/`GuidanceImports` fields and their usage) — `shim.md` is already included
      exactly like the other three, no code change needed.

## 2. Check Pi's AGENTS.md symmetry (design.md's Open Question)

- [x] 2.1 Determine whether `~/.pi/agent/`'s `AGENTS.md` currently imports `recall.md`/`delegate.md`/
      `learn.md` the same way `~/.claude/CLAUDE.md` does, before assuming step 3's Pi line applies.
      Record the finding either way. — **confirmed yes**, same pattern: `AGENTS.md` imports all
      three via `@~/.pi/agent/guidance/<file>.md` (not the Claude Code compat-symlink flat-path
      shape — Pi uses the `guidance/` subdirectory path directly in its import line).
      `~/.pi/agent/guidance/shim.md` is already deployed and present.

## 3. Document the exact activation edit (do not apply — orchestrator does this directly)

- [x] 3.1 State the exact line to add to `~/.claude/CLAUDE.md`, positioned alongside the existing
      `@~/.claude/engram/recall.md` / `@~/.claude/engram/delegate.md` / `@~/.claude/engram/learn.md`
      block: `@~/.claude/engram/shim.md`
- [x] 3.2 If 2.1 found Pi's `AGENTS.md` follows the same pattern, state the matching line:
      `@~/.pi/agent/guidance/shim.md` (or the compat-symlink surface path Pi actually uses per
      2.1's finding — do not assume the Claude Code path shape applies unchanged).
- [x] 3.3 This change's own artifacts stop here — the actual edit to `~/.claude/CLAUDE.md` (and, if
      applicable, `AGENTS.md`) is outside `allowedEditRoots` for this change and is applied by the
      orchestrator directly, per design.md D3.

## 4. Close out

- [x] 4.1 Once the orchestrator has applied the documented edit(s) from group 3 and confirmed
      (e.g. by starting a fresh session, or inspecting that the shim's bootstrap-query instruction
      is now part of loaded guidance) that `shim.md` is genuinely active, record that confirmation
      here. — **applied and confirmed**: `@~/.claude/engram/shim.md` and `@~/.pi/agent/guidance/shim.md`
      are now present in `~/.claude/CLAUDE.md` and `~/.pi/agent/AGENTS.md` respectively (verified by
      direct grep against both files); takes effect for sessions started from now on.
- [ ] 4.2 `openspec archive activate-shim-guidance` once 4.1 is confirmed.
