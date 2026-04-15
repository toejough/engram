#!/usr/bin/env bash
set -euo pipefail

# PostToolUse hook — nudge agent to consider /learn and /prepare at boundaries.

jq -n '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: "Remember to call /learn at completion boundaries (task done, bug resolved, direction change, commit) and /prepare when starting new work."}}'
