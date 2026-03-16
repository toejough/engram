#!/usr/bin/env bash
set -euo pipefail

# Read tool details from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // empty')"
TOOL_INPUT="$(echo "$STDIN_JSON" | jq -c '.tool_input // {}')"
TOOL_RESPONSE="$(echo "$STDIN_JSON" | jq -r '.tool_response // empty')"
FILE_PATH="$(echo "$STDIN_JSON" | jq -r '.tool_input.file_path // empty')"

# Only fire for Write and Edit tools (T-213)
if [[ "$TOOL_NAME" != "Write" && "$TOOL_NAME" != "Edit" ]]; then
    exit 0
fi

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
DATA_DIR="${ENGRAM_DATA_DIR:-${ENGRAM_HOME}/data}"

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

# Surface memories relevant to this tool call and its output
if [[ -x "$ENGRAM_BIN" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --tool-output "$TOOL_RESPONSE" \
        --data-dir "$DATA_DIR" --format json 2>/dev/null) || SURFACE_OUTPUT=""
    if [[ -n "$SURFACE_OUTPUT" ]]; then
        echo "$SURFACE_OUTPUT" | jq '{
            continue: true,
            suppressOutput: false,
            hookSpecificOutput: {
                hookEventName: "PostToolUse",
                additionalContext: .context
            }
        }'
        exit 0
    fi
fi
