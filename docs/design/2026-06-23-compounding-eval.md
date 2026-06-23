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

**Two scoping honesties, up front:**
- **The persist arm uses *oracle* intermediates** (pre-written ideal syntheses), so this RED is the
  **best case** for persistence. Outcome A (no help even with ideal stored intermediates) is therefore a
  *decisive* negative. Outcome B (ideal intermediates help) is only **necessary, not sufficient** — a
  later GREEN must show the *real* persist mechanism produces intermediates good enough to reproduce the
  advantage (a noisy/partial mechanism could erase it).
- **This eval measures the TASK-ACCURACY value of persistence only.** The user's deeper point — the vault
  becoming a richer, inspectable, compounding *artifact* over many sessions — is a **separate question**
  (long-run recall coverage / knowledge density) that a single laddered-task hit rate cannot capture. It
  is explicitly **out of scope here** and named as a follow-on (a multi-session vault-coverage eval). If
  this RED is a negative, that does NOT settle the web-as-artifact value — only the task-accuracy value.

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

Both arms are **warm** (same skill, same `build_warm_cfg`); they differ ONLY in **per-trial vault
contents** (built fresh per trial in a tempdir, as `synth_eval.py` does — not two global vaults). No
skill or binary variant needed.

**Two stress axes, escalated to find the breaking point:**
- **depth** k ∈ {4, 6, 8}: deeper chains are harder to re-derive in one pass.
- **scatter**: add ~40 distractor notes so recall may miss a rung. *Caveat (note 72):* multi-phrase recall
  is strong and may surface every rung regardless, so scatter may not bite — if it doesn't, that itself is
  a finding (retrieval is not the limiter).

**Confound to name, not hide:** the persist arm wins (if it does) either because the stored intermediate
is an *easier recall target* or because it *shortens the reasoning chain* — and the metric (hit rate)
can't separate them (nor can the judge tell whether the agent used the intermediate or re-derived). For
the **build decision both collapse to the same action** ("store the intermediate"), so the RED need not
separate them. IF the RED shows headroom, a follow-up diagnostic (intermediate-in-prompt vs in-vault) can
decompose retrieval-relief from reasoning-relief before committing to a mechanism.

## 3. Metric & decision rule (labeled, per cell)

**RED entry point:** `python3 dev/eval/traps/compound_eval.py` builds per-trial chain vaults and runs the
warm `/recall` arms; an independent sonnet judge rules each answer HIT/MISS on the **terminal
consequence** (does it state team-E, having traversed the chain — not guessed).

**Noise floor first (hard gate):** before judging any Δ, run **no-persist twice** (two independent n=5
batches per depth) and take the floor = the |spread| between the two replicates (expected ~10–20pp at
n=5 — binomial variance is large, so n may need to rise to 8–10 to gate cleanly). A Δ is only real if it
exceeds this measured floor; the RED is **not complete** until the floor exists.

**Metric:** end-to-end chain HIT RATE (% of n). n≥5 per cell (raise to 8–10 if the floor is too wide).

| cell (depth, scatter) | no-persist (% hit) | persist (% hit) | Δ = persist − no-persist (pp) |
|---|---|---|---|
| 4, 0 | _measured_ | _measured_ | _vs floor?_ |
| 6, 0 | _measured_ | _measured_ | _vs floor?_ |
| 8, 0 | _measured_ | _measured_ | _vs floor?_ |
| 6, 40 | _measured_ | _measured_ | _retrieval headroom?_ |

- **RED outcome A (redundant):** no-persist stays within the noise floor of persist across all cells →
  re-derivation from raw is reliable even with deeper chains; persisting synthesis buys nothing for task
  accuracy → **do not build** the persist mechanism (record the 4th negative; the web-as-artifact value
  stays open per §1, a separate eval).
- **RED outcome B (headroom):** **confirmed only if** Δ exceeds the noise floor in **all three depth cells
  (4, 6, 8)** AND at least one cell shows **>10pp** headroom (a monotone degradation of no-persist with
  depth is the expected signature). Then persisting synthesis has real task-accuracy value → **build it**
  (GREEN) and re-run **with the real mechanism producing the intermediates** (not oracle) to confirm the
  advantage survives mechanism quality (per §1's necessary-not-sufficient note).

## 4. The persist mechanism (only if RED outcome B)

A recall/learn step that, after the agent composes a conclusion C across surfaced notes, **crystallizes C
as a synthesis note** — with a **validation gate** so the vault doesn't rot (notes 68/69):
- persist C **only if** a truth-preserving / satisfaction mode (deduction/abduction/composition) confirms
  it — analogy-derived C is NOT persisted (note 69);
- record provenance links to the source notes;
- (Gate operationalization — how the agent distinguishes a deductive/compositional path from an
  analogical one — is deferred to the SKILL.md RED/GREEN, not designed here.)

**This eval GATES the decision to build the deferred roadmap "synthesis-note Z"** (it is the predecessor
test, not the mechanism itself). Synthesis-note Z is a *write-side* capability, like slice 1 (which
survived because write-side artifacts grow the graph), not an in-session reasoning scaffold (the agent
doesn't need that). If RED outcome B holds, the follow-on is to wire synthesis-note Z as a learn/recall
step (separate `superpowers:writing-skills` effort).

## 5. Build plan

1. `dev/eval/traps/compound_fixtures.py` — the chain-ladder fixtures (depth-parameterized; raw-only vs
   raw+stored-intermediates; optional distractor pad). Idiosyncratic tokens; terminal consequence in no
   single rung.
2. `dev/eval/traps/compound_eval.py` — the no-persist vs persist warm `/recall` A/B (reuse
   `build_warm_cfg`/`RECALL_PREFIX`/`MODELS`); independent judge for terminal-consequence hit.
3. **Run the RED** across the cells (§3) — INCLUDING the no-persist-twice noise-floor pass. Decide
   outcome A vs B against the measured floor (the locked §3 criterion).
4. **If B:** build the persist step in `skills/recall|learn/SKILL.md` via `superpowers:writing-skills`
   TDD + the validation gate; then re-run with the **real mechanism producing the intermediates** (not
   oracle) — confirm the advantage survives mechanism quality (per §1).

## Self-review
- Tests PERSISTENCE (storage/compounding), not in-session reasoning (already proven redundant)?
- Built to find the breaking point (escalating depth + scatter), not toy-confirm?
- Honest that a 4th negative is likely and itself informative?
- Persist gated on truth-preserving validation (no vault rot, notes 68/69)?
- Decision rule self-contained with a noise floor, labeled per-cell table?
