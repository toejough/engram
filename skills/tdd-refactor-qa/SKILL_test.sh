#!/bin/bash
# tdd-refactor-qa SKILL.md validation tests for TASK-22
# Run: bash skills/tdd-refactor-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/tdd-refactor-qa/SKILL.md"

echo "=== tdd-refactor-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-22 Requirement: Frontmatter has name field
if grep -q '^name: tdd-refactor-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: tdd-refactor-qa"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-22 Requirement: Frontmatter has role: qa
if grep -q '^role: qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: qa"
else
    echo "FAIL: Frontmatter missing role: qa"
    exit 1
fi

# TASK-22 Requirement: Frontmatter has phase: tdd-refactor
if grep -q '^phase: tdd-refactor' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: tdd-refactor"
else
    echo "FAIL: Frontmatter missing phase: tdd-refactor"
    exit 1
fi

# TASK-22 Requirement: References QA-TEMPLATE pattern (REVIEW/RETURN)
if grep -qi 'REVIEW' "$SKILL_FILE" && grep -qi 'RETURN' "$SKILL_FILE"; then
    echo "PASS: References QA-TEMPLATE pattern (REVIEW/RETURN)"
else
    echo "FAIL: Missing QA-TEMPLATE pattern (must include REVIEW, RETURN)"
    exit 1
fi

# TASK-22 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-22 Requirement: Documents approved yield
if grep -q 'approved' "$SKILL_FILE"; then
    echo "PASS: Documents approved yield"
else
    echo "FAIL: Missing approved yield documentation"
    exit 1
fi

# TASK-22 Requirement: Documents improvement-request yield
if grep -q 'improvement-request' "$SKILL_FILE"; then
    echo "PASS: Documents improvement-request yield"
else
    echo "FAIL: Missing improvement-request yield documentation"
    exit 1
fi

# TASK-22 Requirement: Documents escalate-phase yield
if grep -q 'escalate-phase' "$SKILL_FILE"; then
    echo "PASS: Documents escalate-phase yield"
else
    echo "FAIL: Missing escalate-phase yield documentation"
    exit 1
fi

# TASK-22 Requirement: Verifies tests still pass after refactor
if grep -qi 'tests.*pass' "$SKILL_FILE" || grep -qi 'still.*green' "$SKILL_FILE"; then
    echo "PASS: Documents verification that tests still pass after refactor"
else
    echo "FAIL: Missing verification that tests still pass after refactor"
    exit 1
fi

# TASK-22 Requirement: Verifies code quality improved
if grep -qi 'quality' "$SKILL_FILE" || grep -qi 'lint' "$SKILL_FILE"; then
    echo "PASS: Documents verification of code quality improvement"
else
    echo "FAIL: Missing verification of code quality improvement"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
