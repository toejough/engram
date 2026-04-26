---
level: 4
name: hooks-json
parent: "c3-hooks.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — hooks-json (Property/Invariant Ledger)

> Component in focus: **E16 · hooks.json** (refines L3 c3-hooks).
> Source files in scope:
> - [../../hooks/hooks.json](../../hooks/hooks.json)

## Context (from L3)

Scoped slice of [c3-hooks.md](c3-hooks.md): the L3 edges that touch E16. Claude Code reads
the manifest at plugin load (R1) and the manifest registers each lifecycle event to its
script with a per-event timeout (R2/R3/R4).

![C4 hooks-json context diagram](svg/c4-hooks-json.svg)

> Diagram source: [svg/c4-hooks-json.mmd](svg/c4-hooks-json.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-hooks-json.mmd -o architecture/c4/svg/c4-hooks-json.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-valid-json"></a>P1 | Valid JSON manifest | For all plugin loads, `hooks/hooks.json` parses as a JSON object whose top-level key is `"hooks"` mapping to an object of event-name keys. | [hooks/hooks.json:1](../../hooks/hooks.json#L1) | **⚠ UNTESTED** | No automated schema validation; relies on Claude Code's load-time parser. |
| <a id="p2-three-events"></a>P2 | Three lifecycle events registered | For all loads, the manifest registers exactly three events: `SessionStart`, `PostToolUse`, `UserPromptSubmit`. | [hooks/hooks.json:3](../../hooks/hooks.json#L3), [:14](../../hooks/hooks.json#L14), [:25](../../hooks/hooks.json#L25) | **⚠ UNTESTED** | Adding a fourth event requires both updating this file and adding a corresponding component to c3-hooks.md. |
| <a id="p3-script-paths-plugin-rooted"></a>P3 | Script paths plugin-rooted | For all registered hooks, the `command` field is `${CLAUDE_PLUGIN_ROOT}/hooks/<script>.sh`. | [hooks/hooks.json:8](../../hooks/hooks.json#L8), [:19](../../hooks/hooks.json#L19), [:30](../../hooks/hooks.json#L30) | **⚠ UNTESTED** | The plugin-root prefix lets Claude Code resolve scripts relative to wherever the plugin is checked out. |
| <a id="p4-session-start-timeout"></a>P4 | SessionStart timeout 10s | For all SessionStart firings, the manifest declares `timeout: 10` seconds. | [hooks/hooks.json:9](../../hooks/hooks.json#L9) | **⚠ UNTESTED** | Larger than other hooks because session-start.sh forks an async `go build` (R9) and must complete its sync portion within budget. |
| <a id="p5-post-tool-timeout"></a>P5 | PostToolUse timeout 5s | For all PostToolUse firings, the manifest declares `timeout: 5` seconds. | [hooks/hooks.json:20](../../hooks/hooks.json#L20) | **⚠ UNTESTED** | — |
| <a id="p6-user-prompt-timeout"></a>P6 | UserPromptSubmit timeout 5s | For all UserPromptSubmit firings, the manifest declares `timeout: 5` seconds. | [hooks/hooks.json:31](../../hooks/hooks.json#L31) | **⚠ UNTESTED** | — |
| <a id="p7-command-type"></a>P7 | All hooks are command-type | For all registered hooks, `type` is `"command"` (Claude Code's subprocess-exec hook flavor). | [hooks/hooks.json:7](../../hooks/hooks.json#L7), [:18](../../hooks/hooks.json#L18), [:29](../../hooks/hooks.json#L29) | **⚠ UNTESTED** | No prompt-style hooks (`type: prompt`) are registered. |

## Cross-links

- Parent: [c3-hooks.md](c3-hooks.md) (refines **E16 · hooks.json**)
- Siblings:
  - [c4-session-start.md](c4-session-start.md)
  - [c4-user-prompt-submit.md](c4-user-prompt-submit.md)
  - [c4-post-tool-use.md](c4-post-tool-use.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
