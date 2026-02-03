#!/bin/bash
# tdd-green-qa SKILL.md validation tests for TASK-22
# Run: bash skills/tdd-green-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/tdd-green-qa/SKILL.md"

echo "=== tdd-green-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-22 Requirement: Frontmatter has name field
if grep -q '^name: tdd-green-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: tdd-green-qa"
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

# TASK-22 Requirement: Frontmatter has phase: tdd-green
if grep -q '^phase: tdd-green' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: tdd-green"
else
    echo "FAIL: Frontmatter missing phase: tdd-green"
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

# TASK-22 Requirement: Verifies all tests pass
if grep -qi 'all tests pass' "$SKILL_FILE" || grep -qi 'tests.*pass' "$SKILL_FILE"; then
    echo "PASS: Documents verification that all tests pass"
else
    echo "FAIL: Missing verification that all tests pass"
    exit 1
fi

# TASK-22 Requirement: Verifies no regressions in existing tests
if grep -qi 'regress' "$SKILL_FILE" || grep -qi 'existing tests' "$SKILL_FILE"; then
    echo "PASS: Documents verification of no regressions in existing tests"
else
    echo "FAIL: Missing verification of no regressions in existing tests"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
