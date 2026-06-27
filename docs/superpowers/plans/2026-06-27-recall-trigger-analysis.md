# Recall-trigger analysis — when should recall fire, cheaply?

> **For agentic workers:** this is a RESEARCH/ANALYSIS plan, not a code plan. The "tests" are
> evidence thresholds and adversarial review, not unit tests. Steps use `- [ ]` checkboxes.

**The research question (open inquiry — do NOT pre-judge the answer):** Across past correction/failure
moments, *when* should some form of recall have fired to prevent or smooth the failure? Is it as simple
as **"recall before every tool call"**, or do the moments concentrate on other observable signals — and
does any candidate trigger stay within the over-fire budget? Produce **10 concrete proposals to
evaluate**, spanning the PROBLEM space (patterns for when to fire) and SOLUTION space (cheap/fast ways
to trigger on most patterns). The evidence drives the conclusion; if it shows trigger-timing is *not* a
high-leverage lever, that is the finding to report.

**Scope note — triggering vs body-shrink (Gate A docsA-2).** This analysis is about **WHEN** recall
fires (and at what intensity), not **HOW** the ~287-line skill body is restructured. It *informs* the
roadmap's "shrink the procedure / two-speed recall" lever — by identifying which moments warrant a full
recall vs a cheap quick-recall vs none — but the deliverable here is the trigger analysis + proposals,
not the body-split itself. **"Procedure" = the recall skill body run on every fire.**

**Decision recorded (Gate A docsA-1):** the roadmap marks `payload-prune-after-Step-3` as ← NEXT; Joe
explicitly chose the **procedure** lever this turn instead. Authorized; noted, not reordered.

**Prior context (one worked instance — context, NOT a conclusion to confirm; Gate A askA-1).** GH
#654–658 + notes 80/81/82 came from a *single* failure: in a `/please` cost run the agent re-proposed a
lever (haiku recall tier) already built, measured −14%, and rolled back that same day; the disproof was
in-context and crystallized (note 80), yet it shipped. That gives one candidate pattern
(self-generated-recommendation / "recall-before-recommend") with a named fix space and a specced C7
eval. Treat it as **one data point to confirm or refute against the full corpus** — not the frame.

## Core methodological commitments (non-negotiable)

1. **Classify every moment: TRIGGER | CAPTURE | (APPLICATION — flag only).**
   - **TRIGGER** — a helpful memory existed (or plausibly would), and recalling it *at an observable
     prior signal* would have prevented/smoothed the failure. **Only this class is addressable by the
     trigger lever** and counts toward coverage.
   - **CAPTURE** — no memory existed yet (novel). Fixed by writing notes, not firing recall.
   - **APPLICATION** — memory *was* surfaced but ignored/misapplied. **Flag in passing, do not
     enumerate exhaustively** (Gate A askA-3 — beyond the ask). Note 82 warns misses are often
     *structural* (no note / wrong-shaped / note outranked by chunks), so report the TRIGGER share
     honestly: it **bounds** what triggering can deliver, and may be small.

2. **The trigger signal must be OBSERVABLE AT DECISION TIME — never hindsight.** For each moment, name
   the cue visible *before* the failure. **Extract `signal_category` during Phase 1** (Gate A clarityA-2),
   one of: `ask-keyed` (the incoming user ask), `tool-imminent` (a specific tool/command about to run,
   e.g. git push, file delete), `task-type` (e.g. "editing a SKILL.md", "writing an eval"),
   `step-boundary` (a workflow phase transition), `self-recommendation` (a lever/claim the agent invents
   during synthesis — the #654 pattern), `file-type` (path/extension), `phrase-intent` (a phrase in the
   ask signalling intent). Mark ambiguous ones for Phase 2 review.

3. **Precision needs TWO rates — the negative-evidence requirement (Gate A clarityA-3,5).** Phase 2,
   per pattern, in this order: (a) cluster moments by `signal_category`; (b) identify the concrete
   trigger signal from the cluster; (c) **sample the denominator** for that signal from raw logs (e.g.
   total `git push`es, total self-recommendations); (d) compute **over-fire ratio = fires ÷ needed-fires**;
   (e) flag against the bound. Joe's bound: ≤ ~1.1× fine, ~10× not. Report over-fire as a range
   **[min–max] with a nominal (midpoint)**; apply the bound to the **nominal**; if nominal ≤ 1.5× but the
   range reaches ≥ 10×, mark **high-variance — sample deeper before rating**.

