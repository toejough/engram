#!/bin/bash
# SKILL_test.sh - Tests for commit skill yield protocol compliance
# TASK-28: Update commit skill for yield protocol compatibility

set -euo pipefail

SKILL_FILE="$(dirname "$0")/SKILL.md"
SKILL_FULL="$(dirname "$0")/SKILL-full.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

pass() { echo -e "${GREEN}PASS${NC}: $1"; }
fail() { echo -e "${RED}FAIL${NC}: $1"; exit 1; }

echo "=== Commit Skill Yield Protocol Tests ==="
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

# Test 3: References YIELD.md not RESULT.md
echo "Test 3: References YIELD.md (not RESULT.md)"
if grep -q "RESULT.md" "$SKILL_FILE"; then
    fail "Still references old RESULT.md format"
fi
grep -q "YIELD.md" "$SKILL_FILE" || fail "Does not reference YIELD.md"
pass "References YIELD.md"

# Test 4: Documents yield types (complete, error)
echo "Test 4: Documents valid yield types"
grep -qi "complete" "$SKILL_FILE" || fail "Does not mention 'complete' yield type"
grep -qi "error" "$SKILL_FILE" || fail "Does not mention 'error' yield type"
pass "Documents yield types"

# Test 5: Shows TOML yield format
echo "Test 5: Shows TOML yield format example"
grep -q '\[yield\]' "$SKILL_FILE" || fail "Does not show [yield] TOML format"
grep -q 'type = ' "$SKILL_FILE" || fail "Does not show type field in yield"
pass "Shows TOML yield format"

# Test 6: Maintains commit functionality
echo "Test 6: Maintains core commit functionality"
grep -qi "git" "$SKILL_FILE" || fail "Does not mention git"
grep -qi "commit" "$SKILL_FILE" || fail "Does not mention commit"
grep -qi "AI-Used" "$SKILL_FILE" || fail "Does not mention AI-Used trailer"
pass "Maintains commit functionality"

# Test 7: Has TDD phase templates
echo "Test 7: Has TDD phase commit templates"
grep -qi "TDD Red" "$SKILL_FILE" || fail "Missing TDD Red template"
grep -qi "TDD Green" "$SKILL_FILE" || fail "Missing TDD Green template"
grep -qi "TDD Refactor" "$SKILL_FILE" || fail "Missing TDD Refactor template"
pass "Has TDD phase templates"

# Test 8: Documents files_modified in payload
echo "Test 8: Documents files_modified in complete yield"
grep -q "files_modified" "$SKILL_FILE" || fail "Does not document files_modified field"
pass "Documents files_modified"

echo ""
echo "=== All tests passed ==="
