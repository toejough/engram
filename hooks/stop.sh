#!/usr/bin/env bash
set -euo pipefail

# Async stop hook — runs engram evaluate to assess pending memory evaluations.
# Fire-and-forget: always exits 0 so Claude Code is never blocked.

# Require jq for JSON parsing
command -v jq &>/dev/null || exit 0

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

if [[ -z "$TRANSCRIPT_PATH" || -z "$SESSION_ID" ]]; then
    exit 0
fi

# Evaluate pending memories for this session (fire-and-forget)
"$ENGRAM_BIN" evaluate \
    --transcript-path "$TRANSCRIPT_PATH" \
    --session-id "$SESSION_ID" 2>/dev/null || true

exit 0
