#!/usr/bin/env bash
set -euo pipefail

# PostToolUse hook — nudge agent to consider /learn and /prepare at boundaries.

ENGRAM_BIN="${HOME}/.local/bin/engram"
REMINDER=$("$ENGRAM_BIN" reminder post-tool)

jq -n --arg ctx "$REMINDER" \
    '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: $ctx}}'
