#!/bin/bash
# tdd-qa SKILL.md validation tests for TASK-23
# Run: bash skills/tdd-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/tdd-qa/SKILL.md"

echo "=== tdd-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-23 Requirement: Frontmatter has name field
if grep -q '^name: tdd-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: tdd-qa"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-23 Requirement: Frontmatter has role: qa
if grep -q '^role: qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: qa"
else
    echo "FAIL: Frontmatter missing role: qa"
    exit 1
fi

# TASK-23 Requirement: Frontmatter has phase: tdd
if grep -q '^phase: tdd' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: tdd"
else
    echo "FAIL: Frontmatter missing phase: tdd"
    exit 1
fi

# TASK-23 Requirement: References QA-TEMPLATE pattern (REVIEW/RETURN)
if grep -qi 'REVIEW' "$SKILL_FILE" && grep -qi 'RETURN' "$SKILL_FILE"; then
    echo "PASS: References QA-TEMPLATE pattern (REVIEW/RETURN)"
else
    echo "FAIL: Missing QA-TEMPLATE pattern (must include REVIEW, RETURN)"
    exit 1
fi

# TASK-23 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-23 Requirement: Documents validating overall AC compliance
if grep -qi 'acceptance criteria' "$SKILL_FILE"; then
    echo "PASS: Documents validating overall AC compliance"
else
    echo "FAIL: Missing documentation for acceptance criteria validation"
    exit 1
fi

# TASK-23 Requirement: Documents checking TDD discipline was followed
if grep -qi 'TDD discipline' "$SKILL_FILE" || (grep -qi 'tests first' "$SKILL_FILE" && grep -qi 'minimal' "$SKILL_FILE" && grep -qi 'refactor' "$SKILL_FILE"); then
    echo "PASS: Documents checking TDD discipline was followed"
else
    echo "FAIL: Missing documentation for TDD discipline validation"
    exit 1
fi

# TASK-23 Requirement: Documents approved yield
if grep -q 'approved' "$SKILL_FILE"; then
    echo "PASS: Documents approved yield"
else
    echo "FAIL: Missing approved yield documentation"
    exit 1
fi

# TASK-23 Requirement: Documents improvement-request yield
if grep -q 'improvement-request' "$SKILL_FILE"; then
    echo "PASS: Documents improvement-request yield"
else
    echo "FAIL: Missing improvement-request yield documentation"
    exit 1
fi

# TASK-23 Requirement: Documents escalate-phase yield
if grep -q 'escalate-phase' "$SKILL_FILE"; then
    echo "PASS: Documents escalate-phase yield"
else
    echo "FAIL: Missing escalate-phase yield documentation"
    exit 1
fi

# TASK-23 Requirement: Documents RED/GREEN/REFACTOR cycle validation
if grep -qi 'RED' "$SKILL_FILE" && grep -qi 'GREEN' "$SKILL_FILE" && grep -qi 'REFACTOR' "$SKILL_FILE"; then
    echo "PASS: Documents RED/GREEN/REFACTOR cycle validation"
else
    echo "FAIL: Missing RED/GREEN/REFACTOR cycle documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
