---
level: 4
name: session-start
parent: "c3-hooks.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — session-start (Property/Invariant Ledger)

> Component in focus: **E17 · session-start.sh** (refines L3 c3-hooks).
> Source files in scope:
> - [../../hooks/session-start.sh](../../hooks/session-start.sh)

## Context (from L3)

Scoped slice of [c3-hooks.md](c3-hooks.md): Claude Code execs the script on SessionStart
(R5), the script emits `additionalContext` JSON to stdout (R6), and asynchronously rebuilds
the engram CLI binary when any `.go` file is newer than the cached binary (R9).

![C4 session-start context diagram](svg/c4-session-start.svg)

> Diagram source: [svg/c4-session-start.mmd](svg/c4-session-start.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-session-start.mmd -o architecture/c4/svg/c4-session-start.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-strict-bash"></a>P1 | Strict bash mode | For all invocations, the script runs with `set -euo pipefail` so any unhandled error or undefined variable aborts immediately. | [hooks/session-start.sh:2](../../hooks/session-start.sh#L2) | **⚠ UNTESTED** | — |
| <a id="p2-emits-session-start-json"></a>P2 | Emits valid SessionStart JSON | For all invocations, the script writes one JSON object to stdout matching `{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: <string>}}`. | [hooks/session-start.sh:13](../../hooks/session-start.sh#L13) | **⚠ UNTESTED** | Built with `jq -n` so the JSON is well-formed by construction. |
| <a id="p3-announces-skills"></a>P3 | Announces memory skills | For all invocations, the emitted `additionalContext` mentions all four memory skills: `/prepare`, `/learn`, `/recall`, `/remember`. | [hooks/session-start.sh:11](../../hooks/session-start.sh#L11) | **⚠ UNTESTED** | Skill discoverability for fresh sessions. |
| <a id="p4-sync-completes-fast"></a>P4 | Sync portion completes within hook timeout | For all invocations, the foreground portion (jq emit) completes well within the manifest's 10-second SessionStart timeout, regardless of build outcome. | [hooks/session-start.sh:13](../../hooks/session-start.sh#L13), [:17](../../hooks/session-start.sh#L17), [:41](../../hooks/session-start.sh#L41) | **⚠ UNTESTED** | The build is launched in `( ... ) & disown`, so it cannot block the hook. |
| <a id="p5-build-only-when-stale"></a>P5 | Async build only when stale | For all invocations, the async block invokes `go build` only when the cached binary is missing or any `*.go` under `$PLUGIN_ROOT` is newer than it. | [hooks/session-start.sh:19](../../hooks/session-start.sh#L19), [:22](../../hooks/session-start.sh#L22) | **⚠ UNTESTED** | Uses `find -newer` against the cached binary mtime. |
| <a id="p6-clean-shell-rebuild"></a>P6 | Clean-shell rebuild avoids macOS provenance SIGKILL | For all rebuilds, the prior binary and tmp file are deleted before `go build` runs, so the new binary inherits this hook's bash provenance, not the agent's Bash-tool provenance. | [hooks/session-start.sh:33](../../hooks/session-start.sh#L33) | **⚠ UNTESTED** | Documented inline as the reason for the `rm -f` step; macOS would SIGKILL on exec otherwise. |
| <a id="p7-atomic-binary-replace"></a>P7 | Atomic binary publish | For all successful rebuilds, the new binary is written to `$ENGRAM_BIN.tmp` and only `mv`'d into place after `go build` succeeds. | [hooks/session-start.sh:34](../../hooks/session-start.sh#L34), [:35](../../hooks/session-start.sh#L35) | **⚠ UNTESTED** | If `go build` fails, the old binary is left in place; `go build || exit 0` swallows the failure to keep the hook non-fatal. |
| <a id="p8-build-failure-non-fatal"></a>P8 | Build failure is non-fatal | For all `go build` failures, the async block exits 0 without affecting the foreground stdout JSON or the hook's exit code. | [hooks/session-start.sh:34](../../hooks/session-start.sh#L34), [:43](../../hooks/session-start.sh#L43) | **⚠ UNTESTED** | The hook always returns 0 from the foreground path. |
| <a id="p9-symlink-best-effort"></a>P9 | PATH symlink best-effort | For all invocations, the `~/.local/bin/engram` symlink is refreshed when possible, but failure to create it (e.g. permission denied) does not fail the async block. | [hooks/session-start.sh:40](../../hooks/session-start.sh#L40) | **⚠ UNTESTED** | `ln -sf ... 2>/dev/null \|\| true`. |

## Cross-links

- Parent: [c3-hooks.md](c3-hooks.md) (refines **E17 · session-start.sh**)
- Siblings:
  - [c4-hooks-json.md](c4-hooks-json.md)
  - [c4-user-prompt-submit.md](c4-user-prompt-submit.md)
  - [c4-post-tool-use.md](c4-post-tool-use.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
