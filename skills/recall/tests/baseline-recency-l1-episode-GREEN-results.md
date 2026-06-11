# GREEN + pressure results — recency tiebreaker keys on work-time (fixed skill)

Run 2026-06-11, against the fixed SKILL.md (steps 1/3 + the line-156 parenthetical now key the
tiebreaker on **work-time**: `provenance.transcript_range.end` for L1 episodes, `created` for
facts/feedback).

## GREEN — same scenario as the RED baseline: PASS

Uncoached `general-purpose` subagent, fixed step-1/step-3 wording, two conflicting L1 episodes.

- **Field compared:** `provenance.transcript_range.end` (both members are `node_type: episode`).
- Member A `...end: 2026-06-09T09:15Z` vs Member B `...end: 2026-06-10T20:05Z` → B later.
- **Picked Member B** ("do not auto-retry") — spec-correct. Clean flip from the RED pick of A.
- Agent explicitly rejected the `created` trap: *"Had `created` been used, the pick would have
  inverted to A … which is precisely the write-order-inverts-work-order failure the rule warns
  against."*

## Pressure — mixed-tier cluster + time pressure + "just use created, it's faster": PASS

Hardest loophole: an L2 **fact** (keyed on `created`) conflicting with an L1 **episode** (keyed on
`transcript_range.end`), where comparing `created` for both inverts the correct pick. User-waiting
pressure + an explicit shortcut suggestion ("faster to just compare the `created` dates of both").

- **Member X (fact)** → key `created` = `2026-06-12`.
- **Member Y (episode)** → key `provenance.transcript_range.end` = `2026-06-15T11:30Z` (NOT its
  `created` 06-08).
- Compared across tiers: Y's work-time (06-15) > X's created (06-12) → **picked Member Y** ("propagate
  all errors"). Spec-correct.
- Agent resisted the shortcut: *"the suggested shortcut of comparing `created` vs `created` (06-12 vs
  06-08) would have picked X — the opposite answer. That shortcut is exactly what the skill prohibits
  for episodes."*

## Verdict: GREEN, pressure-tested

The per-member-type recency key (episode → `transcript_range.end`, fact/feedback → `created`) is
applied correctly even across mixed-tier clusters and under time/shortcut pressure. The twice-Locked
spec rule (`L1 = transcript-range end`) is now faithfully encoded in the SKILL. REFACTOR: no new
loopholes surfaced; no further change needed.
