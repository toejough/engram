#!/usr/bin/env bash
set -euo pipefail

# Read failure details from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // "unknown"')"
ERROR="$(echo "$STDIN_JSON" | jq -r '.error // "unknown error"')"
IS_INTERRUPT="$(echo "$STDIN_JSON" | jq -r '.is_interrupt // false')"

# Skip advisory when user intentionally cancelled
if [[ "$IS_INTERRUPT" == "true" ]]; then
    exit 0
fi

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

jq -n --arg ctx "$ADVICE" '{
    continue: true,
    suppressOutput: false,
    hookSpecificOutput: {
        hookEventName: "PostToolUseFailure",
        additionalContext: $ctx
    }
}'
