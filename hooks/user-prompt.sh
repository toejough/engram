#!/usr/bin/env bash
# UserPromptSubmit hook: posts user prompt to engram API server.
# In stage 1 (pre-MCP), uses engram intent (blocking) to get surfaced memories.
# The agent name is set during /use-engram setup via ENGRAM_AGENT_NAME.

set -euo pipefail

if [ -z "${ENGRAM_AGENT_NAME:-}" ]; then
  exit 0  # Engram not active for this session.
fi

engram intent \
  --from "${ENGRAM_AGENT_NAME}:user" \
  --to engram-agent \
  --situation "${PROMPT:-}" \
  --planned-action ""
