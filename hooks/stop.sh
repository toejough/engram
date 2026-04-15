#!/usr/bin/env bash
set -euo pipefail

# Stop hook — nudge agent to consider /learn after completing work.

jq -n '{hookSpecificOutput: {hookEventName: "Stop", additionalContext: "You just finished responding. Consider: did you just complete a task, resolve a bug, change direction, or make a commit? If so, call /learn to capture what was discovered."}}'
