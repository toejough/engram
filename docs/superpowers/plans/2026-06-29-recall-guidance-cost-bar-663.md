# Relax the decision-moment recall guidance's cost bar (#663) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans (or subagent-driven-development).
> This edits **global guidance** (`~/.claude/CLAUDE.md`), not a SKILL.md — so the writing-skills *Skill* does
> not apply, but its **headless RED/GREEN discipline does** (the precedent that shipped the current guidance —
> vault notes 138/139). Steps use `- [ ]`. Gate B/C/D markers are run by the `/please` orchestrator.

**Goal:** Point the three decision-moment cues at the cheap `/recall glance` rung (shipped in #662) and relax the
*cost* hesitation so it fires readily — while keeping the *value* gate (idiosyncratic-only) and the Gate-A
hardening intact.

**Architecture:** One surgical edit to the `~/.claude/CLAUDE.md` "Recall at the decision moments" section
(lines 25-33). The current filter "fire it only when you expect a vault-specific gotcha … each call costs real
minutes" *conflates* cost and value into one phrase. #663 **separates** them: relax COST (glance is cheap → fire
readily), preserve VALUE (skip routine/non-idiosyncratic — memory is net-negative there even when cheap, notes
91/95/99). Then scrub the ROADMAP Track-A entry and amend vault note 139 (its `sources` point at this section).

**Tech Stack:** Markdown guidance; the headless `claude -p` revalidation harness
(`docs/design/2026-06-29-recall-moments-revalidation-data/`); `engram amend` (note 139).

## Global Constraints

- **Headless `claude -p`, fresh process, fictional domains — NEVER subagents** for the RED/GREEN (subagents
  inherit session context and invalidate the control — vault note `feedback_headless_not_subagents…` / 138).
