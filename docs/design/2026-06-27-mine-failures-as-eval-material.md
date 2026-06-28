# Mine failures as eval material — subagent *and* user, one pipeline

> **EXECUTED 2026-06-28** → `2026-06-28-failure-eval-material.md`. The semantic detector replaced the
> word-match plan below (Joe's steer: catch subtle corrections, not "no" word-matches). 137 confirmed
> failures from a 40-transcript sample; headline = the **candidate new recall moments** (uncovered cues),
> not the lessons. This note is kept for the original framing + the mechanism rationale.

> **Direction note (Joe, 2026-06-27).** Not a plan yet — the idea + the mechanism, so it isn't lost.
> Born from the cross-repo backup-recovery work: while deciding whether to drop subagent transcripts as
> noise, Joe reframed them as signal — *"anything a subagent did that failed is worth collecting and
> putting through the same eval as explicit user corrections."*

## The idea

Every transcript — **user sessions and subagent transcripts alike** — contains **failure moments**: a
turn where the agent did something wrong. Today we mine only **explicit user corrections** (the
recall-trigger analysis, `2026-06-27-recall-trigger-patterns-and-proposals.md`). That throws away the
larger, untapped source: **the agent's own subagents failing in isolation** — a subagent that guessed an
API, wrote a test that passed in the failure state, took a wrong design, or got its result discarded by
the parent. Those failures are recorded and then forgotten.

**Collect them and run them through the same eval/classification as user corrections.** The system then
learns from its own mistakes, not just the human's.

## Why this is now cheap to do

Two things we just established make this tractable:

1. **Granularity is already right (no re-chunking).** Chunking is **turn-grained** — a chunk ≈ *one user
   turn + the assistant's response to it*, anchored `turn-N` (`internal/chunk`, target ~500 chars / max
   1500). So a failure moment is **its own retrievable chunk**; you can pull and eval the failing turn
   directly, not a whole session.
2. **The transcripts are in memory.** `engram --auto` ingests subagent transcripts (Joe confirmed: keep
   them), and the **byte cap is now removed** (`ingestBudgetBytes = 0`, 2026-06-27) so even giant sessions
   are ingested whole — no truncated tails. The raw material is all there.

## The missing piece: a failure classifier

The chunker gives the *units*; what's missing is **a classifier that flags which turns are failures.**

- **User sessions** — the explicit correction signal we already mine (`no,` / `don't` / `why are you` /
  `that won't scale` …). Done.
- **Subagent transcripts** — failure has no human "no", so key on **observable in-transcript signals**:
  a `tool_result` error/non-zero exit; a self-correction turn ("that didn't work, let me…"); a later turn
  contradicting an earlier claim; a test going RED then being forced green; a gate-reviewer emitting
  *blocking/reversed*; the **parent discarding the subagent's result**. These are detectable the same way
  the cheap correction-signal extractor worked — pattern-match candidates, then agent-classify.

## Same pipeline as corrections — reuse, don't rebuild

Once flagged, a failure turn flows through the **existing** machinery from the recall-trigger work:

1. **Classify** trigger / capture / application + signal-category (was a memory available? would recall at
   an observable prior cue have prevented it?).
2. **Feed the eval** — each confirmed, generalizable failure becomes either (a) **trap-eval material**
   (reproduce the failure as a RED case; the eval passes only if memory/discipline now prevents it — the
   C-series / C7 anti-amnesia pattern), or (b) a **candidate lesson** to crystallize.

## Why it matters (connection to the trigger analysis)

The cross-repo correction analysis found the dominant real failures are **discipline/application** —
verify-don't-guess (41%), wrong design-direction (35%), skipped TDD-RED (15%) — failures to apply a rule
that *already exists*. **Subagents make the same mistakes**, at far higher volume (≈1,060 subagent
transcripts in the engram index alone). Mining them gives a **bigger, self-generated corpus** for exactly
the enforcement-gate + trap-eval work that analysis pointed to — and it compounds: every subagent run
becomes training data.

## Caveats / open questions

- **Detection precision is the hard part** (same as the recall-trigger proposals): "a subagent failed
  here" is harder to detect cheaply than a human "no". The observable-signal classifier needs its own
  precision pass before it's trusted; budget for a false-positive rate like the ~29% the user-correction
  detector showed.
- **Volume → cost.** Subagents are numerous; classification must stay cheap (signal-match → batch
  agent-classify), not a full-LLM pass per transcript.
- **Generalizability filter.** Many subagent failures are task-specific; the trigger/capture/application
  classification is what separates a reusable lesson from a one-off.
- **Survivorship still applies** — a discarded subagent result isn't always a *failure* (could be a valid
  path not taken); the classifier must distinguish "failed" from "not chosen".

## Status / next step

Direction captured. The natural first build is the **subagent failure-signal detector** (the analogue of
`extract_moments.py`, keyed to in-transcript failure signals) run over the now-ingested subagent corpus,
then route its hits through the existing correction-classification + trap-eval pipeline. Worth a `/please`
brainstorm to pin the detector's signals and precision target before building.
