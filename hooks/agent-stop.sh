#!/usr/bin/env bash
# Stop hook: posts agent output to engram API server.
# In stage 1 (pre-MCP), uses engram intent (blocking) to get surfaced memories.

set -euo pipefail

if [ -z "${ENGRAM_AGENT_NAME:-}" ]; then
  exit 0
fi

engram intent \
  --from "${ENGRAM_AGENT_NAME}" \
  --to engram-agent \
  --situation "${STOP_RESPONSE:-}" \
  --planned-action ""
