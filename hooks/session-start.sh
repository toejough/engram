#!/usr/bin/env bash
set -euo pipefail

# SessionStart hook — announce memory skills, build binary if needed.

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_BIN="${HOME}/.local/bin/engram"

# Sync portion: skills announce.
# First-install fallback hardcode: used only when binary doesn't exist yet
# (chicken-and-egg with build-self). Once bootstrapped, the binary's
# canonical reminder text is used.
if [[ -x "$ENGRAM_BIN" ]]; then
    STATIC_MSG=$("$ENGRAM_BIN" reminder session-start 2>/dev/null || true)
fi
: "${STATIC_MSG:=[engram] Memory skills available. Call /prepare before starting new work. Call /learn after completing work. Call /recall to load previous session context. Call /remember to save something explicitly.}"

jq -n --arg ctx "$STATIC_MSG" \
    '{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: $ctx}}'

# Async portion: bootstrap with go build on first install, otherwise self-heal via build-self.
(
  if [[ ! -x "$ENGRAM_BIN" ]]; then
    mkdir -p "$(dirname "$ENGRAM_BIN")"
    cd "$PLUGIN_ROOT" && go build -o "$ENGRAM_BIN" ./cmd/engram/ 2>/dev/null || true
  else
    "$ENGRAM_BIN" build-self --if-stale --plugin-root "$PLUGIN_ROOT" --bin-path "$ENGRAM_BIN" 2>/dev/null || true
  fi
) & disown

exit 0
