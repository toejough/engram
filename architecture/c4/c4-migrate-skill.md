---
level: 4
name: migrate-skill
parent: "c3-skills.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — migrate-skill (Property/Invariant Ledger)

> Component in focus: **E14 · migrate skill** (refines L3 c3-skills).
> Source files in scope:
> - [../../skills/migrate/SKILL.md](../../skills/migrate/SKILL.md)

## Context (from L3)

Scoped slice of [c3-skills.md](c3-skills.md): the L3 edges that touch E14. R-edges cite the
P-list each edge backs.

![C4 migrate-skill context diagram](svg/c4-migrate-skill.svg)

> Diagram source: [svg/c4-migrate-skill.mmd](svg/c4-migrate-skill.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-migrate-skill.mmd -o architecture/c4/svg/c4-migrate-skill.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-ten-step-flow"></a>P1 | Ten-step flow | For all invocations of the migrate skill, the body instructs the agent through ten numbered steps: locate, read, classify, rewrite situation, build file, pick source, drop fields, write destination, verify, archive. | [skills/migrate/SKILL.md:30](../../skills/migrate/SKILL.md#L30) | **⚠ UNTESTED** | No behavioral test under `skills/migrate/`. |
| <a id="p2-read-before-classify"></a>P2 | Read before classify | For all legacy files, the skill instructs the agent to read the full file before classifying — never batch-migrate by filename alone. | [skills/migrate/SKILL.md:43](../../skills/migrate/SKILL.md#L43) | **⚠ UNTESTED** | Red flag listed explicitly. |
| <a id="p3-skip-current-format"></a>P3 | Skip current-format files | For all files with `schema_version = 2` and a `[content]` sub-table, the skill instructs the agent to skip them — they are already migrated. | [skills/migrate/SKILL.md:21](../../skills/migrate/SKILL.md#L21) | **⚠ UNTESTED** | Idempotency property. |
| <a id="p4-classify-or-stop"></a>P4 | Ambiguity stops | For all candidates whose feedback-vs-fact classification is unclear, the skill instructs the agent to STOP and ask the user — never to default. | [skills/migrate/SKILL.md:59](../../skills/migrate/SKILL.md#L59) | **⚠ UNTESTED** | Misclassification poisons recall. |
| <a id="p5-rewrite-situation"></a>P5 | Rewrite hindsight-biased situations | For all legacy situations that describe the problem already hit (rather than the task an agent would be starting), the skill instructs the agent to rewrite to task shape (activity + domain) before writing the new file. | [skills/migrate/SKILL.md:62](../../skills/migrate/SKILL.md#L62) | **⚠ UNTESTED** | Core principle of the skill. |
| <a id="p6-source-default-agent"></a>P6 | Source defaults to agent | For all legacy files lacking a direct user quotation and lacking `confidence = "A"`, the skill instructs the agent to set `source = "agent"`; otherwise `"human"`. | [skills/migrate/SKILL.md:107](../../skills/migrate/SKILL.md#L107) | **⚠ UNTESTED** | Reported speech does NOT count as direct quotation. |
| <a id="p7-drop-legacy-fields"></a>P7 | Drop legacy fields | For all legacy files, the skill instructs the agent to drop `surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count`, `project_scoped`, `confidence`, `keywords`, `concepts`, and `title` unconditionally — without asking the user. | [skills/migrate/SKILL.md:117](../../skills/migrate/SKILL.md#L117) | **⚠ UNTESTED** | Removed by design. |
| <a id="p8-fully-qualified-paths"></a>P8 | Fully qualified destination paths | For all writes, the skill instructs the agent to use absolute paths under `~/.local/share/engram/memory/{feedback,facts}/`, never bare relative paths. | [skills/migrate/SKILL.md:128](../../skills/migrate/SKILL.md#L128) | **⚠ UNTESTED** | Explicit "always fully qualified" section. |
| <a id="p9-verify-each-file"></a>P9 | Verify each file | For all migrated files, the skill instructs the agent to run `engram show --name <slug>` and fix any parse error before moving to the next file. | [skills/migrate/SKILL.md:134](../../skills/migrate/SKILL.md#L134) | **⚠ UNTESTED** | Backs L3 R6. |
| <a id="p10-archive-not-delete"></a>P10 | Archive, do not delete | For all completed migrations, the skill instructs the agent to rename the legacy directory to `memories.legacy-migrated-<date>` and keep it until at least one real session has confirmed surfacing — never delete. | [skills/migrate/SKILL.md:146](../../skills/migrate/SKILL.md#L146) | **⚠ UNTESTED** | Recovery hatch. |

## Cross-links

- Parent: [c3-skills.md](c3-skills.md) (refines **E14 · migrate skill**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
