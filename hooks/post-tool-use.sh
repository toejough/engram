#!/usr/bin/env bash
set -euo pipefail

# Read tool name and input from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // empty')"
FILE_PATH="$(echo "$STDIN_JSON" | jq -r '.tool_input.file_path // empty')"

# Check if Write/Edit on skill or command files
if [[ "$TOOL_NAME" == "Write" || "$TOOL_NAME" == "Edit" ]]; then
    if [[ "$FILE_PATH" == */skills/* || "$FILE_PATH" == */.claude/commands/* ]]; then
        jq -n '{
            continue: true,
            suppressOutput: false,
            hookSpecificOutput: {
                hookEventName: "PostToolUse",
                additionalContext: "You just edited a skill/command file — did you pressure-test the changes? Verify it still triggers correctly and handles edge cases."
            }
        }'
    fi
fi
