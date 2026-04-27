---
level: 1
name: engram-system
parent: null
children: []
last_reviewed_commit: df51bc93
---

# C1 — Engram plugin (System Context)

Engram is a Claude Code plugin that gives the agent persistent, query-ranked memory.
This diagram shows who and what Engram interacts with at the system boundary; it
deliberately hides the CLI binary, hooks, on-disk stores, and skills (those live at L2).

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    s1([S1 · Joe<br/>developer using Claude Code])
    s2[S2 · Engram plugin]
    s3(S3 · Claude Code<br/>agent harness)
    s4(S4 · Claude Code memory surfaces<br/>CLAUDE.md, .claude/rules, auto-memory, skills)
    s5(S5 · Anthropic API<br/>Haiku rank + extract)
    s6(S6 · Local filesystem<br/>~/.local/share/engram/)

    s1 -->|"R1: invokes /prepare /learn /recall /remember /migrate"| s3
    s3 -->|"R2: loads skills + fires hooks (SessionStart, UserPromptSubmit, PostToolUse)"| s2
    s2 -->|"R3: ranks + extracts via Haiku"| s5
    s2 -->|"R4: reads memories, rules, skills, auto-memory"| s4
    s2 <-->|"R5: reads + writes feedback/ and facts/ TOML"| s6
    s2 -->|"R6: injects briefings and reminders back into context"| s3

    class s1 person
    class s3,s4,s5,s6 external
    class s2 container

    click s1 href "#s1-joe" "Joe"
    click s2 href "#s2-engram-plugin" "Engram plugin"
    click s3 href "#s3-claude-code" "Claude Code"
    click s4 href "#s4-claude-code-memory-surfaces" "Claude Code memory surfaces"
    click s5 href "#s5-anthropic-api" "Anthropic API"
    click s6 href "#s6-local-filesystem" "Local filesystem"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="s1-joe"></a>S1 | Joe | Person | Developer who triggers `/prepare`, `/recall`, `/remember`, `/learn`, `/migrate` and authors the work that produces memories | Human, at a Claude Code session |
| <a id="s2-engram-plugin"></a>S2 | Engram plugin | The system in scope | Plugin providing persistent, query-ranked memory: skills decide when to load context, a slim Go binary computes recall/learn, hooks remind the agent at session and tool-use boundaries | This repository (`github.com/toejough/engram`) |
| <a id="s3-claude-code"></a>S3 | Claude Code | External system | Agent harness that loads the plugin, dispatches skills, fires hooks, and exposes its own memory surfaces | Anthropic Claude Code CLI |
| <a id="s4-claude-code-memory-surfaces"></a>S4 | Claude Code memory surfaces | External system | Read-only sources Engram merges into recall: project + user `CLAUDE.md` (with `@`-imports), `.claude/rules/*.md`, auto-memory under `~/.claude/projects/<slug>/memory/`, and project + user + plugin skill frontmatter | Files owned by Claude Code and the user; never written by Engram |
| <a id="s5-anthropic-api"></a>S5 | Anthropic API | External system | LLM service used by the recall pipeline for Haiku ranking and extraction; also the classification step in `/learn` and `/remember` quality gates | `api.anthropic.com` |
| <a id="s6-local-filesystem"></a>S6 | Local filesystem | External system | Engram's own writable data directory: `~/.local/share/engram/memory/feedback/*.toml` and `~/.local/share/engram/memory/facts/*.toml`; also the cached binary at `~/.claude/engram/bin/engram` | XDG data home on the user's machine |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-joe-claude-code"></a>R1 | Joe | Claude Code | invokes /prepare /learn /recall /remember /migrate | Claude Code CLI |
| <a id="r2-claude-code-engram-plugin"></a>R2 | Claude Code | Engram plugin | loads skills + fires hooks (SessionStart, UserPromptSubmit, PostToolUse) | plugin loader + hooks |
| <a id="r3-engram-plugin-anthropic-api"></a>R3 | Engram plugin | Anthropic API | ranks + extracts via Haiku | HTTPS |
| <a id="r4-engram-plugin-claude-code-memory-surfaces"></a>R4 | Engram plugin | Claude Code memory surfaces | reads memories, rules, skills, auto-memory | filesystem |
| <a id="r5-engram-plugin-local-filesystem"></a>R5 | Engram plugin | Local filesystem | reads + writes feedback/ and facts/ TOML | filesystem |
| <a id="r6-engram-plugin-claude-code"></a>R6 | Engram plugin | Claude Code | injects briefings and reminders back into context | stdout |

## Cross-links

- Parent: none (L1 is the root).
- Refined by: *(none yet)*
