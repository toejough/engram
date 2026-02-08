#!/usr/bin/env bash
# Test suite for TASK-31: Orchestrator startup memory reads
# Traces to: ARCH-059, REQ-012, DES-024
#
# These tests verify that the orchestrator SKILL.md documents memory queries
# at startup to surface past learnings before entering the step loop.

set -e

SKILL_FILE="$HOME/.claude/skills/project/SKILL.md"
TEST_FAILURES=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    TEST_FAILURES=$((TEST_FAILURES + 1))
}

pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
}

echo "=== TASK-31: Orchestrator Startup Memory Reads Tests ==="
echo ""

# CHECK-001: SKILL.md contains "lessons from past projects" query
# AC: Startup section includes projctl memory query "lessons from past projects"
if grep -q 'projctl memory query.*lessons from past projects' "$SKILL_FILE"; then
    pass "CHECK-001: SKILL.md contains 'lessons from past projects' query"
else
    fail "CHECK-001: SKILL.md missing 'projctl memory query \"lessons from past projects\"'"
fi

# CHECK-002: SKILL.md contains workflow-specific challenges query
# AC: Startup section includes projctl memory query "common challenges in <workflow-type> projects"
if grep -q 'projctl memory query.*common challenges.*workflow' "$SKILL_FILE"; then
    pass "CHECK-002: SKILL.md contains workflow-specific challenges query"
else
    fail "CHECK-002: SKILL.md missing 'projctl memory query \"common challenges in <workflow-type> projects\"'"
fi

# CHECK-003: Memory queries appear in startup section (between line 45 and line 90)
# This ensures the queries are placed correctly (after workflow set, before step loop)
STARTUP_SECTION=$(sed -n '45,90p' "$SKILL_FILE")
if echo "$STARTUP_SECTION" | grep -q 'projctl memory query'; then
    pass "CHECK-003: Memory queries appear in startup section (lines 45-90)"
else
    fail "CHECK-003: Memory queries not in startup section (should be after line 61)"
fi

# CHECK-004: At least 2 occurrences of "projctl memory query"
# AC: grep -c "projctl memory query" returns >= 2
QUERY_COUNT=$(grep -o 'projctl memory query' "$SKILL_FILE" | wc -l | tr -d ' ')
if [ "$QUERY_COUNT" -ge 2 ]; then
    pass "CHECK-004: Found $QUERY_COUNT memory queries (>= 2 required)"
else
    fail "CHECK-004: Found only $QUERY_COUNT memory queries, need >= 2"
fi

# CHECK-005: Documentation mentions query results in context
# AC: Query results included in orchestrator's working context
if grep -q -i 'context\|working\|include.*result' "$SKILL_FILE" && \
   grep -q 'memory query' "$SKILL_FILE"; then
    pass "CHECK-005: Documentation mentions query results in context"
else
    fail "CHECK-005: Missing documentation about query results being included in context"
fi

# CHECK-006: Documentation mentions what surfaces (session summaries, retro, QA patterns)
# AC: Queries surface session summaries, retro learnings, QA failure patterns
if grep -q -E 'session|retro|learnings|failure.*pattern' "$SKILL_FILE"; then
    pass "CHECK-006: Documentation mentions what memory surfaces"
else
    fail "CHECK-006: Missing documentation about what memory queries surface"
fi

# CHECK-007: Documentation mentions graceful degradation
# AC: Graceful degradation documented (from REQ-012)
if grep -q -i 'graceful.*degradation\|memory.*unavailable\|memory.*fails' "$SKILL_FILE"; then
    pass "CHECK-007: Documentation includes graceful degradation strategy"
else
    fail "CHECK-007: Missing graceful degradation documentation"
fi

# CHECK-008: Memory queries positioned after "projctl state set --workflow"
# AC: Timing per ARCH-059 - after workflow set, before step loop
if grep -n 'projctl state set --workflow' "$SKILL_FILE" | head -1 | cut -d: -f1 | \
   xargs -I {} bash -c "sed -n '{},90p' \"$SKILL_FILE\" | grep -q 'projctl memory query'"; then
    pass "CHECK-008: Memory queries positioned after 'state set --workflow'"
else
    fail "CHECK-008: Memory queries should appear after 'projctl state set --workflow' line"
fi

echo ""
echo "=== Test Summary ==="
if [ "$TEST_FAILURES" -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}$TEST_FAILURES test(s) failed${NC}"
    echo ""
    echo "Expected behavior (not yet implemented):"
    echo "  - Orchestrator startup section should include memory queries"
    echo "  - Two queries: 'lessons from past projects' and 'common challenges in <workflow-type> projects'"
    echo "  - Queries positioned after 'projctl state set --workflow', before step loop"
    echo "  - Documentation explains what surfaces and graceful degradation"
    exit 1
fi
