#!/bin/bash
# breakdown-qa SKILL.md validation tests for TASK-16
# Run: bash skills/breakdown-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/breakdown-qa/SKILL.md"

echo "=== breakdown-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-16 Requirement: Frontmatter has name field
if grep -q '^name: breakdown-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: breakdown-qa"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-16 Requirement: Frontmatter has role: qa
if grep -q '^role: qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: qa"
else
    echo "FAIL: Frontmatter missing role: qa"
    exit 1
fi

# TASK-16 Requirement: Frontmatter has phase: breakdown
if grep -q '^phase: breakdown' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: breakdown"
else
    echo "FAIL: Frontmatter missing phase: breakdown"
    exit 1
fi

# TASK-16 Requirement: References QA-TEMPLATE pattern (REVIEW/RETURN)
if grep -qi 'REVIEW' "$SKILL_FILE" && grep -qi 'RETURN' "$SKILL_FILE"; then
    echo "PASS: References QA-TEMPLATE pattern (REVIEW/RETURN)"
else
    echo "FAIL: Missing QA-TEMPLATE pattern (must include REVIEW and RETURN)"
    exit 1
fi

# TASK-16 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-16 Requirement: Documents approved yield
if grep -q 'approved' "$SKILL_FILE"; then
    echo "PASS: Documents approved yield"
else
    echo "FAIL: Missing approved yield documentation"
    exit 1
fi

# TASK-16 Requirement: Documents improvement-request yield
if grep -q 'improvement-request' "$SKILL_FILE"; then
    echo "PASS: Documents improvement-request yield"
else
    echo "FAIL: Missing improvement-request yield documentation"
    exit 1
fi

# TASK-16 Requirement: Documents escalate-phase yield
if grep -q 'escalate-phase' "$SKILL_FILE"; then
    echo "PASS: Documents escalate-phase yield"
else
    echo "FAIL: Missing escalate-phase yield documentation"
    exit 1
fi

# TASK-16 Requirement: Validates task decomposition completeness
if grep -qi 'decomposition' "$SKILL_FILE" && grep -qi 'complete' "$SKILL_FILE"; then
    echo "PASS: Validates task decomposition completeness"
else
    echo "FAIL: Missing task decomposition completeness validation"
    exit 1
fi

# TASK-16 Requirement: Checks dependency graph for cycles
if grep -qi 'cycle' "$SKILL_FILE" && grep -qi 'dependency' "$SKILL_FILE"; then
    echo "PASS: Checks dependency graph for cycles"
else
    echo "FAIL: Missing dependency graph cycle detection"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
