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

# Only surface memories for Bash tool calls — non-Bash tools produce near-random BM25 matches
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
                    hookEventName: "PreToolUse",
                    permissionDecision: "allow",
                    permissionDecisionReason: "",
                    additionalContext: $ctx
                }
            }'
    fi
    exit 0
fi

# UC-2: Surface relevant memories before tool use
if [[ -n "$TOOL_NAME" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --format json) || true
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
                    hookEventName: "PreToolUse",
                    permissionDecision: "allow",
                    permissionDecisionReason: "",
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
                hookEventName: "PreToolUse",
                permissionDecision: "allow",
                permissionDecisionReason: "",
                additionalContext: $ctx
            }
        }'
fi
