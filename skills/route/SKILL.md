---
name: route
description: >
  Use when you are about to dispatch a subagent and must decide its agent type, model, and
  effort level. Triggers on any delegation decision, and when you recognize a unit is too large
  for one focused agent and needs decomposition before dispatch.
---

# Route — default to the cheapest tier, escalate on evidence, remember what works

You are an orchestrator. You route, decompose, and synthesize; you do not do object-level work
yourself. There is no inline escape — easy work is delegated to a cheap model, not skipped.

**The rubric is memory, not a hard-coded table.** Every unit starts at the cheapest / fastest
available tier. The *only* thing that raises the starting tier is **recalled evidence** that this
kind of work has failed cheaper before. When a dispatch fails, you fix the spec and — if it fails
again — escalate one tier. Every dispatch is recorded, and those records are what `/learn`
crystallizes and `/recall` surfaces, so the starting tier for similar work reflects real evidence
next time. Cold-start is cheapest-for-everything; as evidence accrues, the *effective* rubric warms
up on its own — via recall, not by editing this file.

## Orchestration work vs object-level work

The line that keeps "delegate everything" from collapsing into either "delegate nothing" or
"delegate the act of delegating":

- **You do (orchestration):** routing/decomposition decisions, dispatching subagents, sequencing
  steps, updating the task list, running the meta-skills that ARE the workflow (`/recall`,
  `/learn`, planning), and synthesizing subagents' returned results into the next decision or the
  user-facing report.
- **You delegate (object-level):** writing code or prose, running tests/builds, judgment calls on
  the artifact, reviewing the artifact — anything that produces or evaluates the deliverable.

## How to pick a tier

