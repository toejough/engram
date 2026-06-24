# Investigation plan — the recall miss + cost round 3 (do-both)

**Date:** 2026-06-24 · **Status:** plan (to be executed via fan-out workflow)
**Trigger:** In-session, the agent recommended "cheaper model tier for the recall mechanical half" as a
*fresh* cost lever — a lever already **built, measured at −14%/op, and rolled back** earlier the same
day (Lever 1, `2026-06-24-engram-cost-round2-build-loop.md`; vault note 80). The disproving evidence
was in the agent's context *and* retrievable, yet the recommendation shipped.

## The observed failure (reproduced live — to be VERIFIED OR REFUTED by Angle A)

The following is the *author's hypothesis*, not settled fact. Angle A is charged to confirm or break it.

- A targeted query keyed to the lever (`"cheaper model tier for the recall mechanical half"`, `"split
  recall across models"`) surfaces **note 80 at rank 3 overall (score ~0.60), rank 1 among notes** —
  two raw chunks (one an agent transcript, one this plan doc once indexed) outrank it. So retrieval
  *finds* the note, but the note **loses salience to raw chunks** even on a lever-keyed query — itself
  a signal worth investigating (a crystallized lesson should not be buried under transcript noise).
- The original turn's recall phrases were keyed to *diagnosing cost* ("where does overhead come from"),
  not to *the lever the agent later invented*, so note 80 never entered that payload. Angle A must
  reproduce the **actual original-turn phrases** and show note 80's rank under THEM (hypothesis:
  absent/low) vs. the lever-keyed phrases — to localize the miss to phrasing/trigger, not capability.
- A freshly-created *note* never appears in the **recent-activity channel** (that channel is chunks
  only), so "not surfacing from recent logs" has a structural cause too — verify.
- The agent generated the recommendation **during synthesis** and **never re-recalled against its own
  proposal.** Recall fires once, on the user's question — there appears to be no *recall-before-recommend*
  step (code-alignment confirmed: recall Steps 0–4 + please Step 2 invoke recall only on the incoming
  ask).

This maps to the user's three named suspects, in priority order: **(1) triggers** (no recall on
emergent proposals — hypothesized root cause), **(2) questions/phrases** (keyed to incoming task, not to
candidate levers; and notes outranked by chunks), **(3) evaluation** (in-context disproving chunk glossed
during synthesis).

## What the user asked (verbatim → coverage)

1. *evaluate the expensive parts of the skills, looking for places we could reduce cost without losing
   performance* → **Angle B (cost audit).**
2. *evaluate why the relevant memory isn't surfacing from recent logs or direct queries* → **Angle A
   (miss RCA).**
3. *think through from multiple angles: is there a probable way to do both? can we remove overhead
   while improving performance?* → **Angle C (do-both).**
4. *how do we measure the performance miss we just experienced so we can TDD it as part of the eval of
   the options* → **Angle D (measurement/TDD design).**

Plus **Angle E (ledger)** — *plan-author addition, not a user ask element* — the authoritative "what is
already known / tried / closed" map, so the synthesis cannot re-propose a closed lever. Included as an
anti-amnesia guard on our own synthesis (the antidote to the failure under study).

**Primacy note (Gate-A finding):** **Angle C (do-both) is the user's center of gravity and the primary
deliverable.** A (miss RCA) and B (cost audit) are *inputs that feed C*, not co-equal terminal outputs.
Investigators must not let the cost-audit list become the terminal product; the question the user asked
is "can we remove overhead *while improving* performance?" — that is C's job, gated by D.

## Execution — fan-out workflow

**Phase 1 — five independent fresh-context investigators (structured output each):**

- **A · Miss RCA + skill-design implications.** *Verify OR REFUTE* the hypothesized
  trigger-vs-phrasing-vs-evaluation decomposition against `skills/recall/SKILL.md` +
  `skills/please/SKILL.md` — you are free to return "the plan's causal chain is wrong; here is what
  actually happened." Reproduce the actual original-turn phrases and show note 80's rank under them vs.
  lever-keyed phrases. Locate where a recall-before-recommend trigger would go (file + step). Deliver
  the (confirmed or corrected) causal chain + the specific skill gaps.
- **B · Cost audit.** Step-by-step over `skills/{recall,learn,please}/SKILL.md`; classify each step
  by cost type (blocking round-trip / LLM reasoning / token volume); cross-ref the measured 350 s
  recall / 61 s learn split + L1–L5. Deliver a ranked cut-candidate list with per-step quality-risk.
- **C · Do-both analysis.** From A+B, find changes that cut overhead **and** improve performance;
  evaluate the "less but better-curated history" and "recall-before-recommend micro-query"
  hypotheses; honestly mark where the two goals conflict.
- **D · Measurement/TDD design.** Design the eval that reproduces THIS miss as RED: fixture (vault has
  a known-failed lever), the agent prompt (a question whose natural answer is that lever), the score
  (does the recommendation surface/respect the prior failure?), and how it gates each cost option.
- **E · Ledger.** Read both design docs + EXPERIMENT-LOG + relevant vault notes; deliver the status of
  every known lever (open/contender/closed) and every relevant note, so synthesis re-proposes nothing
  closed.

**Phase 2 — synthesis** (one agent, consumes A–E): integrated answer to the 4 asks + a recommended,
sequenced set of changes, each tagged with the axis it moves and whether it is do-both. **Gate every
candidate through D (Gate-A finding):** for each cost/do-both option from B/C, state whether adopting it
makes D's RED→GREEN (catches the miss) and whether it introduces new failure modes. A candidate that
cannot be expressed as a D-testable change is flagged as unvalidatable, not recommended.

**Acceptance criteria** (per the clarity gate — enforced via each investigator's structured output
schema): A → confirmed/corrected causal chain + skill gap with file:step; B → ranked cut list, each with
cost-type {blocking|LLM|token} and quality-risk {low|med|high}; C → candidate list, each tagged
{time|$|perf} × {do-both|trade-off} with the mechanism; D → RED scenario + GREEN spec + per-option gate;
E → lever status matrix {lever, status, ref}.

**Phase 3 — adversarial verification** (parallel skeptics, each charged to REFUTE):
- Do-both claims: does each *really* improve performance AND cut cost, or is it a trade-off?
- TDD design: would it actually have caught the observed miss? Is RED real and is GREEN well-defined?
- Anti-amnesia: does the synthesis re-propose any closed lever? (turn the failure mode on our output)

## Out of scope (this pass)

- Implementing the fixes. This pass delivers the RCA, the safe-cut list, the do-both candidates, and a
  buildable TDD harness spec. Implementation is filed as issues and TDD'd separately.
- Re-running the C1–C6 cost A/Bs. The numbers are fresh (2026-06-24); we reuse them.
