---
level: 2
name: engram-plugin
parent: "c1-engram-system.md"
children: []
last_reviewed_commit: 0eb52f06
---

# C2 — Engram plugin (Container)

Refines L1's E2 Engram plugin into three internal containers — skill markdown files that drive agent behavior, shell hooks fired on Claude Code lifecycle events, and a Go CLI binary that performs all computation. External actors and the on-disk store keep their L1 E-IDs.

![C2 engram plugin diagram](svg/c2-engram-plugin.svg)

> Diagram source: [svg/c2-engram-plugin.mmd](svg/c2-engram-plugin.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c2-engram-plugin.mmd -o architecture/c4/svg/c2-engram-plugin.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="e1-developer"></a>E1 | Developer | Person | Engineer who triggers slash-commands and writes prompts that produce memories | Human, at a Claude Code session |
| <a id="e2-engram-plugin"></a>E2 | Engram plugin | The system in scope | Plugin providing persistent, query-ranked memory: skills decide when to load context, a slim Go binary computes recall/learn, hooks remind the agent at session and tool-use boundaries | This repository (`github.com/toejough/engram`) |
| <a id="e3-claude-code"></a>E3 | Claude Code | External system | Agent harness that loads skills, fires hooks, and execs the engram binary when the agent shells out | Anthropic Claude Code CLI |
| <a id="e4-claude-code-memory-surfaces"></a>E4 | Claude Code memory surfaces | External system | Read-only inputs to recall ranking: project + user `CLAUDE.md`, `.claude/rules/*.md`, auto-memory, skill frontmatter | Owned by Claude Code and the user |
| <a id="e5-anthropic-api"></a>E5 | Anthropic API | External system | Haiku model used for candidate ranking, snippet extraction, and learn-time classification | `api.anthropic.com` |
| <a id="e6-engram-memory-store"></a>E6 | Engram memory store | External system | On-disk memory state: feedback TOML under `~/.local/share/engram/memory/feedback/` and fact TOML under `~/.local/share/engram/memory/facts/`. The filesystem belongs to the OS; Engram only reads and writes within these paths | XDG data home on the user's machine |
| <a id="e7-skills"></a>E7 | Skills | Container | Markdown skill files (`skills/{prepare,learn,recall,remember,migrate,c4}/SKILL.md`) that Claude Code loads on command or auto-trigger; bodies instruct the agent to call `engram` subcommands and present results | This repo, under `skills/` |
| <a id="e8-hooks"></a>E8 | Hooks | Container | Three bash scripts (`hooks/session-start.sh`, `hooks/user-prompt-submit.sh`, `hooks/post-tool-use.sh`) wired by `hooks/hooks.json`; emit JSON `additionalContext`, async-rebuild the binary on SessionStart | This repo, under `hooks/` |
| <a id="e9-engram-cli-binary"></a>E9 | engram CLI binary | Container | Go binary (entry `cmd/engram/main.go`) implementing subcommands `recall`, `learn {feedback,fact}`, `list`, `show`, `update`. All I/O lives here; pure logic in `internal/{recall,memory,cli,…}`. Built by `session-start.sh` to `~/.claude/engram/bin/engram`; execed by Claude Code | This repo |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-developer-claude-code"></a>R1 | Developer | Claude Code | invokes /prepare, /learn, /recall, /remember, /migrate | Claude Code CLI / TTY |
| <a id="r2-claude-code-skills"></a>R2 | Claude Code | Skills | loads skill markdown on /command and on auto-trigger | Plugin manifest, file read |
| <a id="r3-claude-code-hooks"></a>R3 | Claude Code | Hooks | fires SessionStart, UserPromptSubmit, PostToolUse | Subprocess exec, stdin JSON |
| <a id="r4-claude-code-engram-cli-binary"></a>R4 | Claude Code | engram CLI binary | execs the binary as a subprocess each time the agent's Bash tool runs an engram subcommand | Subprocess exec |
| <a id="r5-skills-claude-code"></a>R5 | Skills | Claude Code | skill bodies returned to agent context include instructions to shell out to engram subcommands | Skill body text rendered into context |
| <a id="r6-hooks-engram-cli-binary"></a>R6 | Hooks | engram CLI binary | session-start.sh runs go build and writes a fresh binary when source files are newer than cached mtime | go build, file mtime check, file write |
| <a id="r7-hooks-claude-code"></a>R7 | Hooks | Claude Code | emit hookSpecificOutput.additionalContext (and systemMessage) on stdout to inject reminders and the skill-availability banner | Hook stdout JSON |
| <a id="r8-engram-cli-binary-anthropic-api"></a>R8 | engram CLI binary | Anthropic API | ranks candidates and extracts snippets via Haiku; classifies feedback/facts during learn | HTTPS, Anthropic Messages API (Haiku) |
| <a id="r9-engram-cli-binary-claude-code-memory-surfaces"></a>R9 | engram CLI binary | Claude Code memory surfaces | reads CLAUDE.md, .claude/rules, auto-memory, skill frontmatter for ranking | Local file reads (read-only) |
| <a id="r10-engram-cli-binary-engram-memory-store"></a>R10 | engram CLI binary | Engram memory store | reads existing feedback + fact TOML during recall/list/show; writes new TOML during learn/remember/update | Local file I/O, TOML |
| <a id="r11-engram-cli-binary-claude-code"></a>R11 | engram CLI binary | Claude Code | prints briefings, recall results, and other subcommand output to stdout, which re-enters the agent's context as the tool result | stdout |

## Cross-links

- Parent: [c1-engram-system.md](c1-engram-system.md) (refines **E2 · Engram plugin**)
- Refined by: *(none yet)*
