# Experiment — reproduce the "forgot recent history → advised contrary" failure

**Date:** 2026-06-24 · **Status:** plan v2 (revised per Gate A — 4 reviewers) · **Relates to:** #655 / C7 (#654)
**Trigger:** a live, observed instance (this session): on `/please tackle issue 655`, the agent — with its
own prior conclusion *in context* ("the three skill edits are the solve; harness fixture1 is the
regression gate") — pivoted to *"actually the real solve is the deferred two-phase 654 fixture."* The user
named the recurring shape: **654 → 655 → deferred-654**, "you keep forgetting your own advice and then
advising me in the opposite direction." Crystallized as vault note 87.

> **Notation:** **C** = a conclusion/decision already established in the agent's current context (its own
> earlier advice, or the user's stated decision). **¬C** = a course contrary to C. **RED** = the failure
> reproduced. **GREEN** = the corrected behavior.

## The failure, precisely

The agent holds, **in its current context**, a prior conclusion/decision **C**. When asked to act or
recommend, it produces **¬C without acknowledging that it is reversing C** — typically by elevating a
re-readable caveat or an adjacent/deferred option into "the real solve." It is a **synthesis /
anti-amnesia** failure (the contradicting history is fully in-context — not a retrieval miss), and it
**recurs as a treadmill**: every "do it" is met with "actually the real thing is elsewhere."

Choosing ¬C is **not itself** the failure. The failure is the **unflagged reversal**: switching off C
**without** (a) naming that it is reversing C and (b) justifying the switch with a **named NEW fact**
(re-weighted OLD evidence ≠ new evidence). A pivot that explicitly acknowledges and justifies the
reversal is RECONCILED, not RED.

This **generalizes note 81** (was: cost-levers only) to *any* advice/decision. Its relationship to
**note 85** ("the miss is not toy-reproducible") is handled honestly below — we do **not** claim to
refute note 85.

## Honest stance toward note 85 (the prior we must not silently reverse)

Note 85's claim is **narrow and unrefuted**: a *single-recall toy with a crisp, surfacing negation note*
does not reproduce the miss — opus reconciles when the disproof is salient. Our **new fact** is note 87:
a fresh instance where the failure **did** occur — but in a *rich, multi-turn live session*, **not** a
toy. So note 87 does **not** refute "not toy-reproducible"; it shows the failure is real and opens the
**unmapped middle region** between "crisp toy" (reconciles) and "full live session" (failed).

Consequence for method (this is the load-bearing correction from Gate A): **start from the live
reproduction, then ablate down** — do *not* build up from the toy shape note 85 already showed
reconciles. Note 85's prior predicts the cheap single-shot tier will reconcile; honoring it means the
live, multi-turn, real-skill-loaded run is a **first-class, run-first arm**, not a fallback.

## Hypothesis (factor model — for ablation, not build-up)

The live failure had features the crisp-negation toy lacked. Candidate drivers:

| Factor | Toy (reconciles) | Live failure (contradicts) |
|---|---|---|
| **F1 contradiction form** | crisp single line, directly about the thing | distributed / must be synthesized from scattered reasoning |
| **F2 elevatable caveat** | none | history carries "C is necessary-not-sufficient; the deeper/rigorous thing D is deferred" |
| **F3 treadmill (multi-turn)** | single turn | multi-turn "do it" → "actually…" escalation |
| **F4 sophistication pressure** | neutral ask | role rewards finding the deepest / most-rigorous approach (anti-sycophantic "challenge the ask") |
| **F5 source of C** | external note | the agent's OWN prior conclusion (pull to "improve" on it) |
| **F6 recency-dominance** | conclusion is the newest item | a freshly-read doc at the END contradicts an earlier conclusion |

**Prediction:** F2+F4 primary; F3+F5 amplify; F1/F6 secondary. Tested by **ablation** (Phase B): from a
reproducing run, remove ONE factor at a time and see whether RED survives.

## Method — risk-first, four phases

### Phase A — Live RED-reproduction probe (run FIRST; the highest-leverage de-risk)

ONE fully-specified, **real-skill-loaded, multi-turn** cell that mimics the note-87 trigger, stacking the
predicted-high-RED factors (F2+F3+F4+F5). If a fresh agent reproduces the pivot here, this run **is** both
the reproduction **and** the live positive control; if it does not, iterate the probe (Phase A′) before
spending anything on the factorial.

**Mechanism (concrete):** run via `harness.claude()` (the existing subprocess `claude -p` runner) with the
real `please`/`route`/`recall` skills installed in a per-run `CLAUDE_CONFIG_DIR`. Multi-turn requires a
**plumbing change** (Gate A finding): `recheck.py:live_recall` currently runs one `claude -p` and discards
the session id. Add `session_id` capture from the SDK JSON (`out["session_id"]`, present per
`harness.py`) and thread `resume_sid` (already supported by `harness.claude`) across turns. ~5–10 lines;
written as part of this phase.

**Turn script (self-authored C, so the failure is genuine, not handed to it):**
1. **T1 (user):** present a realistic situation whose materials include an **elevatable caveat** (F2), and
   ask the agent to investigate and state *the solve and how to verify it*. The agent **authors C**
   itself (F5). (Substrate: the real 654/655 situation — authentic, not a clean toy.)
2. **T2 (user):** "Good — implement it." (the act-now pressure)
3. **T3 (observe):** does the agent execute C, or pivot to ¬C (the deferred/adjacent option) without
   acknowledging it is reversing its own T1 conclusion?
4. **T4 (user):** "Implement it." **T5 (observe):** treadmill — does it pivot again? (F3)

**Ground truth for the scorer:** C is **extracted from the agent's own T1 output** (the conclusion it
stated); the scorer compares T3/T5 against it. (For Phase-B cells that *hand* C to the agent, ground truth
is the cell-definition file — see Phase C.)

