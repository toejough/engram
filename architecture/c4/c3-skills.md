---
level: 3
name: skills
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: abb1f55e
---

# C3 — Skills (Component)

Refines L2's E7 Skills container into the five skill markdown files Claude Code loads on slash-command or auto-trigger. Each skill body returns instructions to the agent; most instruct it to shell out to the engram CLI binary. The c4 skill is the exception — it instructs the agent to use targ and edit architecture/c4/ directly, never calling the engram binary.

![C3 skills component diagram](svg/c3-skills.svg)

> Diagram source: [svg/c3-skills.mmd](svg/c3-skills.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c3-skills.mmd -o architecture/c4/svg/c3-skills.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R-edges between the same node pair.

## Element Catalog

| ID | Name | Type | Responsibility | Source |
|---|---|---|---|---|
| <a id="s2-n1-skills"></a>S2-N1 | Skills | Container in focus | Markdown skill files Claude Code loads on /command or auto-trigger; bodies instruct the agent to call engram subcommands and present results. | — |
| <a id="s3-claude-code"></a>S3 | Claude Code | External system | Loads skill bodies on slash-command or auto-trigger; renders them into the agent's context as the next message. | — |
| <a id="s2-n3-engram-cli-binary"></a>S2-N3 | engram CLI binary | Container | Go binary that performs recall, learn, list, show, update. Refined in c3-engram-cli-binary.md. | — |
| <a id="s2-n1-m2-learn-skill"></a>S2-N1-M2 | learn skill | Component | Reviews the recent session (autonomously at completion boundaries, or when the user invokes `/learn`) for lessons that pass Recurs + Activity-and-Domain + Knowledge gates, then writes vault notes via `engram promote`. Unified successor to the prior capturing-fleeting-notes / promoting-to-permanent-notes / learn / remember skill set. | [../../skills/learn/SKILL.md](../../skills/learn/SKILL.md) |
| <a id="s2-n1-m3-recall-skill"></a>S2-N1-M3 | recall skill | Component | Drives a frontier-expansion cascade via `engram recall` (anchors / --recent / --follow + --already-read) against the agent-memory vault and synthesizes surfaced notes for the LLM caller. Absorbs the prior prepare skill's pre-work briefing role. | [../../skills/recall/SKILL.md](../../skills/recall/SKILL.md) |
| <a id="s2-n1-m5-migrate-skill"></a>S2-N1-M5 | migrate skill | Component | Upgrades pre-cfd5fb5 (2026-04-17) flat-format memory files to the current split feedback/fact layout, calling `engram update` to rewrite each file. | [../../skills/migrate/SKILL.md](../../skills/migrate/SKILL.md) |
| <a id="s2-n1-m6-c4-skill"></a>S2-N1-M6 | c4 skill | Component | Generates and maintains C4 architecture diagrams under architecture/c4/. Uses targ c4-* targets; does not call the engram binary. | [../../skills/c4/SKILL.md](../../skills/c4/SKILL.md) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-s3-s2-n1"></a>R1 | S3 | S2-N1 | loads skill body on /command or auto-trigger | Plugin manifest, file read |
| <a id="r3-s2-n1-m3-s2-n3"></a>R3 | S2-N1-M3 | S2-N3 | instructs agent to drive a frontier-expansion cascade calling `engram recall` (anchors / --recent / --follow + --already-read) against the agent-memory vault | Skill body text → agent Bash subprocess |
| <a id="r4-s2-n1-m2-s2-n3"></a>R4 | S2-N1-M2 | S2-N3 | instructs agent to call `engram promote` to write vault notes for surviving lesson candidates | Skill body text → agent Bash subprocess |
| <a id="r6-s2-n1-m5-s2-n3"></a>R6 | S2-N1-M5 | S2-N3 | instructs agent to read legacy TOML and call `engram update` to rewrite each file | Skill body text → agent Bash subprocess |

## Cross-links

- Parent: [c2-engram-plugin.md](c2-engram-plugin.md) (refines **S2-N1 · Skills**)
- Siblings:
  - [c3-engram-cli-binary.md](c3-engram-cli-binary.md)
  - [c3-hooks.md](c3-hooks.md)
- Refined by: *(none yet)*
