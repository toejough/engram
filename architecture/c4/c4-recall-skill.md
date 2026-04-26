---
level: 4
name: recall-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — recall-skill (Property/Invariant Ledger)

> Component in focus: **E12 · recall skill** (refines L3 c3-skills).
> Source files in scope:
> - [../../skills/recall/SKILL.md](../../skills/recall/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E12. R-edges cite the
P-list each edge backs.

![C4 recall-skill context diagram](svg/c4-recall-skill.svg)

> Diagram source: [svg/c4-recall-skill.mmd](svg/c4-recall-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-recall-skill.mmd -o architecture/c4/svg/c4-recall-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-two-modes"></a>P1 | Two-mode dispatch | For all invocations, the skill instructs the agent to dispatch on whether the user supplied a query: no-args runs `engram recall`; query-mode runs `engram recall --query "<text>"`. | [skills/recall/SKILL.md:13](../../skills/recall/SKILL.md#L13) | **⚠ UNTESTED** | No behavioral test under `skills/recall/`. |
| <a id="p2-no-args-bare"></a>P2 | No-args runs bare command | For all `/recall` invocations with no query, the skill instructs the agent to call `engram recall` with no flags. | [skills/recall/SKILL.md:17](../../skills/recall/SKILL.md#L17) | **⚠ UNTESTED** | Bare command surfaces prior session context. |
| <a id="p3-query-mode-flag"></a>P3 | Query mode passes `--query` | For all `/recall <query>` invocations, the skill instructs the agent to pass the user's literal query string to `engram recall --query`. | [skills/recall/SKILL.md:30](../../skills/recall/SKILL.md#L30) | **⚠ UNTESTED** | Backs L3 R3. |
| <a id="p4-summarizes-output"></a>P4 | Summarizes for user | For all no-args runs, the skill instructs the agent to summarize what was discussed/decided, what work was done, and what memories were active — filtering mundane tool calls. | [skills/recall/SKILL.md:21](../../skills/recall/SKILL.md#L21) | **⚠ UNTESTED** | Output presentation, not raw dump. |
| <a id="p5-query-mode-presents"></a>P5 | Query-mode presents results | For all query-mode runs, the skill instructs the agent to present the filtered recall results to the user. | [skills/recall/SKILL.md:34](../../skills/recall/SKILL.md#L34) | **⚠ UNTESTED** | Closes the loop. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **E12 · recall skill**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
