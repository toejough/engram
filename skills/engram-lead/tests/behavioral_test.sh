#!/usr/bin/env bash
# Behavioral tests for skills/engram-lead/SKILL.md
# TDD: run before creating SKILL.md (RED), then after (GREEN).
# Usage: bash behavioral_test.sh

SKILL="$(dirname "$0")/../SKILL.md"
PASS=0
FAIL=0
FAILURES=()

pass() { echo "PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); FAILURES+=("$1"); }

assert_contains() {
  local desc="$1" pattern="$2"
  if grep -qF "$pattern" "$SKILL"; then pass "$desc"; else fail "$desc [pattern not found: '$pattern']"; fi
}

assert_not_contains() {
  local desc="$1" pattern="$2"
  if ! grep -qF "$pattern" "$SKILL"; then pass "$desc"; else fail "$desc [pattern should NOT be present: '$pattern']"; fi
}

echo "=== engram-lead SKILL.md behavioral tests ==="
echo ""

echo "--- Group 1: Identity ---"
assert_contains     "name is engram-lead"                       'name: engram-lead'
assert_not_contains "name is NOT engram-tmux-lead"              'name: engram-tmux-lead'
assert_not_contains "description does not mention 'via tmux'"   'via tmux'
assert_contains     "description triggers on /engram-lead"      '/engram-lead'
assert_contains     "description triggers on orchestrate agents" 'orchestrate agents'

echo ""
echo "--- Group 2: No tmux agent management ---"
assert_not_contains "no tmux send-keys for agents" 'tmux send-keys'
assert_not_contains "no pane-count tracking"  'RIGHT_PANE_COUNT'
assert_not_contains "no pane layout rules"    'main-vertical'
assert_not_contains "no tmux kill-pane for agents" 'kill-pane'

echo ""
echo "--- Group 3: Dispatch-based agent management ---"
assert_contains "dispatch assign used for spawning"  'engram dispatch assign'
assert_contains "dispatch stop used for stopping"    'engram dispatch stop'
assert_contains "dispatch drain in shutdown"         'engram dispatch drain'
assert_contains "dispatch status for compaction"     'engram dispatch status'

echo ""
echo "--- Group 4: Optional tmux tail pane ---"
assert_contains "TMUX env var check present"          '$TMUX'
assert_contains "tail -f used for chat observer"      'tail -f'
assert_contains "tmux command for tail pane"          'tmux'
assert_contains "@engram_name set for tail pane"      '@engram_name'

echo ""
echo "--- Group 5: Routing table ---"
assert_contains "routing table header"         '| User Request |'
assert_contains "Implement X route"            'Implement X'
assert_contains "Fix bug X route"              'Fix bug'
assert_contains "Review route"                 'Reviewer'
assert_contains "Tackle issue route"           'Planner'
assert_contains "parallel executors row"       'Parallel executor'

echo ""
echo "--- Group 6: Hold patterns ---"
assert_contains "hold acquire command"         'engram hold acquire'
assert_contains "hold release command"         'engram hold release'
assert_contains "hold check command"           'engram hold check'
assert_contains "Pair (Review) pattern"        'Pair (Review)'
assert_contains "Handoff pattern"              'Handoff'
assert_contains "Fan-In pattern"               'Fan-In'
assert_contains "Barrier pattern"              'Barrier'

echo ""
echo "--- Group 7: Spawn prompt template ---"
assert_contains "spawn prompt template"        'active <role>'
assert_contains "spawn template task field"    'Your task:'
assert_contains "spawn template DONE field"    'Post DONE:'
assert_contains "role naming convention"       'exec-auth'

echo ""
echo "--- Group 8: Operational sections ---"
assert_contains "escalation section"           'Escalation'
assert_contains "TIMEOUT from dead worker"     'TIMEOUT'
assert_contains "context pressure section"     'Context Pressure'
assert_contains "compaction recovery section"  'Compaction Recovery'
assert_contains "shutdown section"             'Shutdown'
assert_contains "never do implementation rule" 'Never do implementation yourself'

echo ""
echo "--- Group 9: use-engram-chat-as required ---"
assert_contains "use-engram-chat-as required"  'engram:use-engram-chat-as'

echo ""
echo "=== Results ==="
echo "PASS: $PASS"
echo "FAIL: $FAIL"
if [ "${#FAILURES[@]}" -gt 0 ]; then
  echo ""
  echo "Failed tests:"
  for f in "${FAILURES[@]}"; do echo "  - $f"; done
fi
if [ "$FAIL" -gt 0 ]; then exit 1; else echo "All tests passed."; exit 0; fi