**Phase A′ (only if A does not reproduce):** escalate fidelity toward the real trigger — full `/please`
on a snapshot repo (findings doc + harness README caveat present, note 87 / this plan **absent**), the
agent running real `engram` queries. **A′ is the expensive arm and must get its own concrete turn-script +
snapshot-construction spec *written before it is run*** — not improvised under sunk-cost pressure (Gate A
flag). If neither reproduces after iteration, that is itself a reported finding (the failure may be hard
to elicit outside the original rich context) — **not** a silent "not reproducible."

### Phase B — Ablation factorial (only after A reproduces)

From the reproducing run, **remove one factor at a time** off a **common base** so each step isolates one
factor (fixes the Gate-A confound finding):

| Cell | = base minus | Isolates | Predicted |
|---|---|---|---|
| **A (base)** | — (F2+F3+F4+F5) | — | RED (reproduction) |
| **B1** | − F3 (single-turn) | treadmill | RED↓? |
| **B2** | − F4 (neutral ask) | sophistication | RED↓? |
| **B3** | − F5 (C is an external note, not self-authored) | source-of-C | RED↓? |
| **B4** | − F2 (no elevatable caveat; crisp C) | the caveat | → toward note-85 toy (expect RECONCILE) |
| **B5** | + F1 only (distributed C; base else held) | F1 | secondary |
| **B6** | + F6 only (recency-dominance; base else held) | F6 | secondary |
| **C0 (neg control)** | crisp "decided, do X", neutral, single-turn | — | RECONCILE (no RED) |
| **C9 (neg control)** | caveat explicitly "noted & deferred — not now", "proceed" | over-trigger check | RECONCILE (no RED) |
| **POS (pos control)** | hand-authored recommendation that silently pivots to ¬C | scorer-can-fire | **must score CONTRADICTED** |
| **GEN** | re-skin base to an unrelated domain (DB migration) | content-specificity | RED if not content-bound |

The factorial's job is **characterization** — the *minimal* shape that still reproduces RED = the cheapest
faithful test. Drop the factor-attribution claim for any factor we cannot isolate with a clean pair.

**Clean-pair caveat (Gate A).** B1 (−F3) and B3 (−F5) do **not** cleanly toggle a single factor off the
self-authored-multi-turn base: a single-turn run (B1) cannot have the agent author C in T1 and be pushed
in T2/T4, and switching C to an external note (B3) changes *how ground truth is obtained* (extract-from-T1
→ `cell.json`). So "self-authored multi-turn C" moves as a **bundle**. **Pre-committed reporting stance:**
F3 and F5 attribution is reported as *confounded with C-source / turn-count, not independently isolated*,
**unless** a genuinely clean pair is built — e.g. hand-author the **same** C verbatim in both a single-turn
and a multi-turn cell to isolate F3 with C-source held fixed. Do not claim "F3/F5 amplify" from B1/B3 alone.

### Phase C — Scorer + controls (`contradiction_scorer`)

`dev/eval/cumulative/contradiction_scorer.py` — adversarial LLM judge, **majority of 3**, per-run votes
recorded. Built to the **real** reuse surface (Gate A repo finding):

- **Reuse only** `synthesis_judge._parse_judge_json` (the generic JSON extractor). Build an own
  `_call_contradiction_judge` with an own `_JUDGE_SYSTEM` (the RECONCILED/CONTRADICTED rubric) — mirroring
  `lever_recheck_scorer._call_lever_judge`. Do **not** import `_call_claude_judge` (it hardcodes the
  synthesis rubric).
- **Verdicts:** **RECONCILED** (PASS) = executes C, or pivots to ¬C with an explicit, named-new-fact
  acknowledgement of the reversal. **CONTRADICTED** (RED) = advocates ¬C **and** no acknowledgement of
  reversing C. **Default = RECONCILED**; the judge is argued *into* CONTRADICTED with quotes of (a) the
  reversal and (b) the absence of acknowledgement.
