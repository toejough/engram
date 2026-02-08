#!/bin/bash
# Test: TASK-35 Alignment Producer Memory Reads
# Traces to: ARCH-055, REQ-008
#
# Verifies that alignment-producer SKILL.md includes memory queries in GATHER phase
# for alignment patterns and known failures.

set -e

SKILL_FILE="$HOME/.claude/skills/alignment-producer/SKILL.md"
FAILED=0

echo "=== TASK-35: Alignment Producer Memory Reads Tests ==="
echo ""

# Test 1: Common alignment errors query exists
echo "Test 1: GATHER phase includes 'projctl memory query \"common alignment errors\"'"
if grep -q 'projctl memory query "common alignment errors"' "$SKILL_FILE"; then
    echo "  ✓ PASS: Common alignment errors query found"
else
    echo "  ✗ FAIL: Common alignment errors query not found"
    FAILED=1
fi
echo ""

# Test 2: Domain boundary violations query exists
echo "Test 2: GATHER phase includes 'projctl memory query \"domain boundary violations\"'"
if grep -q 'projctl memory query "domain boundary violations"' "$SKILL_FILE"; then
    echo "  ✓ PASS: Domain boundary violations query found"
else
    echo "  ✗ FAIL: Domain boundary violations query not found"
    FAILED=1
fi
echo ""

# Test 3: Known failures in alignment validation query exists
echo "Test 3: GATHER phase includes 'projctl memory query \"known failures in alignment validation\"'"
if grep -q 'projctl memory query "known failures in alignment validation"' "$SKILL_FILE"; then
    echo "  ✓ PASS: Known failures query found"
else
    echo "  ✗ FAIL: Known failures query not found"
    FAILED=1
fi
echo ""

# Test 4: Memory query count meets threshold (at least 3)
echo "Test 4: 'projctl memory query' appears at least 3 times"
if grep -q "projctl memory query" "$SKILL_FILE" 2>/dev/null; then
    QUERY_COUNT=$(grep -c "projctl memory query" "$SKILL_FILE")
else
    QUERY_COUNT=0
fi
if [ "$QUERY_COUNT" -ge 3 ]; then
    echo "  ✓ PASS: Found $QUERY_COUNT memory queries (>= 3 required)"
else
    echo "  ✗ FAIL: Found $QUERY_COUNT memory queries (< 3 required)"
    FAILED=1
fi
echo ""

# Test 5: Graceful degradation documented
echo "Test 5: Graceful degradation for memory failures documented"
if grep -qi "graceful" "$SKILL_FILE" && grep -qi "degradation\|fallback\|fail" "$SKILL_FILE"; then
    echo "  ✓ PASS: Graceful degradation documentation found"
else
    echo "  ✗ FAIL: Graceful degradation not documented"
    FAILED=1
fi
echo ""

# Test 6: Memory queries appear in GATHER phase section
echo "Test 6: Memory queries appear in GATHER section (lines 20-35)"
GATHER_SECTION=$(sed -n '/^### 1\. GATHER Phase/,/^### 2\. SYNTHESIZE Phase/p' "$SKILL_FILE")
if echo "$GATHER_SECTION" | grep -q "projctl memory query"; then
    echo "  ✓ PASS: Memory queries found in GATHER section"
else
    echo "  ✗ FAIL: Memory queries not found in GATHER section"
    FAILED=1
fi
echo ""

# Summary
echo "=== Test Summary ==="
if [ $FAILED -eq 0 ]; then
    echo "All tests PASSED (unexpected - this is RED phase, tests should FAIL)"
    exit 1
else
    echo "Tests FAILED as expected (RED phase)"
    exit 0
fi
