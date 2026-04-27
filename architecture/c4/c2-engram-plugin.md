---
level: 2
name: engram-plugin
parent: "c1-engram-system.md"
children: []
last_reviewed_commit: ff9fc22a
---

# C2 — Engram plugin (Container)

Refines L1's E2 Engram plugin into three internal containers — skill markdown files that drive agent behavior, shell hooks fired on Claude Code lifecycle events, and a Go CLI binary that performs all computation. External actors and the on-disk store keep their L1 E-IDs.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff

    s1([S1 · Developer<br/>uses Claude Code])
    s3(S3 · Claude Code<br/>agent harness)
    s4(S4 · Claude Code memory surfaces)
    s5(S5 · Anthropic API<br/>Haiku)
    s6(S6 · Engram memory store<br/>~/.local/share/engram/memory/)

    subgraph s2 [S2 · Engram plugin]
        s2-n1[S2-N1 · Skills<br/>prepare / learn / recall / remember / migrate / c4]
        s2-n2[S2-N2 · Hooks<br/>session-start / user-prompt-submit / post-tool-use]
        s2-n3[S2-N3 · engram CLI binary<br/>recall · learn · list · show · update]
    end

    s1 -->|"R1: invokes /prepare, /learn, /recall, /remember, /migrate"| s3
    s3 -->|"R2: loads skill markdown on /command and on auto-trigger"| s2-n1
    s3 -->|"R3: fires SessionStart, UserPromptSubmit, PostToolUse"| s2-n2
    s3 -->|"R4: execs the binary as a subprocess each time the agent's Bash tool runs an engram subcommand"| s2-n3
    s2-n1 -->|"R5: skill bodies returned to agent context include instructions to shell out to engram subcommands"| s3
    s2-n2 -->|"R6: session-start.sh runs go build and writes a fresh binary when source files are newer than cached mtime"| s2-n3
    s2-n2 -->|"R7: emit hookSpecificOutput.additionalContext (and systemMessage) on stdout to inject reminders and the skill-availability banner"| s3
    s2-n3 -->|"R8: ranks candidates and extracts snippets via Haiku; classifies feedback/facts during learn"| s5
    s2-n3 -->|"R9: reads CLAUDE.md, .claude/rules, auto-memory, skill frontmatter for ranking"| s4
    s2-n3 -->|"R10: reads existing feedback + fact TOML during recall/list/show; writes new TOML during learn/remember/update"| s6
    s2-n3 -->|"R11: prints briefings, recall results, and other subcommand output to stdout, which re-enters the agent's context as the tool result"| s3

    class s1 person
    class s3,s4,s5,s6 external
    class s2-n1,s2-n2,s2-n3 container
    class s2 container

    click s1 href "#s1-developer" "Developer"
    click s2 href "#s2-engram-plugin" "Engram plugin"
    click s3 href "#s3-claude-code" "Claude Code"
    click s4 href "#s4-claude-code-memory-surfaces" "Claude Code memory surfaces"
    click s5 href "#s5-anthropic-api" "Anthropic API"
    click s6 href "#s6-engram-memory-store" "Engram memory store"
    click s2-n1 href "#s2-n1-skills" "Skills"
    click s2-n2 href "#s2-n2-hooks" "Hooks"
    click s2-n3 href "#s2-n3-engram-cli-binary" "engram CLI binary"
