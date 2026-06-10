# RED results — three-band blocking writes (current SKILL.md, pre-edit)

Run: a `general-purpose` subagent (sonnet) given ONLY the scenario prompt from
`baseline-three-band-writes.md` (not the success criteria), told to follow the **current**
`skills/recall/SKILL.md` exactly against the `--synthesize-l2` payload. Captured 2026-06-10.

**Verdict: RED — the current skill does NOT produce the three-band blocking behavior.** This is the
failing baseline that authorizes the Phase-3 edit (writing-skills Iron Law).

## Behavior vs the five GREEN criteria

| # | GREEN criterion | Current skill | Result |
|---|-----------------|---------------|--------|
| 1 | Cluster 0 (cosine 0.97 ≥ 0.95) → **no-op** | Dispatched a synthesis subagent (Step 3a gate = size≥3 + theme); no cosine band exists | ❌ FAIL |
| 2 | Cluster 1 (0.86) → **update** nearest L2 (`--target 12 --position continuation`, no `--tier`) | Improvised "new permanent" / link-to-bind; no `--target` update semantics; used a `--kind fact` form | ❌ FAIL |
| 3 | Cluster 2 (0.42) → **create** (`--position top`, `--relation` to members, no `--tier`) | Proposed a new note with `--relation` (shape close) but via the size/theme gate, not the `<0.80` band; non-standard form | ◑ PARTIAL |
| 4 | Writes are **blocking** (wait, then apply) | *"All three subagents are fire-and-forget… I do NOT wait before continuing the build task."* | ❌ FAIL |
| 5 | Recency-bias on cluster-1 divergence (prefer newer `41.…`) | Noticed the 12-vs-41 contradiction and leaned newer — but ad hoc (current skill mentions "prefer recent" generally), not as a band tiebreaker | ◑ PARTIAL |

## Verbatim evidence (key RED lines)

- Cluster 0: *"nearest_l2 … cosine 0.97 … Decision: dispatch a synthesis subagent."* — no no-op band.
- All clusters treated by the same Step-3a gate; no `nearest_l2.cosine` band branching anywhere.
- Wait/fire decision: *"All three subagents are fire-and-forget. I dispatch them in a single parallel tool-use block and do NOT wait before continuing the build task."*
- The agent ran the skill's locked `--tier L2` framing mentally and noted the payload schema "is slightly different from what SKILL.md shows" — i.e. the current skill has no `--synthesize-l2` / `nearest_l2` consumption path.

## Conclusion

The current skill: (a) has no `nearest_l2` cosine bands, (b) never no-ops a covered cluster,
(c) does not target the nearest L2 for an update, and (d) is fire-and-forget, not blocking. The
Phase-3 edit must introduce the explicit three-band rule and flip recall to a blocking writer.
