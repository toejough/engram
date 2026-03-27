# Tool-Call Frecency Gating for Memory Surfacing

## Problem

Engram's PreToolUse/PostToolUse hooks surface 2 memory advisories on every tool call. In a typical session, non-Bash tools (Grep, Read, Glob, Edit, etc.) fire dozens of times. BM25 matching against file paths and regex patterns produces near-random results — analysis of this session showed 0% relevance for non-Bash tool advisories. The noise consumes context window and clutters the user's terminal.

Bash commands carry semantic intent (`go test`, `targ check-full`, `git push --force`) that memories are actually about. But even within Bash, high-frequency commands (grep, cat) don't benefit from repeated surfacing.

## Design

### 1. Non-Bash tools: skip surfacing entirely

`pre-tool-use.sh`, `post-tool-use.sh`, and `post-tool-use-failure.sh` skip memory surfacing for non-Bash tools. No Go binary invocation, no memory loading.

**Hook-specific placement:**
- `pre-tool-use.sh`: guard at the top (entire hook is memory surfacing)
- `post-tool-use.sh`: guard placed *after* the existing Write/Edit skill-file advisory block, which is unrelated to memory surfacing and must be preserved
- `post-tool-use-failure.sh`: guard wraps only the memory surfacing block, preserving the existing static failure advice (Read path hints, Edit match hints, etc.) for all tool types

**Rationale:** Non-Bash tool inputs (file paths, glob patterns, regex) don't carry the semantic intent that memories match against. Zero followed memories observed for non-Bash tool surfacing.

### 2. Bash commands: frecency-gated surfacing

#### Command key extraction

Extract a stable identity from the Bash command string:

1. Strip leading environment variable assignments (`VAR=val`)
2. Take the first two space-separated tokens
3. If the second token starts with `-`, drop it (flags aren't identity)

Examples:
- `go test ./...` → `go test`
- `targ check-full` → `targ check-full`
- `grep -r foo src/` → `grep`
- `FOO=bar git push origin main` → `git push`
- `ls -la` → `ls`

#### Persistent frequency counter

File: `{data-dir}/tool-frecency.json` (uses the same `--data-dir` flag as all other engram commands)

```json
{
  "go test": {"count": 12, "last": "2026-03-21T10:30:00Z"},
  "grep": {"count": 45, "last": "2026-03-21T10:31:00Z"},
  "targ check-full": {"count": 3, "last": "2026-03-21T09:00:00Z"}
}
```

Updated atomically (write-tmp-rename) on every Bash tool call, regardless of whether surfacing occurs. Ordering: read current count, compute probability and roll, then increment and persist.

#### Surfacing probability

Smooth logarithmic decay:

```
P(surface) = 1 / (1 + ln(1 + count))
```

| count | probability |
|-------|------------|
| 0     | 1.00       |
| 1     | 0.59       |
| 2     | 0.48       |
| 5     | 0.36       |
| 10    | 0.29       |
| 50    | 0.20       |
| 100   | 0.18       |

The curve flattens around 0.18, so even very common commands still surface ~1 in 5 times. No cleanup/reset mechanism — counters grow forever.

The probability roll happens in the Go binary at the top of `runTool()`, before memory loading or BM25 scoring. If the roll says "skip", return empty result immediately.

The Go binary also short-circuits for non-Bash tool names as defense-in-depth (the shell layer is the primary filter, but direct CLI invocations and tests should behave correctly too).

### 3. Unchanged hooks

- **UserPromptSubmit** — matches against user's words; genuinely useful. No gating.
- **SessionStart** — runs `maintain` + shows `/recall` hint. No memory surfacing.
- **PreCompact** — already a no-op since #350.
- **Stop** — flush pipeline. No memory surfacing.

## Implementation

### Shell layer (3 files)

Add early exit guard to `pre-tool-use.sh`, `post-tool-use.sh`, `post-tool-use-failure.sh`:

```bash
# Only surface memories for Bash tool calls
if [[ "$TOOL_NAME" != "Bash" ]]; then
    exit 0
fi
```

### Go layer

New functionality in `internal/surface/` or a new `internal/toolgate/` package:

1. **Command key extraction** — parse the `.command` field from tool input JSON, strip env vars, extract first two tokens, drop flag-like second token
2. **Counter read/update** — load `tool-frecency.json` from `{data-dir}`, increment count for key, write back atomically
3. **Probability gate** — compute `1 / (1 + ln(1 + count))`, roll via injected random source (`func() float64`), short-circuit if skip
4. **Non-Bash defense-in-depth** — short-circuit for `ToolName != "Bash"` before any of the above

The random source is injected to satisfy DI requirements and enable deterministic testing.

Called at the top of `Surfacer.runTool()` before `ListMemories()`.

### State file

- Location: `{data-dir}/tool-frecency.json`
- Created on first Bash tool call
- Atomic writes (temp file + rename)
- No TTL, no cleanup, no max size

## Risks

- **False negatives for rare-but-important commands**: A command called for the first time in a session still checks the persistent counter. If `go test` was called 100 times last week, it gets P=0.18 even if this is the first call today. Acceptable because the curve never drops below ~0.18.
- **Command key collisions**: `npm install` and `npm install express` both map to `npm install`. This is intentional — the subcommand is the identity, not the arguments.
