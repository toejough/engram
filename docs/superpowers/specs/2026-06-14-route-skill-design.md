# Route skill + please integration — design

**Goal:** a reusable `route` skill that tells the orchestrator how to delegate every unit of
object-level work to an appropriately-scoped subagent — picking agent type, model, and effort —
and a please update so its review gates consult the router instead of carrying hardcoded model
pins.

**Decided with Joe (2026-06-14):**

- **Delegate everything — no inline escape.** The top-level agent orchestrates only: it routes,
  decomposes, and synthesizes. It does not do object-level work itself. Easy work is delegated
  to haiku (cheap, not skipped); the cost lever is the model axis, never a "do it inline" branch.
- **Three delegation shapes by task character:**
  - *Easy / mechanical* → delegate to a cheap model (haiku), low effort.
  - *Complex* → decompose into smaller units first, then delegate each unit.
  - *Deep thinking* → delegate to a fresh-context subagent that can focus, at high effort.
- **Subagents always `/recall` first** — every delegated subagent's first action is `/recall`
  before doing its own work (mirrors the gate reviewer protocol).
- **Router controls agentType + model + effort.**
- **please gates consult the router dynamically** — the gate table's per-angle models become
  *starting recommendations* the router confirms or overrides per task, not fixed pins.

**Considered and chosen (dissent recorded):** delegating *everything* means even a trivial
lookup costs a subagent round-trip. Joe chose this deliberately to keep the top-level agent a
pure orchestrator; the haiku tier is the cost mitigation. Not relitigated.

## What the router is — and is not

The `route` skill is **guidance the orchestrator applies when it chooses Agent-tool
parameters.** It is not an enforcement mechanism: Claude Code cannot let a skill or hook change
the main-loop model, and a subagent's model/effort are set only at dispatch time (agent
frontmatter or the per-invocation `model` parameter). So the router's output is a *decision* the
orchestrator then encodes into its `Agent(...)` call. This is the only viable form; the skill
says so plainly to forestall "why doesn't it just switch the model" confusion.

**What "dynamically" means here (explicit scope-down).** Joe asked the gates to consult the
router "dynamically." What is achievable is *advisory, not enforced*: the orchestrator must
choose to consult the route skill before each dispatch; no hook forces it. "Dynamic" therefore
means "the model/effort is selected per-artifact at dispatch time from the rubric" — not "the
running session reconfigures itself." This reduction is inherent to the platform (confirmed
prior turn), not a shortcut.

### Orchestration work vs object-level work (the boundary)

"Delegate everything" needs a crisp line, or it collapses into either "delegate nothing" or
"delegate the act of delegating." The boundary:

- **Orchestration work the top agent does itself:** routing/decomposition decisions, dispatching
  subagents, sequencing workflow steps, updating the task list, running the meta-skills that ARE
  the workflow (`/recall`, `/learn`, planning), and synthesizing subagents' returned results
  into the next decision or the user-facing report.
- **Object-level work the top agent delegates:** writing code or prose, running tests/builds,
  making creative or judgment calls on the artifact, reviewing an artifact (the gates) — anything
  that produces or evaluates the deliverable itself.

So please still *dispatches* its gate reviewers itself (orchestration); what changes is that it
*consults the router for each reviewer's model/effort* instead of reading a fixed pin.

The rubric deliberately aligns with the existing `~/.claude/commands/audit.md#Model Level
Selection` doctrine (Haiku = predictable/mechanical; Sonnet = moderate reasoning; Opus =
complex/nuanced; default cheap, upgrade on failure, reserve Opus) so the repo has one routing
doctrine, not two.

## The routing rubric

Given a unit of work, classify it and dispatch accordingly:

