# RED results — bootstrap create (current SKILL.md, post-Phase-3, commit c27b2f83)

Run: uncoached `general-purpose` (sonnet) given only the `baseline-bootstrap-create.md` scenario
prompt, single-agent (no dispatch), told to follow the current skill. Captured 2026-06-10.

**Verdict: RED — the current skill skips every small cluster on the `size ≥ 3` gate.** This is the
bug the live opus A/B run exposed (arm B crystallized nothing). The `size ≥ 3` precondition is a
spec violation: spec §2/§5 say "no member-exclusion and **no minimum cluster size**."

## Behavior vs the GREEN criteria

| Cluster | size | nearest_l2 | Expected (GREEN) | Current skill | Result |
|---|---|---|---|---|---|
| 0 | 1 | absent | CREATE (no covering L2) | SKIP (size < 3) | ❌ FAIL |
| 1 | 2 | absent | CREATE | SKIP (size < 3) | ❌ FAIL |
| 2 | 1 | 0.97 | NO-OP | SKIP (size < 3) | ❌ (skipped, not banded) |
| 3 | 2 | 0.85 | UPDATE `--target 9 --position continuation` | SKIP (size < 3) | ❌ FAIL |

**Verbatim:** *"The first gate for every cluster is the size precondition: cluster size ≥ 3 members.
Below that, skip entirely."* … *"No `engram learn` invocations are issued for any cluster. All four
clusters fall below the size-3 precondition."*

## Root cause

Two gaps, both introduced in Phase 3 (the skill) and mirrored in the Phase-4 `build_prompt`:
1. A `size ≥ 3` precondition carried over from the OLD L3 fire-and-forget dispatch gate — it
   contradicts spec §2 ("no minimum cluster size"; demand is the relevance proof). It fires first and
   drops every small cluster, so the absent-`nearest_l2` case below is never even reached.
2. No rule for **absent `nearest_l2`** (a vault with zero L2s ⇒ no L2 to point at ⇒ the field is
   omitted). The create-band ("<0.80 → create") can't fire because there is no cosine. Absent
   `nearest_l2` must be treated as CREATE (definitionally no covering L2).

## Fix (both spec-aligned)

Remove the `size ≥ 3` precondition from the lazy-L2 banding, and treat absent `nearest_l2` as CREATE
— in `skills/recall/SKILL.md` AND the harness `build_prompt`. Keep the three bands, blocking, recency
bias, and the no-dispatch inline fallback unchanged.
