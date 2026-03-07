#!/usr/bin/env bash
set -euo pipefail

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
ENGRAM_DATA="${ENGRAM_HOME}/data"

# Build if missing or stale (source newer than binary)
NEEDS_BUILD=false
if [[ ! -x "$ENGRAM_BIN" ]]; then
    NEEDS_BUILD=true
elif [[ -d "$PLUGIN_ROOT" ]]; then
    # Rebuild if any Go source file is newer than the binary
    if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" -print -quit 2>/dev/null | grep -q .; then
        NEEDS_BUILD=true
    fi
fi

if [[ "$NEEDS_BUILD" == "true" ]]; then
    mkdir -p "${ENGRAM_HOME}/bin"
    cd "$PLUGIN_ROOT"
    go build -o "$ENGRAM_BIN" ./cmd/engram/ 2>/dev/null || { echo "[engram] build failed — is Go installed?" >&2; exit 0; }
fi

# UC-2: Surface relevant memories at session start
SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode session-start --data-dir "$ENGRAM_DATA" --format json) || true

# UC-14: Restore session context
CONTEXT_FILE="${ENGRAM_DATA}/session-context.md"
SESSION_CONTEXT=""
if [[ -f "$CONTEXT_FILE" ]]; then
    # Extract summary (skip HTML comment on first line)
    SESSION_CONTEXT=$(tail -n +3 "$CONTEXT_FILE")
fi

# Static guidance for mid-turn message capture (issue #54)
MIDTURN_NOTE="[engram] Mid-turn user messages (delivered via system-reminder) bypass engram hooks. If you receive a mid-turn correction or instruction, capture it by running: ~/.claude/engram/bin/engram correct --message '<the user message>' --data-dir ~/.claude/engram/data"

if [[ -n "$SURFACE_OUTPUT" ]]; then
    echo "$SURFACE_OUTPUT" | jq \
        --arg note "$MIDTURN_NOTE" \
        --arg ctx "$SESSION_CONTEXT" \
        '{systemMessage: .summary, additionalContext: (.context + "\n" + $note + (if $ctx != "" then "\n[engram] Previous session context:\n" + $ctx else "" end))}'
else
    jq -n \
        --arg note "$MIDTURN_NOTE" \
        --arg ctx "$SESSION_CONTEXT" \
        '{additionalContext: ($note + (if $ctx != "" then "\n[engram] Previous session context:\n" + $ctx else "" end))}'
fi
