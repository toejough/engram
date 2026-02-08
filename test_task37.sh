#!/usr/bin/env bash
# Test suite for TASK-37: Universal yield capture in orchestrator
# Acceptance criteria:
# - Orchestrator runs `projctl memory extract -f .claude/projects/<issue>/result.toml -p <issue-id>` after producer yields
# - Extract runs BEFORE `projctl step complete`
# - Failures logged but do not block step completion
# - `grep -c "memory extract" ~/.claude/skills/project/SKILL.md` returns at least 1

set -e

SKILL_FILE="$HOME/.claude/skills/project/SKILL.md"
TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
FAIL_COUNT=0

echo "=== TASK-37: Universal Yield Capture Tests ==="
echo "Testing file: $SKILL_FILE"
echo ""

# Helper function to report test results
report_test() {
    local test_name="$1"
    local result="$2"

    if [ "$result" = "PASS" ]; then
        echo "✓ PASS: $test_name"
    else
        echo "✗ FAIL: $test_name"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# Test 1: memory extract command appears in SKILL.md
echo "Test 1: Memory extract command exists in SKILL.md"
if grep -q "memory extract" "$SKILL_FILE"; then
    report_test "memory extract command documented" "PASS"
    echo "   Unexpected: Feature already implemented!"
else
    report_test "memory extract command documented" "FAIL"
    echo "   Expected: Feature not yet implemented (RED phase)"
fi
echo ""

# Test 2: memory extract appears at least once
echo "Test 2: memory extract appears at least once"
EXTRACT_COUNT=$(grep -c "memory extract" "$SKILL_FILE" || true)
if [ "$EXTRACT_COUNT" -ge 1 ]; then
    report_test "memory extract count >= 1 (found: $EXTRACT_COUNT)" "PASS"
    echo "   Unexpected: Feature already implemented!"
else
    report_test "memory extract count >= 1 (found: $EXTRACT_COUNT)" "FAIL"
    echo "   Expected: Count is $EXTRACT_COUNT (RED phase)"
fi
echo ""

# Test 3: memory extract includes -f flag with result.toml path
echo "Test 3: memory extract uses -f flag with result.toml"
if grep -q "memory extract.*-f.*result\.toml" "$SKILL_FILE"; then
    report_test "memory extract with -f result.toml" "PASS"
    echo "   Unexpected: Feature already implemented!"
else
    report_test "memory extract with -f result.toml" "FAIL"
    echo "   Expected: Pattern not found (RED phase)"
fi
echo ""

# Test 4: memory extract includes -p flag with project/issue ID
echo "Test 4: memory extract uses -p flag for project ID"
if grep -q "memory extract.*-p" "$SKILL_FILE"; then
    report_test "memory extract with -p flag" "PASS"
    echo "   Unexpected: Feature already implemented!"
else
    report_test "memory extract with -p flag" "FAIL"
    echo "   Expected: Pattern not found (RED phase)"
fi
echo ""

# Test 5: memory extract appears in spawn-producer context
echo "Test 5: memory extract in spawn-producer handler section"
if grep -B20 -A20 "spawn-producer" "$SKILL_FILE" | grep -q "memory extract"; then
    report_test "memory extract in spawn-producer section" "PASS"
    echo "   Unexpected: Feature already implemented!"
else
    report_test "memory extract in spawn-producer section" "FAIL"
    echo "   Expected: Not in spawn-producer section (RED phase)"
fi
echo ""

# Test 6: memory extract appears BEFORE step complete
echo "Test 6: memory extract appears before 'step complete'"
# Search for a pattern where memory extract comes before step complete in spawn-producer section
if awk '/spawn-producer/,/^####/ {
    if (/memory extract/) extract_line=NR;
    if (/step complete/ && extract_line && NR > extract_line) found=1;
}
END { exit !found }' "$SKILL_FILE" 2>/dev/null; then
    report_test "memory extract before step complete" "PASS"
    echo "   Unexpected: Feature already implemented!"
else
    report_test "memory extract before step complete" "FAIL"
    echo "   Expected: Ordering not found (RED phase)"
fi
echo ""

# Test 7: Error handling documentation (failures non-blocking)
echo "Test 7: Extract failures are non-blocking (documented)"
if grep -A10 -B10 "memory extract" "$SKILL_FILE" 2>/dev/null | grep -qi "fail.*continue\|non-blocking\|best-effort\|warning"; then
    report_test "extract failures non-blocking" "PASS"
    echo "   Unexpected: Feature already implemented!"
else
    report_test "extract failures non-blocking" "FAIL"
    echo "   Expected: Error handling not documented (RED phase)"
fi
echo ""

# Test 8: Full command pattern with both flags
echo "Test 8: Complete memory extract command pattern"
if grep -q "projctl memory extract -f.*\.claude/projects.*result\.toml.*-p" "$SKILL_FILE"; then
    report_test "complete memory extract command" "PASS"
    echo "   Unexpected: Feature already implemented!"
else
    report_test "complete memory extract command" "FAIL"
    echo "   Expected: Complete command not found (RED phase)"
fi
echo ""

# Summary
echo "==================================="
echo "Test Summary:"
echo "  Total tests: 8"
echo "  Expected to fail: 8 (TDD red phase)"
echo "  Actual failures: $FAIL_COUNT"
echo ""

# In TDD red phase, we WANT all tests to fail
# So we verify that all tests ARE failing
if [ "$FAIL_COUNT" -eq 0 ]; then
    # All tests passed - this is BAD in red phase (means feature already exists)
    echo "❌ RED PHASE VERIFICATION FAILED"
    echo "   All tests passed, but feature should not exist yet!"
    exit 1
else
    # Tests are failing - this is GOOD in red phase
    echo "✓ RED PHASE VERIFIED"
    echo "  Tests correctly fail (feature not yet implemented)"
    exit 0
fi
