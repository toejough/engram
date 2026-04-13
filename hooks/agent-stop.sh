#!/usr/bin/env bash
# Stop hook: posts agent output to engram API server.
# Uses engram intent (blocking) to get surfaced memories.
# Agent name is read from a marker file written by /engram-up.

set -euo pipefail

ENGRAM_DATA="${XDG_DATA_HOME:-$HOME/.local/share}/engram"
SLUG=$(echo "$PWD" | tr '/' '-')
MARKER="$ENGRAM_DATA/chat/${SLUG}.agent-name"

if [ ! -f "$MARKER" ]; then
  exit 0
fi

ENGRAM_AGENT_NAME=$(cat "$MARKER")
if [ -z "$ENGRAM_AGENT_NAME" ]; then
  exit 0
fi

engram intent \
  --from "${ENGRAM_AGENT_NAME}" \
  --to engram-agent \
  --situation "${STOP_RESPONSE:-}" \
  --planned-action ""
