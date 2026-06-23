# Design — the compounding eval (does PERSISTING synthesis pay off?)

> **Status: spec + RED.** Grounded in note 74 (persisting synthesis ≠ in-session reasoning), note 73
> (engram value is memory, not reasoning scaffolds), and the synthesis RED (`EXPERIMENT-LOG` 2026-06-23:
> warm composes 18/18 in-session but stores nothing). This eval tests the ONE synthesis capability not
> yet ruled out: **persisting the validated conclusion so the web of lessons compounds.**

## 1. The question

In-session synthesis is redundant to scaffold (opus composes A+B→C 18/18 on its own). But C **evaporates** —
it is never written back, so the vault never accumulates higher-order notes and the same composition is
re-derived every session. Does **persisting** a validated C (so later reasoning stands on stored
intermediates) make deep/laddered reasoning more reliable than re-deriving from raw facts each time?

**The honest prior:** every test this session showed opus reasons fine once notes are co-surfaced, so a
*shallow* ladder will likely show **no** headroom (4th negative). The eval is therefore built to **escalate
until no-persist breaks** — find the depth/scatter where re-deriving from raw fails, or prove it never does.

## 2. The laddering A/B

A **chain ladder** of idiosyncratic facts (so cold opus can't shortcut; recall must supply them):
`f1: flag-A enables flow-B` → `f2: flow-B writes table-C` → `f3: table-C triggers job-D` →
`f4: job-D pages team-E` → … (extend to depth k). The **depth-k question** asks the end-to-end
consequence ("if flag-A is enabled, who ultimately gets paged?" → team-E), which requires chaining all k
rungs.

| arm | vault contents | what the agent must do |
|---|---|---|
| **no-persist** (current behavior) | the k raw rungs only | recall all k rungs, chain them in one pass |
| **persist** (the candidate) | the k raw rungs **plus** the stored intermediate syntheses (e.g. "enabling flag-A ultimately triggers job-D") — simulating prior sessions that persisted each rung | recall the near-complete stored chain + the last hop → one step |

**Two stress axes, escalated to find the breaking point:**
- **depth** k ∈ {4, 6, 8}: deeper chains are harder to re-derive in one pass.
- **scatter**: add D distractor notes (0, then ~40) so recall may miss a rung (tests whether a single stored
  intermediate is a more robust recall target than co-surfacing all raw rungs — note: slice 2 found recall
  strong, so this may not bite).

## 3. Metric & decision rule (labeled, per cell)

**Metric:** end-to-end chain hit rate — independent judge: does the answer state the correct terminal
consequence (team-E), having actually traversed the chain (not guessed)? n≥5 per cell.

| chain hit rate (% of n) | no-persist | persist | Δ = persist − no-persist |
|---|---|---|---|
| depth 4, scatter 0 | _measured_ | _measured_ | _headroom?_ |
| depth 6, scatter 0 | _measured_ | _measured_ | _headroom?_ |
| depth 8, scatter 0 | _measured_ | _measured_ | _headroom?_ |
| depth 6, scatter 40 | _measured_ | _measured_ | _retrieval headroom?_ |

- **RED outcome A (redundant):** no-persist stays at-or-near persist across all cells → re-derivation from
  raw is reliable; persisting synthesis buys nothing for task accuracy → **do not build** the persist
  mechanism (record the 4th negative; note the web-as-artifact value is then a *separate, non-accuracy*
  argument).
- **RED outcome B (headroom):** no-persist degrades with depth/scatter while persist holds (Δ above the
  noise floor, sized from no-persist-vs-no-persist) → persisting synthesis has real value → **build it**
  (GREEN), then re-run to confirm the built mechanism reproduces the persist arm's advantage.

## 4. The persist mechanism (only if RED outcome B)

A recall/learn step that, after the agent composes a conclusion C across surfaced notes, **crystallizes C
as a synthesis note** — with a **validation gate** so the vault doesn't rot (notes 68/69):
- persist C **only if** a truth-preserving / satisfaction mode (deduction/abduction/composition) confirms
  it — analogy-derived C is NOT persisted (note 69);
- record provenance links to the source notes;
- this is the deferred roadmap "synthesis-note Z" — a *write-side* capability, like slice 1 (which
  survived because write-side artifacts grow the graph), not an in-session reasoning scaffold (which the
  agent doesn't need).

## 5. Build plan

1. `dev/eval/traps/compound_fixtures.py` — the chain-ladder fixtures (depth-parameterized; raw-only vs
   raw+stored-intermediates; optional distractor pad). Idiosyncratic tokens; terminal consequence in no
   single rung.
2. `dev/eval/traps/compound_eval.py` — the no-persist vs persist warm `/recall` A/B (reuse
   `build_warm_cfg`/`RECALL_PREFIX`/`MODELS`); independent judge for terminal-consequence hit.
3. **Run the RED** across the cells (§3). Decide outcome A vs B against the noise floor.
4. **If B:** build the persist step in `skills/recall|learn/SKILL.md` via `superpowers:writing-skills`
   TDD + the validation gate; re-run to confirm.

## Self-review
- Tests PERSISTENCE (storage/compounding), not in-session reasoning (already proven redundant)?
- Built to find the breaking point (escalating depth + scatter), not toy-confirm?
- Honest that a 4th negative is likely and itself informative?
- Persist gated on truth-preserving validation (no vault rot, notes 68/69)?
- Decision rule self-contained with a noise floor, labeled per-cell table?
