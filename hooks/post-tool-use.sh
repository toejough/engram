#!/usr/bin/env bash
set -euo pipefail

# Read tool details from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // empty')"
TOOL_INPUT="$(echo "$STDIN_JSON" | jq -c '.tool_input // {}')"
TOOL_RESPONSE="$(echo "$STDIN_JSON" | jq -r '.tool_response // empty')"
FILE_PATH="$(echo "$STDIN_JSON" | jq -r '.tool_input.file_path // empty')"

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# Don't surface memories for any engram CLI calls (#352, #369)
if [[ "$TOOL_NAME" == "Bash" ]]; then
    BASH_CMD="$(echo "$STDIN_JSON" | jq -r '.tool_input.command // empty')"
    # Normalize ~/... to $HOME/... so both path forms match (#369)
    BASH_CMD_NORMALIZED="${BASH_CMD//\~\//$HOME/}"
    if [[ "$BASH_CMD_NORMALIZED" == *"$ENGRAM_BIN"* ]]; then
        exit 0
    fi
fi

# Skill/command file advisory for Write/Edit
if [[ ("$TOOL_NAME" == "Write" || "$TOOL_NAME" == "Edit") && \
      ("$FILE_PATH" == */skills/* || "$FILE_PATH" == */.claude/commands/*) ]]; then
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

# Only surface memories for Bash tool calls
if [[ "$TOOL_NAME" != "Bash" ]]; then
    exit 0
fi

# Surface memories relevant to this tool call and its output
if [[ -x "$ENGRAM_BIN" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --tool-output "$TOOL_RESPONSE" \
        --format json 2>/dev/null) || SURFACE_OUTPUT=""
    if [[ -n "$SURFACE_OUTPUT" ]]; then
        echo "$SURFACE_OUTPUT" | jq '{
            systemMessage: (.summary // empty),
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
