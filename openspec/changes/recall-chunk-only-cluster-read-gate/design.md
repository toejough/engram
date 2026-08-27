## Context

Issue #733: C5b (honoring a recency-channel standard under conflict) measured ~80% (8/10)
across two independent full-tier runs, against an originally-verified 100%. Root-cause
investigation (session 2026-08-26) read the preserved trial transcripts from the actual run
(`gate-C5-6l_mjvl1/`) and found the one failing trial (idx=3) never called `engram show-chunk`
on the load-bearing matched chunk, unlike all 4 passing trials — a recall-procedure compliance
gap, not a model capability ceiling. Full evidence: vault note 794 and this session's #733 GH
comment.

This repo already specs `agent-instructions/skills/recall/SKILL.md`'s own agent-facing
procedure as capability requirements — see `openspec/specs/recall-glance-deep-dial/spec.md`
("Glance SHALL execute Steps 0–3.5..."). That precedent makes a capability spec the right
mechanism for capturing this fix's target behavior, not just a SKILL.md diff description.

## Goals / Non-Goals

**Goals:**
- Specify the exact agent-behavior requirement precisely enough that a future
  `writing-skills` TDD cycle can implement it against a concrete RED baseline (the failing
  trial's own transcript).
- Name explicitly why this is a repo/skill change (rung 2, vault note 495's ladder) and not a
  vault-note fix (rung 1) — not skip that reasoning silently.
- Name the ceiling on what a wording-only fix can guarantee (vault note 198), so the eventual
  `gate_verdict.py` decision is made with that constraint in view, not blind to it.

**Non-Goals:**
- Editing `agent-instructions/skills/recall/SKILL.md` — deferred to a follow-up cycle (vault
  note 26: skill-text behavior changes require `writing-skills` TDD).
- Changing `gate_verdict.py`'s C5 pass criterion — deferred; needs its own decision informed by
  a real post-fix measured rate (see Open Questions).
- Investigating or fixing cluster representative selection (`internal/cli/query.go`,
  `pickRepresentative`) — verified this session (subagent trace to
  `internal/cli/query.go:1524-1540`) as working as designed: representative = nearest to the
  cluster's k-means centroid, independent of raw query-match score by design, confirmed
  against `docs/GLOSSARY.md:397-398`. Not implicated in the miss (the failing trial's own
  reasoning never referenced "representative"). No action needed.

## Decisions

### Decision: rung-2 (repo/skill change), not rung-1 (vault note)

Vault note 495 (Joe's own design direction) sets a two-rung default: prefer a vault note
(immediately user-actionable) over a repo/skill change ("a last resort"). Considered: could a
vault note stating "read every chunk before judging, even at 0 notes" close this gap on its
own? No, structurally — recall's Step 2.5 only consults notes/chunks matching the CURRENT
task's own query phrases; it has no step that queries "am I executing Step 2.5 correctly." A
note about correctly running Step 2.5 has no reliable trigger surface inside Step 2.5's own
execution; it would only resurface by coincidence (explore-sampling near some unrelated future
query), not reliably at the moment it's needed. The gap is in the procedure's own text, so the
fix has to be too.

Alternative considered: rely on vault note 794 alone (already crystallized this session), no
repo change. Rejected for the identical structural reason — 794's `situation:` field is
phrased for a diagnostic moment ("diagnosing why a recall-based agent failed to honor..."), not
for the live Step 2.5 moment, so it will not surface as a Step 2.5 candidate during a future
live recall run either.

### Decision: spec the requirement now; implement the SKILL.md text in a follow-up cycle

`writing-skills` TDD (note 26) requires a baseline (RED) run showing the failure mode on the
OLD text before editing. That baseline already exists, and is stronger than a synthetic one:
the actual failing trial's transcript (`gate-C5-6l_mjvl1/warm-cfg/projects/.../warm-3-*/*.jsonl`)
is a real RED artifact — cite its exact turns rather than re-deriving a synthetic repro. This
change's job is to (a) spec-record the target behavior and (b) hand the follow-up cycle a
concrete `tasks.md` plus a pointer to the RED evidence — not to run that TDD cycle itself,
since a SKILL.md edit is its own gated, test-driven unit.

### Decision: name the ~95% ceiling explicitly; do not silently assume the wording fix "solves" C5

Vault note 198 measured (a different skill-text mechanism, 3 wording iterations): prose/wording
mechanisms in skill text plateau below ~95% per-trial adherence — good for raising a rate,
wrong tool for guaranteeing every-trial compliance. `gate_verdict.py`'s current C5 verdict
(`axis_verdict`) requires every valid trial in a run to pass for GREEN — at a generous 95% true
compliance rate, a 5-trial `--tier full` run has only ~77% (0.95^5) chance of showing all-GREEN;
at 90% only ~59%. A wording-only fix, even one that genuinely raises true compliance well above
today's measured 80%, should be EXPECTED to still occasionally show C5 as RED with zero real
regression. Naming this now so it isn't rediscovered as a fresh mystery later; the actual
disposition (accept the residual flakiness under the existing same-tree-recheck discipline —
notes 209/793 — vs. move `gate_verdict.py`'s C5 criterion to a threshold test) is left as an
Open Question, decided from the ACTUAL post-fix rate once measured.

## Risks / Trade-offs

- [Risk] The eventual SKILL.md wording fix may not fully close the gap (per the ~95% ceiling
  above) → [Mitigation] The spec requirement + RED transcript give the follow-up cycle a
  concrete target and baseline; measure the post-fix rate with `c5.py --arms warm` before
  declaring the fix sufficient, and revisit `gate_verdict.py`'s C5 threshold based on that
  measurement rather than an assumption that 100% is achievable.
- [Risk] Strengthening Step 2.5A's wording could add verbosity/cost to every recall
  invocation, not only chunks-only-cluster ones → [Mitigation] Scope the wording addition to
  the chunks-only-cluster branch specifically (already a distinct sub-case in the existing
  text), not a blanket rewrite of Step 2.5A.
- [Risk] This requirement could go stale if a future, unrelated recall refactor removes the
  `show-chunk` mechanism (`--lazy-chunks`) entirely → [Mitigation] The requirement is written
  against `recall-payload-cuts`, the capability that already owns `show-chunk`'s existence; if
  that capability is retired, this requirement retires with it via the normal spec-sync
  process.

## Migration Plan

N/A — this change is spec-only; no code or skill edits ship in this cycle, so there is nothing
to deploy or roll back.

## Open Questions

1. Should `gate_verdict.py`'s C5 axis verdict move from an exact-bar
   (`passed == len(valid)`) to a threshold-based test (e.g. a confidence bound against a
   target rate) once the post-fix true compliance rate is measured? Decide in the follow-up
   implementation cycle, with real post-fix data — not now, on a guess.
2. Does the eventual SKILL.md wording fix belong only in Step 2.5A's existing chunk-reading
   sentence, or does the zero-note branch need its own more visually distinct callout? (The
   failing trial's exact words — "0 notes; crystallized 0 lessons" — suggest the current
   text's flow makes "no notes" read as a stopping point.) Leave to the `writing-skills` TDD
   cycle to decide by pressure-testing candidate wordings.
