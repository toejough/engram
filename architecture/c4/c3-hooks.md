---
level: 3
name: hooks
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: 1ba7e162
---

# C3 — Hooks (Component)

Refines L2's E8 Hooks container into the manifest plus three bash scripts wired to Claude Code lifecycle events. Hooks emit hookSpecificOutput.additionalContext JSON on stdout to inject reminders into the agent's context. The SessionStart hook additionally rebuilds the Go binary asynchronously when any source file is newer than the cached binary, isolating the build from the agent's own Bash provenance so macOS doesn't SIGKILL it on exec.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    e3(E3 · Claude Code<br/>agent harness)
    e9[E9 · engram CLI binary<br/>Go binary built by session-start.sh]

    subgraph e8 [E8 · Hooks]
        e16[E16 · hooks.json<br/>manifest]
        e17[E17 · session-start.sh<br/>skill announcement + async rebuild]
        e18[E18 · user-prompt-submit.sh<br/>/prepare /learn nudge]
        e19[E19 · post-tool-use.sh<br/>/prepare /learn nudge]
    end

    e3 -->|"R1: reads manifest at plugin load"| e16
    e16 -->|"R2: registers SessionStart -> session-start.sh (timeout 10s)"| e17
    e16 -->|"R3: registers UserPromptSubmit -> user-prompt-submit.sh (timeout 5s)"| e18
    e16 -->|"R4: registers PostToolUse -> post-tool-use.sh (timeout 5s)"| e19
    e3 -->|"R5: fires lifecycle events as subprocess execs with JSON stdin"| e8
    e17 -->|"R6: emits SessionStart additionalContext announcing memory skills"| e3
    e18 -->|"R7: emits UserPromptSubmit additionalContext nudging /prepare and /learn"| e3
    e19 -->|"R8: emits PostToolUse additionalContext nudging /prepare and /learn"| e3
    e17 -->|"R9: async go build to ~/.claude/engram/bin/engram when any *.go is newer than the cached binary"| e9

    class e3 external
    class e9 container
    class e16,e17,e18,e19 component
    class e8 container

    click e8 href "#e8-hooks" "Hooks"
    click e3 href "#e3-claude-code" "Claude Code"
    click e9 href "#e9-engram-cli-binary" "engram CLI binary"
    click e16 href "#e16-hooks-json" "hooks.json"
    click e17 href "#e17-session-start-sh" "session-start.sh"
    click e18 href "#e18-user-prompt-submit-sh" "user-prompt-submit.sh"
    click e19 href "#e19-post-tool-use-sh" "post-tool-use.sh"
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="e8-hooks"></a>E8 | Hooks | Container in focus | Three bash scripts wired by hooks/hooks.json; emit JSON additionalContext, async-rebuild the binary on SessionStart. | — |
| <a id="e3-claude-code"></a>E3 | Claude Code | External system | Reads the hook manifest at plugin load and execs the registered scripts on lifecycle events; consumes their stdout JSON to inject context. | — |
| <a id="e9-engram-cli-binary"></a>E9 | engram CLI binary | Container | Go binary built and refreshed by session-start.sh. Refined in c3-engram-cli-binary.md. | — |
| <a id="e16-hooks-json"></a>E16 | hooks.json | Component | Manifest mapping SessionStart, UserPromptSubmit, and PostToolUse events to their scripts via ${CLAUDE_PLUGIN_ROOT} paths and per-event timeouts (10s / 5s / 5s). | [../../hooks/hooks.json](../../hooks/hooks.json) |
| <a id="e17-session-start-sh"></a>E17 | session-start.sh | Component | Synchronously emits the memory-skill announcement. Asynchronously rebuilds the Go binary when any *.go is newer than the cached binary mtime, deleting the prior binary first to avoid macOS provenance SIGKILL, then symlinks ~/.local/bin/engram to it. | [../../hooks/session-start.sh](../../hooks/session-start.sh) |
| <a id="e18-user-prompt-submit-sh"></a>E18 | user-prompt-submit.sh | Component | Emits additionalContext reminding the agent of /learn and /prepare boundaries on every user prompt. | [../../hooks/user-prompt-submit.sh](../../hooks/user-prompt-submit.sh) |
| <a id="e19-post-tool-use-sh"></a>E19 | post-tool-use.sh | Component | Emits the same /learn / /prepare reminder after each tool use. | [../../hooks/post-tool-use.sh](../../hooks/post-tool-use.sh) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-claude-code-hooks-json"></a>R1 | Claude Code | hooks.json | reads manifest at plugin load | File read |
| <a id="r2-hooks-json-session-start-sh"></a>R2 | hooks.json | session-start.sh | registers SessionStart -> session-start.sh (timeout 10s) | Manifest entry |
| <a id="r3-hooks-json-user-prompt-submit-sh"></a>R3 | hooks.json | user-prompt-submit.sh | registers UserPromptSubmit -> user-prompt-submit.sh (timeout 5s) | Manifest entry |
| <a id="r4-hooks-json-post-tool-use-sh"></a>R4 | hooks.json | post-tool-use.sh | registers PostToolUse -> post-tool-use.sh (timeout 5s) | Manifest entry |
| <a id="r5-claude-code-hooks"></a>R5 | Claude Code | Hooks | fires lifecycle events as subprocess execs with JSON stdin | Subprocess exec, stdin JSON |
| <a id="r6-session-start-sh-claude-code"></a>R6 | session-start.sh | Claude Code | emits SessionStart additionalContext announcing memory skills | Hook stdout JSON |
| <a id="r7-user-prompt-submit-sh-claude-code"></a>R7 | user-prompt-submit.sh | Claude Code | emits UserPromptSubmit additionalContext nudging /prepare and /learn | Hook stdout JSON |
| <a id="r8-post-tool-use-sh-claude-code"></a>R8 | post-tool-use.sh | Claude Code | emits PostToolUse additionalContext nudging /prepare and /learn | Hook stdout JSON |
| <a id="r9-session-start-sh-engram-cli-binary"></a>R9 | session-start.sh | engram CLI binary | async go build to ~/.claude/engram/bin/engram when any *.go is newer than the cached binary | Subprocess (go build), file I/O |

## Cross-links

- Parent: [c2-engram-plugin.md](c2-engram-plugin.md) (refines **E8 · Hooks**)
- Siblings:
  - [c3-engram-cli-binary.md](c3-engram-cli-binary.md)
  - [c3-skills.md](c3-skills.md)
- Refined by: *(none yet)*
