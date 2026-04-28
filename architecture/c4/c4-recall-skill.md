---
level: 4
name: recall-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 658e2ee3
---

# C4 — recall-skill (Property/Invariant Ledger)

> Component in focus: **S2-N1-M3 · recall skill**.
> Source files in scope:
> - [skills/recall/SKILL.md](skills/recall/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E12. R-edges cite the
P-list each edge backs.

![C4 recall-skill context diagram](svg/c4-recall-skill.svg)

> Diagram source: [svg/c4-recall-skill.mmd](svg/c4-recall-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-recall-skill.mmd -o architecture/c4/svg/c4-recall-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R-edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="s2-n1-m3-p1-two-invocation-modes"></a>S2-N1-M3-P1 | Two invocation modes | For all invocations of the recall skill, the body distinguishes exactly two modes: a no-args mode triggered by `/recall` with no query, and a query mode triggered by `/recall <query>`. | [skills/recall/SKILL.md:13](../../skills/recall/SKILL.md#L13) | **⚠ UNTESTED** | No behavioral test under `skills/recall/`. |
| <a id="s2-n1-m3-p2-calls-engram-recall-no-args"></a>S2-N1-M3-P2 | Calls `engram recall` (no-args) | For all no-args invocations, the skill instructs the agent to run `engram recall` with no flags. | [skills/recall/SKILL.md:17](../../skills/recall/SKILL.md#L17) | **⚠ UNTESTED** | Backs L3 R3 for the no-args branch. |
| <a id="s2-n1-m3-p3-calls-engram-recall-query-query-mode"></a>S2-N1-M3-P3 | Calls `engram recall --query` (query mode) | For all query-mode invocations, the skill instructs the agent to run `engram recall --query "<the user's query>"` with the user-supplied query string. | [skills/recall/SKILL.md:30](../../skills/recall/SKILL.md#L30) | **⚠ UNTESTED** | Backs L3 R3 for the query branch. |
| <a id="s2-n1-m3-p4-summarizes-no-args-output"></a>S2-N1-M3-P4 | Summarizes no-args output | For all completed no-args invocations, the skill instructs the agent to summarize the output covering: what was being discussed/decided, what work was done (filtering mundane tool calls), and which memories were active. | [skills/recall/SKILL.md:21](../../skills/recall/SKILL.md#L21) | **⚠ UNTESTED** | Three-bullet summary contract; output is not silently consumed. |
| <a id="s2-n1-m3-p5-presents-query-results"></a>S2-N1-M3-P5 | Presents query results | For all completed query-mode invocations, the skill instructs the agent to present the filtered results to the user. | [skills/recall/SKILL.md:34](../../skills/recall/SKILL.md#L34) | **⚠ UNTESTED** | Closes the loop in query mode — results surface to the user. |
| <a id="s2-n1-m3-p6-trigger-phrases-declared"></a>S2-N1-M3-P6 | Trigger phrases declared | For all skill loads, the description frontmatter declares the trigger phrases (`/recall`, "what was I working on", "load previous context", "search session history", resume work from a previous session) used by Claude Code's skill router to dispatch to this skill. | [skills/recall/SKILL.md:3](../../skills/recall/SKILL.md#L3) | **⚠ UNTESTED** | Discoverability contract; routing is performed by the harness, not the skill body. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **S2-N1-M3 · recall skill**)
- Siblings:
  - [c4-c4-skill.md](c4-c4-skill.md)
  - [c4-learn-skill.md](c4-learn-skill.md)
  - [c4-migrate-skill.md](c4-migrate-skill.md)
  - [c4-prepare-skill.md](c4-prepare-skill.md)
  - [c4-remember-skill.md](c4-remember-skill.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

