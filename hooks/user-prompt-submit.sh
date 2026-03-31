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
    if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" \
        -print -quit 2>/dev/null | grep -q .; then
        NEEDS_BUILD=true
    fi
fi

if [[ "$NEEDS_BUILD" == "true" ]]; then
    mkdir -p "${ENGRAM_HOME}/bin"
    cd "$PLUGIN_ROOT"
    go build -o "$ENGRAM_BIN" ./cmd/engram/ 2>/dev/null \
        || { echo "[engram] build failed — is Go installed?" >&2; exit 0; }
fi

# Read hook JSON from stdin — capture full input for multiple field reads
HOOK_JSON="$(cat)"
USER_MESSAGE="$(echo "$HOOK_JSON" | jq -r '.prompt // empty')"
TRANSCRIPT_PATH="$(echo "$HOOK_JSON" | jq -r '.transcript_path // empty')"
SESSION_ID="$(echo "$HOOK_JSON" | jq -r '.session_id // empty')"

# Consume pending maintenance results from async SessionStart (#370)
# Consumed here (not Pre/PostToolUse) so subagent tool calls don't eat it.
PENDING_SYS=""
PENDING_CTX=""
PENDING_FILE="$ENGRAM_HOME/pending-maintenance.json"
PENDING_TMP="$PENDING_FILE.consuming.$$"
if [[ -f "$PENDING_FILE" ]] && mv "$PENDING_FILE" "$PENDING_TMP" 2>/dev/null; then
    PENDING_SYS="$(jq -r '.systemMessage // empty' "$PENDING_TMP")"
    PENDING_CTX="$(jq -r '.additionalContext // empty' "$PENDING_TMP")"
    rm -f "$PENDING_TMP"
fi

# Skip surfacing for engram skill invocations (#369)
# Still consume pending file above so it's not stuck forever.
SKILL_CMD="${USER_MESSAGE%% *}"
if [[ "$SKILL_CMD" == /* ]]; then
    SKILL_NAME="${SKILL_CMD#/}"
    if [[ -d "$PLUGIN_ROOT/skills/$SKILL_NAME" ]]; then
        # Emit pending content even when skipping surfacing
        if [[ -n "$PENDING_SYS" || -n "$PENDING_CTX" ]]; then
            jq -n \
                --arg sys "$PENDING_SYS" \
                --arg ctx "$PENDING_CTX" \
                '{systemMessage: $sys, hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: $ctx}}'
        fi
        exit 0
    fi
fi

# UC-3: Check for inline correction (with transcript context)
CORRECT_ARGS=(correct --message "$USER_MESSAGE")
if [[ -n "$TRANSCRIPT_PATH" ]]; then
    CORRECT_ARGS+=(--transcript-path "$TRANSCRIPT_PATH")
fi

# Capture correct output to avoid mixing plain text + JSON on stdout
CORRECT_OUTPUT=""
if [[ -n "$USER_MESSAGE" ]]; then
    CORRECT_OUTPUT=$("$ENGRAM_BIN" "${CORRECT_ARGS[@]}") || true
fi

# UC-2: Surface relevant memories
SURFACE_OUTPUT=""
if [[ -n "$USER_MESSAGE" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode prompt \
        --message "$USER_MESSAGE" --session-id "$SESSION_ID" --format json) || true
fi

# Combine into single JSON output — merge pending maintenance, surface, correct
FINAL_SYS="$PENDING_SYS"
FINAL_CTX="$PENDING_CTX"

if [[ -n "$SURFACE_OUTPUT" ]]; then
    SURFACE_SYS="$(echo "$SURFACE_OUTPUT" | jq -r '.summary // empty')"
    SURFACE_CTX="$(echo "$SURFACE_OUTPUT" | jq -r '.context // empty')"
    FINAL_SYS="${FINAL_SYS:+$FINAL_SYS
}$SURFACE_SYS"
    FINAL_CTX="${FINAL_CTX:+$FINAL_CTX
}$SURFACE_CTX"
fi

if [[ -n "$CORRECT_OUTPUT" ]]; then
    FINAL_SYS="${FINAL_SYS:+$FINAL_SYS
}$CORRECT_OUTPUT"
fi

if [[ -n "$FINAL_SYS" || -n "$FINAL_CTX" ]]; then
    jq -n \
        --arg sys "$FINAL_SYS" \
        --arg ctx "$FINAL_CTX" \
        '{systemMessage: $sys, hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: $ctx}}'
fi
