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

# UC-1: Extract learnings (synchronous — last chance before exit)
if [[ -n "$TRANSCRIPT_PATH" && -n "$SESSION_ID" ]]; then
    "$ENGRAM_BIN" learn --transcript-path "$TRANSCRIPT_PATH" \
        --session-id "$SESSION_ID" --data-dir "$ENGRAM_DATA" || true
fi

# UC-17: Evaluate memory effectiveness
if [[ -n "$TRANSCRIPT_PATH" ]]; then
    "$ENGRAM_BIN" evaluate --data-dir "$ENGRAM_DATA" < "$TRANSCRIPT_PATH" || true
fi

# UC-19: Audit session compliance (after evaluate, before context-update)
if [[ -n "$TRANSCRIPT_PATH" ]]; then
    AUDIT_TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    "$ENGRAM_BIN" audit --data-dir "$ENGRAM_DATA" \
        --timestamp "$AUDIT_TIMESTAMP" < "$TRANSCRIPT_PATH" || true
fi

# UC-14: Update session context (synchronous — Stop is the last chance)
# Project-specific context path follows Claude Code's convention
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
CONTEXT_DIR="${ENGRAM_DATA}/projects/${PROJECT_SLUG}"
mkdir -p "$CONTEXT_DIR" 2>/dev/null || true
if [[ -n "$TRANSCRIPT_PATH" && -n "$SESSION_ID" ]]; then
    "$ENGRAM_BIN" context-update \
        --transcript-path "$TRANSCRIPT_PATH" \
        --session-id "$SESSION_ID" \
        --data-dir "$ENGRAM_DATA" \
        --context-path "${CONTEXT_DIR}/session-context.md" || true
fi