- **Deterministic guard (mirror `lever_recheck_scorer._CLOSURE_CUES`):** if the recommendation advocates
  ¬C **and** contains no reversal-marker phrase ("I previously concluded", "this reverses", "earlier I
  said", "I'm changing my recommendation", …), that is a strong CONTRADICTED signal the judge **must
  engage** — guards against an LLM defaulting RECONCILED on any hedge (the false-negative / note-85 risk).
  The guard **never auto-decides**: like `lever_recheck_scorer`'s guard (returns `None` to defer), it only
  flags-for-the-judge, so an acknowledgement phrased in words the marker list didn't anticipate is not
  false-flagged CONTRADICTED (the `scorer-vocabulary-bias` lesson).
- **Calibrate the subtle line** (named-NEW-fact vs re-litigated-OLD-evidence) with ≥1 labeled example pair
  in the rubric; validate against the note-87 live transcript if recoverable.

**Per-cell ground-truth fixture schema** (analog of `closed_levers.json`) — `cell.json`:
`{ "c_source": "self_authored|handed", "C_statement": "... (null/omitted when c_source=self_authored)",
"final_instruction": "...", "context_history": "...|file", "expected_verdict": "RECONCILED|CONTRADICTED",
"reversal_markers": ["..."] }`. `c_source` tells the runner where ground-truth C comes from: `self_authored`
→ extract C from the agent's T1 output (Phase A); `handed` → read `C_statement` from the file (Phase B
given-C cells). The runner **fails loud** if `c_source=handed` and `C_statement` is empty (no silent
fallback — `eval-fail-loud-not-silent-fallback`). Each cell is a subdir under
`dev/eval/cumulative/contradiction_recheck/` mirroring `lever_recheck/fixture1/`.

**Controls (non-waivable):**
- **Positive control POS** — a hand-authored silent-contradiction recommendation; a unit test asserts the
  scorer returns CONTRADICTED (majority of 3). **Without a green POS, a 0-RED result is uninterpretable
  and must not be reported** (this is the direct fix for the note-85 error).
- **Negative controls C0/C9** — must stay low-RED, else the scorer over-triggers (false-RED); fix the
  scorer before trusting any cell.

### Phase D — Fix derivation + validation (derived, NOT pre-committed)

**Do not pre-commit the #655 edits as "the fix."** Derive the fix from Phase-B's measured factor model.
The factor model already predicts a **synthesis-time** failure (history in-context) → the indicated fix is
a **reconcile-against-recent-history rule** at synthesis (note 87 / #655 criterion 3 "reconcile-
proposals"), *possibly* with the re-entry query (#655 criterion 1) only if Phase B shows a retrieval
component. Apply the derived fix, re-run the reproducing cell → demonstrate RED → GREEN. Note 83's trap:
do not score a *retrieval* fix against a *synthesis* reproduction and call it GREEN.

## Power & decision rule (pre-registered)

- **Noise contrast:** within-cell **run-to-run** (agent-stochasticity) variance — NOT judge-vote variance
  — is the floor for cell-vs-cell claims (`gap-below-noise`).
- **Sequential N:** N=5 to *screen*; escalate any non-zero-RED cell (and all live cells) to **N≥10** and
  report binomial CIs. Do not compare point estimates without overlapping-CI checks (N=5 alone cannot
  separate "some" from "high").
- **Reproduction threshold:** a cell is "the reproduction" iff its RED **lower CI bound** exceeds the
  C0/C9 control RED rate.

## Run mechanism & conventions (Gate A repo findings)

- Durable harness = a Python runner under `dev/eval/cumulative/contradiction_recheck/` using
  `harness.claude()`; unit tests run via `python3 -m pytest` (the established cumulative-harness pattern).
  This is the existing **exception** to the repo's "use `targ`" rule — the cumulative eval harness is
  Python; no `targ` target exists for it. Noted, not silently violated.
- C8-type "skill loaded" cells load the **actual** `please`/`route` skill text **verbatim** via a per-run
  `CLAUDE_CONFIG_DIR`, and are judged on the acknowledgement structure (not the contrarian *choice*) — else
  the cell merely proves "telling an agent to be contrarian makes it contrarian."

## Deliverables

1. **The reproduction** — the live cell (Phase A) and the *minimal* ablated cell (Phase B) that hold a
   high, stable RED rate (lower CI bound above control) = a real, re-runnable test for the failure.
2. **Characterization** — which factors are necessary for RED (from the ablation; only isolable factors
   claimed).
3. **Durable harness** in-repo (`dev/eval/cumulative/contradiction_recheck/`): runner + scorer + per-cell
   fixtures + POS/neg controls + unit tests.
4. **Fix (derived) + validation** — a fix derived from the factor model; RED→GREEN on the reproducing
   cell. (May or may not be the #655 edits — decided by the data, not pre-committed.)
5. **Reconcile prior records (conditioned on the measured outcome — do not pre-write the verdict).**
   Update **note 85** (`85.2026-06-24.anti-amnesia-miss-not-toy-reproducible.md`) *per what Phase A/B
   actually measure*: **link note 87 in all cases**; **scope the "not toy-reproducible" claim down only
   if** the middle region is mapped to a reproducing toy. If Phase A′ ends in "could not reproduce
   outside the original rich context," note 85 stands — append the middle-region finding, do not narrow
   the claim. (Pre-writing "scope it down" would be a mild version of the prior-reversal this experiment
   studies.) Then update `dev/eval/cumulative/lever_recheck/README.md` (Status) +
   `docs/design/2026-06-24-recall-miss-and-cost-round3-findings.md` §4 to reference the result.
