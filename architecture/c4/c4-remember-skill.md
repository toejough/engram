---
level: 4
name: remember-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — remember-skill (Property/Invariant Ledger)

> Component in focus: **E13 · remember skill** (refines L3 c3-skills).
> Source files in scope:
> - [../../skills/remember/SKILL.md](../../skills/remember/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E13. R-edges cite the
P-list each edge backs.

![C4 remember-skill context diagram](svg/c4-remember-skill.svg)

> Diagram source: [svg/c4-remember-skill.mmd](svg/c4-remember-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-remember-skill.mmd -o architecture/c4/svg/c4-remember-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-five-step-flow"></a>P1 | Five-step flow | For all invocations of the remember skill, the body instructs the agent to classify, run quality gates, draft + present, save, and handle results. | [skills/remember/SKILL.md:11](../../skills/remember/SKILL.md#L11) | **⚠ UNTESTED** | No behavioral test under `skills/remember/`. |
| <a id="p2-classify-feedback-or-fact"></a>P2 | Classify feedback vs fact | For all candidates, the skill instructs the agent to classify each as feedback (SBIA) or fact (SPO), and to split a multi-memory utterance into multiple candidates. | [skills/remember/SKILL.md:15](../../skills/remember/SKILL.md#L15) | **⚠ UNTESTED** | Direct quotes can yield two facts. |
| <a id="p3-quality-gates"></a>P3 | Three quality gates | For all candidates, the skill instructs the agent to apply Recurs, Actionable, and Right-home gates in order; a single failure drops the candidate with explanation to the user. | [skills/remember/SKILL.md:21](../../skills/remember/SKILL.md#L21) | **⚠ UNTESTED** | Mirrors learn skill's gates. |
| <a id="p4-user-approval"></a>P4 | User approval before save | For all surviving candidates, the skill instructs the agent to draft all fields and present them for explicit user approval before invoking `engram learn`. | [skills/remember/SKILL.md:60](../../skills/remember/SKILL.md#L60) | **⚠ UNTESTED** | Distinguishes /remember from autonomous /learn. |
| <a id="p5-task-shaped-situation"></a>P5 | Task-shaped situation | For all drafts, the skill instructs the agent to phrase `situation` as activity + domain matching how `/prepare` would query — not as diagnosis/symptom/fix. | [skills/remember/SKILL.md:63](../../skills/remember/SKILL.md#L63) | **⚠ UNTESTED** | Same hindsight-bias rule as learn. |
| <a id="p6-source-human"></a>P6 | `--source human` | For all saves driven by `/remember`, the skill instructs the agent to pass `--source human` (the user dictated the memory). | [skills/remember/SKILL.md:73](../../skills/remember/SKILL.md#L73) | **⚠ UNTESTED** | Distinguishes user-explicit from agent-derived memories. |
| <a id="p7-calls-engram-learn-or-update"></a>P7 | Calls `engram learn` / `engram update` | For all approved candidates, the skill instructs invocation of `engram learn feedback`, `engram learn fact`, or (on DUPLICATE) `engram update`. | [skills/remember/SKILL.md:73](../../skills/remember/SKILL.md#L73) | **⚠ UNTESTED** | Backs L3 R5. |
| <a id="p8-contradiction-asks"></a>P8 | CONTRADICTION asks user | For all `engram learn` results returning CONTRADICTION, the skill instructs the agent to present the conflict and ask the user whether to update existing, replace, or keep both. | [skills/remember/SKILL.md:81](../../skills/remember/SKILL.md#L81) | **⚠ UNTESTED** | User-facing skill must not auto-resolve contradictions. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **E13 · remember skill**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
