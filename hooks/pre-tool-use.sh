#!/usr/bin/env bash
set -euo pipefail

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# Build if missing or stale (source newer than binary)
NEEDS_BUILD=false
if [[ ! -x "$ENGRAM_BIN" ]]; then
    NEEDS_BUILD=true
elif [[ -d "$PLUGIN_ROOT" ]]; then
    # Rebuild if any Go source file is newer than the binary
    if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" -print -quit 2>/dev/null | grep -q .; then
        NEEDS_BUILD=true
    fi
fi

if [[ "$NEEDS_BUILD" == "true" ]]; then
    mkdir -p "${ENGRAM_HOME}/bin"
    cd "$PLUGIN_ROOT"
    go build -o "$ENGRAM_BIN" ./cmd/engram/ 2>/dev/null || { echo "[engram] build failed — is Go installed?" >&2; exit 0; }
fi

# Read tool name and input from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // empty')"
TOOL_INPUT="$(echo "$STDIN_JSON" | jq -c '.tool_input // {}')"

# Don't surface memories for any engram CLI calls (#352, #369)
if [[ "$TOOL_NAME" == "Bash" ]]; then
    BASH_CMD="$(echo "$STDIN_JSON" | jq -r '.tool_input.command // empty')"
    if [[ "$BASH_CMD" == *"$ENGRAM_BIN"* ]]; then
        exit 0
    fi
fi

# Only surface memories for Bash tool calls — non-Bash tools produce near-random BM25 matches
if [[ "$TOOL_NAME" != "Bash" ]]; then
    exit 0
fi

# UC-2: Surface relevant memories before tool use
if [[ -n "$TOOL_NAME" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --format json) || true
    if [[ -n "$SURFACE_OUTPUT" ]]; then
        echo "$SURFACE_OUTPUT" | jq '{
            systemMessage: (.summary // empty),
            continue: true,
            suppressOutput: false,
            hookSpecificOutput: {
                hookEventName: "PreToolUse",
                permissionDecision: "allow",
                permissionDecisionReason: "",
                additionalContext: .context
            }
        }'
    fi
fi
