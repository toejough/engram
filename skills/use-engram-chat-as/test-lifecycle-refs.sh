#!/usr/bin/env bash
# Behavior baseline test for issue #545
# Each check greps for a pattern that MUST NOT appear in the final skill.
# Returns 0 (pass) when all patterns are absent, 1 (fail) when any are found.

set -euo pipefail
SKILL="$(dirname "$0")/SKILL.md"
FAILED=0

check_absent() {
  local description="$1"
  local pattern="$2"
  if grep -qF "$pattern" "$SKILL"; then
    echo "FAIL: Found forbidden pattern: $description"
    echo "      Pattern: $pattern"
    FAILED=1
  else
    echo "PASS: $description"
  fi
}

check_absent \
  'Step 7 references Background Monitor Pattern' \
  'Background Monitor Pattern, above'

check_absent \
  'Step 7 says spawn background monitor Agent' \
  'Spawn background monitor Agent'

check_absent \
  'Step 8 info message says Monitor active' \
  'Initialization complete. Monitor active.'

check_absent \
  'Step 9 says wait for monitor Agent notification' \
  'Wait for monitor Agent notification'

check_absent \
  'Step 10 says Monitor Agent returns' \
  'Monitor Agent returns semantic event'

check_absent \
  'Step 13 says Go to step 9' \
  'Go to step 9 -- ALWAYS'

check_absent \
  'The watch only ends when section present' \
  'The watch only ends when:'

check_absent \
  'Ready Messages says spawning the monitor (line 423)' \
  'or spawning the monitor. Announcing'

check_absent \
  'Ready Messages says before its monitor is watching (line 428)' \
  'before its monitor is watching'

check_absent \
  'Ready Messages says before spawning the monitor (line 430)' \
  'before spawning the monitor and posting'

check_absent \
  'Shutdown Protocol says Exit the monitor Agent loop' \
  'Exit the monitor Agent loop.'

check_absent \
  'Compaction Recovery Step 6 says Re-enter the fswatch loop' \
  'Re-enter the fswatch loop.'

check_absent \
  'Compaction Recovery says Continue from step 9 of the Agent Lifecycle' \
  'Continue the lifecycle from step 9 of the Agent Lifecycle'

check_absent \
  'Compaction Recovery guard note says in your watch loop' \
  'in your watch loop:'

check_absent \
  'Compaction Recovery guard note says waiting for fswatch' \
  'while the agent is waiting for fswatch.'

check_absent \
  'Common Mistakes row: Poll with sleep 2 loop / fswatch' \
  '| Poll with `sleep 2` loop | Use `fswatch -1`'

check_absent \
  'Common Mistakes row: Run fswatch/wc/grep / background monitor Agent' \
  '| Run fswatch/wc/grep directly in main agent context | Use background monitor Agent'

check_absent \
  'Common Mistakes row: Always re-enter the fswatch after posting' \
  'Always re-enter the fswatch after posting'

check_absent \
  'Common Mistakes row: Watch for next assignment (conflicts with stateless)' \
  'Completing a task != dismissed. Watch for next assignment'

check_absent \
  'Common Mistakes row: Exit monitor Agent loop' \
  'Exit monitor Agent loop after completing in-flight work'

if [ "$FAILED" -eq 0 ]; then
  echo ""
  echo "All checks passed — no stale monitor references found."
  exit 0
else
  echo ""
  echo "Some checks failed — stale monitor references remain."
  exit 1
fi
