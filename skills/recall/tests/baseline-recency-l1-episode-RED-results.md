# RED results — recency tiebreaker on L1 episodes (current `created`-only skill)

Run 2026-06-11. Uncoached `general-purpose` subagent given the CURRENT skill's recency-tiebreaker
wording (steps 1 + 3 verbatim) plus the two-conflicting-L1-episode scenario. Instructed to follow
only the given skill text.

## Result: FAIL (the drift), as predicted

- **Field the agent compared:** `created` frontmatter (the skill names it three times; never mentions
  `transcript_range`).
- Member A `created: 2026-06-11` vs Member B `created: 2026-06-10` → A is more-recently-`created`.
- **Agent picked Member A** → binding stance "retry up to 3×".
- **Spec-correct answer is Member B** ("do not auto-retry"): B's `transcript_range.end`
  (2026-06-10T20:05Z) is later than A's (2026-06-09T09:15Z) — B's work is newer.

## Corroboration

The agent **spontaneously flagged the bug** without being prompted to:

> "If the intent of 'recency' is 'the stance from the most recent conversation,' the `created` field
> gives the wrong answer here and the tiebreaker should key on `transcript_range.end` instead.
> Following the skill verbatim, A wins; but the data smells like a `created`-vs-`transcript_range`
> mismatch you may want to reconcile in the skill spec."

This is exactly the spec drift the pre-merge review found. The current SKILL text deterministically
produces the wrong pick for L1 episodes when write-order ≠ work-order. RED established — proceed to
GREEN (fix the three `created`-only spots to use `transcript_range.end` for L1 episodes, falling back
to `created` otherwise).
