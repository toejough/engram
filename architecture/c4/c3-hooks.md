---
level: 3
name: hooks
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: 7da51d0c
---

# C3 — Hooks (Component)

Refines L2's E8 Hooks container into the manifest plus three bash scripts wired to Claude Code lifecycle events. Hooks emit hookSpecificOutput.additionalContext JSON on stdout to inject reminders into the agent's context. The SessionStart hook additionally rebuilds the Go binary asynchronously when any source file is newer than the cached binary, isolating the build from the agent's own Bash provenance so macOS doesn't SIGKILL it on exec.

```mermaid
flowchart LR
    classDef person      fill:#08427b,stroke:#052e56,color:#fff
    classDef external    fill:#999,   stroke:#666,   color:#fff
    classDef container   fill:#1168bd,stroke:#0b4884,color:#fff
    classDef component   fill:#85bbf0,stroke:#5d9bd1,color:#000

    s3(S3 · Claude Code<br/>agent harness)
    s2-n3[S2-N3 · engram CLI binary<br/>Go binary built by session-start.sh]

    subgraph s2-n2 [S2-N2 · Hooks]
        s2-n2-m1[S2-N2-M1 · hooks.json<br/>manifest]
        s2-n2-m2[S2-N2-M2 · session-start.sh<br/>skill announcement + async rebuild]
        s2-n2-m3[S2-N2-M3 · user-prompt-submit.sh<br/>/prepare /learn nudge]
        s2-n2-m4[S2-N2-M4 · post-tool-use.sh<br/>/prepare /learn nudge]
    end

    s3 -->|"R1: reads manifest at plugin load"| s2-n2-m1
    s2-n2-m1 -->|"R2: registers SessionStart -> session-start.sh (timeout 10s)"| s2-n2-m2
    s2-n2-m1 -->|"R3: registers UserPromptSubmit -> user-prompt-submit.sh (timeout 5s)"| s2-n2-m3
    s2-n2-m1 -->|"R4: registers PostToolUse -> post-tool-use.sh (timeout 5s)"| s2-n2-m4
    s3 -->|"R5: fires lifecycle events as subprocess execs with JSON stdin"| s2-n2
    s2-n2-m2 -->|"R6: emits SessionStart additionalContext announcing memory skills"| s3
    s2-n2-m3 -->|"R7: emits UserPromptSubmit additionalContext nudging /prepare and /learn"| s3
    s2-n2-m4 -->|"R8: emits PostToolUse additionalContext nudging /prepare and /learn"| s3
    s2-n2-m2 -->|"R9: async go build to ~/.claude/engram/bin/engram when any *.go is newer than the cached binary"| s2-n3

    class s3 external
    class s2-n3 container
    class s2-n2-m1,s2-n2-m2,s2-n2-m3,s2-n2-m4 component
    class s2-n2 container

    click s2-n2 href "#s2-n2-hooks" "Hooks"
    click s3 href "#s3-claude-code" "Claude Code"
    click s2-n3 href "#s2-n3-engram-cli-binary" "engram CLI binary"
    click s2-n2-m1 href "#s2-n2-m1-hooks-json" "hooks.json"
    click s2-n2-m2 href "#s2-n2-m2-session-start-sh" "session-start.sh"
    click s2-n2-m3 href "#s2-n2-m3-user-prompt-submit-sh" "user-prompt-submit.sh"
    click s2-n2-m4 href "#s2-n2-m4-post-tool-use-sh" "post-tool-use.sh"
```

## Element Catalog

