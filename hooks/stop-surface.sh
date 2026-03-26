#!/usr/bin/env bash
set -euo pipefail

# Stop hook: surface relevant memories when agent finishes responding.
# Uses the advanced JSON hook API (exit 0 + JSON stdout) required for
# plugin Stop hooks per anthropics/claude-code#10875.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# Build if missing or stale (source newer than binary)
NEEDS_BUILD=false
if [[ ! -x "$ENGRAM_BIN" ]]; then
    NEEDS_BUILD=true
elif [[ -d "$PLUGIN_ROOT" ]]; then
    if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" -print -quit 2>/dev/null | grep -q .; then
        NEEDS_BUILD=true
    fi
fi

if [[ "$NEEDS_BUILD" == "true" ]]; then
    mkdir -p "${ENGRAM_HOME}/bin"
    cd "$PLUGIN_ROOT"
    go build -o "$ENGRAM_BIN" ./cmd/engram/ 2>/dev/null || { echo "[engram] build failed" >&2; exit 0; }
fi

# Read hook JSON from stdin
HOOK_JSON="$(cat)"
TRANSCRIPT_PATH="$(echo "$HOOK_JSON" | jq -r '.transcript_path // empty')"
SESSION_ID="$(echo "$HOOK_JSON" | jq -r '.session_id // empty')"
STOP_HOOK_ACTIVE="$(echo "$HOOK_JSON" | jq -r '.stop_hook_active // false')"

# Prevent infinite loop: if we already blocked once, allow the stop
if [[ "$STOP_HOOK_ACTIVE" == "true" ]]; then
    exit 0
fi

if [[ -z "$TRANSCRIPT_PATH" ]]; then
    exit 0
fi

# Surface memories based on agent's recent output
SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode stop \
    --transcript-path "$TRANSCRIPT_PATH" \
    --session-id "$SESSION_ID" \
    --format json 2>/dev/null) || SURFACE_OUTPUT=""

if [[ -n "$SURFACE_OUTPUT" ]]; then
    CONTEXT=$(echo "$SURFACE_OUTPUT" | jq -r '.context // empty')
    if [[ -n "$CONTEXT" ]]; then
        jq -n --arg reason "$CONTEXT" \
            '{"decision":"block","reason":$reason}'
        exit 0
    fi
fi
