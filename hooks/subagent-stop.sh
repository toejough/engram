#!/usr/bin/env bash
# SubagentStop hook: posts subagent output to engram chat.
# Addressed to engram-agent (the lead already sees subagent output natively).
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

engram post \
  --from "${ENGRAM_AGENT_NAME}:subagent:${SUBAGENT_ID:-unknown}" \
  --to engram-agent \
  --text "${SUBAGENT_OUTPUT:-}"
