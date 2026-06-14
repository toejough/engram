# Route skill + please integration — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:writing-skills for both SKILL.md
> artifacts (RED baseline → GREEN → pressure tests). Steps use checkbox (`- [ ]`) syntax.

**Goal:** create `skills/route/SKILL.md` (+ `commands/route.md`) encoding the delegate-everything
routing doctrine, and update `skills/please/SKILL.md` so its gates consult the router. Per
`docs/superpowers/specs/2026-06-14-route-skill-design.md`.

**Architecture:** pure skill-text + one command pointer. No Go change — `engram update` scans
`skills/` and `commands/` by directory, so both new files deploy automatically.

**Tech stack:** markdown skill text; subagent pressure tests; `engram update` for deployment.

---

### Task 1: RED — baseline without the route skill

**Files:** Create `/tmp/route-red/batch.md` (throwaway).

- [ ] **Step 1: Write the batch scenario**

```markdown
# Scenario
You are a top-level orchestrator. You have four units of work queued:
1. Reformat a commit message to Conventional Commits style.
2. Review a 40-line diff for correctness given surrounding context.
3. Refactor authentication across 6 files to a new session model.
4. Decide the architecture for a new caching layer (open-ended).

For EACH unit, state exactly how you would staff it: do it yourself or delegate?
If delegate, which agent type, which model, which effort level? List your decisions.
```

- [ ] **Step 2: Run baseline.** Dispatch a fresh general-purpose subagent with ONLY the batch
scenario (no route skill). Record verbatim. Expected RED: does several inline, picks one model
for all or omits model/effort, no decomposition of unit 3, no recall-first instruction.

### Task 2: GREEN — write `skills/route/SKILL.md`

**Files:** Create `/Users/joe/repos/personal/engram/skills/route/SKILL.md`.

- [ ] **Step 1: Write the skill** with this structure (frontmatter `name: route`, `description`
starting "Use when..." describing triggering conditions only, no workflow summary):

```markdown
---
name: route
description: >
  Use when you are about to dispatch a subagent and must decide its agent type, model, and
  effort level. Triggers on any delegation decision, and when you recognize a unit is too large
  for one focused agent and needs decomposition before dispatch.
---

# Route — delegate every unit of work to the right-sized agent

You are an orchestrator. You route, decompose, and synthesize; you do not do object-level
work yourself. There is no inline escape — easy work is delegated to a cheap model, not skipped.
The cost lever is the model axis.

## Orchestration work vs object-level work

The line that keeps "delegate everything" from collapsing:

- **You do (orchestration):** routing/decomposition decisions, dispatching subagents, sequencing
  steps, updating the task list, running the meta-skills that ARE the workflow (`/recall`,
  `/learn`, planning), and synthesizing subagents' returned results into the next decision or
  the user-facing report.
- **You delegate (object-level):** writing code or prose, running tests/builds, judgment calls
  on the artifact, reviewing the artifact — anything that produces or evaluates the deliverable.

## Not an enforcement mechanism

This skill is guidance you apply when you choose `Agent` tool parameters. A skill cannot change
the main-loop model, and a subagent's model/effort are fixed only at dispatch (agent frontmatter
or the per-invocation `model` parameter). The router's output is a decision YOU encode into the
dispatch call.

## The rubric

Classify each unit and dispatch accordingly. Aligns with `audit.md`'s Model Level Selection.

| Task character | agentType | model | effort | Note |
| --- | --- | --- | --- | --- |
| Mechanical / predictable (formatting, status checks, template-fill, single-file lookup) | `Explore` (read-only) or `general-purpose` | haiku | low | The default. Cheap, not skipped. |
| Moderate reasoning (code review with context, a TDD unit, triage, structured edit) | `general-purpose` or a domain agent | sonnet | medium | |
| Complex / nuanced judgment (architecture, cross-cutting refactor, hard debugging) | decompose first → delegate the pieces; if irreducible, one focused agent | opus (or sonnet at high effort) | high | Decomposition is your job, before dispatch. |
| Deep thinking (open-ended analysis, design exploration) | `general-purpose`, fresh context | opus | high | Delegated so it is not diluted by orchestrator context. |

**Resolution:** default to the cheapest tier that can plausibly do the unit; upgrade a tier if
the cheaper one fails; reserve opus for units that genuinely need it.

## Two rules every dispatch obeys

1. **Subagent recalls first.** Instruct every dispatched subagent that its FIRST action is
   `/recall`, with phrases drawn from its unit, before doing the work.
2. **Decompose before dispatch.** A unit too large for one focused subagent (spans multiple
   files/concerns, or needs more than one clear deliverable) is not dispatched as-is — break it
   into smaller units and route each.

## Red flags — STOP

| Sign you're off | What to do |
| --- | --- |
| "This one's trivial, I'll just answer it" | Delegate to haiku. No inline escape; cheap ≠ skipped. |
| "Tokens are tight, I'll do it myself" | Delegate-everything holds; the haiku tier is the cost lever. |
| "The complex task can go as one big agent" | Decompose first, then delegate the pieces. |
| "The subagent has the prompt, skip its recall" | Recall-first is non-waivable; vault memory is part of the work. |
| You picked one model for a mixed batch | Classify each unit independently; tiers differ per unit. |
```

