---
level: 1
name: audit-clean
parent: null
children: []
last_reviewed_commit: a7b7fc34
---

# C1 — Engram (System Context)

Engram is a Claude Code plugin that gives the agent persistent, query-ranked memory.
This diagram shows who and what Engram interacts with at the system boundary; it
deliberately hides the CLI binary, hooks, on-disk stores, and skills (those live at L2).

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    user([S1 · Joe<br/>developer using Claude Code])
    cc(S3 · Claude Code<br/>agent harness)
    ccmem(S4 · Claude Code memory surfaces<br/>CLAUDE.md, .claude/rules, auto-memory, skills)
    anth(S5 · Anthropic API<br/>Haiku rank + extract)
    fs(S6 · Local filesystem<br/>~/.local/share/engram/)
    engram[S2 · Engram plugin]

    user -->|R1: invokes /prepare /learn /recall /remember /migrate| cc
    cc -->|R2: loads skills + fires SessionStart, UserPromptSubmit, PostToolUse hooks| engram
    engram -->|R3: ranks + extracts via Haiku| anth
    engram -->|R4: reads memories, rules, skills, auto-memory| ccmem
    engram <-->|R5: reads + writes feedback/ and facts/ TOML| fs
    engram -->|R6: injects briefings and reminders back into context| cc

    class user person
    class cc,ccmem,anth,fs external
    class engram container

    click user href "#s1-joe" "Joe"
    click engram href "#s2-engram-plugin" "Engram plugin"
    click cc href "#s3-claude-code" "Claude Code"
    click ccmem href "#s4-claude-code-memory-surfaces" "Claude Code memory surfaces"
    click anth href "#s5-anthropic-api" "Anthropic API"
    click fs href "#s6-local-filesystem" "Local filesystem"
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
| <a id="r1-joe-cc"></a>R1 | Joe | Claude Code | Invokes slash-commands and writes prompts that trigger skill auto-invocation | Claude Code CLI / TTY |
| <a id="r2-cc-engram"></a>R2 | Claude Code | Engram plugin | Loads skill markdown, executes hooks (`SessionStart`, `UserPromptSubmit`, `PostToolUse`), invokes `engram` binary subcommands | Plugin manifest, shell hooks (stdin JSON), subprocess exec |
| <a id="r3-engram-anth"></a>R3 | Engram plugin | Anthropic API | Ranks memory/skill/auto-memory candidates and extracts snippets during recall; classifies feedback/facts during learn | HTTPS, Anthropic Messages API (Haiku) |
| <a id="r4-engram-ccmem"></a>R4 | Engram plugin | Claude Code memory surfaces | Discovers and reads CLAUDE.md (+ `@`-imports), `.claude/rules/*.md`, auto-memory topic files, and skill frontmatter for ranking | Local file reads (read-only; "read everywhere, write only what you own") |
| <a id="r5-engram-fs"></a>R5 | Engram plugin | Local filesystem | Reads and writes Engram's own feedback/fact TOML; reads/writes the cached binary | Local file I/O, TOML |
| <a id="r6-engram-cc"></a>R6 | Engram plugin | Claude Code | Returns briefings (`/prepare`), recall results (`/recall`), and hook reminders that re-enter the agent's context | Hook stdout JSON (`systemMessage`, `additionalContext`) |

## Cross-links

- Parent: none (L1 is the root).
- Refined by: *(none yet — to be authored at L2)*. Expected next file: `c2-engram-containers.md` decomposing **S2 · Engram plugin** into the `engram` Go binary (`cmd/engram` + `internal/`), the skill set (`skills/{prepare,recall,learn,remember,migrate,c4}`), the hook scripts (`hooks/{session-start,user-prompt-submit,post-tool-use}.sh`), and the on-disk memory store under `~/.local/share/engram/memory/`.
