#!/usr/bin/env bash
# UserPromptSubmit hook: posts user prompt to engram API server.
# Uses engram intent (blocking) to get surfaced memories.
# Agent name is read from a marker file written by /engram-up.

set -euo pipefail

ENGRAM_DATA="${XDG_DATA_HOME:-$HOME/.local/share}/engram"
SLUG=$(echo "$PWD" | tr '/' '-')
MARKER="$ENGRAM_DATA/chat/${SLUG}.agent-name"

if [ ! -f "$MARKER" ]; then
  exit 0  # Engram not active for this session.
fi

ENGRAM_AGENT_NAME=$(cat "$MARKER")
if [ -z "$ENGRAM_AGENT_NAME" ]; then
  exit 0
fi

engram intent \
  --from "${ENGRAM_AGENT_NAME}:user" \
  --to engram-agent \
  --situation "${PROMPT:-}" \
  --planned-action ""
