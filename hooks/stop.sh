#!/usr/bin/env bash
set -euo pipefail

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
ENGRAM_DATA="${ENGRAM_HOME}/data"

# Skip if binary not built yet
[[ -x "$ENGRAM_BIN" ]] || exit 0

# Read hook JSON from stdin
HOOK_JSON="$(cat)"
TRANSCRIPT_PATH="$(echo "$HOOK_JSON" | jq -r '.transcript_path // empty')"
SESSION_ID="$(echo "$HOOK_JSON" | jq -r '.session_id // empty')"

# Debug: log what we received
echo "[engram-debug-stop] keys=$(echo "$HOOK_JSON" | jq -r 'keys[]' | tr '\n' ',')" >&2

# UC-14: Update session context
if [[ -n "$TRANSCRIPT_PATH" && -n "$SESSION_ID" ]]; then
    "$ENGRAM_BIN" context-update \
        --transcript-path "$TRANSCRIPT_PATH" \
        --session-id "$SESSION_ID" \
        --data-dir "$ENGRAM_DATA" || true
fi
