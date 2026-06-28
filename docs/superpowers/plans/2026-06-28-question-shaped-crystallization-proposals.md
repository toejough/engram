# Question-shaped crystallization + reasoning-from-memory — proposal-generation plan

> **For agentic workers:** RESEARCH/DESIGN plan. The deliverable is a *diverse, evaluable set of proposed
> fixes* (CONTENDER/PARK + a recommendation), not an implementation. "Tests" are the prior-evidence
> constraints + adversarial review, not unit tests. Steps use `- [ ]`.

**The ask (Joe, 2026-06-28).** Propose some fixes to evaluate for *question-shaped crystallization*,
considering everything learned about note quality + the situations we want notes surfaced in + the reasoning
research — ideally so we can **reason from memories toward better problem-understanding and correct outcomes
in fewer passes.**

## The reframing the reasoning research forces (the spine) — read the SCOPE carefully

The prior research evaluated *some* "make the system reason" mechanisms and rejected them — but the evidence
is **scoped**, and over-reading it would dodge half the ask (Gate-A ask-alignment finding):

- **Notes 72/73 (measured) — scoped to *emergent synthesis* + *graph expansion*.** Retrieval-side machinery
  for **C6 emergent synthesis** (A+B→C compositional join) and **graph expansion** has **no measured
  headroom** (C6: warm 18/18 vs cold 1/9; opus reasons fine over what surfaces). → For *that* case, retrieval
  scaffolds are **PARK**, admissible only behind a warm-vs-warm+scaffold A/B. **But 72/73 did NOT measure
  diagnostic/abductive surfacing** (symptom→likely-cause) — the ask's *"better understanding problems"* goal.
  That case is **UNMEASURED, not closed** — so it earns an angle, not a tombstone.
- **Note 74 (the live write-side lever):** persisting *validated synthesis* — capture the reasoned conclusion
  so a *future* session recalls it already-reasoned; the compounding lever for "fewer passes."
- **Note 76 — three independent values of persistence:** inspection/audit, correction, **AND reuse by a
  weaker model** (a cheaper model recalls the pre-computed conclusion instead of re-deriving). The third is a
  distinct, directly-evaluable "fewer passes" (and cost) path.
- **Note 120 / the audit — the *worth-surfacing* gap:** cluster-driven notes are topic-shaped, ~half as
  question-useful. Note 119 (linked from 120): **surfacing** (the floor, shipped) and **worth-surfacing**
  (note quality) are the two halves of retrieval value — the proposals must touch both.
- **Persistence rigor (76/68/69), separate from question-shaping:** persist only *sound* conclusions, hedged
  by inference mode + linked to inputs, never aggregation/analogy-as-proof; gate against vault-rot.

**Net:** write-side quality (question-shaped + capture-reasoning) is the primary lever; the
**diagnostic-surfacing** and **weaker-model-reuse** threads are *live, not parked*; only
emergent-synthesis/graph-expansion retrieval scaffolds are PARK (scoped to exactly what 72/73 measured).

## Design space — the angles to generate across

