#!/usr/bin/env bash
set -euo pipefail

# SessionStart hook — announce recall skill, build binary if needed.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# --- Sync portion: announce skills and surface memories ---
STATIC_MSG="[engram] Memory skills available. Call /prepare before starting new work. Call /learn after completing work. Call /recall to load previous session context. Call /remember to save something explicitly."

if [[ -x "$ENGRAM_BIN" ]]; then
    PREP_MEMORIES=$("$ENGRAM_BIN" recall --memories-only --query "when to call /prepare" 2>/dev/null || true)
    LEARN_MEMORIES=$("$ENGRAM_BIN" recall --memories-only --query "when to call /learn" 2>/dev/null || true)
    MEMORIES=""
    [[ -n "$PREP_MEMORIES" ]] && MEMORIES="${PREP_MEMORIES}"
    if [[ -n "$LEARN_MEMORIES" ]]; then
        [[ -n "$MEMORIES" ]] && MEMORIES="${MEMORIES}\n"
        MEMORIES="${MEMORIES}${LEARN_MEMORIES}"
    fi
    if [[ -n "$MEMORIES" ]]; then
        jq -n --arg ctx "${STATIC_MSG}\n\n${MEMORIES}" \
            '{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: $ctx}}'
    else
        jq -n --arg ctx "${STATIC_MSG}" \
            '{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: $ctx}}'
    fi
else
    jq -n --arg ctx "${STATIC_MSG}" \
        '{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: $ctx}}'
fi

# --- Async portion: build if needed ---
(
    NEEDS_BUILD=false
    if [[ ! -x "$ENGRAM_BIN" ]]; then
        NEEDS_BUILD=true
    elif [[ -d "$PLUGIN_ROOT" ]]; then
        if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" -print -quit 2>/dev/null | grep -q .; then
            NEEDS_BUILD=true
        fi
    fi

    if [[ "$NEEDS_BUILD" == "true" ]]; then
        mkdir -p "${ENGRAM_HOME}/bin"
        cd "$PLUGIN_ROOT"
        # Remove existing binary before rebuilding. If a Claude Code agent ran go build,
        # the binary would have inherited com.apple.provenance, causing macOS to SIGKILL
        # it on execution. Deleting first ensures we always rebuild from the clean shell.
        rm -f "$ENGRAM_BIN" "$ENGRAM_BIN.tmp"
        go build -o "$ENGRAM_BIN.tmp" ./cmd/engram/ || exit 0
        mv "$ENGRAM_BIN.tmp" "$ENGRAM_BIN"
    fi

    # Ensure symlink on PATH — replace stale regular files too
    SYMLINK_TARGET="$HOME/.local/bin/engram"
    if [[ -L "$SYMLINK_TARGET" ]] && [[ "$(readlink "$SYMLINK_TARGET")" == "$ENGRAM_BIN" ]]; then
        : # already correct
    else
        mkdir -p "$HOME/.local/bin"
        ln -sf "$ENGRAM_BIN" "$SYMLINK_TARGET" 2>/dev/null || true
    fi
) & disown

exit 0
