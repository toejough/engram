# Question-shaped crystallization + reasoning-from-memory — proposal-generation plan

> **For agentic workers:** RESEARCH/DESIGN plan. The deliverable is a *diverse, evaluable set of proposed
> fixes* (CONTENDER/PARK + a recommendation), not an implementation. "Tests" are the prior-evidence
> constraints + adversarial review, not unit tests. Steps use `- [ ]`.

**The ask (Joe, 2026-06-28).** Propose some fixes to evaluate for *question-shaped crystallization*,
considering everything learned about note quality + the situations we want notes surfaced in + the reasoning
research — ideally so we can **reason from memories toward better problem-understanding and correct outcomes
in fewer passes.**

## The reframing the reasoning research forces (the spine)

The prior research already evaluated the obvious "make the system reason" mechanisms and rejected them — so
the proposals must be **write-side**, not retrieval-scaffold:

- **Notes 72/73 (measured):** retrieval-side reasoning machinery (graph expansion, multi-hop, a
  synthesis/compose step, reranking) has **no measured headroom** — engram's value is the *memory itself*
  (C6: warm 18/18 vs cold 1/9), and *opus already reasons excellently over what surfaces*. Default to NO
  reasoning/retrieval scaffold; any scaffold is **PARK**, admissible only behind a warm-vs-warm+scaffold A/B
  that shows real headroom (prior attempts found none).
- **Note 74 (the live lever):** *persisting validated synthesis* is the distinct, compounding **write-side**
  capability — capture the reasoned conclusion so a *future* session recalls it already-reasoned.
- **"Reason from memories, fewer passes"** is therefore served by **capturing the reasoning into memory**
  (write side) + surfacing it (the floor, shipped) — NOT by reasoning for the agent at retrieval time.
- **Note 120 / the audit:** cluster-driven notes are topic-shaped + ~half as question-useful; the fix is
  question-shaped `situation` handles. **Notes 76/68/69:** persist only *sound* conclusions, hedged by
  inference mode + linked to inputs, never aggregation/analogy-as-proof; gate against vault-rot.

## Design space — the write-side angles to generate across

1. **Question-shaped `situation` handles** — derive the handle from the *anticipated question/failure the
   note answers*, not the cluster topic (route cluster-driven candidates through the learn path's
   question-shaping; a question-template; a resituate pass).
2. **Capture reasoning, not just facts** — make persisting the *reasoned conclusion* (problem→diagnosis→fix
   chain; recall Step 4) the norm, and verify it actually fires (is Step 4 producing synthesis notes?).
3. **Crystallization quality gate at write time** — a self-check when a note is written (question-shaped?
   distinct-actionable, not vague-aggregation/narrow-impl?) that rejects/improves a topic-shaped note before
   it lands — the audit's rubric as a write-side gate.
4. **Inference-mode / diagnostic framing** — tag/shape notes by the reasoning they support (abductive
   symptom→likely-cause for *understanding problems*; deductive rule→consequence), so a diagnostic situation
   surfaces a diagnosis pattern (helps "understand the problem").
5. **Coverage capture** — feed the failure-mining pipeline (68% uncovered) into crystallization in
   question-shaped form, so the uncovered lessons get written.
6. **Abstraction pairing** — crystallize the generalizable principle *with* a concrete instance, so the note
   matches both abstract and concrete queries.
- **PARK (with the A/B gate):** graph-expanded / relational retrieval, synthesis-at-retrieval, multi-hop —
  per 72/73, only admissible if a warm-vs-warm+scaffold A/B shows headroom.

## Pipeline (Step 4)
- [ ] **Generate** — workflow: one designer agent per write-side angle (1–6) produces a concrete proposal
      (mechanism, where it changes the skill/binary, how it's evaluated, the failure mode it prevents) + a
      designer for the PARK direction (forced to state the A/B that would justify it).
- [ ] **Judge** — a panel scores each proposal against: (a) does it improve note quality / reasoning-from-
      memory-via-capture / fewer-passes; (b) does it respect the prior evidence (no retrieval scaffold without
      an A/B); (c) cost/risk/effort; (d) is it cheaply evaluable. Flag CONTENDER vs PARK.
- [ ] **Synthesize** — collapse near-duplicates, surface the diverse set (all angles represented, honest
      CONTENDER/PARK per the deliver-full-diverse-set rule), each with how to evaluate it.

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
- **Over-collapsing to one idea** — the deliver-full-diverse-set rule: give all angles, rate honestly.
- **Re-proposing a parked lever as fresh** (note 82) — the PARK section + the A/B gate guard this.
- **Proposals that scaffold reasoning** — reject unless they carry the warm-vs-warm+scaffold A/B (72/73).
