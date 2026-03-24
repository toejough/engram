#!/usr/bin/env bash
set -euo pipefail

# Fast sync SessionStart hook — emits static context only (#370).
# Slow work (build, maintain) runs in the async session-start.sh hook.

SYSTEM_MSG="[engram] Say /recall to load context from previous sessions, or /recall <query> to search session history."
ADDITIONAL_CTX="[engram] Mid-turn user messages (delivered via system-reminder) bypass engram hooks. If you receive a mid-turn correction or instruction, capture it by running: ~/.claude/engram/bin/engram correct --message '<the user message>'"

jq -n \
    --arg sys "$SYSTEM_MSG" \
    --arg add "$ADDITIONAL_CTX" \
    '{systemMessage: $sys, additionalContext: $add}'
