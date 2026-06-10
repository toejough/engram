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

## Post-refactor re-run

(appended after the REFACTOR + re-test)
