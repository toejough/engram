# R26 acceptance walk: does the sketch support the workflow?

Date: 2026-05-23. Companion artifact to the research log.

The premise (from Synthesis 2026-05-23c): the research log we're
building *is* a working prototype of the memory system we're
designing. If the L0/L1/L2 sketch can't support the six
operations the workflow uses, that's a strong signal we're wrong.

This walk takes each operation, asks "what would the sketch do
here?", and surfaces what works vs. what's a gap. Severity:
**low** = mechanism missing but easy to add; **medium** = needs
real design; **high** = sketch is structurally incomplete.

---

## Op 1 — Query-driven retrieval

**Workflow example.** "What did I learn yesterday about CLS?"
parses to: facts/feedback × temporal. "What about CLS?" parses
to: facts/feedback × topic. "We're about to refactor auth
middleware" parses to: feedback × situation.

**What works.**

- The 2×3 query × kind matrix from R15 maps cleanly to indexes:
  episodes get temporal + entity indexing; facts get semantic
  embedding; feedback gets situation embedding.
- The cascade pattern (L2 first, descend on miss) from the
  current spec still works.

**Gaps.**

- **Query router is unspecified.** Severity: medium. Who decides
  "this is topic × facts" vs. "this is situation × feedback"? An
  LLM intent-parsing step? A query DSL? The current `engram
  recall` takes a raw string and runs the cascade — no kind
  routing exists.
- **Multi-kind fan-out.** Severity: medium. "What have we
  learned about X" wants both facts and feedback. The sketch is
  silent on whether we fan out, and how.
- **Result fusion across kinds.** Severity: medium. Episodes
  rank by recency × relevance; facts by semantic similarity;
  feedback by situation match. Combining into one ranked output
  needs a fusion rule beyond RRF (kinds have different
  qualitative signals, not just different scores).
- **Reconstructive synthesis (R22).** Severity: medium. The
  workflow shows that responses are synthesized from atoms, not
  fetched — but R22's mechanism is still TBD.

---

## Op 2 — Write/extraction without query

**Workflow example.** During this design conversation, I emit a
synthesis note (consolidation), spawn an R entry (a new question
crystallized), and refine an existing R entry (R24). None of
these were triggered by a query.

**What works.**

- L1 emission at task boundaries is well-defined in the current
  spec.
- The change-log + synthesis-note pattern in the research log
  shows that write-without-query is already happening
  organically — what's missing is the formal hook.

**Gaps.**

- **Salience gate for L1 emission (R19).** Severity: medium.
  Mechanism unspecified. Without it, L1 grows unboundedly; with
  it, some sessions silently fail to emit.
- **L1→L2 analysis trigger (R23).** Severity: medium.
  Hybrid hypothesis on the table but no concrete spec.
