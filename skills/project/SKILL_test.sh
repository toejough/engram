#!/bin/bash
# SKILL_test.sh - Tests for project orchestrator yield protocol compliance
# TASK-29: Update /project skill for new dispatch

set -euo pipefail

SKILL_FILE="$(dirname "$0")/SKILL.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass() { echo -e "${GREEN}PASS${NC}: $1"; }
fail() { echo -e "${RED}FAIL${NC}: $1"; exit 1; }

echo "=== Project Skill Yield Protocol Tests ==="
echo ""

# Test 1: Skill file exists
echo "Test 1: SKILL.md exists"
[[ -f "$SKILL_FILE" ]] || fail "SKILL.md not found"
pass "SKILL.md exists"

# Test 2: Has required frontmatter
echo "Test 2: Has required frontmatter fields"
grep -q "^name: project" "$SKILL_FILE" || fail "Missing name field"
grep -q "^user-invocable: true" "$SKILL_FILE" || fail "Missing user-invocable field"
pass "Has required frontmatter"

# Test 3: Dispatches to new producer skill names
echo "Test 3: Dispatches to producer skills"
grep -qi "pm-interview-producer\|pm-infer-producer" "$SKILL_FILE" || fail "Missing PM producer skill dispatch"
pass "Dispatches to producer skills"

# Test 4: Dispatches to new QA skill names
echo "Test 4: Dispatches to QA skills"
grep -qi "pm-qa\|design-qa\|arch-qa" "$SKILL_FILE" || fail "Missing QA skill dispatch"
pass "Dispatches to QA skills"

# Test 5: Documents pair loop pattern
echo "Test 5: Documents PAIR LOOP pattern"
grep -qi "pair.*loop\|producer.*qa" "$SKILL_FILE" || fail "Missing pair loop documentation"
pass "Documents pair loop pattern"

# Test 6: Handles yield types
echo "Test 6: Handles yield protocol types"
grep -qi "complete\|approved" "$SKILL_FILE" || fail "Missing complete/approved handling"
grep -qi "improvement-request" "$SKILL_FILE" || fail "Missing improvement-request handling"
grep -qi "escalate" "$SKILL_FILE" || fail "Missing escalation handling"
pass "Handles yield types"

# Test 7: Handles need-context via context-explorer
echo "Test 7: Dispatches need-context to context-explorer"
grep -qi "need-context\|context-explorer" "$SKILL_FILE" || fail "Missing need-context handling"
pass "Dispatches need-context"

# Test 8: Provides yield_path in context
echo "Test 8: Documents yield_path context provision"
grep -qi "yield_path\|output\.yield" "$SKILL_FILE" || fail "Missing yield_path documentation"
pass "Documents yield_path"

# Test 9: References YIELD.md
echo "Test 9: References YIELD.md"
grep -q "YIELD.md" "$SKILL_FILE" || fail "Does not reference YIELD.md"
pass "References YIELD.md"

# Test 10: Maintains state machine phases
echo "Test 10: Maintains core state machine"
grep -qi "pm\|design\|arch\|implementation" "$SKILL_FILE" || fail "Missing phase references"
pass "Maintains state machine"

echo ""
echo "=== All tests passed ==="
