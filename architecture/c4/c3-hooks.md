---
level: 3
name: hooks
parent: "c2-engram-plugin.md"
children: []
last_reviewed_commit: 1ba7e162
---

# C3 — Hooks (Component)

Refines L2's E8 Hooks container into the manifest plus three bash scripts wired to Claude Code lifecycle events. Hooks emit hookSpecificOutput.additionalContext JSON on stdout to inject reminders into the agent's context. The SessionStart hook additionally rebuilds the Go binary asynchronously when any source file is newer than the cached binary, isolating the build from the agent's own Bash provenance so macOS doesn't SIGKILL it on exec.

![C3 hooks diagram](svg/c3-hooks.svg)

> Diagram source: [svg/c3-hooks.mmd](svg/c3-hooks.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c3-hooks.mmd -o architecture/c4/svg/c3-hooks.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

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
