## Why

The Luhmann ID scheme (`internal/cli/luhmann.go`) implements sibling and child placement
(`--position continuation|sibling`), but commit f620bfaf (2026-06-12) removed the only call sites
that ever chose those positions from the `learn` skill. Every capture since has used
`--position top`, so all 447 vault notes today carry flat top-level integer IDs — the branching
structure the ID scheme was named for is unused capability. This restores the disposition step
that decides sibling vs. child vs. fresh top-level, grounded in canonical zettelkasten branching
rules rather than the pre-removal ad hoc logic.

## What Changes

- Add a disposition step to the `learn` skill's crystallization flow (Step 2): given the note
  being written and its relationship to recently-written or recalled notes, decide
  `--position top|continuation|sibling` (+ `--target <id>` when not top-level).
- Ground the decision rule in cited canonical Luhmann/zettelkasten literature: continuation
  (child, deeper ID) when the new note branches into a sub-point raised by an existing note;
  sibling (same level, next letter/digit) when it continues/extends the same thought at the same
  level; fresh top-level when it starts an unrelated thread.
- Follow `superpowers:writing-skills` TDD for the SKILL.md change: RED baseline showing notes
  land top-level today, GREEN with the disposition step applied, REFACTOR as needed.
- No change to the binary (`internal/cli/luhmann.go` already supports all three positions) — this
  is a skill-behavior change only.

## Capabilities

### New Capabilities
- `learn-branching-disposition`: the learn skill's Step 2 crystallization flow decides note
  placement (top/continuation/sibling + target) using canonical zettelkasten branching rules
  before invoking write-memory.

### Modified Capabilities
(none — `write-memory` and the underlying `luhmann` ID allocation are unaffected; this only adds
a decision step upstream of the existing `--position`/`--target` flags that write-memory already
accepts)

## Impact

- `agent-instructions/skills/learn/SKILL.md` (Step 2 crystallization instructions)
- `agent-instructions/skills/write-memory/SKILL.md` (if the handoff contract needs the
  position/target fields made explicit — verify during design)
- No Go code changes expected; `internal/cli/luhmann.go` and its CLI flags already implement the
  mechanism end to end
- ADR-0025 (tag retrieval) and ADR-0012 (supersession) are unaffected — this restores note
  *placement* structure, not a new retrieval mechanism (per ADR-0007, wikilink-graph traversal
  for retrieval remains rejected; that finding does not apply to hierarchical placement)
