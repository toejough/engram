#!/usr/bin/env bash
set -euo pipefail

# Read failure details from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // "unknown"')"
TOOL_INPUT="$(echo "$STDIN_JSON" | jq -c '.tool_input // {}')"
ERROR="$(echo "$STDIN_JSON" | jq -r '.error // "unknown error"')"
IS_INTERRUPT="$(echo "$STDIN_JSON" | jq -r '.is_interrupt // false')"

# Skip advisory when user intentionally cancelled
if [[ "$IS_INTERRUPT" == "true" ]]; then
    exit 0
fi

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
DATA_DIR="${ENGRAM_DATA_DIR:-${ENGRAM_HOME}/data}"

# Build targeted advice based on tool type
case "$TOOL_NAME" in
    Read)
        ADVICE="Tool failed. Check that the file path exists and is correct, then retry or try an alternative. Continue working toward the intended outcome."
        ;;
    Bash)
        ADVICE="Tool failed. Diagnose the error from the output, fix the command or try an alternative approach. Continue working toward the intended outcome."
        ;;
    Edit)
        ADVICE="Tool failed. The old_string likely didn't match — re-read the file to get the exact current content, then retry. Continue working toward the intended outcome."
        ;;
    Write)
        ADVICE="Tool failed. Check that the directory exists and you have the correct path, then retry. Continue working toward the intended outcome."
        ;;
    Grep|Glob)
        ADVICE="Tool failed. Check the pattern syntax and path, then retry or try a different search approach. Continue working toward the intended outcome."
        ;;
    *)
        ADVICE="Tool failed. Diagnose the error, fix or try an alternative. Continue working toward the intended outcome."
        ;;
esac

# Surface relevant memories about this failure
MEMORY_CONTEXT=""
MEMORY_SUMMARY=""
if [[ "$TOOL_NAME" == "Bash" && -x "$ENGRAM_BIN" ]]; then
    SURFACE_OUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --tool-output "$ERROR" --tool-errored \
        --data-dir "$DATA_DIR" --format json 2>/dev/null) || SURFACE_OUT=""
    MEMORY_CONTEXT="$(echo "$SURFACE_OUT" | jq -r '.context // empty' 2>/dev/null)" || MEMORY_CONTEXT=""
    MEMORY_SUMMARY="$(echo "$SURFACE_OUT" | jq -r '.summary // empty' 2>/dev/null)" || MEMORY_SUMMARY=""
fi

# Combine static advisory with memory context
if [[ -n "$MEMORY_CONTEXT" ]]; then
    COMBINED="${ADVICE}
${MEMORY_CONTEXT}"
else
    COMBINED="$ADVICE"
fi

if [[ -n "$MEMORY_SUMMARY" ]]; then
    jq -n --arg sys "$MEMORY_SUMMARY" --arg ctx "$COMBINED" '{
        systemMessage: $sys,
        continue: true,
        suppressOutput: false,
        hookSpecificOutput: {
            hookEventName: "PostToolUseFailure",
            additionalContext: $ctx
        }
    }'
else
    jq -n --arg ctx "$COMBINED" '{
        continue: true,
        suppressOutput: false,
        hookSpecificOutput: {
            hookEventName: "PostToolUseFailure",
            additionalContext: $ctx
        }
    }'
fi
