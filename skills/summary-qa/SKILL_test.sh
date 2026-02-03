#!/bin/bash
# summary-qa SKILL.md validation tests for TASK-24
# Run: bash skills/summary-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/summary-qa/SKILL.md"

echo "=== summary-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-24 Requirement: Frontmatter has name field
if grep -q '^name: summary-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: summary-qa"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-24 Requirement: Frontmatter has role: qa
if grep -q '^role: qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: qa"
else
    echo "FAIL: Frontmatter missing role: qa"
    exit 1
fi

# TASK-24 Requirement: Frontmatter has phase: summary
if grep -q '^phase: summary' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: summary"
else
    echo "FAIL: Frontmatter missing phase: summary"
    exit 1
fi

# TASK-24 Requirement: References QA-TEMPLATE pattern (REVIEW/RETURN)
if grep -qi 'REVIEW' "$SKILL_FILE" && grep -qi 'RETURN' "$SKILL_FILE"; then
    echo "PASS: References QA-TEMPLATE pattern (REVIEW/RETURN)"
else
    echo "FAIL: Missing QA-TEMPLATE pattern (must include REVIEW, RETURN)"
    exit 1
fi

# TASK-24 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-24 Requirement: Documents approved yield
if grep -q 'approved' "$SKILL_FILE"; then
    echo "PASS: Documents approved yield"
else
    echo "FAIL: Missing approved yield documentation"
    exit 1
fi

# TASK-24 Requirement: Documents improvement-request yield
if grep -q 'improvement-request' "$SKILL_FILE"; then
    echo "PASS: Documents improvement-request yield"
else
    echo "FAIL: Missing improvement-request yield documentation"
    exit 1
fi

# TASK-24 Requirement: Documents escalate-phase yield
if grep -q 'escalate-phase' "$SKILL_FILE"; then
    echo "PASS: Documents escalate-phase yield"
else
    echo "FAIL: Missing escalate-phase yield documentation"
    exit 1
fi

# TASK-24 Requirement: Reviews summary accuracy
if grep -qi 'accuracy' "$SKILL_FILE"; then
    echo "PASS: Reviews summary accuracy"
else
    echo "FAIL: Missing accuracy validation"
    exit 1
fi

# TASK-24 Requirement: Reviews summary completeness
if grep -qi 'completeness' "$SKILL_FILE"; then
    echo "PASS: Reviews summary completeness"
else
    echo "FAIL: Missing completeness validation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
