#!/usr/bin/env bash
set -euo pipefail

# UC-14: Async context-update — runs via "async": true hook entry.
# Claude Code manages the background lifecycle (timeout, cleanup).

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
ENGRAM_DATA="${ENGRAM_HOME}/data"

# Skip silently if binary not yet built (sync hook builds it)
[[ -x "$ENGRAM_BIN" ]] || exit 0

# Platform-aware OAuth token retrieval (DES-3)
TOKEN=""
if [[ "$(uname)" == "Darwin" ]]; then
    TOKEN=$(security find-generic-password \
        -s "Claude Code-credentials" -w 2>/dev/null \
        | python3 -c \
        "import sys,json; print(json.load(sys.stdin)['claudeAiOauth']['accessToken'])" \
        2>/dev/null) || true
fi
export ENGRAM_API_TOKEN="${TOKEN:-${ENGRAM_API_TOKEN:-}}"

# Read hook JSON from stdin
HOOK_JSON="$(cat)"
TRANSCRIPT_PATH="$(echo "$HOOK_JSON" | jq -r '.transcript_path // empty')"
SESSION_ID="$(echo "$HOOK_JSON" | jq -r '.session_id // empty')"

if [[ -n "$TRANSCRIPT_PATH" && -n "$SESSION_ID" ]]; then
    "$ENGRAM_BIN" context-update \
        --transcript-path "$TRANSCRIPT_PATH" \
        --session-id "$SESSION_ID" \
        --data-dir "$ENGRAM_DATA" || true
fi