- **Feedback-from-behavior.** Severity: medium. The user
  explicitly *captures* meta-lessons (e.g., "don't claim CLS
  authorization for things outside its scope"). If the agent
  doesn't notice the pattern, it silently disappears. The
  current sketch has no automatic feedback-recognition step.
- **The R24 refinement** ("procedural → situation-cued") was a
  write triggered by *reasoning*, not by a task closing. The
  sketch's task-close trigger doesn't cover this. Either
  reasoning-driven refinements need a new trigger, or the agent
  treats every meaningful realization as a "task close" — both
  workable, neither specified.

---

## Op 3 — Consolidation without query (R21)

**Workflow example.** When R15 and R20 were reframed together
(Synthesis 2026-05-22), neither was a fresh question — we
consolidated across two existing entries. The R24 refinement
also has a consolidation flavor (cross-entry pattern
recognition: "CLS doesn't authorize procedural" applies in
multiple contexts).

**What works.**

- Append-only at L0–L2 means consolidation is non-destructive.
- The current spec's `engram synthesize` pattern is a starting
  point.

**Gaps.**

- **What does consolidation *do* at each tier?** Severity: high.
  The sketch is mostly silent. Candidate actions:
  - L0/L1: probably nothing (substrate is raw).
  - L2 facts: merge near-duplicates (already in current spec);
    update confidence (R16); surface contradictions; promote
    common-thread facts upward.
  - L2 feedback: re-evaluate situation embeddings as new
    situations accumulate.
  - L3 (if it exists, R25): regenerate.
- **Trigger choice.** Severity: medium. Idle / Stop hook / cron
  / user-invoked. The current spec offers `engram synthesize`
  as user-invoked + Stop hook; R21 hypothesizes a maintenance
  pass; nothing decided.
- **Cost model.** Severity: medium. Consolidation requires LLM
  calls over many entries. Cost = (entries touched) × (LLM
  call cost). Without budget caps, this can dominate.

---

## Op 4 — Proactive surfacing

**Workflow example.** When I started replying about CLS just
now, I *should* have surfaced "remember, don't over-claim CLS
authorization" — a feedback item that already exists in the log.
I didn't, because there's no mechanism for "this looks like a
situation where prior feedback applies."

**What works.**

- R24's situation-cued retrieval pipeline supports this in
  principle: feedback indexed by situation embedding; query the
  index with current-situation embedding.

**Gaps.**

- **No "situation start" event.** Severity: **high**. R24's
  proactive surfacing requires a hook the sketch doesn't have.
  The skill currently fires only on explicit `/learn` or
  `/recall`. Candidate hook: SessionStart, PreToolUse on certain
  tools, or a per-turn lightweight probe.
- **"Situation" is fuzzy.** Severity: medium. Is a situation a
  task? A turn? A tool call sequence? A change of topic? Needs
  definition before situation embedding is well-formed.
- **No engram API for "what feedback applies?"** Severity: low.
  Would be a small extension to `engram recall`.
- **Cost per situation.** Severity: medium. If every turn fires
  a situation probe, that's a lot of recalls. Need a cheap
  proxy or a coarser trigger.

**This is the biggest structural gap in the sketch.** R24's
hypothesis depends on a hook that doesn't exist.

---

## Op 5 — Refinement of stored items (R16)

**Workflow example.** R24 was refined from "feedback is
procedural" to "feedback is situation-cued semantic" — a content
change driven by reasoning, not by usage signal.

**What works.**

- Append-only with `supersedes:` covers content refinement
  cleanly: write a new entry with `supersedes: <old-uuid>`. The
  old entry stays for history; retrieval follows the redirect.
- The change log naturally records the refinement event.

**Gaps.**

- **R16 functional form unspecified.** Severity: medium. Decay
  function (Ebbinghaus exponential? Power law? ACT-R
  activation?), strength updates on retrieval, demotion
  threshold — all open.
- **Reasoning-driven refinement has no formal trigger.**
  Severity: medium. The R24 refinement happened because we
  *agreed* to refine. Without a discipline / hook, the agent
  has to notice and act. The skill could enforce this ("on
  every retrieval, ask: is this still accurate?") but that's a
  per-recall cost.
- **Distinction between use-based decay and reason-based
  supersession.** Severity: low. These are different mechanisms
  but both update confidence. The sketch should treat them as
  one signal feeding into a single "current confidence" field.

---

## Op 6 — Spawning new items from stored items

**Workflow example.** R2's lit summary spawned R15–R22. R15
spawned R23–R25. The meta-observation about the log spawned R26.
Each spawn is a cheap forward reference (a new entry citing the
parent).

**What works.**

- Wikilinks + UUID resolver from the current spec make cheap
  references possible.
- Edge types from R18 (temporal, causal, contradicts,
  supersedes) cover most relationships.

**Gaps.**

- **No "spawned-from" / "motivated-by" edge.** Severity: low.
  The change log captures spawn relationships in prose, not as
  structured edges. R18 should add `derives-from:` or
  `motivated-by:`.
- **The R# entries themselves don't fit the L2 split cleanly.**
  Severity: **high** (or: a real finding about scope). An R
  entry is:
  - A *question* (no analog in episodes/facts/feedback).
  - Plus a current *hypothesis* (fact-shaped).
  - Plus *priority/status* metadata (no analog).
  - Plus *dependencies* (edges).
  - Plus *resolution criteria* (no analog).

  Two interpretations:

  - **(a) Open questions are a fourth L2 kind.** Add
    `questions` alongside episodes/facts/feedback. Has its own
    indexing (status × priority × deps × resolution criteria).
    Risk: bespoke schema, more retrieval pipelines.
  - **(b) Open questions are facts with low confidence + a
    "needs work" flag.** Reuses R16's confidence field. The
    hypothesis IS a fact; the question is just "this fact has
    low confidence and we should work on it." Status/priority
    become tags. Resolution criteria becomes a sibling fact:
    "this question resolves when ...".

  (b) is simpler and reuses existing machinery. (a) is more
  explicit but more substrate. **My lean: (b).** Open questions
  are low-confidence facts the system has flagged for active
  work. R16's confidence + a `status: open` tag is enough.

---

## Cross-cutting findings

1. **Open-questions/active-situations** — the L2 split is OK if
   we adopt interpretation (b) above: questions are
   low-confidence facts with an open-work flag. R16 needs to
   support this; no new L2 kind required.
2. **Proactive surfacing has no hook** — biggest single
   structural gap. R24's hypothesis depends on a "situation
   start" event the sketch doesn't define.
3. **Query routing + multi-kind fusion** — three pipelines need
   a router and a fusion rule. R15 needs to extend.
4. **Consolidation actions per tier** — R21 needs concrete
   per-tier action lists, not just "maintenance pass."
5. **R16 remains the largest single unspecified mechanism** —
   functional form, triggers, distinction between use-based and
   reason-based updates.
6. **The sketch handles writes-without-query, refinement,
   spawning, and query-driven retrieval reasonably well.** The
   weakest area is proactive (Op 4) and the structural
   ambiguity around questions (Op 6).

---

## Verdict

The sketch is **viable but incomplete**. No fatal flaws — it
supports the workflow in principle — but five mechanisms need
sharpening before it's buildable:

1. **R16** — confidence/decay/supersedes mechanism, including
   support for low-confidence "open work" items.
2. **R19** — salience gate at L0/L1.
3. **R21** — per-tier consolidation actions + trigger.
4. **R22** — reconstructive retrieval mechanism.
5. **R23** — L1→L2 analysis trigger.

And one **new R** to add:

- **R27** — situation-start hook + situation definition (the
  R24 prerequisite that the sketch doesn't have).

Recommended order: R16 → R27 → R21 → R19/R22/R23 in parallel.
R16 first because it underlies refinement, open-questions, and
consolidation. R27 next because it's the biggest structural gap
and unblocks R24. The rest are independent mechanism specs.

---

## What this walk *didn't* test

- **Scale.** The research log is ~30 R entries plus a handful of
  synthesis notes. The real vault is 100× larger. Some gaps
  might be invisible at this scale (e.g., consolidation cost).
- **Adversarial cases.** Contradicting feedback, identical
  episodes from different sessions, situations that match
  multiple feedback items strongly. We've assumed cooperative
  data.
- **Cross-session continuity.** The log is one conversation. A
  multi-session workflow (resume after a week) would exercise
  L0 retrieval and feedback re-surfacing differently.

These are out of scope for this walk; flag for later validation.
