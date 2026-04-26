---
level: 4
name: user-prompt-submit
parent: "c3-hooks.md"
children: []
last_reviewed_commit: cd55eab2
---

# C4 — user-prompt-submit (Property/Invariant Ledger)

> Component in focus: **E18 · user-prompt-submit.sh** (refines L3 c3-hooks.md).
> Source files in scope:
> - [hooks/user-prompt-submit.sh](hooks/user-prompt-submit.sh)

## Context (from L3)

user-prompt-submit.sh is the simplest of the three engram hooks. Claude Code execs it on every UserPromptSubmit lifecycle event (registered via R3 in hooks.json with a 5s timeout). The script emits a single fixed JSON document on stdout via `jq -n`, populating `hookSpecificOutput.hookEventName` with the literal `"UserPromptSubmit"` and `hookSpecificOutput.additionalContext` with a constant reminder string nudging the agent to call `/learn` at completion boundaries and `/prepare` at new-work boundaries. The script reads no stdin, performs no I/O beyond stdout, exits non-zero on any failure courtesy of `set -euo pipefail`, and has no inputs that influence its output — every invocation emits the same bytes.

![C4 user-prompt-submit context diagram](svg/c4-user-prompt-submit.svg)

> Diagram source: [svg/c4-user-prompt-submit.mmd](svg/c4-user-prompt-submit.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-user-prompt-submit.mmd -o architecture/c4/svg/c4-user-prompt-submit.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-strict-bash-mode"></a>P1 | strict-bash-mode | Script runs under `set -euo pipefail`, so any command failure, unset variable, or pipe failure aborts with non-zero exit before stdout is emitted. | [hooks/user-prompt-submit.sh:2](../../hooks/user-prompt-submit.sh#L2) | **⚠ UNTESTED** |   |
| <a id="p2-fixed-event-name"></a>P2 | fixed-event-name | Emitted JSON has `hookSpecificOutput.hookEventName` set to the literal string `"UserPromptSubmit"`, matching the lifecycle event Claude Code dispatched. | [hooks/user-prompt-submit.sh:6](../../hooks/user-prompt-submit.sh#L6) | **⚠ UNTESTED** |   |
| <a id="p3-constant-reminder-payload"></a>P3 | constant-reminder-payload | `hookSpecificOutput.additionalContext` is a constant string nudging the agent to call `/learn` at completion boundaries (task done, bug resolved, direction change, commit) and `/prepare` when starting new work; nothing in the input or environment alters its content. | [hooks/user-prompt-submit.sh:6](../../hooks/user-prompt-submit.sh#L6) | **⚠ UNTESTED** |   |
| <a id="p4-single-json-document-on-stdout"></a>P4 | single-json-document-on-stdout | Script writes exactly one well-formed JSON document to stdout (via `jq -n`) and writes nothing else, so Claude Code's stdout-JSON consumer (R7) always receives a parseable payload. | [hooks/user-prompt-submit.sh:6](../../hooks/user-prompt-submit.sh#L6) | **⚠ UNTESTED** |   |
| <a id="p5-no-stdin-no-side-effects"></a>P5 | no-stdin-no-side-effects | Script reads no stdin, performs no file I/O, spawns no background work, and depends only on `jq` being on PATH; output is a pure function of the script source. | [hooks/user-prompt-submit.sh:6](../../hooks/user-prompt-submit.sh#L6) | **⚠ UNTESTED** |   |
| <a id="p6-fits-within-5s-timeout"></a>P6 | fits-within-5s-timeout | Single `jq -n` invocation completes well within the 5s timeout registered by R3 in hooks.json, so Claude Code never has to kill the hook for taking too long. | [hooks/user-prompt-submit.sh:6](../../hooks/user-prompt-submit.sh#L6) | **⚠ UNTESTED** |   |

## Cross-links

- Parent: [c3-hooks.md](c3-hooks.md) (refines **E18 · user-prompt-submit.sh**)
- Siblings:
  - [c4-hooks-json.md](c4-hooks-json.md)
  - [c4-post-tool-use.md](c4-post-tool-use.md)
  - [c4-session-start.md](c4-session-start.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

