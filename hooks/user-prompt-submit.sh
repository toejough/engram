#!/usr/bin/env bash
set -euo pipefail

# UserPromptSubmit hook — nudge agent to consider /prepare before new work.

jq -n '{hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: "A new user message just arrived. Consider: is this new work, a task switch, a new issue, or a debugging session? If so, call /prepare to load relevant context. Is this a completion boundary — task done, bug resolved, direction change, or commit? If so, call /learn to capture what was discovered."}}'
