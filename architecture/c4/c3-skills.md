---
level: 3
name: skills
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: 1ba7e162
---

# C3 — Skills (Component)

Refines L2's E7 Skills container into the six skill markdown files Claude Code loads on slash-command or auto-trigger. Each skill body returns instructions to the agent; most instruct it to shell out to the engram CLI binary. The c4 skill is the exception — it instructs the agent to use targ and edit architecture/c4/ directly, never calling the engram binary.

![C3 skills diagram](svg/c3-skills.svg)

> Diagram source: [svg/c3-skills.mmd](svg/c3-skills.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c3-skills.mmd -o architecture/c4/svg/c3-skills.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="e7-skills"></a>E7 | Skills | Container in focus | Markdown skill files Claude Code loads on /command or auto-trigger; bodies instruct the agent to call engram subcommands and present results. | — |
| <a id="e3-claude-code"></a>E3 | Claude Code | External system | Loads skill bodies on slash-command or auto-trigger; renders them into the agent's context as the next message. | — |
| <a id="e9-engram-cli-binary"></a>E9 | engram CLI binary | Container | Go binary that performs recall, learn, list, show, update. Refined in c3-engram-cli-binary.md. | — |
| <a id="e10-prepare-skill"></a>E10 | prepare skill | Component | Tells the agent to make 2–3 targeted `engram recall` queries by task and present a summary. | [../../skills/prepare/SKILL.md](../../skills/prepare/SKILL.md) |
| <a id="e11-learn-skill"></a>E11 | learn skill | Component | Reviews the recent session for learnable feedback/facts and walks the agent through saving them via `engram learn feedback` / `engram learn fact`. | [../../skills/learn/SKILL.md](../../skills/learn/SKILL.md) |
| <a id="e12-recall-skill"></a>E12 | recall skill | Component | Calls `engram recall` against the project's session transcripts and surfaces relevant memories. | [../../skills/recall/SKILL.md](../../skills/recall/SKILL.md) |
| <a id="e13-remember-skill"></a>E13 | remember skill | Component | Captures explicit knowledge the user dictates as feedback or fact memories with user approval, via `engram learn` or `engram update`. | [../../skills/remember/SKILL.md](../../skills/remember/SKILL.md) |
| <a id="e14-migrate-skill"></a>E14 | migrate skill | Component | Upgrades pre-cfd5fb5 (2026-04-17) flat-format memory files to the current split feedback/fact layout, calling `engram update` to rewrite each file. | [../../skills/migrate/SKILL.md](../../skills/migrate/SKILL.md) |
| <a id="e15-c4-skill"></a>E15 | c4 skill | Component | Generates and maintains C4 architecture diagrams under architecture/c4/. Uses targ c4-* targets; does not call the engram binary. | [../../skills/c4/SKILL.md](../../skills/c4/SKILL.md) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-claude-code-skills"></a>R1 | Claude Code | Skills | loads skill body on /command or auto-trigger | Plugin manifest, file read |
| <a id="r2-prepare-skill-engram-cli-binary"></a>R2 | prepare skill | engram CLI binary | instructs agent to call `engram recall` (with --query per topic) | Skill body text → agent Bash subprocess |
| <a id="r3-recall-skill-engram-cli-binary"></a>R3 | recall skill | engram CLI binary | instructs agent to call `engram recall` for prior session context | Skill body text → agent Bash subprocess |
| <a id="r4-learn-skill-engram-cli-binary"></a>R4 | learn skill | engram CLI binary | instructs agent to call `engram learn feedback` / `engram learn fact` | Skill body text → agent Bash subprocess |
| <a id="r5-remember-skill-engram-cli-binary"></a>R5 | remember skill | engram CLI binary | instructs agent to call `engram learn` or `engram update` | Skill body text → agent Bash subprocess |
| <a id="r6-migrate-skill-engram-cli-binary"></a>R6 | migrate skill | engram CLI binary | instructs agent to read legacy TOML and call `engram update` to rewrite each file | Skill body text → agent Bash subprocess |

## Cross-links

- Parent: [c2-engram-plugin.md](c2-engram-plugin.md) (refines **E7 · Skills**)
- Siblings:
  - [c3-engram-cli-binary.md](c3-engram-cli-binary.md)
  - [c3-hooks.md](c3-hooks.md)
- Refined by: *(none yet)*
