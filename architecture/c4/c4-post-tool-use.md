---
level: 4
name: post-tool-use
parent: "c3-hooks.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — post-tool-use (Property/Invariant Ledger)

> Component in focus: **E19 · post-tool-use.sh** (refines L3 c3-hooks).
> Source files in scope:
> - [../../hooks/post-tool-use.sh](../../hooks/post-tool-use.sh)

## Context (from L3)

Scoped slice of [c3-hooks.md](c3-hooks.md): Claude Code execs the script after each tool
use (R5) and consumes the emitted `additionalContext` JSON (R8).

![C4 post-tool-use context diagram](svg/c4-post-tool-use.svg)

> Diagram source: [svg/c4-post-tool-use.mmd](svg/c4-post-tool-use.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-post-tool-use.mmd -o architecture/c4/svg/c4-post-tool-use.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-strict-bash"></a>P1 | Strict bash mode | For all invocations, the script runs with `set -euo pipefail`. | [hooks/post-tool-use.sh:2](../../hooks/post-tool-use.sh#L2) | **⚠ UNTESTED** | — |
| <a id="p2-emits-post-tool-json"></a>P2 | Emits valid PostToolUse JSON | For all invocations, the script writes a single JSON object to stdout matching `{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: <string>}}`. | [hooks/post-tool-use.sh:6](../../hooks/post-tool-use.sh#L6) | **⚠ UNTESTED** | Built with `jq -n`. |
| <a id="p3-nudges-prepare-learn"></a>P3 | Nudges /prepare and /learn | For all invocations, the emitted `additionalContext` mentions both `/prepare` and `/learn` and the boundaries at which each should be called. | [hooks/post-tool-use.sh:6](../../hooks/post-tool-use.sh#L6) | **⚠ UNTESTED** | Static literal; identical text to the user-prompt-submit reminder. |
| <a id="p4-non-destructive"></a>P4 | Non-destructive | For all invocations, the script performs no filesystem writes, network calls, or subprocess execs other than `jq`. | [hooks/post-tool-use.sh:1](../../hooks/post-tool-use.sh#L1) | **⚠ UNTESTED** | Hook fires after every tool use — must be cheap and side-effect-free. |
| <a id="p5-fast-completion"></a>P5 | Completes within timeout | For all invocations, the script completes within the manifest's 5-second PostToolUse timeout. | [hooks/post-tool-use.sh:6](../../hooks/post-tool-use.sh#L6) | **⚠ UNTESTED** | Single `jq -n` call. |

## Cross-links

- Parent: [c3-hooks.md](c3-hooks.md) (refines **E19 · post-tool-use.sh**)
- Siblings:
  - [c4-hooks-json.md](c4-hooks-json.md)
  - [c4-session-start.md](c4-session-start.md)
  - [c4-user-prompt-submit.md](c4-user-prompt-submit.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
