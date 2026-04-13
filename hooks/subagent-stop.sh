#!/usr/bin/env bash
# SubagentStop hook: posts subagent output to engram chat.
# Addressed to engram-agent (the lead already sees subagent output natively).

set -euo pipefail

if [ -z "${ENGRAM_AGENT_NAME:-}" ]; then
  exit 0
fi

engram post \
  --from "${ENGRAM_AGENT_NAME}:subagent:${SUBAGENT_ID:-unknown}" \
  --to engram-agent \
  --text "${SUBAGENT_OUTPUT:-}"
