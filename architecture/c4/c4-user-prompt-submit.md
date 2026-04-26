---
level: 4
name: user-prompt-submit
parent: "c3-hooks.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — user-prompt-submit (Property/Invariant Ledger)

> Component in focus: **E18 · user-prompt-submit.sh** (refines L3 c3-hooks).
> Source files in scope:
> - [../../hooks/user-prompt-submit.sh](../../hooks/user-prompt-submit.sh)

## Context (from L3)

Scoped slice of [c3-hooks.md](c3-hooks.md): Claude Code execs the script on each user
prompt submission (R5) and consumes the emitted `additionalContext` JSON (R7).

![C4 user-prompt-submit context diagram](svg/c4-user-prompt-submit.svg)

> Diagram source: [svg/c4-user-prompt-submit.mmd](svg/c4-user-prompt-submit.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-user-prompt-submit.mmd -o architecture/c4/svg/c4-user-prompt-submit.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-strict-bash"></a>P1 | Strict bash mode | For all invocations, the script runs with `set -euo pipefail`. | [hooks/user-prompt-submit.sh:2](../../hooks/user-prompt-submit.sh#L2) | **⚠ UNTESTED** | — |
| <a id="p2-emits-user-prompt-json"></a>P2 | Emits valid UserPromptSubmit JSON | For all invocations, the script writes a single JSON object to stdout matching `{hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: <string>}}`. | [hooks/user-prompt-submit.sh:6](../../hooks/user-prompt-submit.sh#L6) | **⚠ UNTESTED** | Built with `jq -n` so the JSON is well-formed by construction. |
| <a id="p3-nudges-prepare-learn"></a>P3 | Nudges /prepare and /learn | For all invocations, the emitted `additionalContext` mentions both `/prepare` and `/learn` and the boundaries at which each should be called. | [hooks/user-prompt-submit.sh:6](../../hooks/user-prompt-submit.sh#L6) | **⚠ UNTESTED** | Static literal; identical to the post-tool-use reminder. |
| <a id="p4-non-destructive"></a>P4 | Non-destructive | For all invocations, the script performs no filesystem writes, network calls, or subprocess execs other than `jq`. | [hooks/user-prompt-submit.sh:1](../../hooks/user-prompt-submit.sh#L1) | **⚠ UNTESTED** | Hook fires on every user prompt — must be cheap and side-effect-free. |
| <a id="p5-fast-completion"></a>P5 | Completes within timeout | For all invocations, the script completes within the manifest's 5-second UserPromptSubmit timeout. | [hooks/user-prompt-submit.sh:6](../../hooks/user-prompt-submit.sh#L6) | **⚠ UNTESTED** | Single `jq -n` call; bounded by jq startup. |

## Cross-links

- Parent: [c3-hooks.md](c3-hooks.md) (refines **E18 · user-prompt-submit.sh**)
- Siblings:
  - [c4-hooks-json.md](c4-hooks-json.md)
  - [c4-session-start.md](c4-session-start.md)
  - [c4-post-tool-use.md](c4-post-tool-use.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
