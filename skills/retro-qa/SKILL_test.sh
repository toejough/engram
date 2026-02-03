#!/bin/bash
# retro-qa SKILL.md validation tests for TASK-24
# Run: bash skills/retro-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/retro-qa/SKILL.md"

echo "=== retro-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-24 Requirement: Frontmatter has name field
if grep -q '^name: retro-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: retro-qa"
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

# TASK-24 Requirement: Frontmatter has phase: retro
if grep -q '^phase: retro' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: retro"
else
    echo "FAIL: Frontmatter missing phase: retro"
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

# TASK-24 Requirement: Reviews retro completeness
if grep -qi 'complete\|thorough\|comprehensive' "$SKILL_FILE"; then
    echo "PASS: Reviews retro completeness"
else
    echo "FAIL: Missing completeness review documentation"
    exit 1
fi

# TASK-24 Requirement: Validates successes and challenges are documented
if grep -qi 'success\|went well' "$SKILL_FILE" && grep -qi 'challenge\|improve' "$SKILL_FILE"; then
    echo "PASS: Validates successes and challenges"
else
    echo "FAIL: Missing validation of successes and challenges"
    exit 1
fi

# TASK-24 Requirement: Validates actionable recommendations
if grep -qi 'action\|recommendation' "$SKILL_FILE"; then
    echo "PASS: Validates actionable recommendations"
else
    echo "FAIL: Missing validation of actionable recommendations"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