1. **Recall first (you, the orchestrator).** Before dispatching, check recalled memory for
   tier-performance evidence on *this kind of work* ("cheap tier sufficed for X" / "cheap failed on
   Y — needs mid"). Recalled evidence sets the starting tier.
2. **Absent evidence, default to the cheapest / fastest tier — for EVERY unit, no exceptions.**
   This includes work that *feels* genuinely hard: hard debugging, cross-cutting refactors,
   correctness-critical reviews, greenfield design. Your sense that a unit "needs a strong model"
   is a **prediction of difficulty, which is not evidence** — "genuinely complex" and "looks hard"
   are the same hunch in different words, and the hunch is exactly what this loop replaces. You
   learn a unit needs a higher tier by watching the cheap tier **fail** (step 3), never by
   forecasting it. On a cold start with no recalled evidence, a race condition, an 8-file refactor,
   and a new API design all start **cheap** — same as a variable rename.
3. **Escalate on failure, spec-first:**
   - **First fail** → the failure is usually a *spec* failure, not a model failure. Rewrite the
     handoff (sharper files, acceptance checks, tighter do-NOT-touch), retry the **same** tier. The
     builder never gets to guess twice off the same spec.
   - **Second fail on the same unit** → escalate **one** tier and retry.
   - Repeat until it passes or you reach the deep tier.
4. **Memory discounts the tier (a special case of evidence lowering it).** A unit whose needed
   knowledge is recallable — a known convention, prior decision, crystallized diagnostic — drops one
   tier (floored at cheap), because the model *applies* recalled knowledge instead of *deriving* it.
   (Measured 2026-06-28 at the deep→mid boundary — vault note 135.)

## The handoff is the unlock

A cheap-tier model matches an expensive one when the spec is exact; when it fails, suspect the spec
before the model. Every dispatch MUST hand the subagent:

- **Exact files and paths** to create/modify (not "the auth code" — `internal/foo/bar.go:20-45`).
- **Acceptance checks** — the concrete, verifiable conditions that mean "done" (a command + its
  expected output; a test that must pass; the property that must hold).
- **Explicit do-NOT-touch bounds** — files, interfaces, and behaviors the unit must leave alone.
- **The subagent's recall-first instruction** (see the "Two rules every dispatch obeys" section below).

Vague handoffs are why cheap tiers "fail." Fix the handoff first.

## Record every dispatch (the evidence)

After each dispatch resolves, record one line of evidence. Build it from what you already know as
orchestrator plus the review's verdict. No privileged telemetry needed — this keeps the loop
harness-agnostic. A **work-kind** is your classification of the unit's shape and concept ("single-file
refactor", "cross-cutting lint", "API integration") — kept consistent enough that the same kind
reuses prior dispatches' evidence.

| field | source |
| --- | --- |
| **work-kind** | your classification of the unit (see above) |
| **tier used** | your routing decision (cheap / mid / deep) |
| **model (roster @ dispatch)** | the concrete model the tier resolved to *at dispatch time* (e.g. `cheap (haiku)`); provenance, not a routing rule — lets a later audit ask "did swapping the cheap model change the failure rate on this work-kind?" |
| **why** | source of the tier choice: "recalled evidence (kind K passed at tier T)", "memory-discount applied", or "cold-start default" |
| **outcome** | **the review/gate verdict** — PASS/FAIL. Never the subagent's self-report (it confabulates — vault notes 148, 162). |
| **escalation** | if it failed and escalated, the tier it finally passed at |
| **duration** | observed wall-clock, best-effort |
| **cost** | only if the harness exposes it; mark "n/a" otherwise — never invent it |

Produce this record per-dispatch and collect the rows into a compact table in your user-facing
report (the mini-report). The table is the audit trail for route's own tier guidance over time.

## The loop that improves the rubric

1. **Route** picks a tier (cheapest-first, or evidence-raised).
2. **Record** the dispatch outcome (above).
3. The record lands in your session transcript, which **`/learn`**'s sweep auto-ingests as
   recallable memory — so even uncrystallized, a tier outcome is retrievable next time. When a
   finding is strong and general ("cheap failed N× on work-kind K — needs mid"), crystallize it via
   `/learn`: a save-request to yourself, or a reversal if it overturns a prior tier assumption. (A
   dedicated confirmed-approach capture kind — issue #668 — would be its cleaner home.)
4. **`/recall`** surfaces that evidence next time similar work is routed.
5. The **starting tier** now reflects evidence — the rubric improved without editing this file.

Periodic consolidation: when strong repeated evidence accrues for a work-kind, fold it into the
cold-start priors below via `superpowers:writing-skills` TDD, so a cold session starts warm. Until
then, recall does the improving.

## Cold-start priors (unproven — evidence overwrites these)

Absent recalled evidence, these are the *starting* tiers — hypotheses the dispatch record confirms
or overturns, not prescriptions. The posture is **cheapest for everything**; no entry starts above
cheap without recorded evidence (see "How to pick a tier" step 2 — including its no-exception for
work that merely *feels* hard), and none has earned it yet.

| Work character | Cold-start tier | Status |
| --- | --- | --- |
| Everything, by default | cheap | the posture |
| Memory-backed unit (answer is recallable) | one tier down, floored at cheap | **evidence-backed** (note 135) |

Current roster: cheap = haiku, mid = sonnet, deep = opus. A new model re-fills a tier without
changing this table.

## Two rules every dispatch obeys

1. **The subagent recalls first.** Instruct every dispatched subagent that its FIRST action is
   `/recall`, with phrases drawn from its unit, before doing the work. Vault memory is part of the
   job, not an optional warm-up.
2. **Decompose before dispatch.** A unit too large for one focused subagent — it spans multiple
   files or concerns, or needs more than one clear deliverable — is not dispatched as-is. Break it
   into smaller units and route each. Decomposition is orchestration; you do it yourself.

## Red flags — STOP and re-read

| Sign you're off | What to do |
| --- | --- |
| You picked mid/deep because the unit "looks hard" | Default cheapest; only recalled evidence raises the start. A surface hunch is the guess the loop replaces. |
| You picked deep because it's "genuinely hard / complex," not merely "looks hard" | Same hunch, fancier words. Predicted difficulty is not recorded evidence. Cold start = cheap for hard debugging, refactors, correctness reviews, and design too. |
| You reasoned "correctness-critical / broad impact / needs careful reasoning → deep" | That is forecasting difficulty. Start cheap; let a *failed review* prove it needs more — don't pre-provision. |
| You escalated a tier on the first failure | First fail → rewrite the spec, retry same tier. Escalate only on the second fail. |
| You dispatched without exact files, acceptance checks, and do-NOT-touch bounds | Fix the handoff — a vague spec is why cheap tiers "fail." |
| You skipped recording the dispatch outcome | The record IS the rubric; no record → no evidence → the rubric never improves. |
| You wrote the outcome from the subagent's "done" claim | Outcome = the review verdict, a ground-truth artifact. Subagent reports confabulate (notes 148, 162). |
| You invented a cost/duration number | Duration is best-effort observed; cost only if the harness exposes it. Never fabricate. |
| You treated the cold-start table as a prescription | It's a set of overwritable priors; recalled evidence wins. |
| You hardcoded a model name as the rule | Route by tier; names are just the current roster. |
| "The subagent has the prompt, skip its recall" | Recall-first is non-waivable. |
| "The complex task can go as one big agent" | Decompose first, then delegate the pieces. |
