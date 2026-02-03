#!/bin/bash
# arch-qa SKILL.md validation tests for TASK-15
# Run: bash skills/arch-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/arch-qa/SKILL.md"

echo "=== arch-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-15 Requirement: Frontmatter has name: arch-qa
if grep -q '^name: arch-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: arch-qa"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-15 Requirement: Frontmatter has role: qa
if grep -q '^role: qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: qa"
else
    echo "FAIL: Frontmatter missing role: qa"
    exit 1
fi

# TASK-15 Requirement: Frontmatter has phase: arch
if grep -q '^phase: arch' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: arch"
else
    echo "FAIL: Frontmatter missing phase: arch"
    exit 1
fi

# TASK-15 Requirement: References QA-TEMPLATE pattern (REVIEW/RETURN)
if grep -qi 'REVIEW' "$SKILL_FILE" && grep -qi 'RETURN' "$SKILL_FILE"; then
    echo "PASS: References QA-TEMPLATE pattern (REVIEW/RETURN)"
else
    echo "FAIL: Missing QA-TEMPLATE pattern (must include REVIEW, RETURN)"
    exit 1
fi

# TASK-15 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-15 Requirement: Documents approved yield
if grep -q 'approved' "$SKILL_FILE"; then
    echo "PASS: Documents approved yield"
else
    echo "FAIL: Missing approved yield documentation"
    exit 1
fi

# TASK-15 Requirement: Documents improvement-request yield
if grep -q 'improvement-request' "$SKILL_FILE"; then
    echo "PASS: Documents improvement-request yield"
else
    echo "FAIL: Missing improvement-request yield documentation"
    exit 1
fi

# TASK-15 Requirement: Documents escalate-phase yield
if grep -q 'escalate-phase' "$SKILL_FILE"; then
    echo "PASS: Documents escalate-phase yield"
else
    echo "FAIL: Missing escalate-phase yield documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
