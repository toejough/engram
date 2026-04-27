---
level: 1
name: engram-system
parent: null
children: []
last_reviewed_commit: 653b5e77
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

    s1([S1 · Developer<br/>uses Claude Code])
    s2[S2 · Engram plugin]
    s3(S3 · Claude Code<br/>agent harness)
    s4(S4 · Claude Code memory surfaces<br/>CLAUDE.md, .claude/rules, auto-memory, skills)
    s5(S5 · Anthropic API<br/>Haiku rank + extract)
    s6(S6 · Engram memory store<br/>~/.local/share/engram/memory/)

    s1 -->|"R1: Invokes slash-commands and writes prompts that trigger skill auto-invocation"| s3
    s3 -->|"R2: Loads skill markdown, executes hooks (`SessionStart`, `UserPromptSubmit`, `PostToolUse`), invokes `engram` binary subcommands"| s2
    s2 -->|"R3: Ranks memory/skill/auto-memory candidates and extracts snippets during recall; classifies feedback/facts during learn"| s5
    s2 -->|"R4: Discovers and reads CLAUDE.md (+ `@`-imports), `.claude/rules/*.md`, auto-memory topic files, and skill frontmatter for ranking"| s4
    s2 <-->|"R5: Reads and writes Engram's own feedback/fact TOML; reads/writes the cached binary"| s6
    s2 -->|"R6: Returns briefings (`/prepare`), recall results (`/recall`), and hook reminders that re-enter the agent's context"| s3

    class s1 person
    class s3,s4,s5,s6 external
    class s2 container

    click s1 href "#s1-developer" "Developer"
    click s2 href "#s2-engram-plugin" "Engram plugin"
    click s3 href "#s3-claude-code" "Claude Code"
    click s4 href "#s4-claude-code-memory-surfaces" "Claude Code memory surfaces"
    click s5 href "#s5-anthropic-api" "Anthropic API"
    click s6 href "#s6-engram-memory-store" "Engram memory store"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="s1-developer"></a>S1 | Developer | Person | Developer who triggers `/prepare`, `/recall`, `/remember`, `/learn`, `/migrate` and authors the work that produces memories | Human, at a Claude Code session |
| <a id="s2-engram-plugin"></a>S2 | Engram plugin | The system in scope | Plugin providing persistent, query-ranked memory: skills decide when to load context, a slim Go binary computes recall/learn, hooks remind the agent at session and tool-use boundaries | This repository (`github.com/toejough/engram`) |
| <a id="s3-claude-code"></a>S3 | Claude Code | External system | Agent harness that loads the plugin, dispatches skills, fires hooks, and exposes its own memory surfaces | Anthropic Claude Code CLI |
| <a id="s4-claude-code-memory-surfaces"></a>S4 | Claude Code memory surfaces | External system | Read-only sources Engram merges into recall: project + user `CLAUDE.md` (with `@`-imports), `.claude/rules/*.md`, auto-memory under `~/.claude/projects/<slug>/memory/`, and project + user + plugin skill frontmatter | Files owned by Claude Code and the user; never written by Engram |
| <a id="s5-anthropic-api"></a>S5 | Anthropic API | External system | LLM service used by the recall pipeline for Haiku ranking and extraction; also the classification step in `/learn` and `/remember` quality gates | `api.anthropic.com` |
| <a id="s6-engram-memory-store"></a>S6 | Engram memory store | External system | Engram's own writable data directory: `~/.local/share/engram/memory/feedback/*.toml` and `~/.local/share/engram/memory/facts/*.toml`. The filesystem belongs to the OS; Engram only reads and writes within these paths | XDG data home on the user's machine |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-developer-claude-code"></a>R1 | Developer | Claude Code | Invokes slash-commands and writes prompts that trigger skill auto-invocation | Claude Code CLI / TTY |
| <a id="r2-claude-code-engram-plugin"></a>R2 | Claude Code | Engram plugin | Loads skill markdown, executes hooks (`SessionStart`, `UserPromptSubmit`, `PostToolUse`), invokes `engram` binary subcommands | Plugin manifest, shell hooks (stdin JSON), subprocess exec |
| <a id="r3-engram-plugin-anthropic-api"></a>R3 | Engram plugin | Anthropic API | Ranks memory/skill/auto-memory candidates and extracts snippets during recall; classifies feedback/facts during learn | HTTPS, Anthropic Messages API (Haiku) |
| <a id="r4-engram-plugin-claude-code-memory-surfaces"></a>R4 | Engram plugin | Claude Code memory surfaces | Discovers and reads CLAUDE.md (+ `@`-imports), `.claude/rules/*.md`, auto-memory topic files, and skill frontmatter for ranking | Local file reads (read-only; "read everywhere, write only what you own") |
| <a id="r5-engram-plugin-engram-memory-store"></a>R5 | Engram plugin | Engram memory store | Reads and writes Engram's own feedback/fact TOML; reads/writes the cached binary | Local file I/O, TOML |
| <a id="r6-engram-plugin-claude-code"></a>R6 | Engram plugin | Claude Code | Returns briefings (`/prepare`), recall results (`/recall`), and hook reminders that re-enter the agent's context | Hook stdout JSON (`systemMessage`, `additionalContext`) |

## Cross-links

- Parent: none (L1 is the root).
- Refined by: *(none yet)*
