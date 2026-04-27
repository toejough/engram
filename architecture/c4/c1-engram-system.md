---
level: 1
name: engram-system
parent: null
children: []
last_reviewed_commit: 1caf804c
---

# C1 — Engram plugin (System Context)

Engram is a Claude Code plugin that gives the agent persistent, query-ranked memory.
This diagram shows who and what Engram interacts with at the system boundary; it
deliberately hides the CLI binary, hooks, on-disk stores, and skills (those live at L2).

![C1 engram-system system context](svg/c1-engram-system.svg)

> Diagram source: [svg/c1-engram-system.mmd](svg/c1-engram-system.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c1-engram-system.mmd -o architecture/c4/svg/c1-engram-system.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

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
