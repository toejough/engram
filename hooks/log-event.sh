#!/usr/bin/env bash
set -euo pipefail

# Logs hook event data to a file for observability.
# Used to understand which events fire at which session boundaries.

LOG_DIR="${HOME}/.claude/engram/data/hook-events"
mkdir -p "$LOG_DIR"

HOOK_JSON="$(cat)"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%S.%NZ 2>/dev/null || date -u +%Y-%m-%dT%H:%M:%SZ)"
EVENT_NAME="$(echo "$HOOK_JSON" | jq -r '.hook_event_name // "unknown"' 2>/dev/null || echo "unknown")"

echo "${TIMESTAMP} ${EVENT_NAME} ${HOOK_JSON}" >> "${LOG_DIR}/events.log"
