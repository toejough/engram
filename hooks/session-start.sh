#!/usr/bin/env bash
set -euo pipefail

# SessionStart hook — announce memory skills, surface memories, build binary if needed.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_BIN="${HOME}/.local/bin/engram"

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
        mkdir -p "$(dirname "$ENGRAM_BIN")"
        cd "$PLUGIN_ROOT"
        rm -f "$ENGRAM_BIN" "$ENGRAM_BIN.tmp"
        go build -o "$ENGRAM_BIN.tmp" ./cmd/engram/ || exit 0
        mv "$ENGRAM_BIN.tmp" "$ENGRAM_BIN"
    fi
) & disown

exit 0
