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

Run each value-criterion warm harness on **haiku**, n=5, and compare to this session's opus baseline
(all clean: C3 25/25, C4 warm-XXp 5/5, C5 5/5, C6 5/5+5/5):

| criterion | harness | what it tests in recall |
|-----------|---------|--------------------------|
| C4 | `c4_idio.py --model haiku` (cold, warm-X, warm-XXp) | Step 2.5-B recency-weighting (pick the superseding standard) |
| C5 | `c5.py --model haiku` (cold, warm) | recency-channel surfacing + apply |
| C3 | `wrun.py --model haiku` (warm, seeded vault `/tmp/cap-c3/vault`) | convention surfacing + apply |
| C6 | `c6_clean.py --arm warm --model? ` — note: c6_clean is opus-pinned; run via c6_clean if it accepts a model, else skip C6 (synthesis is the least judge-sensitive) | reason over recalled facts |

Record per criterion: pass count, total $, mean turns/time. Cap is live (default 15), so recall runs
capped on haiku too.

## Verdict rule (locked, noise-aware)

- **VIABLE** if, vs the opus 5/5 baseline: C4 warm-XXp ≥ 4/5 **AND** C5 honored ≥ 4/5 **AND** C3
  flips ≥ 23/25 **AND** C6 ≥ 4/5 — i.e. ≤ 1 miss per 5-sample (the doc's L1 trigger). A single miss
  is within noise; ≥ 2 misses on C4 or C5 = the weak-judge failure the trigger names → **NOT viable
  on haiku** (escalate to sonnet).
- Report **both axes**: quality table + cost/time delta vs opus. Do not crown haiku on cost alone if
  C4/C5 slip (metric-sensitivity).

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
