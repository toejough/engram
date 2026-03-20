#!/usr/bin/env bash
set -euo pipefail

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
ENGRAM_DATA="${ENGRAM_HOME}/data"

# Skip if binary not built yet
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

# Project-specific context path
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
CONTEXT_DIR="${ENGRAM_DATA}/projects/${PROJECT_SLUG}"
mkdir -p "$CONTEXT_DIR" 2>/dev/null || true

# Unified flush pipeline: learn → context-update (#309, #348)
FLUSH_ARGS=(--data-dir "$ENGRAM_DATA")
[[ -n "$TRANSCRIPT_PATH" ]] && FLUSH_ARGS+=(--transcript-path "$TRANSCRIPT_PATH")
[[ -n "$SESSION_ID" ]] && FLUSH_ARGS+=(--session-id "$SESSION_ID")
FLUSH_ARGS+=(--context-path "${CONTEXT_DIR}/session-context.md")

"$ENGRAM_BIN" flush "${FLUSH_ARGS[@]}" || true
