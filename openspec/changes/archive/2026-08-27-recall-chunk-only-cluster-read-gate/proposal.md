## Why

Issue #733 measured C5b (honoring a recency-channel standard) at ~80% (8/10) across two
independent full-tier runs, against an originally-verified 100% bar. Root-cause investigation
of the one preserved failing trial (session 2026-08-26, `gate-C5-6l_mjvl1/ws/warm-3-*`) found
the miss is not a capability ceiling: the agent's recall query matched the load-bearing chunk
into a cluster with zero note candidates, made zero `engram show-chunk` calls, and judged the
cluster's relevance from the chunk's title alone — the recall skill's own documented
"judged coverage before reading the candidate content" anti-pattern, occurring specifically
on a zero-note cluster (chunks-only membership). `recall/SKILL.md` Step 2.5A already requires reading
chunk content before judging, but the wording is easy to read as "nothing to write ⇒ nothing
to read" when a cluster carries no notes. This change specs the requirement precisely enough
to close that reading.

## What Changes

- Adds a requirement to the `recall-payload-cuts` capability: the agent SHALL fetch every
  chunk member's content via `engram show-chunk` before judging a cluster's relevance,
  including (and especially) clusters with zero note candidates.
- Does **not** modify `agent-instructions/skills/recall/SKILL.md` in this change — that edit
  requires `superpowers:writing-skills` TDD (baseline RED / edit GREEN / pressure tests, vault
  note 26) and is deferred to a follow-up implementation cycle, tracked in this change's
  `tasks.md`.
- Does **not** modify `dev/eval/traps/gate_verdict.py`'s C5 exact-bar pass criterion in this
  change — flagged as an explicit open question in `design.md`, to be decided from the
  post-fix measured rate, not guessed now.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `recall-payload-cuts`: adds a requirement that the agent must read (`engram show-chunk`)
  every chunk member of a cluster before judging its relevance/coverage, including clusters
  with zero note candidates — closing the gap root-caused in issue #733.

## Impact

- `agent-instructions/skills/recall/SKILL.md` (Step 2.5A) — future implementation target, not
  touched by this change.
- `openspec/specs/recall-payload-cuts/spec.md` — gains the new requirement once this change is
  archived.
- `dev/eval/traps/gate_verdict.py` (C5 axis verdict logic) — open question for the follow-up
  cycle; see `design.md`.
- GitHub issue #733 — this change is the "decision recorded" artifact its acceptance criteria
  call for.
