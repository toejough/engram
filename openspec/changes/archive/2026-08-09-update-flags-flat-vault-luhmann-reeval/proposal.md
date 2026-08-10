## Why

The `restore-luhmann-branching-disposition` change (2026-08-09, issue #701) restored the
`learn` skill's ability to place NEW notes as `continuation`/`sibling` instead of always
`top`, but does nothing for the 447 existing notes already flat since f620bfaf (2026-06-12).
`engram update` already detects and notifies about other vault-shape drift it does not
silently fix (old-format vocab files, untagged vocab definitions, a duplicate chunk-index
backlog) — but a notice that only points at a mechanism affecting future notes is impotent for
an already-flat vault: there must also be a way to actually fix it. This change adds both the
detection notice AND a one-shot batch command that re-evaluates and re-parents existing
top-level notes, so the notice has a real remedy to point at.

## What Changes

- `engram update` gains a vault diagnostic: when every note ID in the vault is a top-level
  Luhmann ID (no `continuation`/`sibling` children exist anywhere), report this in the update
  Report (mirroring `VaultHasOldVocabFiles` / `VaultHasUntaggedVocabDefinitions`) and print a
  one-line notice naming `engram update --reparent-luhmann`.
- **BREAKING (vault-mutating, opt-in)**: new `engram update --reparent-luhmann` one-shot
  command, mirroring the existing `--regen-vocab` migration-flag precedent:
  - **Derive phase**: the binary finds candidate parent/sibling relationships among top-level
    notes using their existing embedding sidecars (nearest-neighbor similarity), and emits a
    JSON payload of candidate pairs needing a disposition judgment (mirrors `vocab refit`'s
    `naming_requests`/`fingerprint` flow) — the binary does not decide continuation vs. sibling
    vs. no-relation on its own; that judgment requires reading note content.
  - **Answer phase**: the calling agent (the `learn` skill, in a new batch-re-eval mode) applies
    the same disposition test from #701 to each candidate pair and writes an answers file.
  - **Apply phase**: `engram update --reparent-luhmann --answers <file>` computes new Luhmann
    IDs (reusing `nextChild`/`nextSibling`/`nextTopLevel`), renames each affected note file and
    its `.vec.json` sidecar, and rewrites every incoming wikilink/`supersedes` reference
    vault-wide from the old basename to the new one, so no link breaks.
  - `--dry-run` prints the full rename map (old basename → new basename, and every reference
    that would be rewritten) without writing anything, exactly like `vocab refit --dry-run`.
  - Stale-answers safety: if the vault changed between derive and apply, the apply is void and
    must be re-derived (mirrors `vocab refit`'s stale-names error).
- No change to `internal/cli/luhmann.go`'s ID-computation logic — reused as-is.

## Capabilities

### New Capabilities
- `update-flat-vault-luhmann-notice`: `engram update` detects an all-top-level Luhmann ID tree
  across the vault and surfaces a one-line notice naming the remedy command.
- `update-reparent-luhmann-batch`: `engram update --reparent-luhmann` derives candidate
  parent/sibling relationships, accepts agent-authored disposition answers, and applies a
  vault-wide rename + backlink rewrite that re-parents existing top-level notes.

### Modified Capabilities
(none — additive to the existing `engram update` vault-diagnostic and migration-flag patterns;
no existing requirement's behavior changes)

## Impact

- `internal/cli/update.go` — detector function + notice-writer (as before); new
  `--reparent-luhmann`/`--answers`/`--dry-run` flag handling and derive/apply command bodies
- `internal/update/update.go` — new opaque bool field on `Report`
  (`VaultHasOnlyTopLevelNotes`), following the `VaultHasOldVocabFiles` pattern
- `internal/vaultgraph/` — likely needs a new rename-aware backlink-rewrite helper (no such
  helper exists today; `ParseWikilinks`/`ScanVault`/`ParseBasename` are read-only) — confirm
  during design
- `internal/cli/luhmann.go` — reused, not modified, for ID computation
- `docs/GLOSSARY.md` — document both the notice and the new `--reparent-luhmann` command
- `agent-instructions/skills/learn/SKILL.md` — new batch-re-eval mode that answers the derive
  payload (separate from the normal per-capture Step 2 disposition flow)
