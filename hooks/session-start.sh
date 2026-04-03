#!/usr/bin/env bash
set -euo pipefail

# SessionStart hook — announce recall skill, build binary if needed.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# --- Sync portion: announce recall ---
jq -n '{systemMessage: "[engram] Say /recall to load context from previous sessions, or /recall <query> to search session history."}'

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
        go build -o "$ENGRAM_BIN.tmp" ./cmd/engram/ || exit 0
        mv "$ENGRAM_BIN.tmp" "$ENGRAM_BIN"
    fi

    # Ensure symlink on PATH
    SYMLINK_TARGET="$HOME/.local/bin/engram"
    if [[ ! -e "$SYMLINK_TARGET" ]] && [[ ! -L "$SYMLINK_TARGET" ]]; then
        mkdir -p "$HOME/.local/bin"
        ln -s "$ENGRAM_BIN" "$SYMLINK_TARGET" 2>/dev/null || true
    fi
) & disown

exit 0
