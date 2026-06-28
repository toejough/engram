# Mine main + subagent failures as eval material

> **For agentic workers:** RESEARCH/ANALYSIS plan. The "tests" are detector-recall measurement + a
> realism check + adversarial review, not unit tests. Steps use `- [ ]` checkboxes.

**The ask (Joe, 2026-06-28).** Get the list of places we could have done better — from **both main session
AND subagent transcripts** — and for each, find **where a memory (if it existed) would have been most
helpfully recalled to do better on first pass**. Cluster into patterns. Turn the result into **eval
material**: realistic trap-RED reproductions + candidate lessons. Executes
`docs/design/2026-06-27-mine-failures-as-eval-material.md`, reusing the method (but not the regex detector)
from `docs/design/2026-06-27-recall-trigger-patterns-and-proposals.md`.

## The deliverable IS the list + per-failure first-pass injection point

The product is: **(1) the failure list, and (2) for each, the single earliest decision point where a
recalled memory would have prevented it on first pass — and whether that point is one our process already
covers or a NEW candidate moment.** The *uncovered* injection points are the improvement findings (the
whole point: do better than current). The detector is the **quality gatekeeper** that makes the list
trustworthy — not the product.

## Detector philosophy (Joe's two steers — the spine)

1. **Semantic, not word-match.** Users (and subagents) signal failure in subtle, conversational ways with
   **no signal word at all** ("hmm, what if we tried it the other way?", "let's step back", a skeptical
   clarifying question, polite redirection, quiet disappointment). The detector **reads each turn in
   context and judges semantically** — "is this a correction / redirection / dissatisfaction with what was
   just done?" (main) or "did this go wrong / error / get walked back?" (subagent). **No regex pre-filter**
   — a pre-filter silently drops exactly the word-less corrections. (Consequence: the recall-trigger
   analysis used a *word-match* extractor, so its correction corpus was **incomplete and skewed toward
   blunt corrections**; re-detect semantically here and flag that limitation when comparing.)
2. **Over-match, then prune.** Tune semantic judgment for **recall**: flag anything that plausibly reads
   as a failure; the closer classification pass prunes false positives to `NOT-A-FAILURE`. A missed real
   failure is invisible and lost; a false positive is cheap to prune. **We do NOT gate on false-positive
   rate.** The metric that matters is the **false-NEGATIVE rate** (§Phase 0).

## Survivorship rubric — failed vs not-chosen (operational)

A discarded/not-chosen subagent result is **NOT a failure**. Decide:
- **FAILED** — wrong, errored (non-zero exit / test FAIL / "command not found"), corrected, or required
  **>~20% rework** by the parent; a test written that passes in the failure state; a claim later
  contradicted/undone.
- **NOT-CHOSEN** — error-free and achieved the stated objective; would pass review without substantive
  rework; the parent simply picked a different valid path or merged a stylistic cleanup.
- Examples: *subagent wrote a test green in the failure state* → **FAILED**; *subagent chose approach A,
  parent used B but A worked* → **NOT-CHOSEN**; *subagent guessed an import path that didn't exist* →
  **FAILED**. **Default a genuine borderline to NOT-A-FAILURE** (over-match is about casting wide on
  *candidates*; the prune is conservative on *confirmation*).

## Injection-point — "most helpfully, on first pass" (OPEN — not limited to existing recall sites)