| Task character | agentType | model | effort | Notes |
| --- | --- | --- | --- | --- |
| Mechanical / predictable (formatting, status checks, template-fill, single-file lookup) | `Explore` (read-only) or `general-purpose` | haiku | low | The default. Cheap, not skipped. |
| Moderate reasoning (code review with context, TDD unit, triage, structured edit) | `general-purpose` or a domain agent | sonnet | medium | |
| Complex / nuanced judgment (architecture, cross-cutting refactor, hard debugging) | decompose first → delegate the pieces; if irreducible, a single focused agent | opus (or sonnet at high effort) | high | Decomposition is the orchestrator's job, done before dispatch. |
| Deep thinking (open-ended analysis, design exploration) | `general-purpose`, fresh context | opus | high | Delegated to a focused subagent precisely so it is not diluted by orchestrator context. |

Resolution doctrine (from audit.md): **default to the cheapest tier that can plausibly do the
unit; upgrade a tier if the cheaper one fails; reserve opus for units that genuinely need it.**

Every dispatched subagent is instructed to **run `/recall` as its first action**, with phrases
drawn from its unit, before doing the work.

Decomposition rule: a unit too large for one focused subagent is not dispatched as-is — the
orchestrator breaks it into smaller units and routes each. "Too large" = the unit spans multiple
files/concerns or needs more than one clear deliverable.

## please integration

- The gate table's "Angles (model)" column changes from fixed pins to **defaults the router may
  override**: the wording becomes "starting model; the orchestrator routes each angle via the
  `route` skill, which may upgrade/downgrade for the specific artifact." The current pins
  (sonnet for ask/code/design-fit; haiku for docs/clarity) remain the documented defaults
  because they are already correct applications of the rubric.
- A short pointer is added to the please overview: object-level work and gate reviews are
  delegated per the `route` skill; the orchestrator itself does not do the object-level work.
- The reviewer protocol's existing "first action is `/recall`" already matches the router's
  recall-first rule — cross-reference rather than duplicate.

## Files

- **Create** `skills/route/SKILL.md` — the rubric, the delegate-everything doctrine, the
  not-an-enforcement-mechanism caveat, the recall-first rule, the decomposition rule, a red-flags
  table.
- **Create** `commands/route.md` — thin slash-command pointer (mirrors `commands/please.md`).
- **Modify** `skills/please/SKILL.md` — gate table wording + overview pointer.
- **Modify** docs: `CLAUDE.md` (skills list), `README.md` (skills table), and the C4 L1
  `c1-system-context.md` if the please flow description references gate model pins.
- **No Go change** — `engram update` discovers skills/commands by directory scan
  (`planSkillCopies` / `planCommandCopies`), so `skills/route/` and `commands/route.md`
  deploy automatically. Note the harness asymmetry (update.go `supportedHarnesses`): the Claude
  Code harness has `SkillsTargetRel` but no `CommandsTargetRel`, so `commands/route.md` deploys
  to OpenCode only. In Claude Code the `/route` slash command is provided by the skill `name:
  route` itself — exactly how `/please` works today (there is no `~/.claude/commands/please.md`).
  So the command file is the OpenCode pointer, not dead weight.

## Testing (writing-skills TDD)

- **RED:** a fresh agent, given a mixed batch of units (one mechanical, one moderate, one
  complex-and-decomposable, one deep-thinking) WITHOUT the route skill, is asked how it would
  staff them. Baseline: it does them inline or picks a single model for all, with no
  decomposition and no recall-first instruction.
- **GREEN:** same batch WITH the route skill — it delegates each, picks the tiered model+effort,
  decomposes the complex one, and instructs each subagent to recall first; nothing is done
  inline.
- **Pressure tests:** "this one's trivial, just answer it" (must still delegate to haiku);
  "tokens are tight, do it yourself" (delegate-everything holds; haiku is the lever); "the
  complex task is fine as one big agent" (must decompose); "skip the subagent's recall, it has
  the prompt" (recall-first holds).

## Out of scope

- No change to how subagents are dispatched mechanically (the Agent tool is unchanged).
- No attempt to switch the main-loop model — impossible by design; the skill documents this.
- please's seven-step structure and the four gates are unchanged; only model-selection wording
  inside them moves to the router.
