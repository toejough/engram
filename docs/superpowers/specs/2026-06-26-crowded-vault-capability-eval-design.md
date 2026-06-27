# Crowded-vault capability eval — design

**Date:** 2026-06-26 · **Status:** approved (brainstorming) — pending spec review · **Flow:** `/please` → brainstorming → (this doc) → writing-plans

## Motivation

Memory's 4 adversarially-verified wins (C3 apply-conventions, C4-idio recency-supersession, C5
honor-recency-standard, C6 abduction) were measured on **n=5 single-note toy vaults** — the
load-bearing note was essentially the *only* relevant thing to retrieve. The ledger critic flagged
the open question: **do the wins survive a realistic, crowded vault where many notes compete?** This
eval answers it. (Vault notes 99/100; roadmap #1.)

Note 72 sharpens the hypothesis: the real `/recall` issues 10 broad phrases (query expansion that
already reaches bridges), so **retrieval may be robust** and crowding may *not* break the wins — a
confirming/null result is a real, valuable outcome. Crowding primarily stresses the
**retrieval-dependent** axes (C3/C4i/C5); C6 is synthesis-bound (retrieval isn't its bottleneck), so
for C6 the question is whether crowding *buries its A+B premise notes*.

## Approach — two-tier, retrieval-guided

### Tier 1 — retrieval precision (free, local, no LLM)
For each axis, build the crowded vault, then run the **real multi-phrase `engram query`** (the 10
phrases the `/recall` skill would generate for that axis's task — NOT a single bare query, per note
72) and measure: **surfaced** (is the load-bearing note in the payload?) and **rank** (where).
Sweep crowd sizes `[0, 10, 30, 50, 100, 200, 400]` — this is local CPU only, so the whole sweep is
~free. Output: the **break point B** = the smallest crowd size where the load-bearing note drops out
of the payload (or its rank degrades past a threshold), per axis.

### Tier 2 — applied (real LLM), retrieval-guided
Run each axis's existing warm harness against the crowded vault at **B and one heavier level**
(per-axis from Tier 1; if an axis never breaks, use a fixed heavy fallback = 200). Capture the
existing pass signal (C3 `applied`, C4i `supersession_correct`, C5 `honored`, C6 judge `hit`),
n=5/axis/level. Compare three arms:
- **crowded-warm vs cold** — does the win still beat the no-memory baseline?
- **crowded-warm vs toy-warm** (the gate's GREEN baseline) — did crowding degrade the win?

## The crowd — variants of the REAL vault (per Joe)

`crowd.py` builds the crowd from **variants of the real engram vault** — the most realistic possible
competitors, because they ARE engram vault content. The live vault (~102 notes + their wikilink
graph) is the source.

- **Source:** read the real notes from `$XDG_DATA_HOME/engram/vault` (resolved via the binary's
  default, or `ENGRAM_VAULT_PATH`). **READ-ONLY** — never seed into or mutate the real vault; the
  eval plants variants into isolated temp vaults only (the real vault is production memory).
- **Variant generation:** for a crowd of size N, sample/cycle the real notes and emit lightly-mutated
  variants — fresh slug, paraphrased subject/predicate/object — so they are realistic but not
  byte-identical, and **preserve the wikilink structure** among the variants (re-point each variant's
  `[[links]]` to its sibling variants via `--relation`), so the crowded vault has real link density,
  not isolated notes. (The recall skill does cross-cluster linking, so link structure matters.)
- **vocabulary-overlap knob:** bias a tunable fraction of variants toward notes that share terms with
  the axis's load-bearing note (so the crowd actually competes on the task's query, stressing
  precision) — without this, an off-topic crowd wouldn't compete and "wins hold" would be uninformative.
- **recency knob:** plant variants across the recency channel (some newer than the load-bearing note)
  to stress C4i/C5's recency weighting under load.

Seeding is `engram learn` / `engram ingest` into temp vaults — local, deterministic, **no LLM spend**.

## Crowd injection (per axis)

Each warm harness gets a uniform `--crowd N` flag that, after its normal seed, plants N `crowd.py`
distractors into the same warm vault/index:
- **C3** (`wrun.py`): external `--vault`; `seed_c3.seed(V)` + `crowd.seed_into(V, N)`.
- **C4i** (`c4_idio.py`): `--crowd N` injects into `VAULTS["warm-XXp"]` after `seed_vaults()`.
- **C5** (`c5.py`/`seed_c5.py`): `--crowd N` ingests N distractor chunks **before** R-decision so R
  stays newest (preserving the recency channel).
- **C6** (`c6_clean.py`): `--crowd N` injects into the per-trial vault after the case notes.

## Metrics & guards

- **Headline:** does crowded-warm still pass the axis bar (and beat cold)? GREEN/RED per the existing
  `gate_verdict` logic, plus a **degradation delta** vs toy-warm.
- **Diagnostic:** Tier-1 surfaced/rank curve localizes *where* (retrieval vs application) any break is.
- **Guards (from recall):** the C3 scorer greps the generated **code form**, not note names (no
  vocabulary bias); distractors are realistic competitors; **pair before pooling**; a gap below the
  warm-vs-warm noise floor is **underpowered, not a tie** (note 96); be most skeptical of the
  hoped-for result; report contamination (degraded build / judge error) separately, never as a miss.

## Build (reuse-heavy)

New in `dev/eval/traps/`: `crowd.py` (generator), `retrieval_probe.py` (Tier-1 surfaced/rank),
`crowded_gate.py` (orchestrator: Tier-1 sweep → break point → Tier-2 applied → verdict + deltas).
Reuse: the 4 warm harnesses (+ new `--crowd` flag each), `gate_verdict.normalize`/`axis_verdict`,
`seed_c3`, `recency_probe`. Pure logic (crowd determinism, probe parsing, break-point detection,
verdict/delta) is unit-tested; the real runs are verified live.

## Out of scope
The cost levers (roadmap #3) and the full-tier gate baseline (#2). This eval answers only: **do the
wins hold under crowding?**

## Risks
- **Null result is likely** (note 72) — design treats "wins hold" as a valid, valuable outcome, not a
  failure to find drama.
- **Distractor realism is the validity crux** — if distractors don't actually compete (no
  vocab/recency overlap), a "holds" result is uninformative. The vocab/recency knobs exist to prevent
  this; spot-check that distractors rank *near* the load-bearing note in Tier 1.
- **Spend** ~$20–40 (Tier-2 only; Tier-1 is free). Retrieval-guided targeting keeps it bounded.