**This investigation surfaces candidates for doing BETTER than our current process — so the injection-
point analysis is UNCONSTRAINED.** For each failure, find the observable decision cue where a recalled
memory would have most helpfully prevented it on first pass, **regardless of whether our current process
recalls there.** Do NOT limit the answer to our existing memory-pull moments (start-of-task recall, the
subagent's recall-first, the parent brief) — that would make the investigation circular, only
rediscovering where we already look. **The highest-value findings are injection points our process does
NOT cover today** — the candidate *new* recall moments.

Per confirmed failure, record:
- **`first_pass_trigger`** — the single earliest **observable decision cue** at which a recalled memory
  would have prevented the failure **without rework**, observable AT DECISION TIME (never hindsight) and
  at **ANY** decision point (e.g. "before asserting an API/path/format exists", "at the design-approach
  choice", "before the destructive command", "before declaring done", "mid-synthesis before proposing a
  lever", "at task-init"). Where several exist, surface the earliest/clearest; note alternatives.
- **`coverage`** — does our current process recall/brief at that cue? `covered` (an existing recall site)
  | `uncovered` (a NEW candidate moment — the high-value "do better" finding) | `partial`. **The
  `uncovered` set is a primary output, not a side note.**
- **`klass`** = *reachability* (note 82), relative to the IDEAL injection: `TRIGGER` (a memory existed/
  should and firing at the cue prevents it) | `CAPTURE` (no memory existed — the ceiling) | `APPLICATION`
  (memory present but ignored). Classified **independently** of where the cue is or whether it's covered.
- **`ideal_locus`** (descriptive, not constraining) — would the ideal recall fire in the subagent's own
  recall, the parent's brief, a main-session recall, or a **new moment** none of those currently covers?

## Corpus + strata (explicit)

- **Main user-correction transcripts (semantic re-detect):** engram main sessions
  (`~/.claude/projects/-Users-joe-repos-personal-engram/*.jsonl`, top-level) + restored cross-repo main
  sessions (`~/restic-restore-claude/Users/joe/.claude/projects/-Users-joe-repos-personal-{imptest,glowsync,targ,traced}/*.jsonl`).
- **Subagent transcripts (~1,420):** engram `~/.claude/projects/-Users-joe-repos-personal-engram/**/subagents/*.jsonl`
  (~1,022) + restored `~/restic-restore-claude/.../{targ 239, traced 111, imptest 37, glowsync 11}/**/subagents/*.jsonl`
  (398). **Correct count: ~1,420 subagent transcripts** (the design doc's "~1,060" was engram-only).
- **Strata:** by **repo** (engram, targ, traced, imptest, glowsync) × **role** (main-session vs subagent).
  **Sampling:** *main sessions exhaustive* (smaller set); *subagents stratified — proportional to count,
  min 20 per repo-stratum, target ~250 subagent transcripts for Phase 1* (note 86 — a representative
  sample sizes the patterns + the false-negative rate; the full 1,420 sweep is not needed to answer the
  ask, and we label honestly that the subagent list is a representative sample, not exhaustive).

## Pipeline (Step 4)

### Phase 0 — Prove the detector's RECALL (not its precision)
- [ ] Pin the strata + sample sizes above.
- [ ] On a calibration sample, run the **semantic detect+classify** prompt (below). Then run the
      **false-negative check**: a *separate* agent pass over a sample of windows where detection flagged
      **nothing**, asking "did this window contain a correction/redirection/failure the detector missed?"
      Report the **miss rate (DERIVED)**. Gate: if the detector misses real failures, broaden the prompt
      and re-check. (We do NOT gate on false-positives — those are pruned in classification.)

### Phase 1 — Mine failure moments (semantic, high-recall, fan-out)
- [ ] **Detect+classify** over the sampled corpus. Each agent reads a transcript window **in full**
      (assistant text + `tool_result` content + consecutive-turn deltas) and, high-recall, emits EVERY
      plausible failure as a record: `{summary, source, role(main|subagent), repo, signal_cue,
      first_pass_trigger, coverage(covered|uncovered|partial), ideal_locus(subagent-recall|parent-brief|
      main|new-moment), klass(TRIGGER|CAPTURE|APPLICATION|NOT-A-FAILURE), memory_could_help(y|maybe|n),
      lesson, is_real_failure_confidence}`.
- [ ] **Prune:** drop `NOT-A-FAILURE` and low-confidence borderlines (per the rubric).
- [ ] **Parent-discard signal: scoped OUT by default** — the subagent output payload is a transient
      `/tmp` file (gone), and the parent JSONL holds only a summary string, so "parent discarded the
      result" can't be detected from payloads (only file-edit-overlap inference via the `meta.json`
      dispatch link, which is error-prone). Default: rely on **in-transcript** subagent cues only;
      document this exclusion in the deliverable. (Fallback only if cheap: timestamp+edit-overlap,
      flagged `confidence=LOW`.)

### Phase 2 — Patterns
- [ ] Cluster confirmed failures by `signal_category`; report the distribution + the
      **TRIGGER/CAPTURE/APPLICATION split AND the `memory_could_help` y/maybe/n ceiling** (what fraction
      is un-addressable by any memory).
- [ ] **Report the `coverage` split — the headline improvement finding:** what fraction of failures point
      at an **uncovered** injection cue (a moment our current process does not recall at)? Cluster the
      uncovered cues into **candidate new recall moments** (e.g. before-external-assert, at-design-choice,
      before-declaring-done) — these are the "do better than current" deliverable.
- [ ] *Optional (researcher-added, label as such):* a light comparison to the recall-trigger
      user-correction distribution — same patterns at higher volume vs genuinely new (subagent-specific:
      thin parent brief, subagent recall not firing). Not part of the core ask; include only if cheap.

### Phase 3 — Eval material (with the behavioral/tactical honesty)
- [ ] For each generalizable confirmed failure: a **trap-RED candidate** — `{failing situation,
      observable prior cue, the memory that would prevent it, the RED reproduction, the pass condition
      (memory/discipline prevents it; NOT retrieval-vs-synthesis-conflated — note 83)}` + a **candidate
      lesson**.
- [ ] **Flag each trap tactical vs behavioral.** Per trap-gate RESULTS.md (behavioral traps 0/5 cold,
      tactical 5/5) + the behavioral-traps-need-context note: **discipline/behavioral failures (verify-
      don't-guess, design-direction, step-boundary) do NOT reproduce in cheap isolated fixtures** — they
      need rich session context. Mark each trap candidate **tactical (context-free, cheap to eval)** vs
      **behavioral (needs a rich-context harness; flag as expensive / currently un-evalable)**. Realistic
      reproduction per note 85 (stub-scale for retrieval traps; no toy that a clean model passes).
- [ ] **Annotate** which capability axis each trap would extend (C3–C6 exist; "C7" would be new) — do NOT
      claim it "slots into" the harness (a new axis needs a seeder + scorer + driver + gate row; out of
      scope here).

## Deliverable
`docs/design/2026-06-28-failure-eval-material.md`, tables leading each section:
- **(0)** the detector + its **measured false-negative (miss) rate** (the headline honesty number).
- **(1)** the failure list (main + subagent), each with `first_pass_trigger`, `coverage`, `klass`.
- **(2)** patterns + the **`coverage` split (candidate new recall moments — the headline "do better"
  output)** + the TRIGGER/CAPTURE/APPLICATION split + the `memory_could_help` ceiling.
- **(3)** eval material: trap-RED candidates + lessons, **each flagged tactical vs behavioral** + the
  axis it would extend.
Commit the detector prompt + classified moments as the data trail (`2026-06-28-failure-eval-data/`).

## Out of scope (this run = analysis + eval material to EVALUATE, not an implementation)
- Building the detector into engram, or wiring any new trap into `dev/eval/traps/` (we produce specs +
  lessons, not committed traps/code). The prune-durability fix (#659) and the recall-trigger proposals.

## Risks
- **Detector recall is the load-bearing unknown** — Phase 0 must measure the *miss* rate honestly; a high
  miss rate is a finding (broaden, don't ship a list with a blind spot).
- **Parent-discard is structurally hard** (payload gone) — scoped out by default; the deliverable says so.
- **Behavioral traps may be un-evalable cheaply** — flagged per trap, not hidden.
- **Semantic detection is agent-work** — bounded by the sampling above; the subagent list is a labeled
  representative sample, not exhaustive.
- **Same-patterns outcome** — if subagents just reproduce the known patterns, the value is volume/eval-
  material, not new insight; state that plainly.
