## Why

A runbook's `red_flags` field can be silently dropped before it ever reaches an agent: Claude Code's own Bash-tool-output persistence truncates large tool results at roughly 2KB, and a runbook with enough accumulated `red_flags` entries exceeds that on its own, regardless of field position. Confirmed on the production route runbook (25KB total, red_flags block alone ~4.1KB across 15 entries): the newest entry, added specifically to fix a different defect (#757), was cut and never delivered — not ignored, never seen (#763).

## What Changes

- `internal/cli/query.go` and `internal/cli/show.go` now pass a runbook note's content through a new `capRedFlagsForPreview` transform before returning it, so an oversized `red_flags` list is truncated safely (newest entries kept, an explicit in-band marker names the omission) instead of silently losing entries to an external truncation boundary neither command controls.
- `engram show <basename>` is no longer treated as an unconditionally-safe fallback for a truncated query preview — it faces the identical external truncation for a large note, so it now gets the same protection as `engram query`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `recall-runbook-surfacing`: the existing requirement "A surfaced runbook SHALL render its `red_flags` and be retrievable in full" assumed `engram show` is an unconditionally complete fallback and did not address what happens when `red_flags` itself is too large to render in full under external output truncation. Adds a normative sentence and a new scenario covering the truncation-safety guarantee; existing requirement/scenario text is kept verbatim (vault notes 643/651's header-matching convention).

## Impact

- `internal/cli/query.go`, `internal/cli/show.go`: wired to the new transform.
- `internal/cli/redflags_truncation.go` (new): `capRedFlagsForPreview` and its helpers.
- `internal/cli/redflags_truncation_internal_test.go` (new, whitebox), plus one new test each in `internal/cli/query_runbook_test.go` and `internal/cli/show_test.go` (blackbox).
- No schema or on-disk note-format change — this is a display-time transform only, matching the existing `stripWikilinks` precedent (served content already legitimately differs from stored file bytes).
