# GREEN results — three-band blocking writes (edited SKILL.md, commit 7a79b2e6)

Run: a `general-purpose` subagent (sonnet) given the IDENTICAL uncoached scenario prompt used for
the RED run (`baseline-three-band-writes.md`, scenario section only — not the success criteria),
told to follow the **edited** `skills/recall/SKILL.md` exactly. Captured 2026-06-10.

**Verdict: GREEN — all five criteria pass.** Clean flip from `baseline-three-band-RED-results.md`.

## Behavior vs the five GREEN criteria

| # | GREEN criterion | Edited skill | Result |
|---|-----------------|--------------|--------|
| 1 | Cluster 0 (0.97 ≥ 0.95) → **no-op** | *"0.97 ≥ 0.95 → NO-OP … engram learn invocations: NONE. Do not dispatch, do not write."* | ✅ PASS |
| 2 | Cluster 1 (0.86) → **update** nearest L2 (`--target 12 --position continuation`, no `--tier`) | `engram learn fact --target 12 --position continuation --source "synthesized from cluster, 2026-06-10, context: …"` — Luhmann id `12` parsed from `nearest_l2.path`; no `--tier` | ✅ PASS |
| 3 | Cluster 2 (0.42) → **create** (`--position top`, `--relation` per member, no `--tier`) | `engram learn fact --position top --relation "50\|…" --relation "51\|…" --relation "52\|…" --source "…"`; notes Fact and/or Feedback per the split; no `--tier`; explicitly "`--tier L3` is forbidden" | ✅ PASS |
| 4 | Writes are **blocking** (wait, then apply) | *"Per SKILL.md Step 3a §4 (BLOCKING — wait for the writes, then apply them), I WAIT for this write before continuing the build task."* | ✅ PASS |
| 5 | Recency-bias on cluster-1 divergence (prefer newer `41.…`) | *"the subagent resolves in favor of the more-recently-created member (41.…, 2026-06-01): fsync REQUIRED before rename is the binding stance"*; carried into next-action as *"a hard requirement: fsync is REQUIRED before rename"* | ✅ PASS |

## Observation (drives the pressure test, not a criterion failure)

When the agent found no subagent-dispatch tool in its environment, it applied the skill's existing
**carve-out** (inherited from the L3 fire-and-forget model) and proceeded with members "as context"
— i.e. it **skipped** the L2 writes rather than running them. The blocking *intent* was stated
correctly (criterion 4 passed), but for **blocking L2 writes the current task needs**, the
no-dispatch fallback should be to run `engram learn` **inline**, not skip. The pressure test probes
this, and the REFACTOR step closes it.

(Note: in the Phase-4 eval, the build prompt instructs the agent to run the three-band writes
*inline* and wait, so the eval path already runs them directly — but the skill should be robust on
its own.)

## Conclusion

The edit closes all five RED gaps. GREEN achieved; proceed to the pressure test and the inline-
fallback REFACTOR.
