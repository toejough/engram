#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# Platform-aware OAuth token retrieval (DES-3)
TOKEN=""
if [[ "$(uname)" == "Darwin" ]]; then
    TOKEN=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['claudeAiOauth']['accessToken'])" 2>/dev/null) || true
fi
export ENGRAM_API_TOKEN="${TOKEN:-${ENGRAM_API_TOKEN:-}}"

# UC-3: Check for inline correction
"$ENGRAM_BIN" correct --message "$CLAUDE_USER_MESSAGE" --data-dir "$ENGRAM_DATA"
