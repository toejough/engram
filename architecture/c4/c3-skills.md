---
level: 3
name: skills
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: 7da51d0c
---

# C3 — Skills (Component)

Refines L2's E7 Skills container into the six skill markdown files Claude Code loads on slash-command or auto-trigger. Each skill body returns instructions to the agent; most instruct it to shell out to the engram CLI binary. The c4 skill is the exception — it instructs the agent to use targ and edit architecture/c4/ directly, never calling the engram binary.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    s3(S3 · Claude Code<br/>agent harness)
    s2-n3[S2-N3 · engram CLI binary<br/>recall / learn / list / show / update]

    subgraph s2-n1 [S2-N1 · Skills]
        s2-n1-m1[S2-N1-M1 · prepare skill<br/>briefs the agent before new work]
        s2-n1-m2[S2-N1-M2 · learn skill<br/>saves session lessons]
        s2-n1-m3[S2-N1-M3 · recall skill<br/>loads prior session context]
        s2-n1-m4[S2-N1-M4 · remember skill<br/>explicit-knowledge capture]
        s2-n1-m5[S2-N1-M5 · migrate skill<br/>legacy memory upgrade]
        s2-n1-m6[S2-N1-M6 · c4 skill<br/>architecture diagrams]
    end

    s3 -->|"R1: loads skill body on /command or auto-trigger"| s2-n1
    s2-n1-m1 -->|"R2: instructs agent to call `engram recall` (with --query per topic)"| s2-n3
    s2-n1-m3 -->|"R3: instructs agent to call `engram recall` for prior session context"| s2-n3
    s2-n1-m2 -->|"R4: instructs agent to call `engram learn feedback` / `engram learn fact`"| s2-n3
    s2-n1-m4 -->|"R5: instructs agent to call `engram learn` or `engram update`"| s2-n3
    s2-n1-m5 -->|"R6: instructs agent to read legacy TOML and call `engram update` to rewrite each file"| s2-n3

    class s3 external
    class s2-n3 container
    class s2-n1-m1,s2-n1-m2,s2-n1-m3,s2-n1-m4,s2-n1-m5,s2-n1-m6 component
    class s2-n1 container

    click s2-n1 href "#s2-n1-skills" "Skills"
    click s3 href "#s3-claude-code" "Claude Code"
    click s2-n3 href "#s2-n3-engram-cli-binary" "engram CLI binary"
    click s2-n1-m1 href "#s2-n1-m1-prepare-skill" "prepare skill"
    click s2-n1-m2 href "#s2-n1-m2-learn-skill" "learn skill"
    click s2-n1-m3 href "#s2-n1-m3-recall-skill" "recall skill"
    click s2-n1-m4 href "#s2-n1-m4-remember-skill" "remember skill"
    click s2-n1-m5 href "#s2-n1-m5-migrate-skill" "migrate skill"
    click s2-n1-m6 href "#s2-n1-m6-c4-skill" "c4 skill"
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="s2-n1-skills"></a>S2-N1 | Skills | Container in focus | Markdown skill files Claude Code loads on /command or auto-trigger; bodies instruct the agent to call engram subcommands and present results. | — |
| <a id="s3-claude-code"></a>S3 | Claude Code | External system | Loads skill bodies on slash-command or auto-trigger; renders them into the agent's context as the next message. | — |
| <a id="s2-n3-engram-cli-binary"></a>S2-N3 | engram CLI binary | Container | Go binary that performs recall, learn, list, show, update. Refined in c3-engram-cli-binary.md. | — |
| <a id="s2-n1-m1-prepare-skill"></a>S2-N1-M1 | prepare skill | Component | Tells the agent to make 2–3 targeted `engram recall` queries by task and present a summary. | [../../skills/prepare/SKILL.md](../../skills/prepare/SKILL.md) |
| <a id="s2-n1-m2-learn-skill"></a>S2-N1-M2 | learn skill | Component | Reviews the recent session for learnable feedback/facts and walks the agent through saving them via `engram learn feedback` / `engram learn fact`. | [../../skills/learn/SKILL.md](../../skills/learn/SKILL.md) |
| <a id="s2-n1-m3-recall-skill"></a>S2-N1-M3 | recall skill | Component | Calls `engram recall` against the project's session transcripts and surfaces relevant memories. | [../../skills/recall/SKILL.md](../../skills/recall/SKILL.md) |
| <a id="s2-n1-m4-remember-skill"></a>S2-N1-M4 | remember skill | Component | Captures explicit knowledge the user dictates as feedback or fact memories with user approval, via `engram learn` or `engram update`. | [../../skills/remember/SKILL.md](../../skills/remember/SKILL.md) |
| <a id="s2-n1-m5-migrate-skill"></a>S2-N1-M5 | migrate skill | Component | Upgrades pre-cfd5fb5 (2026-04-17) flat-format memory files to the current split feedback/fact layout, calling `engram update` to rewrite each file. | [../../skills/migrate/SKILL.md](../../skills/migrate/SKILL.md) |
| <a id="s2-n1-m6-c4-skill"></a>S2-N1-M6 | c4 skill | Component | Generates and maintains C4 architecture diagrams under architecture/c4/. Uses targ c4-* targets; does not call the engram binary. | [../../skills/c4/SKILL.md](../../skills/c4/SKILL.md) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-s3-s2-n1"></a>R1 | S3 | S2-N1 | loads skill body on /command or auto-trigger | Plugin manifest, file read |
| <a id="r2-s2-n1-m1-s2-n3"></a>R2 | S2-N1-M1 | S2-N3 | instructs agent to call `engram recall` (with --query per topic) | Skill body text → agent Bash subprocess |
| <a id="r3-s2-n1-m3-s2-n3"></a>R3 | S2-N1-M3 | S2-N3 | instructs agent to call `engram recall` for prior session context | Skill body text → agent Bash subprocess |
| <a id="r4-s2-n1-m2-s2-n3"></a>R4 | S2-N1-M2 | S2-N3 | instructs agent to call `engram learn feedback` / `engram learn fact` | Skill body text → agent Bash subprocess |
| <a id="r5-s2-n1-m4-s2-n3"></a>R5 | S2-N1-M4 | S2-N3 | instructs agent to call `engram learn` or `engram update` | Skill body text → agent Bash subprocess |
| <a id="r6-s2-n1-m5-s2-n3"></a>R6 | S2-N1-M5 | S2-N3 | instructs agent to read legacy TOML and call `engram update` to rewrite each file | Skill body text → agent Bash subprocess |

## Cross-links

- Parent: [c2-engram-plugin.md](c2-engram-plugin.md) (refines **S2-N1 · Skills**)
- Siblings:
  - [c3-engram-cli-binary.md](c3-engram-cli-binary.md)
  - [c3-hooks.md](c3-hooks.md)
- Refined by: *(none yet)*
