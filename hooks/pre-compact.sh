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

# Platform-aware OAuth token retrieval (DES-3)
TOKEN=""
if [[ "$(uname)" == "Darwin" ]]; then
    TOKEN=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['claudeAiOauth']['accessToken'])" 2>/dev/null) || true
fi
export ENGRAM_API_TOKEN="${TOKEN:-${ENGRAM_API_TOKEN:-}}"

# Read transcript from file path provided in stdin JSON
STDIN_JSON="$(cat)"
TRANSCRIPT_PATH="$(echo "$STDIN_JSON" | jq -r '.transcript_path // empty')"
TRANSCRIPT=""
if [[ -n "$TRANSCRIPT_PATH" && -f "$TRANSCRIPT_PATH" ]]; then
    TRANSCRIPT="$(cat "$TRANSCRIPT_PATH")"
fi

if [[ -z "$TRANSCRIPT" ]]; then
    echo "[engram] Warning: no transcript available — learn/evaluate skipped" >&2
fi

# UC-14: Final session context flush (synchronous — last chance)
SESSION_ID="$(echo "$STDIN_JSON" | jq -r '.session_id // empty')"
if [[ -n "$TRANSCRIPT_PATH" && -n "$SESSION_ID" ]]; then
    "$ENGRAM_BIN" context-update \
        --transcript-path "$TRANSCRIPT_PATH" \
        --session-id "$SESSION_ID" \
        --data-dir "$ENGRAM_DATA" || true
fi

# UC-1: Extract learnings from session transcript (incremental)
if [[ -n "$TRANSCRIPT_PATH" && -n "$SESSION_ID" ]]; then
    "$ENGRAM_BIN" learn --transcript-path "$TRANSCRIPT_PATH" \
        --session-id "$SESSION_ID" --data-dir "$ENGRAM_DATA" || true
elif [[ -n "$TRANSCRIPT" ]]; then
    echo "$TRANSCRIPT" | "$ENGRAM_BIN" learn --data-dir "$ENGRAM_DATA" || true
fi

# UC-15: Evaluate outcome of surfaced memories
if [[ -n "$TRANSCRIPT" ]]; then
    echo "$TRANSCRIPT" | "$ENGRAM_BIN" evaluate --data-dir "$ENGRAM_DATA" || true
fi
