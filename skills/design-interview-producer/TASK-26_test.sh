#!/usr/bin/env bash
# Tests for TASK-26: Design interview producer memory reads
# Validates memory query integration in design-interview-producer GATHER phase
#
# AC:
# - GATHER phase includes `projctl memory query "prior design decisions for <project-domain>"`
# - GATHER phase includes `projctl memory query "UX patterns for <feature-area>"`
# - GATHER phase includes `projctl memory query "known failures in design validation"`
# - Memory queries run BEFORE interview questions
# - `grep -c "projctl memory query" ~/.claude/skills/design-interview-producer/SKILL.md` returns >= 3
#
# Traces to: ARCH-055, REQ-008

set -e

SKILL_FILE="${HOME}/.claude/skills/design-interview-producer/SKILL.md"
PASS_COUNT=0
FAIL_COUNT=0

echo "======================================"
echo "TASK-26: Design interview producer memory reads"
echo "======================================"
echo ""

# Verify SKILL.md exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "ERROR: $SKILL_FILE does not exist"
    exit 1
fi

# TEST 1: At least 3 memory query commands exist
echo "TEST 1: Memory query count >= 3"
count=$(grep -o "projctl memory query" "$SKILL_FILE" 2>/dev/null | wc -l | tr -d ' ')
if [[ "$count" -ge 3 ]]; then
    echo "✓ PASS: found $count queries"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "✗ FAIL: found $count queries, expected >= 3"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo ""

# TEST 2: Query for prior design decisions exists
echo "TEST 2: Prior design decisions query exists"
if grep -q 'projctl memory query.*prior design decisions' "$SKILL_FILE"; then
    echo "✓ PASS"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "✗ FAIL: pattern not found"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo ""

# TEST 3: Query for UX patterns exists
echo "TEST 3: UX patterns query exists"
if grep -q 'projctl memory query.*UX patterns' "$SKILL_FILE"; then
    echo "✓ PASS"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "✗ FAIL: pattern not found"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo ""

# TEST 4: Query for known failures in design validation exists
echo "TEST 4: Known failures in design validation query exists"
if grep -q 'projctl memory query.*known failures.*design validation' "$SKILL_FILE"; then
    echo "✓ PASS"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo "✗ FAIL: pattern not found"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi
echo ""

# TEST 5: Memory queries are in GATHER phase section
echo "TEST 5: Memory queries in GATHER phase section"
gather_section=$(sed -n '/^### 1\. GATHER Phase/,/^### [0-9]/p' "$SKILL_FILE" | sed '$ d')
if [[ -z "$gather_section" ]]; then
    echo "✗ FAIL: GATHER Phase section not found"
    FAIL_COUNT=$((FAIL_COUNT + 1))
else
    count=$(echo "$gather_section" | grep -o "projctl memory query" 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$count" -ge 3 ]]; then
        echo "✓ PASS: found $count queries in GATHER section"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "✗ FAIL: found $count queries in GATHER section, expected >= 3"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
fi
echo ""

# TEST 6: Memory queries documented BEFORE interview questions
echo "TEST 6: Memory queries before interview questions"
gather_section=$(sed -n '/^### 1\. GATHER Phase/,/^### [0-9]/p' "$SKILL_FILE" | sed '$ d')
if [[ -z "$gather_section" ]]; then
    echo "✗ FAIL: GATHER Phase section not found"
    FAIL_COUNT=$((FAIL_COUNT + 1))
else
    memory_line=$(echo "$gather_section" | grep -n "projctl memory query" | head -1 | cut -d: -f1)
    interview_line=$(echo "$gather_section" | grep -n -E "Interview user|AskUserQuestion" | head -1 | cut -d: -f1)

    if [[ -z "$memory_line" ]]; then
        echo "✗ FAIL: no memory query found in GATHER phase"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    elif [[ -z "$interview_line" ]]; then
        echo "✗ FAIL: no interview reference found in GATHER phase"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    elif [[ "$memory_line" -lt "$interview_line" ]]; then
        echo "✓ PASS: memory at line $memory_line, interview at line $interview_line"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "✗ FAIL: memory at line $memory_line comes AFTER interview at line $interview_line"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
fi
echo ""

# Print summary
echo "======================================"
echo "Summary: $PASS_COUNT passed, $FAIL_COUNT failed"
echo "======================================"
echo ""

# Exit with failure if any tests failed (RED phase - tests MUST fail)
if [[ "$FAIL_COUNT" -gt 0 ]]; then
    echo "RED PHASE: Tests failing as expected (feature not implemented)"
    exit 1
fi

# If all tests pass, that's unexpected in RED phase
echo "UNEXPECTED: All tests passed in RED phase!"
echo "Feature may already be implemented or tests are incorrect."
exit 1
