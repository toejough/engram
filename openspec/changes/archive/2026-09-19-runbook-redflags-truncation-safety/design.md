## Context

Issue #763 named two candidate fixes: (a) render `red_flags` before the step body, or (b) rely on agents to run `engram show <basename>` when a truncation notice is visible. Both turned out to be insufficient, measured directly against the production route runbook (vault note `1036.2026-09-18.route-dispatch-tier-selection`, 25KB total):

- **(a) is already true and still insufficient.** `runbookFrontmatterDoc`'s struct field order already places `red_flags` 5th — well before `situation`'s prose ends and long before the step body. Direct byte-offset measurement: `red_flags:` starts at byte 654, but the list itself (15 entries) spans to byte 4766 — ~4.1KB on its own, well past the ~2KB point where Claude Code's Bash-tool-output persistence truncates. Field position cannot fix a field whose own content is too large.
- **(b) does not help for a large note.** `internal/cli/show.go`'s `RunShow` (before this change) printed the raw file body with zero protection. It faces the identical external truncation Claude Code applies to any large tool output — calling `engram show` on this runbook returns the same content `engram query` already showed, truncated at the same point. Rule 7 (in `agent-instructions/guidance/shim.md`) prescribes a remedy that doesn't remedy anything for a note this size.

The actual failure mode: callers append new `red_flags` entries (`engram learn runbook --red-flag`, `engram amend --red-flag`) to the end of the existing list. The newest entry — almost always the one that matters most right now — is therefore the one most likely to sit past a truncation boundary as the list grows.

## Goals / Non-Goals

**Goals:**
- Guarantee the most-recently-added `red_flags` entry survives external output truncation, for both `engram query` and `engram show`.
- Make any omission visible in-band (in the rendered content itself), not silently invisible.

**Non-Goals:**
- Changing the on-disk note format or `runbookFrontmatterDoc`'s field order — already correct, per Context.
- Building a general-purpose "smart truncation" system for note bodies beyond `red_flags` — out of scope; `red_flags` is the safety-critical field this issue is about, not step bodies or other frontmatter.
- Adding a `--red-flags-only` `engram show` mode or any new CLI surface — the fix is a display-time transform on existing commands, not a new capability.
- Fixing agents' compliance with shim rule 7 — a behavioral lever this session's own evidence (issue #762's investigation) found unreliable; this change removes the dependency on that compliance for `red_flags` specifically, rather than trying to improve it.

## Decisions

**D1. Cap the rendered `red_flags` list at display time, not at write time.** The stored file is untouched; `capRedFlagsForPreview` transforms only the string returned to the caller (`engram query`'s item content, `engram show`'s printed output) — the same pattern already established by `stripWikilinks` (query already legitimately serves content that differs from the stored file). Alternative considered: reorder/cap the list at write time (`engram learn`/`amend`) — rejected, because the stored note is the durable record; a human or a different tool reading the file directly should see the true, complete list, and capping only matters for the specific delivery paths that face external truncation.

**D2. Keep entries from the END of the list, not the start.** Callers append new entries; the newest is conventionally last. Dropping from the front (keeping the oldest) would have kept exactly the wrong entries in the reproduced case. Alternative considered: track per-entry recency via a timestamp or ordinal — rejected as unscoped; the array has no per-entry metadata today, and adding one is a larger schema change than this issue calls for. The append-order convention is not enforced anywhere in code, but it is the only convention that exists, and reversing the drop direction directly fixes the reproduced failure without inventing new metadata.

**D3. An explicit in-band marker replaces dropped entries, rather than silently shortening the list.** A future agent (or human) reading the content must be able to tell entries were omitted and how to get them (`engram show <basename>`) — even though D3's own remedy (`engram show`) is now ALSO capped, so for a runbook whose full `red_flags` list exceeds even `engram show`'s protection, the marker is honest about a partial view, not a false promise of completeness. (No runbook observed so far has a `red_flags` list large enough to defeat the cap itself — the budget is sized generously against the one reproduced case.)

**D4. Budget sized at 1200 bytes, comfortably under the ~2KB observed truncation point.** Leaves headroom for the frontmatter fields that precede `red_flags` (`type`, `tier`, `situation`, `done_when`) and for `engram query`'s own wrapping (item metadata, YAML structure) around the note content, which itself consumes some of the 2KB budget before the note's own bytes even start.

## Risks / Trade-offs

- **[Risk] The append-order-implies-recency assumption is a convention, not an enforced contract.** A caller that replaces the whole list in a different order (`engram amend --red-flag` replaces wholesale; nothing stops a caller from re-ordering) would defeat D2's guarantee. → Mitigation: this is a real limitation, named here rather than papered over; a future change could add append-only semantics to `--red-flag` if this recurs. Out of scope for this fix, which addresses the reproduced failure directly.
- **[Risk] A `red_flags` list that itself exceeds the 1200-byte budget will still lose its own oldest-kept entries.** → Mitigation: none needed yet — no observed runbook's list is that large; if one arises, the marker at least makes the omission visible rather than silent, which is strictly better than today's behavior.
- **[Trade-off] This does not fix rule 7 (agents not reliably calling `engram show` on a truncation notice) — issue #762's own investigation found wording fixes for this class of gap unreliable.** → Not attempted here; this change removes the *need* for rule 7 compliance specifically for `red_flags`, which is the safety-critical field, rather than trying to fix compliance itself.

## Migration Plan

1. Add `capRedFlagsForPreview` (`internal/cli/redflags_truncation.go`) with a RED-then-GREEN unit test reproducing the exact byte-budget failure.
2. Wire it into `internal/cli/query.go`'s candidate content assignment and `internal/cli/show.go`'s `RunShow`.
3. Add one integration-level test each to `query_runbook_test.go` and `show_test.go` proving the wiring (not just the unit-level helper) survives an oversized list.
4. Verify against the real production route runbook via the installed binary — confirmed: newest entry now lands at byte offset ~1650 (`engram show`) / ~1940 (`engram query`), both comfortably within any 2KB window.
5. No rollback machinery beyond `git revert` — a pure display-time transform, no schema or stored-data change.

## Open Questions

None outstanding — implemented and verified against real production data before this design doc was written (time-constrained execution order; see tasks.md framing).
