---
level: 4
name: learn-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — learn-skill (Property/Invariant Ledger)

> Component in focus: **E11 · learn skill** (refines L3 c3-skills).
> Source files in scope:
> - [../../skills/learn/SKILL.md](../../skills/learn/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E11. R-edges cite the
P-list each edge backs.

![C4 learn-skill context diagram](svg/c4-learn-skill.svg)

> Diagram source: [svg/c4-learn-skill.mmd](svg/c4-learn-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-learn-skill.mmd -o architecture/c4/svg/c4-learn-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-five-step-flow"></a>P1 | Five-step flow | For all invocations of the learn skill, the body instructs the agent to execute five numbered steps: identify candidates, run quality gates, draft, persist, handle results. | [skills/learn/SKILL.md:13](../../skills/learn/SKILL.md#L13) | **⚠ UNTESTED** | No behavioral test under `skills/learn/`. |
| <a id="p2-recurs-gate"></a>P2 | Recurs gate | For all candidate memories, the skill instructs the agent to drop any whose situation names this project, internals, phase numbers, issue IDs, dates, or one-time events. | [skills/learn/SKILL.md:26](../../skills/learn/SKILL.md#L26) | **⚠ UNTESTED** | Filters out non-portable memories. |
| <a id="p3-actionable-gate"></a>P3 | Actionable gate | For all candidate memories, the skill instructs the agent to drop any that do not name a concrete action that changes future behavior. | [skills/learn/SKILL.md:37](../../skills/learn/SKILL.md#L37) | **⚠ UNTESTED** | Filters out vague observations and inert facts. |
| <a id="p4-right-home-gate"></a>P4 | Right-home gate | For all candidate memories, the skill instructs the agent to name an alternative home (code, doc, skill, CLAUDE.md, `.claude/rules/`, or `docs/` spec/plan) and verify against it via `git log` + read before persisting. | [skills/learn/SKILL.md:41](../../skills/learn/SKILL.md#L41) | **⚠ UNTESTED** | Three-way action table for verification outcome. |
| <a id="p5-task-shaped-situation"></a>P5 | Task-shaped situation | For all surviving candidates, the skill instructs the agent to write the `situation` field as activity + domain (matching how `/prepare` would query), not as diagnosis/symptom/fix. | [skills/learn/SKILL.md:65](../../skills/learn/SKILL.md#L65) | **⚠ UNTESTED** | Hindsight-bias prevention. |
| <a id="p6-source-agent-default"></a>P6 | `--source agent` default | For all autonomous task-boundary runs (not user-invoked /learn), the skill instructs the agent to persist with `--source agent`; user-invoked /learn presents findings for approval first. | [skills/learn/SKILL.md:75](../../skills/learn/SKILL.md#L75) | **⚠ UNTESTED** | Source attribution affects later dedup/contradiction handling. |
| <a id="p7-calls-engram-learn"></a>P7 | Calls `engram learn` | For all surviving candidates, the skill instructs invocation of `engram learn feedback` (corrections/failures) or `engram learn fact` (knowledge/patterns) with the SBIA or SPO field set. | [skills/learn/SKILL.md:79](../../skills/learn/SKILL.md#L79) | **⚠ UNTESTED** | Backs L3 R4. |
| <a id="p8-duplicate-broadens"></a>P8 | DUPLICATE broadens situation | For all `engram learn` results returning DUPLICATE, the skill instructs the agent to diagnose surfacing failure and update the existing memory's situation via `engram update --name ... --situation "..."`, never to dismiss. | [skills/learn/SKILL.md:87](../../skills/learn/SKILL.md#L87) | **⚠ UNTESTED** | Surfacing failure is the cause, not the candidate's redundancy. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **E11 · learn skill**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
