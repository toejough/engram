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

# Consume pending maintenance results from async SessionStart (#370)
PENDING_SYS=""
PENDING_CTX=""
PENDING_FILE="$ENGRAM_HOME/pending-maintenance.json"
PENDING_TMP="$PENDING_FILE.consuming.$$"
if [[ -f "$PENDING_FILE" ]] && mv "$PENDING_FILE" "$PENDING_TMP" 2>/dev/null; then
    PENDING_SYS="$(jq -r '.systemMessage // empty' "$PENDING_TMP")"
    PENDING_CTX="$(jq -r '.additionalContext // empty' "$PENDING_TMP")"
    rm -f "$PENDING_TMP"
fi

# Skill/command file advisory for Write/Edit
if [[ ("$TOOL_NAME" == "Write" || "$TOOL_NAME" == "Edit") && \
      ("$FILE_PATH" == */skills/* || "$FILE_PATH" == */.claude/commands/*) ]]; then
    ADVISORY_CTX="You just edited a skill/command file — did you pressure-test the changes? Verify it still triggers correctly and handles edge cases."
    MERGED_SYS="$PENDING_SYS"
    MERGED_CTX="${PENDING_CTX:+$PENDING_CTX
}$ADVISORY_CTX"
    jq -n \
        --arg sys "$MERGED_SYS" \
        --arg ctx "$MERGED_CTX" \
        '{
            continue: true,
            suppressOutput: false,
            systemMessage: (if $sys == "" then null else $sys end),
            hookSpecificOutput: {
                hookEventName: "PostToolUse",
                additionalContext: $ctx
            }
        }'
    exit 0
fi

# Only surface memories for Bash tool calls
if [[ "$TOOL_NAME" != "Bash" ]]; then
    # Emit pending content if available before exiting (#370)
    if [[ -n "$PENDING_SYS" || -n "$PENDING_CTX" ]]; then
        jq -n \
            --arg sys "$PENDING_SYS" \
            --arg ctx "$PENDING_CTX" \
            '{
                systemMessage: $sys,
                continue: true,
                suppressOutput: false,
                hookSpecificOutput: {
                    hookEventName: "PostToolUse",
                    additionalContext: $ctx
                }
            }'
    fi
    exit 0
fi

# Surface memories relevant to this tool call and its output
if [[ -x "$ENGRAM_BIN" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --tool-output "$TOOL_RESPONSE" \
        --format json 2>/dev/null) || SURFACE_OUTPUT=""
    if [[ -n "$SURFACE_OUTPUT" ]]; then
        # Merge pending maintenance context with surfacing output (#370)
        SURFACE_SYS="$(echo "$SURFACE_OUTPUT" | jq -r '.summary // empty')"
        SURFACE_CTX="$(echo "$SURFACE_OUTPUT" | jq -r '.context // empty')"
        FINAL_SYS="${PENDING_SYS:+$PENDING_SYS
}$SURFACE_SYS"
        FINAL_CTX="${PENDING_CTX:+$PENDING_CTX
}$SURFACE_CTX"
        jq -n \
            --arg sys "$FINAL_SYS" \
            --arg ctx "$FINAL_CTX" \
            '{
                systemMessage: $sys,
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

# No surfacing output — emit pending content standalone if available (#370)
if [[ -n "$PENDING_SYS" || -n "$PENDING_CTX" ]]; then
    jq -n \
        --arg sys "$PENDING_SYS" \
        --arg ctx "$PENDING_CTX" \
        '{
            systemMessage: $sys,
            continue: true,
            suppressOutput: false,
            hookSpecificOutput: {
                hookEventName: "PostToolUse",
                additionalContext: $ctx
            }
        }'
fi
