#!/bin/bash
# SKILL_test.sh - Tests for commit skill

set -euo pipefail

SKILL_FILE="$(dirname "$0")/SKILL.md"
SKILL_FULL="$(dirname "$0")/SKILL-full.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

pass() { echo -e "${GREEN}PASS${NC}: $1"; }
fail() { echo -e "${RED}FAIL${NC}: $1"; exit 1; }

echo "=== Commit Skill Tests ==="
echo ""

# Test 1: Skill file exists
echo "Test 1: SKILL.md exists"
[[ -f "$SKILL_FILE" ]] || fail "SKILL.md not found"
pass "SKILL.md exists"

# Test 2: Has required frontmatter
echo "Test 2: Has required frontmatter fields"
grep -q "^name: commit" "$SKILL_FILE" || fail "Missing name field"
grep -q "^user-invocable: true" "$SKILL_FILE" || fail "Missing user-invocable field"
pass "Has required frontmatter"

# Test 3: No legacy YIELD.md or RESULT.md references
echo "Test 3: No legacy references"
if grep -q "RESULT.md" "$SKILL_FILE"; then
    fail "Still references old RESULT.md format"
fi
if grep -q "YIELD.md" "$SKILL_FILE"; then
    fail "Still references legacy YIELD.md"
fi
pass "No legacy references"

# Test 4: Documents completion and error reporting
echo "Test 4: Documents result reporting"
grep -qi "complete\|success" "$SKILL_FILE" || fail "Does not mention completion reporting"
grep -qi "error\|failure" "$SKILL_FILE" || fail "Does not mention error reporting"
pass "Documents result reporting"

# Test 5: Maintains core commit functionality
echo "Test 5: Maintains core commit functionality"
grep -qi "git" "$SKILL_FILE" || fail "Does not mention git"
grep -qi "commit" "$SKILL_FILE" || fail "Does not mention commit"
grep -qi "AI-Used" "$SKILL_FILE" || fail "Does not mention AI-Used trailer"
pass "Maintains commit functionality"

# Test 6: Has TDD phase templates
echo "Test 6: Has TDD phase commit templates"
grep -qi "TDD Red" "$SKILL_FILE" || fail "Missing TDD Red template"
grep -qi "TDD Green" "$SKILL_FILE" || fail "Missing TDD Green template"
grep -qi "TDD Refactor" "$SKILL_FILE" || fail "Missing TDD Refactor template"
pass "Has TDD phase templates"

# Test 7: Documents files modified in completion
echo "Test 7: Documents files modified in completion"
grep -qi "files.modified\|files modified" "$SKILL_FILE" || fail "Does not document files modified"
pass "Documents files modified"

echo ""
echo "=== All tests passed ==="
