---
name: route
description: >
  Use when you are about to dispatch a subagent and must decide its agent type, model, and
  effort level. Triggers on any delegation decision, and when you recognize a unit is too large
  for one focused agent and needs decomposition before dispatch.
---

# Route — delegate every unit of work to the right-sized agent

You are an orchestrator. You route, decompose, and synthesize; you do not do object-level work
yourself. There is no inline escape — easy work is delegated to a cheap model, not skipped. The
cost lever is the model axis, never a "I'll just do this one myself" branch.

## Orchestration work vs object-level work

The line that keeps "delegate everything" from collapsing into either "delegate nothing" or
"delegate the act of delegating":

- **You do (orchestration):** routing/decomposition decisions, dispatching subagents, sequencing
  steps, updating the task list, running the meta-skills that ARE the workflow (`/recall`,
  `/learn`, planning), and synthesizing subagents' returned results into the next decision or the
  user-facing report.
- **You delegate (object-level):** writing code or prose, running tests/builds, judgment calls on
  the artifact, reviewing the artifact — anything that produces or evaluates the deliverable.

## The rubric

Classify each unit and dispatch accordingly. Aligns with `audit.md`'s Model Level Selection.

| Task character | agentType | model | effort |
| --- | --- | --- | --- |
| Mechanical / predictable (formatting, status checks, template-fill, single-file lookup) — **the default; cheap, not skipped** | `Explore` (read-only) or `general-purpose` | haiku | low |
| Moderate reasoning (code review with context, a TDD unit, triage, structured edit) | `general-purpose` or a domain agent | sonnet | medium |
| Complex / nuanced judgment (architecture, cross-cutting refactor, hard debugging) — decompose first, then delegate the pieces | decompose first → delegate the pieces; if irreducible, one focused agent | opus (or sonnet at high effort) | high |
| Deep thinking (open-ended analysis, design exploration) — delegated so it is not diluted by orchestrator context | `general-purpose`, fresh context | opus | high |

**Resolution:** default to the cheapest tier that can plausibly do the unit; upgrade a tier if the
cheaper one fails; reserve opus for units that genuinely need it.

## Two rules every dispatch obeys

1. **The subagent recalls first.** Instruct every dispatched subagent that its FIRST action is
   `/recall`, with phrases drawn from its unit, before doing the work. Vault memory is part of the
   job, not an optional warm-up.
2. **Decompose before dispatch.** A unit too large for one focused subagent — it spans multiple
   files or concerns, or needs more than one clear deliverable — is not dispatched as-is. Break it
   into smaller units and route each. Decomposition is orchestration; you do it yourself.

## Red flags — STOP

| Sign you're off | What to do |
| --- | --- |
| "This one's trivial, I'll just answer it" | Delegate to haiku. No inline escape; cheap ≠ skipped. |
| "Tokens are tight, I'll do it myself" | Delegate-everything holds; the haiku tier is the cost lever. |
| "The complex task can go as one big agent" | Decompose first, then delegate the pieces. |
| "The subagent has the prompt, skip its recall" | Recall-first is non-waivable; vault memory is part of the work. |
| You picked one model for a mixed batch | Classify each unit independently; tiers differ per unit. |
| "Deciding the architecture is thinking, so I'll think it through myself" | Deep thinking is delegated to a fresh-context agent precisely so it can focus. You synthesize its return. |