- **Do NOT regress the win-nucleus of the guidance:** the explicit action-naming ("the action is recalling — not
  a substitute self-check" — note 137, took 0/5→5/5), the 3 cues, and the Gate-A hardening ("once, not per
  retry"; do NOT re-add the cut "before a final verdict" cue — note 139).
- **The value gate is load-bearing** (notes 91/95/99): relaxing cost must NOT extend firing to clearly
  non-idiosyncratic/routine decisions. That is the regression the value-gate test guards.
- `~/.claude/CLAUDE.md` is a **global file, edited in place** (NOT in the repo — no git commit for it). Only the
  ROADMAP scrub is a repo commit. Commit trailer `AI-Used: [claude]`.

---

## Task 1: Revise the guidance + headless RED/GREEN

**Files:**
- Modify (global, in place): `~/.claude/CLAUDE.md` lines 25-33.
- Reuse: the headless harness under `docs/design/2026-06-29-recall-moments-revalidation-data/`.

**Exact GREEN content** — replace the current section (lines 25-33) with:

```markdown
## Recall at the decision moments, not only at the start

engram `/recall` surfaces vault memory you haven't loaded — most valuable right before a wrong call locks in.
The cheap **`/recall glance`** rung (read-only, no crystallization writes, ~3 phrases) makes this affordable:
it's quick, so fire it **readily** at the cues below — but still only where memory plausibly helps, i.e. when you
expect a **vault-specific gotcha**: a prior decision, a hard-won project lesson, a convention that bites. On
routine, non-idiosyncratic decisions, skip it — memory is net-negative there even when cheap. At these cues,
**run `/recall glance` before you proceed**:

- **Before declaring work done** — **`/recall glance` first** (the action is recalling — not a substitute
  self-check or re-inspection), then verify; the vault names the gotchas you'd otherwise ship past.
- **After a failure you can't immediately explain** — **`/recall glance` once before you start guessing** (not on
  every retry) — a past lesson may name the cause.
- **Before you start building a new approach** — **`/recall glance` prior decisions and standards first**, while
  the path is still cheap to change.

Escalate to **`/recall deep`** when the decision is weighty or irreversible, when `glance` flags an uncovered
gap, or when it turns on honoring a recent-activity standard (glance surfaces such items but won't elevate them
to requirements). `deep` also crystallizes — reach for it when you want recall to *learn*, not just check.

These catch *application* gaps — the lesson existed, just unrecalled at the moment.
```

- [ ] **Step 1 — Inspect the harness.** Read `docs/design/2026-06-29-recall-moments-revalidation-data/scenarios.json`
  and the runner (`scratchpad/run_revalidation.sh` if present, else the data dir's README/results.md) to learn how
  the prior 0/5→4/5 flip was measured (fresh `claude -p` per arm, controlled CLAUDE.md as the only variable,
  fictional domains). Confirm it counts whether the agent fired `/recall` at a cue.
- [ ] **Step 2 — RED (control + current guidance).** For a **plausibly-idiosyncratic** cue scenario (fictional
  domain), run headless `claude -p` with (a) NO recall-guidance CLAUDE.md (control) → expect ~0/5 recall;
  (b) the CURRENT guidance → fires `/recall` but **deep/unspecified** (no glance), with the cost-hesitation
  wording. Record both. This establishes the baseline the revision must beat *without regressing the flip*.
- [ ] **Step 3 — GREEN edit.** Apply the exact GREEN content above to `~/.claude/CLAUDE.md` lines 25-33.
- [ ] **Step 4 — GREEN test 1 (flip preserved + glance used).** Headless `claude -p`, revised guidance, the same
  plausibly-idiosyncratic cue (fictional domain), 5 reps: the agent must still **fire recall at the cue**
  (flip preserved, ≈ control's 4-5/5) AND invoke the **`glance`** rung (not deep) — i.e. the relaxation took.
  Pass-bar: fires ≥4/5, and the fires use `glance`.
- [ ] **Step 5 — GREEN test 2 (value gate holds — the regression guard).** Headless `claude -p`, revised
  guidance, a **clearly non-idiosyncratic / routine** decision scenario (e.g. "rename a local variable",
  "format this JSON" — no plausible vault gotcha), 5 reps: the agent must **NOT fire** recall (the lowered cost
  bar must not induce over-firing on routine work). Pass-bar: ≤1/5 fires (and any fire is defensible).
- [ ] **Step 6 — If either bar fails, REFACTOR + Gate B.** If the flip regressed → the glance/value wording
  buried the action; restore action-prominence (note 137). If the value gate leaked (fires on routine) →
  strengthen the "skip routine/non-idiosyncratic — net-negative even when cheap" clause. Re-run the failed test.
  The refactored guidance passes Gate B (design-fit) before the task is done.
- [ ] **Step 7 — Record results** as a labeled table: arm × {fires/5, rung used} for control / current / revised
  (idiosyncratic) / revised (routine). Spend estimate: ~15-20 `claude -p` runs ≈ $5-10 (no cap; report actual).

## Task 2: Doc + memory scrub (note 64)

**Files:**
- Modify: `docs/ROADMAP.md` — the Track-A "✅ SHIPPED — recall at the decision moments" entry (the cost-filter
  wording) AND the Track-B depth-dial entry's "(3, #663)" line (mark it shipped).
- Amend (CLI): vault note `139.2026-06-29.recall-decision-moments-guidance-not-hooks-shipped.md`.

- [ ] **Step 1 — ROADMAP Track-A.** In the "✅ SHIPPED — recall at the decision moments" entry, update the
  cost-filter description: the cues now fire the cheap **`/recall glance`** rung *readily* (cost bar relaxed by
  #662/#663); the **value** gate is unchanged (idiosyncratic-only — net-negative on routine even when cheap);
  `deep` escalation for weighty/coverage/C5. One or two sentences; do not rewrite the whole entry.
- [ ] **Step 2 — ROADMAP Track-B.** In the depth-dial entry, mark "(3, #663)" as ✅ SHIPPED 2026-06-29
  (guidance relaxed to fire glance readily, value-gate preserved, headless-validated).
- [ ] **Step 3 — Amend note 139** (its `sources` include this CLAUDE.md section; the relaxed wording supersedes
  its "fire only when … costs real minutes" description). Use the bare Luhmann id for `--target` (per #664):
```bash
engram amend --target "139" \
  --relation "140.2026-06-29.recall-depth-dial-relaxes-not-dissolves-overfire|update: #663 relaxed this guidance's COST bar — cues now fire /recall glance readily; value gate (idiosyncratic-only) unchanged" \
  --behavior "<re-state: 3 cues now fire the cheap /recall glance rung readily; deep reserved for weighty/coverage/C5; value gate unchanged>"
```
  (If `amend` rejects re-synthesis flags on a fact note, fall back to a `--relation`-only link recording the
  supersession — do not fabricate a flag the CLI doesn't accept; verify with `engram amend --help` first.)
- [ ] **Step 4 — Gate C** over the ROADMAP changes.

## Task 3: Close out #663

- [ ] **Step 1 — Commit** the ROADMAP changes (the `~/.claude/CLAUDE.md` edit is global, not committed).
  Message scope `docs(663)`; trailer `AI-Used: [claude]`. Gate D over the message + #663 comment first.
- [ ] **Step 2 — Comment + close #663**: guidance relaxed (glance fires readily, value-gate preserved); headless
  flip-and-value-gate results (the Task-1 table); note 139 amended. Depth-dial arc (#661→#662→#663) complete.
- [ ] **Step 3 — Clean** any scratchpad harness copies created.

## Self-review (writing-plans checklist)
- **Coverage:** guidance edit + headless validation = Task 1; doc/memory scrub = Task 2; close-out = Task 3.
- **Scope honesty:** this relaxes the COST bar only; the VALUE gate and the action-naming/Gate-A hardening are
  explicitly preserved (the win-nucleus). The bounded-reach caveat (firing more helps only to the addressable
  share — note 109) is recorded, not re-litigated (Joe chose #663 over payload-prune).
- **Validation correctness:** two headless bars — flip-preserved (don't lose the win) AND value-gate-holds
  (don't over-fire on routine) — both via fresh `claude -p`, never subagents.
