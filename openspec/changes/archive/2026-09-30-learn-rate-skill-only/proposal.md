# Proposal: learn-rate-skill-only

## Why

The memory-loop audit (`dev/eval/audit/REPORT-2026-09-02.md`, gate-reviewed 2026-09-02..06)
measured the writing side as the memory loop's dominant loss: the learn step fired for only
**5.6% of worth-learning moments** (W2, 4/71, CI95 [1.4, 11.3]); **67 lessons were lost**, of
which **40 (59.7%) were judged realistically reachable by a completion-report-lessons step**;
the single largest bucket is **39 missed success-reinforcements** (learn's kind-4 almost never
fires). The audit's read-side finding sharpens the case: the failures memory "would have helped
with" overwhelmingly trace to lessons never captured (9 of 11 fixable failure-family moments had
nothing relevant in the vault at their timestamps) — write loss surfacing downstream as
failures, not read loss.

Joe's constraints, decided in the 2026-09-06 brainstorm: **skill/guidance-markdown only** (no
hooks — not live enough and harness-specific; no new engram code in v1 — this v1 is the
deliberate test of whether the agent can behave reliably from md alone), and a
**capture-then-curate** quality posture (low bar at capture time, quality control at a separate
curation step, both implemented on existing surfaces).

## What Changes

Three skill-text edits, no new code, no new storage:

1. **LESSONS contract (route + please skills):** every dispatched subagent's completion report
   must end with a `LESSONS:` line — `LESSONS: none` or 1–3 one-line lessons (confirmed
   corrections, surprising findings, validated approaches; deliberately low bar — these are
   offers, not vault notes). The orchestrator re-asks once when the line is missing and
   collects lines without judging them in-flight. This is the pre-decided #739 D7 direction
   (vault note 853, 2026-08-30: "measure→inject — … a mandatory LESSONS line in completion
   reports under skill-TDD"); measured precedent: a prior engram effort (#655) measured ~93%
   adherence to a mandated completion-report output contract, so the mechanism is proven
   in-reach for prose-only enforcement.
2. **Curation at the closing learn (learn skill):** the closing learn's sweep gains an explicit
   input — scan the session's collected `LESSONS:` lines and judge each against the learn
   skill's existing Step-2 bar: the line maps to one of the four moment kinds (kind 1
   corrections, kind 2 explicit save-requests, kind 3 reversals, kind 4 confirmed approaches),
   it is confirmed rather than hypothesized (kind 4 requires a resolved uncertainty or explicit
   specific confirmation — never bare success), and it states a general reusable principle, not
   a session-specific narrative. Lines that clear the bar crystallize
   via write-memory as usual; the rest are discarded silently. The report line itself is the
   disposable offer tier: capture-then-curate with zero new plumbing.
3. **Kind-4 recalibration (learn skill):** add 2–3 concrete positive exemplars drawn from the
   audit's real missed success moments, and tie the kind-4 scan explicitly to the completion
   moment. The bar itself (resolved uncertainty or explicit specific confirmation — never bare
   success) does not move; only its recognizability does.

Explicitly parked, each with the same pre-registered revisit condition (post-v1 re-measure via
`dev/eval/audit/run_audit.py`, ~$35, after ≥2–3 weeks of natural usage — W2 still below Joe's
bar is the evidence vault note 495 requires before escalating: note 495's rule is that engram
improvements default to vault-note/skill-text fixes, with a repo/code change justified only
once the note-level fix has demonstrably failed to reach the problem): the
**watcher/metacognition layer**
(v1 batch mining of the append-only chunk index → offers; v2 streaming transcript tail + a
watcher-salience channel in recall's payload — harness-agnostic, but code), the **mechanical
validator** on the LESSONS contract (the guarantee layer per vault note 198's finding that
prose/skill-text mechanisms asymptote below ~95% per-trial adherence — guarantees need a
mechanical layer), and **hooks/harness push** (out for Claude Code entirely; Pi-era).

## Capabilities

### New Capabilities

- `lessons-contract`: dispatched-subagent completion reports carry a mandatory LESSONS line;
  the orchestrator enforces presence (one re-ask) and collects lines for the closing learn.

### Modified Capabilities

- `learn`: closing learn curates collected LESSONS lines through the existing four-kind
  taxonomy and Step-2 quality bar; kind-4's cue gains audit-derived exemplars and a
  completion-moment scan anchor.
- `route`: the dispatch handoff checklist ("The handoff is the unlock" in route's SKILL.md)
  mandates the LESSONS line in every subagent's completion-report instructions.
- `please`: gate/step text collects LESSONS lines from dispatched work and hands them to the
  closing learn.

## Impact

- **Affected specs:** `learn`, `route`, `please` (skill behavior specs); new `lessons-contract`
  capability spec.
- **Affected code:** none in v1 — `agent-instructions/skills/{route,please,learn}/SKILL.md`
  only, each under `superpowers:writing-skills` TDD (repo rule), deployed via `engram update`.
- **Measurement:** pre-registered before/after with the committed audit instrument
  (`run_audit.py`, instrument v2, bootstrap CIs; primary metric W2, secondary lost-lessons
  total and reachable share; per-moment counting unit; instrument version declared on both
  sides).
- **Risk:** prose mechanisms cap below ~95% per-trial adherence (note 198) — v1 buys
  rate-raising, not guarantees; the parked validator/watcher are the escalation path, gated on
  the re-measure rather than pre-built.