```

## Element Catalog

| ID | Name | Type | Responsibility | System of Record |
|---|---|---|---|---|
| <a id="s1-developer"></a>S1 | Developer | Person | Engineer who triggers slash-commands and writes prompts that produce memories | Human, at a Claude Code session |
| <a id="s2-engram-plugin"></a>S2 | Engram plugin | The system in scope | Plugin providing persistent, query-ranked memory: skills decide when to load context, a slim Go binary computes recall/learn, hooks remind the agent at session and tool-use boundaries | This repository (`github.com/toejough/engram`) |
| <a id="s3-claude-code"></a>S3 | Claude Code | External system | Agent harness that loads skills, fires hooks, and execs the engram binary when the agent shells out | Anthropic Claude Code CLI |
| <a id="s4-claude-code-memory-surfaces"></a>S4 | Claude Code memory surfaces | External system | Read-only inputs to recall ranking: project + user `CLAUDE.md`, `.claude/rules/*.md`, auto-memory, skill frontmatter | Owned by Claude Code and the user |
| <a id="s5-anthropic-api"></a>S5 | Anthropic API | External system | Haiku model used for candidate ranking, snippet extraction, and learn-time classification | `api.anthropic.com` |
| <a id="s6-engram-memory-store"></a>S6 | Engram memory store | External system | On-disk memory state: feedback TOML under `~/.local/share/engram/memory/feedback/` and fact TOML under `~/.local/share/engram/memory/facts/`. The filesystem belongs to the OS; Engram only reads and writes within these paths | XDG data home on the user's machine |
| <a id="s2-n1-skills"></a>S2-N1 | Skills | The system in scope | Markdown skill files (`skills/{prepare,learn,recall,remember,migrate,c4}/SKILL.md`) that Claude Code loads on command or auto-trigger; bodies instruct the agent to call `engram` subcommands and present results | This repo, under `skills/` |
| <a id="s2-n2-hooks"></a>S2-N2 | Hooks | The system in scope | Three bash scripts (`hooks/session-start.sh`, `hooks/user-prompt-submit.sh`, `hooks/post-tool-use.sh`) wired by `hooks/hooks.json`; emit JSON `additionalContext`, async-rebuild the binary on SessionStart | This repo, under `hooks/` |
| <a id="s2-n3-engram-cli-binary"></a>S2-N3 | engram CLI binary | The system in scope | Go binary (entry `cmd/engram/main.go`) implementing subcommands `recall`, `learn {feedback,fact}`, `list`, `show`, `update`. All I/O lives here; pure logic in `internal/{recall,memory,cli,…}`. Built by `session-start.sh` to `~/.claude/engram/bin/engram`; execed by Claude Code | This repo |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-s1-s3"></a>R1 | S1 | S3 | invokes /prepare, /learn, /recall, /remember, /migrate | Claude Code CLI / TTY |
| <a id="r2-s3-s2-n1"></a>R2 | S3 | S2-N1 | loads skill markdown on /command and on auto-trigger | Plugin manifest, file read |
| <a id="r3-s3-s2-n2"></a>R3 | S3 | S2-N2 | fires SessionStart, UserPromptSubmit, PostToolUse | Subprocess exec, stdin JSON |
| <a id="r4-s3-s2-n3"></a>R4 | S3 | S2-N3 | execs the binary as a subprocess each time the agent's Bash tool runs an engram subcommand | Subprocess exec |
| <a id="r5-s2-n1-s3"></a>R5 | S2-N1 | S3 | skill bodies returned to agent context include instructions to shell out to engram subcommands | Skill body text rendered into context |
| <a id="r6-s2-n2-s2-n3"></a>R6 | S2-N2 | S2-N3 | session-start.sh runs go build and writes a fresh binary when source files are newer than cached mtime | go build, file mtime check, file write |
| <a id="r7-s2-n2-s3"></a>R7 | S2-N2 | S3 | emit hookSpecificOutput.additionalContext (and systemMessage) on stdout to inject reminders and the skill-availability banner | Hook stdout JSON |
| <a id="r8-s2-n3-s5"></a>R8 | S2-N3 | S5 | ranks candidates and extracts snippets via Haiku; classifies feedback/facts during learn | HTTPS, Anthropic Messages API (Haiku) |
| <a id="r9-s2-n3-s4"></a>R9 | S2-N3 | S4 | reads CLAUDE.md, .claude/rules, auto-memory, skill frontmatter for ranking | Local file reads (read-only) |
| <a id="r10-s2-n3-s6"></a>R10 | S2-N3 | S6 | reads existing feedback + fact TOML during recall/list/show; writes new TOML during learn/remember/update | Local file I/O, TOML |
| <a id="r11-s2-n3-s3"></a>R11 | S2-N3 | S3 | prints briefings, recall results, and other subcommand output to stdout, which re-enters the agent's context as the tool result | stdout |

## Cross-links

- Parent: [c1-engram-system.md](c1-engram-system.md) (refines **S2 · Engram plugin**)
- Refined by: *(none yet)*
