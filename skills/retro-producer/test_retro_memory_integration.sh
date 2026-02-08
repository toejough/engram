#!/usr/bin/env bash
# Test: TASK-30 - Retro producer memory reads and writes
# Traces to: ARCH-057, REQ-011, DES-029
#
# This test verifies that retro-producer SKILL.md includes:
# 1. Memory queries in GATHER phase
# 2. Memory learns in PRODUCE phase
# 3. Proper prefixes for structured learning capture

set -euo pipefail

SKILL_FILE="${HOME}/.claude/skills/retro-producer/SKILL.md"
FAILURES=0

# Helper function for test assertions
assert_grep_count() {
    local pattern="$1"
    local expected_min="$2"
    local description="$3"

    local actual
    actual=$(grep -c "$pattern" "$SKILL_FILE" || true)

    if [[ "$actual" -ge "$expected_min" ]]; then
        echo "✓ PASS: $description (found $actual, expected >= $expected_min)"
    else
        echo "✗ FAIL: $description (found $actual, expected >= $expected_min)"
        FAILURES=$((FAILURES + 1))
    fi
}

# Helper function for exact phrase matching
assert_contains_phrase() {
    local phrase="$1"
    local description="$2"

    if grep -q "$phrase" "$SKILL_FILE"; then
        echo "✓ PASS: $description"
    else
        echo "✗ FAIL: $description - phrase not found: '$phrase'"
        FAILURES=$((FAILURES + 1))
    fi
}

echo "Testing retro-producer memory integration..."
echo ""

# AC-1: GATHER queries "retrospective challenges" and "process improvement recommendations"
echo "## Testing GATHER phase memory queries"
assert_contains_phrase "projctl memory query.*retrospective challenges" \
    "GATHER phase queries retrospective challenges"

assert_contains_phrase "projctl memory query.*process improvement recommendations" \
    "GATHER phase queries process improvement recommendations"

# AC-2: At least 2 memory query calls
assert_grep_count "memory query" 2 \
    "At least 2 memory query commands present"

echo ""
echo "## Testing PRODUCE phase memory writes"

# AC-3: PRODUCE writes successes with "Success:" prefix
assert_contains_phrase "projctl memory learn.*Success:" \
    "PRODUCE phase writes successes with 'Success:' prefix"

# AC-4: PRODUCE writes challenges with "Challenge:" prefix
assert_contains_phrase "projctl memory learn.*Challenge:" \
    "PRODUCE phase writes challenges with 'Challenge:' prefix"

# AC-5: PRODUCE writes recommendations with "Retro recommendation:" prefix
assert_contains_phrase "projctl memory learn.*Retro recommendation:" \
    "PRODUCE phase writes recommendations with 'Retro recommendation:' prefix"

# AC-6: At least 3 memory learn calls
assert_grep_count "memory learn" 3 \
    "At least 3 memory learn commands present"

echo ""
echo "## Testing structure integration"

# Verify memory queries appear in GATHER section
if grep -A 20 "^### GATHER" "$SKILL_FILE" | grep -q "memory query"; then
    echo "✓ PASS: Memory queries appear in GATHER section"
else
    echo "✗ FAIL: Memory queries not found in GATHER section"
    FAILURES=$((FAILURES + 1))
fi

# Verify memory learns appear in PRODUCE section
if grep -A 30 "^### PRODUCE" "$SKILL_FILE" | grep -q "memory learn"; then
    echo "✓ PASS: Memory learns appear in PRODUCE section"
else
    echo "✗ FAIL: Memory learns not found in PRODUCE section"
    FAILURES=$((FAILURES + 1))
fi

echo ""
echo "========================================"
if [[ $FAILURES -eq 0 ]]; then
    echo "ALL TESTS PASSED"
    exit 0
else
    echo "TESTS FAILED: $FAILURES failure(s)"
    exit 1
fi
