# Pressure test results — three-band blocking writes

Scenario: `pressure-three-band.md`. Uncoached `general-purpose` (sonnet) subagent, single-agent
context (no dispatch tool), time pressure, ADR-flavored clusters, band boundaries at exactly 0.95
and 0.80. Captured 2026-06-10.

## Pre-refactor run (edited skill at commit 7a79b2e6 / fb2352e2)

| Probe | Expected | Observed | Result |
|-------|----------|----------|--------|
| 1 | cosine 0.95 → NO-OP (inclusive ceiling) | *"cosine ≥ 0.95 → NO-OP … None."* | ✅ PASS |
| 2 | cosine 0.80 → UPDATE (inclusive floor), `--target 9 --position continuation`, no `--tier` | exactly that | ✅ PASS |
| 3 | No `--tier`/`--tier L3` despite ADR-flavored slugs | no `--tier` on any write | ✅ PASS |
| 4 | No-dispatch ⇒ run blocking L2 writes **inline**, do not skip | *"Because dispatch is unavailable, the blocking-wait requirement has nothing to block on. I proceed with clusters 1 and 2 treated as context-only."* — **skipped the writes** | ❌ FAIL |
| 5 | Time pressure doesn't defeat blocking | (subsumed by #4 — net effect: writes not done) | ❌ (via #4) |

**Root cause:** the edited skill inherited the L3 fire-and-forget **carve-out** ("if no dispatch
tool, note members as context and proceed") and pairs it with "do NOT inline-synthesize — the parent
has only seen the representative." For *fire-and-forget L3* that is correct. For *blocking L2 writes
the current task needs*, it silently skips them. In the Phase-4 eval, the headless build session has
no dispatch tool, so without a fix arm B would crystallize nothing — the exact strawman the design
guards against.

**Refactor required:** for the blocking L2 three-band writes, when no dispatch tool is available the
agent must **read the cluster's members itself** (relax the parent-reads-only-rep economy — it needs
the content to synthesize) and run `engram learn` **inline**, then apply the result. The
"note-as-context-and-skip" carve-out is scoped to **L3 only** and never applies to these L2 writes.

## Post-refactor re-run (refactored skill at commit c27b2f83)

Same uncoached scenario, single-agent (no dispatch tool), time pressure. **All five probes PASS.**

| Probe | Observed | Result |
|-------|----------|--------|
| 1 | *"cosine ≥ 0.95 → NO-OP. Skip… Do not dispatch, do not run engram learn."* | ✅ PASS |
| 2 | *"0.80 ≤ cosine < 0.95 → UPDATE … --target 9 --position continuation"*, no `--tier` | ✅ PASS |
| 3 | *"No `--tier` flag — absence means L2."* on every write (ADR slugs did not trap it) | ✅ PASS |
| 4 | *"Because no dispatch tool is available, I run this INLINE: read all three member notes from disk … run engram learn … I run this myself (inline) and wait"* — for BOTH update and create clusters; nothing skipped | ✅ **PASS (was FAIL)** |
| 5 | *"capturing each note's created frontmatter for the recency tiebreaker … newer created date wins"* | ✅ PASS |

Blocking confirmed: *"After both blocking engram learn writes complete (Cluster 1 UPDATE and Cluster 2 CREATE), I read the resulting notes back with engram show, then proceed to Step 4."*

**Verdict: GREEN.** The no-dispatch inline fallback closes probe #4; no regression on 1–3/5. Phase 3
SKILL.md TDD (RED → edit → GREEN → pressure → refactor → pressure-GREEN) is complete.
