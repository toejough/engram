---
level: 1
name: engram-system
parent: null
children: [c2-engram-plugin.md]
last_reviewed_commit: 63f069ee
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

    e1([E1 · Developer<br/>uses Claude Code])
    e2[E2 · Engram plugin]
    e3(E3 · Claude Code<br/>agent harness)
    e4(E4 · Claude Code memory surfaces<br/>CLAUDE.md, .claude/rules, auto-memory, skills)
    e5(E5 · Anthropic API<br/>Haiku rank + extract)
    e6(E6 · Local filesystem<br/>~/.local/share/engram/)

    e1 -->|"R1: Invokes slash-commands and writes prompts that trigger skill auto-invocation"| e3
    e3 -->|"R2: Loads skill markdown, executes hooks (`SessionStart`, `UserPromptSubmit`, `PostToolUse`), invokes `engram` binary subcommands"| e2
    e2 -->|"R3: Ranks memory/skill/auto-memory candidates and extracts snippets during recall; classifies feedback/facts during learn"| e5
    e2 -->|"R4: Discovers and reads CLAUDE.md (+ `@`-imports), `.claude/rules/*.md`, auto-memory topic files, and skill frontmatter for ranking"| e4
    e2 -->|"R5: Reads and writes Engram's own feedback/fact TOML; reads/writes the cached binary"| e6
    e2 -->|"R6: Returns briefings (`/prepare`), recall results (`/recall`), and hook reminders that re-enter the agent's context"| e3

    class e1 person
    class e3,e4,e5,e6 external
    class e2 container

    click e1 href "#e1-developer" "Developer"
    click e2 href "#e2-engram-plugin" "Engram plugin"
    click e3 href "#e3-claude-code" "Claude Code"
    click e4 href "#e4-claude-code-memory-surfaces" "Claude Code memory surfaces"
    click e5 href "#e5-anthropic-api" "Anthropic API"
    click e6 href "#e6-local-filesystem" "Local filesystem"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="e1-developer"></a>E1 | Developer | Person | Engineer who triggers `/prepare`, `/recall`, `/remember`, `/learn`, `/migrate` and authors the work that produces memories | Human, at a Claude Code session |
| <a id="e2-engram-plugin"></a>E2 | Engram plugin | The system in scope | Plugin providing persistent, query-ranked memory: skills decide when to load context, a slim Go binary computes recall/learn, hooks remind the agent at session and tool-use boundaries | This repository (`github.com/toejough/engram`) |
| <a id="e3-claude-code"></a>E3 | Claude Code | External system | Agent harness that loads the plugin, dispatches skills, fires hooks, and exposes its own memory surfaces | Anthropic Claude Code CLI |
| <a id="e4-claude-code-memory-surfaces"></a>E4 | Claude Code memory surfaces | External system | Read-only sources Engram merges into recall: project + user `CLAUDE.md` (with `@`-imports), `.claude/rules/*.md`, auto-memory under `~/.claude/projects/<slug>/memory/`, and project + user + plugin skill frontmatter | Files owned by Claude Code and the user; never written by Engram |
| <a id="e5-anthropic-api"></a>E5 | Anthropic API | External system | LLM service used by the recall pipeline for Haiku ranking and extraction; also the classification step in `/learn` and `/remember` quality gates | `api.anthropic.com` |
| <a id="e6-local-filesystem"></a>E6 | Local filesystem | External system | Engram's own writable data directory: `~/.local/share/engram/memory/feedback/*.toml` and `~/.local/share/engram/memory/facts/*.toml`; also the cached binary at `~/.claude/engram/bin/engram` | XDG data home on the user's machine |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-developer-claude-code"></a>R1 | Developer | Claude Code | Invokes slash-commands and writes prompts that trigger skill auto-invocation | Claude Code CLI / TTY |
| <a id="r2-claude-code-engram-plugin"></a>R2 | Claude Code | Engram plugin | Loads skill markdown, executes hooks (`SessionStart`, `UserPromptSubmit`, `PostToolUse`), invokes `engram` binary subcommands | Plugin manifest, shell hooks (stdin JSON), subprocess exec |
| <a id="r3-engram-plugin-anthropic-api"></a>R3 | Engram plugin | Anthropic API | Ranks memory/skill/auto-memory candidates and extracts snippets during recall; classifies feedback/facts during learn | HTTPS, Anthropic Messages API (Haiku) |
| <a id="r4-engram-plugin-claude-code-memory-surfaces"></a>R4 | Engram plugin | Claude Code memory surfaces | Discovers and reads CLAUDE.md (+ `@`-imports), `.claude/rules/*.md`, auto-memory topic files, and skill frontmatter for ranking | Local file reads (read-only; "read everywhere, write only what you own") |
| <a id="r5-engram-plugin-local-filesystem"></a>R5 | Engram plugin | Local filesystem | Reads and writes Engram's own feedback/fact TOML; reads/writes the cached binary | Local file I/O, TOML |
| <a id="r6-engram-plugin-claude-code"></a>R6 | Engram plugin | Claude Code | Returns briefings (`/prepare`), recall results (`/recall`), and hook reminders that re-enter the agent's context | Hook stdout JSON (`systemMessage`, `additionalContext`) |

## Cross-links

- Parent: none (L1 is the root).
- Refined by: [c2-engram-plugin.md](c2-engram-plugin.md) — refines **E2 · Engram plugin**
