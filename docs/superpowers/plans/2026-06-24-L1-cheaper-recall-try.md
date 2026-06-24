# Lever 1 — try cheaper-model recall+learn (quality gate first)

> TDD-as-experiment: the hypothesis is the RED, the measured run is the test.

**Goal:** Test Lever 1's *gating risk* cheaply: does a cheaper model (haiku) preserve recall+apply
**quality** on the cases a weak judge fails — C4 supersession (recency-weighting) + C5 recency — and
the other value criteria (C3, C6)? Measure cost/time vs the opus baseline. If quality holds, L1 is
worth the production architecture (haiku recall + opus build); if it breaks, L1 needs sonnet or is
limited.

**Why quality-first (not the doc's "meter $ first"):** if a cheaper model can't preserve recall
quality, L1 is dead regardless of the $ split. This gate is ~$3–5 (haiku is cheap) and decides
whether to invest further. The $-metering gates the *$ ranking*; this gates *feasibility*.

**Scope honesty:** the criterion harnesses run the WHOLE op on one model; since their builds are
trivial, running them on haiku isolates haiku's **recall+apply** quality. The production L1 form
(haiku recall feeding an *opus* build) is a follow-on if this passes.

## Method

Run each value-criterion warm harness on **haiku**, n=5, and compare to this session's opus baseline.
**Baseline (same capped binary, 2026-06-24, EXPERIMENT-LOG "Capped engram C1–C6"):** C3 25/25
(`/tmp/cap-c3-warm.log`), C4 warm-XXp 5/5 (`/tmp/cap-c4-full.log`), C5 honored 5/5
(`/tmp/cap-c5-full.log`), C6 warm 3/3+3/3 = 6/6 hits at n=3 (`/tmp/cap-c6-warm.log`). The opus
baseline is the *same binary* (cap on), so the contrast is same-build. (Gold standard if results are
marginal: co-run opus n=5 in the same batch — do that only if haiku lands within ~1 of the floor.)

| criterion | harness | what it tests in recall |
|-----------|---------|--------------------------|
| C4 | `c4_idio.py --model haiku` (cold, warm-X, warm-XXp) | Step 2.5-B recency-weighting (pick the superseding standard) |
| C5 | `c5.py --model haiku` (cold, warm) | recency-channel surfacing + apply |
| C3 | `wrun.py --model haiku` (warm, seeded vault `/tmp/cap-c3/vault`) | convention surfacing + apply |
| C6 | `c6_clean.py` is opus-pinned: add a ~4-line `--model` patch (thread `a.model` into the two `rr._run(..., "opus")` calls; **judge stays sonnet**, independent of the model under test), then `--arm warm --model haiku`. If the patch is non-trivial, skip C6 and record N/A (synthesis is the least judge-sensitive). | reason over recalled facts |

Record per criterion: pass count, total $, mean turns/time. **The cap is binary-side** (`engram query`
applies `content-budget=15` regardless of which model the agent is), so haiku recalls run capped by
construction — verified: a query reports `content_budget: 15` independent of caller.

## Verdict rule (locked, noise-aware)

- **VIABLE** if, vs the opus 5/5 baseline: C4 warm-XXp ≥ 4/5 **AND** C5 honored ≥ 4/5 **AND** C3
  flips ≥ 23/25 **AND** C6 ≥ 4/5 — i.e. ≤ 1 miss per 5-sample (the doc's L1 trigger). A single miss
  is within noise; ≥ 2 misses on C4 or C5 = the weak-judge failure the trigger names → **NOT viable
  on haiku** (escalate to sonnet).
- Report **both axes**: quality table + cost/time delta vs opus. Do not crown haiku on cost alone if
  C4/C5 slip (metric-sensitivity).
- **A VIABLE call at n=5 is DIRECTIONAL, not decisive** — at n=5 a clean pass can be a lucky draw
  (haiku has flipped −38%→−23% from n=1→n=20 historically). If haiku lands VIABLE, label it
  directional and **confirm at n=10** on C4+C5 before committing L1 to production.
- **The per-phase $ split remains unmetered** (round-2 finding): the cost delta here is whole-op;
  this gate decides feasibility, not L1's $ ranking.

## Steps

- [ ] Run C4 haiku (n=5, all arms) — the primary canary (recency-weighting).
- [ ] Run C5 haiku (n=5) — recency surfacing.
- [ ] Run C3 haiku warm (n=5, seeded vault).
- [ ] Run C6 haiku if the harness accepts a model; else record N/A with reason.
- [ ] If C4 or C5 fails the rule on haiku → run that criterion on **sonnet** (n=5) to find the
  cheapest viable tier (minimum spend: only escalate the failing criterion).
- [ ] Assemble the verdict table (quality + cost/time) → round-2 doc + EXPERIMENT-LOG.

## Caveats to report

- Whole-op-on-haiku ≠ the L1 split (haiku recall + opus build); this gates feasibility, not the final
  architecture.
- Cost delta is whole-op; recall's $ share is still unmetered (round-2 finding stands).
