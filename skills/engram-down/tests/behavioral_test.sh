#!/usr/bin/env bash
# Behavioral tests for skills/engram-down/SKILL.md
# TDD: run before editing SKILL.md (RED), then after (GREEN).
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

echo "=== engram-down SKILL.md behavioral tests ==="
echo ""

echo "--- Group 1: Conditional tmux tail kill ---"
assert_contains     "checks TMUX env var"               '$TMUX'
assert_contains     "uses @engram_name to find pane"    '@engram_name'
assert_not_contains "does not use pane_title format string to find pane" '#{pane_title}'
assert_contains     "kill-pane command present"         'kill-pane'
assert_contains     "conditional: skip if not in tmux"  'not in tmux'

echo ""
echo "--- Group 2: Core shutdown sequence intact ---"
assert_contains "dispatch drain present"   'dispatch drain'
assert_contains "dispatch stop present"    'dispatch stop'
assert_contains "scan LEARNED messages"    'LEARNED'
assert_contains "session summary step"     'session summary'
assert_contains "preserve chat file"       'chat file'

echo ""
echo "--- Group 3: Common mistakes table ---"
assert_contains "common mistakes table"    'Common Mistakes'

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
