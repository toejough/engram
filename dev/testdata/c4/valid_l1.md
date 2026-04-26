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

    e1([E1 · Joe<br/>developer using Claude Code])
    e2[E2 · Engram plugin]
    e3(E3 · Claude Code<br/>agent harness)
    e4(E4 · Claude Code memory surfaces<br/>CLAUDE.md, .claude/rules, auto-memory, skills)
    e5(E5 · Anthropic API<br/>Haiku rank + extract)
    e6(E6 · Local filesystem<br/>~/.local/share/engram/)

    e1 -->|"R1: invokes /prepare /learn /recall /remember /migrate"| e3
    e3 -->|"R2: loads skills + fires hooks (SessionStart, UserPromptSubmit, PostToolUse)"| e2
    e2 -->|"R3: ranks + extracts via Haiku"| e5
    e2 -->|"R4: reads memories, rules, skills, auto-memory"| e4
    e2 -->|"R5: reads + writes feedback/ and facts/ TOML"| e6
    e6 -->|"R5: reads + writes feedback/ and facts/ TOML"| e2
    e2 -->|"R6: injects briefings and reminders back into context"| e3

    class e1 person
    class e3,e4,e5,e6 external
    class e2 container

    click e1 href "#e1-joe" "Joe"
    click e2 href "#e2-engram-plugin" "Engram plugin"
    click e3 href "#e3-claude-code" "Claude Code"
    click e4 href "#e4-claude-code-memory-surfaces" "Claude Code memory surfaces"
    click e5 href "#e5-anthropic-api" "Anthropic API"
    click e6 href "#e6-local-filesystem" "Local filesystem"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="e1-joe"></a>E1 | Joe | Person | Developer who triggers `/prepare`, `/recall`, `/remember`, `/learn`, `/migrate` and authors the work that produces memories | Human, at a Claude Code session |
| <a id="e2-engram-plugin"></a>E2 | Engram plugin | The system in scope | Plugin providing persistent, query-ranked memory: skills decide when to load context, a slim Go binary computes recall/learn, hooks remind the agent at session and tool-use boundaries | This repository (`github.com/toejough/engram`) |
| <a id="e3-claude-code"></a>E3 | Claude Code | External system | Agent harness that loads the plugin, dispatches skills, fires hooks, and exposes its own memory surfaces | Anthropic Claude Code CLI |
| <a id="e4-claude-code-memory-surfaces"></a>E4 | Claude Code memory surfaces | External system | Read-only sources Engram merges into recall: project + user `CLAUDE.md` (with `@`-imports), `.claude/rules/*.md`, auto-memory under `~/.claude/projects/<slug>/memory/`, and project + user + plugin skill frontmatter | Files owned by Claude Code and the user; never written by Engram |
| <a id="e5-anthropic-api"></a>E5 | Anthropic API | External system | LLM service used by the recall pipeline for Haiku ranking and extraction; also the classification step in `/learn` and `/remember` quality gates | `api.anthropic.com` |
| <a id="e6-local-filesystem"></a>E6 | Local filesystem | External system | Engram's own writable data directory: `~/.local/share/engram/memory/feedback/*.toml` and `~/.local/share/engram/memory/facts/*.toml`; also the cached binary at `~/.claude/engram/bin/engram` | XDG data home on the user's machine |

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