| ID | Name | Type | Responsibility | Code Pointer |
|---|---|---|---|---|
| <a id="s2-n2-hooks"></a>S2-N2 | Hooks | Container in focus | Three bash scripts wired by hooks/hooks.json; emit JSON additionalContext, async-rebuild the binary on SessionStart. | — |
| <a id="s3-claude-code"></a>S3 | Claude Code | External system | Reads the hook manifest at plugin load and execs the registered scripts on lifecycle events; consumes their stdout JSON to inject context. | — |
| <a id="s2-n3-engram-cli-binary"></a>S2-N3 | engram CLI binary | Container | Go binary built and refreshed by session-start.sh. Refined in c3-engram-cli-binary.md. | — |
| <a id="s2-n2-m1-hooks-json"></a>S2-N2-M1 | hooks.json | Component | Manifest mapping SessionStart, UserPromptSubmit, and PostToolUse events to their scripts via ${CLAUDE_PLUGIN_ROOT} paths and per-event timeouts (10s / 5s / 5s). | [../../hooks/hooks.json](../../hooks/hooks.json) |
| <a id="s2-n2-m2-session-start-sh"></a>S2-N2-M2 | session-start.sh | Component | Synchronously emits the memory-skill announcement. Asynchronously rebuilds the Go binary when any *.go is newer than the cached binary mtime, deleting the prior binary first to avoid macOS provenance SIGKILL, then symlinks ~/.local/bin/engram to it. | [../../hooks/session-start.sh](../../hooks/session-start.sh) |
| <a id="s2-n2-m3-user-prompt-submit-sh"></a>S2-N2-M3 | user-prompt-submit.sh | Component | Emits additionalContext reminding the agent of /learn and /prepare boundaries on every user prompt. | [../../hooks/user-prompt-submit.sh](../../hooks/user-prompt-submit.sh) |
| <a id="s2-n2-m4-post-tool-use-sh"></a>S2-N2-M4 | post-tool-use.sh | Component | Emits the same /learn / /prepare reminder after each tool use. | [../../hooks/post-tool-use.sh](../../hooks/post-tool-use.sh) |

## Relationships

| ID | From | To | Description | Protocol/Medium |
|---|---|---|---|---|
| <a id="r1-s3-s2-n2-m1"></a>R1 | S3 | S2-N2-M1 | reads manifest at plugin load | File read |
| <a id="r2-s2-n2-m1-s2-n2-m2"></a>R2 | S2-N2-M1 | S2-N2-M2 | registers SessionStart -> session-start.sh (timeout 10s) | Manifest entry |
| <a id="r3-s2-n2-m1-s2-n2-m3"></a>R3 | S2-N2-M1 | S2-N2-M3 | registers UserPromptSubmit -> user-prompt-submit.sh (timeout 5s) | Manifest entry |
| <a id="r4-s2-n2-m1-s2-n2-m4"></a>R4 | S2-N2-M1 | S2-N2-M4 | registers PostToolUse -> post-tool-use.sh (timeout 5s) | Manifest entry |
| <a id="r5-s3-s2-n2"></a>R5 | S3 | S2-N2 | fires lifecycle events as subprocess execs with JSON stdin | Subprocess exec, stdin JSON |
| <a id="r6-s2-n2-m2-s3"></a>R6 | S2-N2-M2 | S3 | emits SessionStart additionalContext announcing memory skills | Hook stdout JSON |
| <a id="r7-s2-n2-m3-s3"></a>R7 | S2-N2-M3 | S3 | emits UserPromptSubmit additionalContext nudging /prepare and /learn | Hook stdout JSON |
| <a id="r8-s2-n2-m4-s3"></a>R8 | S2-N2-M4 | S3 | emits PostToolUse additionalContext nudging /prepare and /learn | Hook stdout JSON |
| <a id="r9-s2-n2-m2-s2-n3"></a>R9 | S2-N2-M2 | S2-N3 | async go build to ~/.claude/engram/bin/engram when any *.go is newer than the cached binary | Subprocess (go build), file I/O |

## Cross-links

- Parent: [c2-engram-plugin.md](c2-engram-plugin.md) (refines **S2-N2 · Hooks**)
- Siblings:
  - [c3-engram-cli-binary.md](c3-engram-cli-binary.md)
  - [c3-skills.md](c3-skills.md)
- Refined by: *(none yet)*
