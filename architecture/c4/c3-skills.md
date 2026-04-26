---
level: 3
name: skills
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: 44cec351
---

# C3 — Skills (Component)

Refines L2's E7 Skills container into the six skill markdown files Claude Code loads on slash-command or auto-trigger. Each skill body returns instructions to the agent; most instruct it to shell out to the engram CLI binary. The c4 skill is the exception — it instructs the agent to use targ and edit architecture/c4/ directly, never calling the engram binary.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    e3(E3 · Claude Code<br/>agent harness)
    e9[E9 · engram CLI binary<br/>recall / learn / list / show / update]

    subgraph e7 [E7 · Skills]
        e10[E10 · prepare skill<br/>briefs the agent before new work]
        e11[E11 · learn skill<br/>saves session lessons]
        e12[E12 · recall skill<br/>loads prior session context]
        e13[E13 · remember skill<br/>explicit-knowledge capture]
        e14[E14 · migrate skill<br/>legacy memory upgrade]
        e15[E15 · c4 skill<br/>architecture diagrams]
    end

    e3 -->|"R1: loads skill body on /command or auto-trigger"| e7
    e10 -->|"R2: instructs agent to call `engram recall` (with --query per topic)"| e9
    e12 -->|"R3: instructs agent to call `engram recall` for prior session context"| e9
    e11 -->|"R4: instructs agent to call `engram learn feedback` / `engram learn fact`"| e9
    e13 -->|"R5: instructs agent to call `engram learn` or `engram update`"| e9
    e14 -->|"R6: instructs agent to read legacy TOML and call `engram update` to rewrite each file"| e9

    class e3 external
    class e9 container
    class e10,e11,e12,e13,e14,e15 component
    class e7 container

    click e7 href "#e7-skills" "Skills"
    click e3 href "#e3-claude-code" "Claude Code"
    click e9 href "#e9-engram-cli-binary" "engram CLI binary"
    click e10 href "#e10-prepare-skill" "prepare skill"
    click e11 href "#e11-learn-skill" "learn skill"
    click e12 href "#e12-recall-skill" "recall skill"
    click e13 href "#e13-remember-skill" "remember skill"
    click e14 href "#e14-migrate-skill" "migrate skill"
    click e15 href "#e15-c4-skill" "c4 skill"
```

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
- Siblings: *(none)*
- Refined by: *(none yet)*