| # | angle | side | mechanism (where it changes) | ask-goal served |
|---|---|---|---|---|
| 1 | **question-shaped `situation` handles** | write | add learn Step-2's "phrase as a future task" rule to recall Step 2.5; a question-template; retroactive `engram resituate` (the command exists — rewrites situation + re-embeds) on existing topic-shaped notes | note quality · fewer passes |
| 2 | **persist reasoned conclusions (Step-4 as norm)** | write | make recall Step 4 fire reliably; capture the problem→diagnosis→fix *chain*, not just a fact; audit how many `source: synthesis` notes actually exist | reason-from-memory · fewer passes |
| 3 | **write-time quality gate** | write | agent self-check before `engram learn` — validates angle-1's output (question-shaped? distinct-actionable, not vague/narrow?); the audit rubric as a write gate (skill prose; `engram check` is structural-only) | note quality |
| 4 | **diagnostic / abductive note shaping** | write | structure notes for *understanding problems*: "when [symptom] appears → likely [cause]" phrasing so diagnostic situations match; optional additive `inference_mode` frontmatter field (note 46: additive, no schema bump) | **understand problems** |
| 5 | **coverage capture from failure-mining** | write | feed the 68%-uncovered mined lessons through angles 1–4 as question-shaped `engram learn` calls | note quality · coverage |
| 6 | **abstraction pairing** | write | crystallize the generalizable principle *with* a concrete instance, so the note matches abstract *and* concrete queries | note quality |
| 7 | **weaker-model reuse** | write | persist conclusions a *cheaper* model can recall instead of re-deriving; measure opus→sonnet viability on persisted-conclusion tasks (note 76 value 3) | **fewer passes** · cost |
| 8 | **diagnostic-surfacing adequacy check** | retrieval (measure) | probe whether the right diagnostic note actually surfaces for symptom→cause queries (untested — 72/73 didn't measure it); propose a fix *only if* a gap shows | **understand problems** (surfacing half) |
| PARK | graph-expanded / synthesis-at-retrieval for **emergent synthesis** | retrieval | A/B-gated only (72/73 measured ~0 headroom for this case) | — |

**Ask → angle traceability:** *quality of notes* → 1,3,5,6 · *situations we want them surfaced in* → 1,4
(worth-surfacing) + 8 (surfacing) · *reason from memories* → 2 · *better understanding problems* → 4 + 8 ·
*fewer passes* → 2,7 (+1) · *consider the reasoning research* → the scoped spine (72/73/74/76).

## Pipeline (Step 4)
- [ ] **Generate** — workflow: one designer agent per angle (1–8) produces a concrete proposal (mechanism,
      where it changes the skill/binary, how it's evaluated, the failure/goal it serves). Angle 8 is a
      *measurement-first* proposal (probe diagnostic surfacing; fix only if a gap shows). A separate designer
      states the scoped PARK direction (emergent-synthesis/graph retrieval) + the exact A/B that would justify
      un-parking it.
- [ ] **Judge** — a panel scores each proposal against: (a) which ask-goal it serves + how well (quality /
      reason-from-memory / understand-problems / fewer-passes); (b) does it respect the *scoped* prior evidence
      (no emergent-synthesis/graph scaffold without an A/B; diagnostic surfacing is fair game); (c)
      cost/risk/effort; (d) cheaply evaluable? Flag CONTENDER vs PARK.
- [ ] **Synthesize** — collapse near-duplicates, surface the diverse set (all angles + the three ask-goals
      represented, honest CONTENDER/PARK per the deliver-full-diverse-set rule), each with how to evaluate it.

## Deliverable
`docs/design/2026-06-28-question-shaped-crystallization-proposals.md` — the evaluable proposal set, table-led:
each proposal = {name, mechanism, where it changes, what it costs/risks, how to evaluate, CONTENDER/PARK,
which goal it serves (quality / reason-from-memory / fewer-passes)} + a recommended first one + the parked
retrieval-scaffolds with their A/B gate. **Propose, don't pick** — deliver the full diverse set; recommend
within it.

## Out of scope
Implementing any proposal (this run only proposes + evaluates-on-paper); building retrieval scaffolds; the
multi-session laddering A/B for persist-synthesis (a proposal *names* it, doesn't run it).

## Risks
- **Over-collapsing to one idea** — must deliver all 8 angles + the PARK as distinct, independently-evaluable
  options with honest CONTENDER/PARK ratings; the recommendation is a *position inside* the full set, never
  achieved by pruning alternatives (note 35).
- **Over-narrowing the ask** (the caught Gate-A failure) — keep the goal-A (understand-problems) thread
  (angles 4, 8) and weaker-model-reuse (angle 7) first-class; do NOT let the scoped 72/73 evidence silently
  swallow diagnostic surfacing, which it never measured.
- **Re-proposing a parked lever as fresh** (note 82) — the PARK is *scoped* to emergent-synthesis/graph
  retrieval + carries the A/B gate; diagnostic surfacing is measured, not assumed.
- **Hygiene:** note 65 (`--synthesize-l2`) is stale vs the shipped `--lazy-chunks` — fix in the closing learn.
