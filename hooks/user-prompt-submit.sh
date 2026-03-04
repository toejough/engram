#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# Get OAuth token from Claude Code Keychain
ENGRAM_API_TOKEN=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['claudeAiOauth']['accessToken'])" 2>/dev/null) || true
export ENGRAM_API_TOKEN

# UC-3: Check for inline correction
"$ENGRAM_BIN" correct --message "$CLAUDE_USER_MESSAGE" --data-dir "$ENGRAM_DATA"
