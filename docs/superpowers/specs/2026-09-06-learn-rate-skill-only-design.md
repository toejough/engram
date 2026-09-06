# Learn-Rate Improvement, Skill-Only v1 — Design

**Date:** 2026-09-06 · **Status:** draft, awaiting Joe's review · **Feeds:** the #739 Phase 2
scope decision (memory-loop-audit tasks 6.3/6.4)

## Problem

The memory-loop audit (dev/eval/audit/REPORT-2026-09-02.md) measured the writing side as the
memory loop's dominant loss: **W2 learn-fired = 5.6%** (4/71 worth-learning moments), **67 lost
lessons** of which **40 (59.7%) are judged reachable by a completion-report-lessons step**, and
the largest bucket is **39 missed success-reinforcements** (learn.md's kind-4 almost never
fires). The audit also showed the failures memory "would have helped with" overwhelmingly trace
to lessons never captured (9 of 11 fixable failure-family moments had nothing relevant in the
vault at their timestamps) — write loss surfacing downstream, not read loss.

## Constraints (Joe's, decided in this brainstorm)

1. **Skill/guidance-markdown only.** No hooks (not live enough, harness-specific), no harness
   integration, no new engram code in v1. The agent must behave reliably from md alone — and
   this v1 is the deliberate test of that hypothesis.
2. **Layered mechanisms, capture-then-curate.** Lower the bar at capture time; keep quality
   control at a separate curation step. In v1 both layers are implemented with existing surfaces.
3. **Prior-decision grounding:** the completion-report LESSONS line is the pre-decided #739 D7
   direction (vault note 853); prose mechanisms cap below ~95% per-trial adherence (note 198) —
   v1 accepts rate-raising, not guarantees; escalation beyond prose requires measured evidence
   (notes 495/824).

## Design — three skill edits, no new infrastructure

### 1. LESSONS contract in dispatch/completion (route + please skills)

- **route** (agent-instructions/skills/route/SKILL.md): the dispatch handoff template gains a
  mandatory instruction — every dispatched subagent ends its completion report with a
  `LESSONS:` line: either `LESSONS: none` or 1–3 one-line lessons (confirmed corrections,
  surprising findings, validated approaches — low bar, offers not vault notes).
- **Orchestrator check (route + please text):** on receiving a report without a `LESSONS:`
  line, the orchestrator asks the subagent once for it (or records `LESSONS: missing` if the
  agent is gone). Prose-level enforcement; measured precedent ~93% for output contracts (#655).
- The orchestrator **collects** LESSONS lines; it does not judge them in-flight.

### 2. Curation at the closing learn (learn skill)

- **learn** (agent-instructions/skills/learn/SKILL.md): the closing learn's Step 2 sweep gains
  an explicit input: scan the session's collected `LESSONS:` lines (they are in the transcript
  by construction) and judge each against the existing four-kind taxonomy and the notes-68/69
  quality gate. Lines that clear the bar are crystallized via write-memory as usual; lines that
  don't are discarded silently — the report line itself is the disposable "offer tier."
- No new storage, no pending-offers plumbing: capture-then-curate implemented as
  report-lines-then-learn-judgment.

### 3. Kind-4 (confirmed approach) recalibration (learn skill)

- The audit shows the "never mere success" bar reads so strict that 39 real confirmed-approach
  moments died as routine. Recalibrate the kind-4 cue: add 2–3 concrete positive exemplars
  drawn from the audit's real missed moments, and tie the scan explicitly to the completion
  moment ("before writing a completion report or closing a cycle, scan for resolved
  uncertainties and validated bets"). The bar itself (resolved uncertainty or explicit specific
  confirmation — never bare success) does not move; only its recognizability does.

## Testing

Every SKILL.md edit under `superpowers:writing-skills` TDD (repo rule, no exceptions):
baseline pressure test RED, edit, behavioral GREEN. The route/please contract edits get a
dispatch-shaped pressure test (does a dispatched agent's report carry LESSONS; does the
orchestrator re-ask on absence); the learn edits get a closing-learn test over a transcript
containing planted LESSONS lines of mixed quality (bar holds: junk discarded, confirmed kept).

## Measurement (pre-registered)

- **Instrument:** `dev/eval/audit/run_audit.py` (committed, instrument v2, bootstrap CIs).
- **When:** after ≥2–3 weeks of natural cross-repo usage of the updated skills.
- **Primary metric:** W2 learn-fired rate (baseline 5.6%, CI95 [1.4, 11.3]); secondary: lost
  lessons total and reachable-fixable share; per-moment counting unit, instrument version
  declared on both sides.
- **Cost:** ~$35/run per the runner's own estimate mode.

## Parked (explicit, with revisit conditions)

- **Watcher (metacognition layer)** — v1 batch mining of the chunk index for missed lessons →
  offers; v2 streaming tail of live transcripts + a watcher-salience channel in recall's
  payload. Harness-agnostic (transcripts on disk; extract.py parses both harnesses; detector
  tech committed). **Revisit when:** the post-v1 re-measure shows W2 still below Joe's bar —
  that measurement is the note-495 evidence for escalating to code. Deferral is safe: the
  chunk index is append-only, so today's misses remain minable later.
- **Hooks / harness push** — out entirely for Claude Code; Pi-era consideration only.
- **Mechanical validator on the LESSONS contract** — the guarantee layer (note 198); same
  revisit condition as the watcher.

## Out of scope

Read-side changes (find loss measured ≈1–3 moments — nothing to fix), D-E fork-credit rule,
dispatch-side memory injection (the other #739 Phase 2 lever, decided separately at 6.3).