- [ ] **Step 2: Re-run the Task 1 batch** WITH the route skill prepended (fresh subagent).
Expected GREEN: all four delegated; unit 1 → haiku/low, unit 2 → sonnet/medium, unit 3 →
decomposed then delegated, unit 4 → opus/high fresh context; each dispatch says recall-first.

- [ ] **Step 3:** If any unit is done inline or unit 3 isn't decomposed, tighten wording and
re-run. Do not proceed on a failing GREEN.

- [ ] **Step 4: Combined please+route GREEN.** Fresh subagent given BOTH the route skill and a
gate-A scenario from please (a committed plan to review). Expected: it consults the router to set
each gate angle's model/effort (not a blind fixed pin) and instructs each reviewer to recall
first. Confirms the integration is behaviorally live, not just textually present.

- [ ] **Step 5: Decomposition-inside-please GREEN.** Fresh subagent given the route skill and a
deliberately complex single ask ("refactor auth across 6 files") framed as a please step-4 unit.
Expected: it decomposes before delegating, rather than dispatching one giant agent.

### Task 3: Pressure tests (fresh subagent each, route skill + one probe)

- [ ] "Unit is one trivial rename — just do it." Expected: delegate to haiku.
- [ ] "We're low on budget, handle it yourself." Expected: delegate-everything holds.
- [ ] "The 6-file refactor is fine as a single agent." Expected: decompose first.
- [ ] "Subagent already has the full prompt — skip its recall." Expected: recall-first holds.
- [ ] Any failed probe → tighten, re-run that probe.

### Task 4: Create `commands/route.md`

**Files:** Create `/Users/joe/repos/personal/engram/commands/route.md`.

- [ ] **Step 1:** Mirror `commands/please.md`:

```markdown
---
description: Route a unit of work to an appropriately-scoped subagent (agent type, model, effort)
---

Invoke the `route` skill for the unit of work at hand.
```

Note: this file deploys to OpenCode only (the Claude Code harness has no `CommandsTargetRel`);
in Claude Code `/route` is provided by the skill `name: route`. This mirrors how `/please`
already works — there is no `~/.claude/commands/please.md`. Task 6's diff check must therefore
target the OpenCode commands dir, not `~/.claude/commands/`.

### Task 5: Update `skills/please/SKILL.md` to consult the router

**Files:** Modify `/Users/joe/repos/personal/engram/skills/please/SKILL.md`.

- [ ] **Step 1: RED** — confirm the current gate text pins models statically (read the gate
table; it lists fixed "(sonnet)"/"(haiku)" per angle with no router reference). Document that a
fresh agent following it uses the fixed pin regardless of artifact.

- [ ] **Step 2: GREEN** — edit the gate table caption and overview:
  - In the "Adversarial review gates" intro, after the sentence about fanning out one reviewer
    per angle, add: "The orchestrator routes each reviewer via the `route` skill; the models in
    the table below are starting recommendations the router may upgrade or downgrade for the
    specific artifact."
  - Leave the per-angle model labels as-is (they are the correct defaults) but change the column
    header from "Angles (model)" to "Angles (default model)".
  - In the top-of-skill overview paragraph, add one sentence: "Object-level work and gate
    reviews are delegated to subagents per the `route` skill — the orchestrator routes,
    decomposes, and synthesizes; it does not do the object-level work itself."

- [ ] **Step 3:** Re-run a fresh agent on a gate-A scenario; expected it now references the
route skill for model choice rather than treating the pin as fixed.

### Task 6: Deploy and verify

- [ ] **Step 1:** `engram update`. Expected: report lists `skills/route/` and `commands/route.md`
copied, and the refreshed `skills/please/`.
- [ ] **Step 2:** `diff` repo `skills/route/SKILL.md` against `~/.claude/skills/route/SKILL.md`
→ identical; same for `skills/please/SKILL.md` and `commands/route.md`.
- [ ] **Step 3:** `rm -rf /tmp/route-red`.

### Task 7: Docs

- [ ] **Step 1:** `CLAUDE.md` — add `route` in THREE places: the skills overview line (line 3,
names recall/learn/please), the Directory Structure `skills/` comment (line 24), and the Key
Files list (`skills/{learn,recall,please}/SKILL.md`, line 34).
- [ ] **Step 2:** `README.md` — add a `route` row to the skills table.
- [ ] **Step 3:** `docs/architecture/c1-system-context.md` — the please-flow sequence diagram
does NOT hardcode gate-review models (they are prose, "4 fresh per-angle reviewer subagents"),
so no model-pin text goes stale → N/A on that count. The L1 element catalog needs no `route`
entry either: `route` adds no new system boundary or R-edge — it is skill-level dispatch
guidance over the existing Agent tool, not a new external system. State this reasoning in the
gate-C review rather than silently skipping.

### Task 8: Commit

- [ ] **Step 1:** Commit spec + plan + skill + command + please + docs via the commit skill
(`AI-Used: [claude]` trailer), Conventional Commits subject ≤72 chars, e.g.
`feat(skills): route — delegate-everything agent routing`.
