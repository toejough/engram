#!/usr/bin/env bash
# Test: TASK-27 Architecture interview producer memory reads
# Traces to: ARCH-055, REQ-008
#
# IMPORTANT: These tests are expected to FAIL until memory query implementation is complete.
# This is the "RED" phase of TDD - tests specify expected behavior.

# Don't exit on first failure - we want to see all test results
set +e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_FILE="$SCRIPT_DIR/SKILL.md"

# Color output helpers
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

FAIL_COUNT=0

fail() {
    echo -e "${RED}FAIL${NC}: $1"
    FAIL_COUNT=$((FAIL_COUNT + 1))
}

pass() {
    echo -e "${GREEN}PASS${NC}: $1"
}

warn() {
    echo -e "${YELLOW}WARN${NC}: $1"
}

echo "=== TASK-27: Memory Query Integration Tests ==="
echo "Expected: These tests should FAIL until memory query implementation is complete"
echo ""

# ============================================================================
# AC 1: GATHER phase includes `projctl memory query "prior architecture decisions for <project-domain>"`
# ============================================================================
echo "Test 1: GATHER phase queries prior architecture decisions"
if grep -q 'projctl memory query.*prior architecture decisions' "$SKILL_FILE"; then
    pass "Prior architecture decisions query found"
else
    fail "Missing: projctl memory query for prior architecture decisions in GATHER phase"
fi

# ============================================================================
# AC 2: GATHER phase includes `projctl memory query "technology patterns for <feature-area>"`
# ============================================================================
echo "Test 2: GATHER phase queries technology patterns"
if grep -q 'projctl memory query.*technology patterns' "$SKILL_FILE"; then
    pass "Technology patterns query found"
else
    fail "Missing: projctl memory query for technology patterns in GATHER phase"
fi

# ============================================================================
# AC 3: GATHER phase includes `projctl memory query "known failures in architecture validation"`
# ============================================================================
echo "Test 3: GATHER phase queries known failures in architecture validation"
if grep -q 'projctl memory query.*known failures in architecture validation' "$SKILL_FILE"; then
    pass "Known failures query found"
else
    fail "Missing: projctl memory query for known failures in architecture validation"
fi

# ============================================================================
# AC 4: Memory queries run BEFORE interview questions (verify in GATHER section, before INTERVIEW)
# ============================================================================
echo "Test 4: Memory queries appear in GATHER phase (before INTERVIEW phase)"
# Extract line numbers for GATHER and INTERVIEW sections
gather_line=$(grep -n '^### GATHER' "$SKILL_FILE" | head -1 | cut -d: -f1 || echo "0")
interview_line=$(grep -n '^### INTERVIEW' "$SKILL_FILE" | head -1 | cut -d: -f1 || echo "0")

if [ "$gather_line" = "0" ]; then
    fail "GATHER section not found in SKILL.md"
fi

if [ "$interview_line" = "0" ]; then
    fail "INTERVIEW section not found in SKILL.md"
fi

# Check that at least one memory query appears between GATHER and INTERVIEW
memory_query_count=$(sed -n "${gather_line},${interview_line}p" "$SKILL_FILE" | grep -c 'projctl memory query' || echo 0)

if [ "$memory_query_count" -ge 1 ]; then
    pass "Memory queries appear in GATHER phase (before INTERVIEW) - found $memory_query_count queries"
else
    fail "No memory queries found in GATHER phase (between GATHER and INTERVIEW sections)"
fi

# ============================================================================
# AC 5: `grep -c "memory query" ~/.claude/skills/arch-interview-producer/SKILL.md` returns at least 3
# ============================================================================
echo "Test 5: At least 3 memory query references in SKILL.md"
count=$(grep -c "memory query" "$SKILL_FILE" || echo 0)
if [ "$count" -ge 3 ]; then
    pass "Found $count memory query references (>= 3 required)"
else
    fail "Only found $count memory query references, need at least 3"
fi

echo ""
echo "=== Test Summary ==="
echo "Failed tests: $FAIL_COUNT"
echo ""
echo -e "${RED}=== Expected result: All tests FAILED (TDD RED phase) ===${NC}"
echo "These tests specify the expected behavior for memory query integration."
echo "Implementation should make these tests pass (TDD GREEN phase)."

# Exit with failure if any tests failed
exit $FAIL_COUNT
