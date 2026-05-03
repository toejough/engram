#!/usr/bin/env bash
set -euo pipefail

# SessionStart hook — announce memory skills, build binary if needed.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_BIN="${HOME}/.local/bin/engram"

# Sync portion: announce skills
STATIC_MSG=$("$ENGRAM_BIN" reminder session-start)

jq -n --arg ctx "$STATIC_MSG" \
    '{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: $ctx}}'

# Async portion: build if needed
(
  export ENGRAM_PLUGIN_ROOT="$PLUGIN_ROOT"
  "$ENGRAM_BIN" build-self --if-stale --plugin-root "$PLUGIN_ROOT" --bin-path "$ENGRAM_BIN" 2>/dev/null || true
) & disown

exit 0
