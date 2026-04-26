---
level: 4
name: prepare-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — prepare-skill (Property/Invariant Ledger)

> Component in focus: **E10 · prepare skill** (refines L3 c3-skills).
> Source files in scope:
> - [../../skills/prepare/SKILL.md](../../skills/prepare/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E10. R-edges cite the
P-list each edge backs.

![C4 prepare-skill context diagram](svg/c4-prepare-skill.svg)

> Diagram source: [svg/c4-prepare-skill.mmd](svg/c4-prepare-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-prepare-skill.mmd -o architecture/c4/svg/c4-prepare-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-three-step-flow"></a>P1 | Three-step flow | For all invocations of the prepare skill, the body instructs the agent to execute exactly three numbered steps: analyze the situation, make targeted recall queries, present a briefing. | [skills/prepare/SKILL.md:13](../../skills/prepare/SKILL.md#L13) | **⚠ UNTESTED** | No behavioral test under `skills/prepare/`. |
| <a id="p2-query-budget"></a>P2 | Bounded query count | For all situations the agent analyzes, the skill instructs the agent to issue 2–3 targeted `engram recall` queries — not zero, not many. | [skills/prepare/SKILL.md:23](../../skills/prepare/SKILL.md#L23) | **⚠ UNTESTED** | Cap is content-level guidance only; no enforcement mechanism. |
| <a id="p3-task-shaped-queries"></a>P3 | Task-shaped queries | For all queries the agent constructs, the skill instructs them to be phrased by task (activity + domain), not by anticipated failure mode ("query by task, not by fear"). | [skills/prepare/SKILL.md:36](../../skills/prepare/SKILL.md#L36) | **⚠ UNTESTED** | Aligns query phrasing with how `learn`/`remember` write situations. |
| <a id="p4-calls-engram-recall"></a>P4 | Calls `engram recall` | For all queries issued by the agent, the skill instructs invocation of `engram recall --query "<topic>"` (no other subcommand). | [skills/prepare/SKILL.md:27](../../skills/prepare/SKILL.md#L27) | **⚠ UNTESTED** | Backs L3 R2. |
| <a id="p5-presents-briefing"></a>P5 | Presents briefing to user | For all completed runs of the skill, the skill instructs the agent to summarize the recalled context for the user before proceeding to work. | [skills/prepare/SKILL.md:41](../../skills/prepare/SKILL.md#L41) | **⚠ UNTESTED** | Closes the loop — recall results are not silently consumed. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **E10 · prepare skill**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