4. **Corpus strategy (raw logs are ~99.9% engram — measured: 374 MB vs ~400 KB non-engram).**
   - **Tier A — vault feedback notes (EXHAUSTIVE).** **Recount first:**
     `grep -l '^type: feedback' <vault>/*.md | wc -l` (**52** at plan time; Gate A codeA-1). Each is a
     crystallized correction whose `situation:` field *is* a distilled trigger signal + the lesson —
     the cleanest counterfactual (memory provably exists) and the cross-domain breadth.
   - **Tier B — raw engram session logs (SAMPLED).** Sample the **18 top-level engram sessions + their
     largest subagent transcripts**; per sampled session extract **all** correction/failure moments (no
     per-session cap). Target the "long list" (~expect 100s of moments) and use these for denominators.
   - **Tier C — non-engram logs (SPOT-CHECK, ~400 KB total).** Check 3–5 for any cross-domain pattern
     the engram corpus misses. Report the engram skew honestly — cross-project generality is weak.

5. **Cost anchor (Gate A docsA-4).** Per `recall-cost-isolation.md` (2026-06-25), recall is **~190s**,
   Step-2 paging dominant (~43–63%), Step-0/1 ~17%, Step-2.5 ~9–33% — *not* the old ~350s. So a full
   fire is expensive; weight patterns toward moments that genuinely need full synthesis vs those a
   cheap quick-recall (or none) would cover. This is the bridge to the two-speed split.

6. **Map findings to prior art + a test SKETCH.** Relate each pattern/proposal to #654–658 + notes
   80/81/82 where applicable. Because Joe asked for "reasonable tests," each proposal carries a
   **one-line test sketch** (what RED reproduction would look like); full C7-style RED→GREEN detail only
   for CONTENDER-rated proposals (Gate A askA-4).

## Pipeline (Step 4 execution)

### Phase 1 — Mine moments (fan-out)
- [ ] Tier A: recount feedback notes, then one exhaustive pass → structured moments.
- [ ] Tier B: fan out over the sampled engram sessions → all correction/failure moments.
- [ ] Tier C: spot-check 3–5 non-engram sessions.
- [ ] **Barrier:** dedup + merge into one moment table. Per-moment schema: `{summary, preceding_signal,
      signal_category, class(TRIGGER|CAPTURE|APPLICATION), memory_existed(y/n/maybe), source}`.

### Phase 2 — Pattern taxonomy + precision + the headline verdict
- [ ] Cluster TRIGGER-class moments by `signal_category` → candidate patterns.
- [ ] Per pattern, run commitment-3 steps (a)–(e): signal → denominator → over-fire range → bound flag.
- [ ] **REQUIRED — Headline hypothesis verdict (Gate A askA-2):** state explicitly whether
      **"recall before every tool call"** passes the ~1.1× bound, with the estimated over-fire ratio. This
      is the deliverable's section (0).
- [ ] **Phase 2c — Completeness critic:** audit for missing `signal_category` classes (list in
      commitment 2); for any absent, note it and decide whether Phase 3 needs a proposal targeting it.

### Phase 3 — 10 proposals (problem × solution)
- [ ] For each of 10: `{pattern targeted, trigger signal, expected coverage (% of TRIGGER moments),
      over-fire [range]+nominal, solution surface (skill-body | frontmatter description | split-skill |
      CLAUDE.md pointer | binary | please-gate), trigger cost (latency/tokens of the trigger itself),
      test sketch (one line), relation to #654–658, rating}`.
- [ ] **CONTENDER/PARK criteria (Gate A clarityA-4):** **CONTENDER** = over-fire nominal ≤ ~10×
      AND coverage ≥ 50% of its target pattern AND the trigger itself is cheap (no extra LLM round-trip,
      or a sub-second/sub-1k-token check). **PARK** = fails any of those, or is high-variance. Deliver all
      10 with honest ratings; recommend within the set, never by deleting members.

## Deliverable
`docs/design/2026-06-27-recall-trigger-patterns-and-proposals.md`:
- **(0) Headline verdict** on "recall before every tool call?" + the TRIGGER/CAPTURE/APPLICATION split
  (the lever's ceiling) — both up front as the negative evidence.
- **(1)** the moment list (long, structured).
- **(2)** the pattern taxonomy: coverage + over-fire [range] per pattern, bound flag.
- **(3)** the 10 proposals as above, with CONTENDER/PARK + recommendation.
Each section leads with a labeled table.

## Out of scope (settled this session)
- async/background relocation (note 108 — not a real-axis lever).
- payload-size cuts (shipped: lazy-chunks, recent-fill — time/paging, not this lever).
- BUILDING any proposal — this run produces analysis + 10 proposals to EVALUATE, not an implementation.

## Risks
- **Over-fire denominators are sampled** — report ranges + nominal, not false precision; flag high-variance.
- **Survivorship bias:** feedback notes are corrections we *caught*; raw-log mining offsets but can't erase it.
- **Counterfactual softness:** "a memory would have helped" is a judgment — anchor to TRIGGER-class
  (memory plausibly exists) and mark maybes.
- **Corpus skew:** ~99.9% engram — cross-project claims are weak; say so.
