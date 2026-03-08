#!/usr/bin/env bash
set -euo pipefail

# Read tool name and input from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // empty')"
FILE_PATH="$(echo "$STDIN_JSON" | jq -r '.tool_input.file_path // empty')"

# Only fire for Write and Edit tools (T-213)
if [[ "$TOOL_NAME" != "Write" && "$TOOL_NAME" != "Edit" ]]; then
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DATA_DIR="${ENGRAM_DATA_DIR:-$HOME/.engram}"

# Skill/command file advisory
if [[ "$FILE_PATH" == */skills/* || "$FILE_PATH" == */.claude/commands/* ]]; then
    jq -n '{
        continue: true,
        suppressOutput: false,
        hookSpecificOutput: {
            hookEventName: "PostToolUse",
            additionalContext: "You just edited a skill/command file — did you pressure-test the changes? Verify it still triggers correctly and handles edge cases."
        }
    }'
    exit 0
fi

# Proactive reminder (UC-18)
if [[ -n "$FILE_PATH" ]]; then
    REMINDER="$("$SCRIPT_DIR/../cmd/engram/engram" remind --data-dir "$DATA_DIR" --file-path "$FILE_PATH" 2>/dev/null || true)"
    if [[ -n "$REMINDER" ]]; then
        jq -n --arg ctx "$REMINDER" '{
            continue: true,
            suppressOutput: false,
            hookSpecificOutput: {
                hookEventName: "PostToolUse",
                additionalContext: $ctx
            }
        }'
        exit 0
    fi
fi
