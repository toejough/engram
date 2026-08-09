## Context

`agent-instructions/skills/route/SKILL.md`'s Cold-start priors table is plain markdown; recorded
dispatch evidence (vault notes tagged `work-kind/go-package-implementation`, `tier/cheap|mid`,
`outcome/pass|fail`) already clears the skill's own escalation thresholds (documented in
`proposal.md`). No architecture, data model, or dependency questions are in play — this is a
single-row text edit to a skill file, landed via the existing `writing-skills` TDD loop.

## Goals / Non-Goals

**Goals:**
- Add the one evidence-backed row to the Cold-start priors table.
- Preserve the table's existing format and its "unproven — evidence overwrites these" framing for
  every other row.

**Non-Goals:**
- No change to the routing algorithm, the evidence-recording workflow, or any other work-kind's
  cold-start tier.
- No new tooling — the refit-trigger mechanism that would make this periodic is tracked
  separately (toejough/engram#723).

## Decisions

- **Row source of truth is the evidence tally in proposal.md**, not a new derivation — the numbers
  (8/12 cheap, 10/10 mid) were computed directly via `engram count --group-by tags --filter
  tags=work-kind/go-package-implementation --filter tags=tier/<t>` and are cited in the row's
  Status column, matching the existing table's convention (e.g. the memory-discount row cites
  "vault note 135").
- **No design alternatives considered** — the table format and evidence-citation convention are
  fixed by the existing skill; this change follows it exactly rather than introducing a new shape.

## Risks / Trade-offs

- [Evidence could be stale by the time this lands] → the row cites the tally date (2026-08-09);
  if `writing-skills` TDD surfaces a materially different count at implementation time, re-run the
  `engram count` queries from `proposal.md` and use the current numbers instead of the ones
  recorded here.
