#!/usr/bin/env bash
# Behavioral tests for skills/engram-up/SKILL.md
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

echo "=== engram-up SKILL.md behavioral tests ==="
echo ""

echo "--- Group 1: Delegates to engram-lead ---"
assert_contains     "references engram-lead"                   'engram-lead'
assert_contains     "references use-engram-chat-as"            'engram:use-engram-chat-as'
assert_not_contains "does not duplicate routing table"         '| User Request |'
assert_not_contains "does not duplicate hold patterns"         'engram hold acquire'
assert_not_contains "does not duplicate spawn template"        'active <role>'
assert_not_contains "does not duplicate shutdown steps"        'dispatch drain'

echo ""
echo "--- Group 2: Triggers ---"
assert_contains "triggers on /engram"       '/engram'
assert_contains "triggers on /engram-up"    '/engram-up'
assert_contains "triggers on start engram"  'start engram'

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
