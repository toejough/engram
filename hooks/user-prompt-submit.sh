#!/usr/bin/env bash
set -euo pipefail

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
ENGRAM_DATA="${ENGRAM_HOME}/data"

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

# Platform-aware OAuth token retrieval (DES-3)
TOKEN=""
if [[ "$(uname)" == "Darwin" ]]; then
    TOKEN=$(security find-generic-password \
        -s "Claude Code-credentials" -w 2>/dev/null \
        | python3 -c \
        "import sys,json; print(json.load(sys.stdin)['claudeAiOauth']['accessToken'])" \
        2>/dev/null) || true
fi
export ENGRAM_API_TOKEN="${TOKEN:-${ENGRAM_API_TOKEN:-}}"

# Read hook JSON from stdin — capture full input for multiple field reads
HOOK_JSON="$(cat)"
USER_MESSAGE="$(echo "$HOOK_JSON" | jq -r '.prompt // empty')"
TRANSCRIPT_PATH="$(echo "$HOOK_JSON" | jq -r '.transcript_path // empty')"

# UC-3: Check for inline correction (with transcript context)
CORRECT_ARGS=(correct --message "$USER_MESSAGE" --data-dir "$ENGRAM_DATA")
if [[ -n "$TRANSCRIPT_PATH" ]]; then
    CORRECT_ARGS+=(--transcript-path "$TRANSCRIPT_PATH")
fi

if [[ -n "$USER_MESSAGE" ]]; then
    "$ENGRAM_BIN" "${CORRECT_ARGS[@]}"
fi

# UC-2: Surface relevant memories
if [[ -n "$USER_MESSAGE" ]]; then
    "$ENGRAM_BIN" surface --mode prompt \
        --message "$USER_MESSAGE" --data-dir "$ENGRAM_DATA"
fi
