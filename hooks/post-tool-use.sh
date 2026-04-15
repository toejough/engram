#!/usr/bin/env bash
set -euo pipefail

# PostToolUse hook — nudge agent to consider /learn and /prepare at boundaries.

jq -n '{hookSpecificOutput: {hookEventName: "PostToolUse", additionalContext: "Important reminders from the user: remember to call /learn at completion boundaries (task done, bug resolved, direction change, commit) and /prepare when starting new work. These are CRITICAL memory boundaries. If you are at one or recently completed work without calling /learn, PAUSE and CALL IT NOW. If you are at one or recently started work without calling /prepare, PAUSE and CALL IT NOW."}}'
