#!/usr/bin/env bash
set -euo pipefail

# SessionStart hook — announce memory skills, surface memories, build binary if needed.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# --- Sync portion: announce skills ---
STATIC_MSG="[engram] Memory skills available. Call /prepare before starting new work. Call /learn after completing work. Call /recall to load previous session context. Call /remember to save something explicitly."

jq -n --arg ctx "$STATIC_MSG" \
    '{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: $ctx}}'

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

    # Ensure symlink on PATH
    mkdir -p "$HOME/.local/bin"
    ln -sf "$ENGRAM_BIN" "$HOME/.local/bin/engram" 2>/dev/null || true
) & disown

exit 0
