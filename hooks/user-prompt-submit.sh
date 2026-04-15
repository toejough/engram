#!/usr/bin/env bash
set -euo pipefail

# UserPromptSubmit hook — nudge agent to consider /prepare before new work.

jq -n '{hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: "Important reminders from the user: remember to call /learn at completion boundaries (task done, bug resolved, direction change, commit) and /prepare when starting new work. These are CRITICAL memory boundaries. If you are at one or recently completed work without calling /learn, PAUSE and CALL IT NOW. If you are at one or recently started work without calling /prepare, PAUSE and CALL IT NOW."}}'
