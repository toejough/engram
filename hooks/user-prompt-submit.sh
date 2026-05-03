#!/usr/bin/env bash
set -euo pipefail

# UserPromptSubmit hook — nudge agent to consider /prepare before new work.

ENGRAM_BIN="${HOME}/.local/bin/engram"
REMINDER=$("$ENGRAM_BIN" reminder user-prompt)

jq -n --arg ctx "$REMINDER" \
    '{hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: $ctx}}'
