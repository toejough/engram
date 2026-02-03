#!/bin/bash
# alignment-qa SKILL.md validation tests for TASK-24
# Run: bash skills/alignment-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/alignment-qa/SKILL.md"

echo "=== alignment-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-24 Requirement: Frontmatter has name field
if grep -q '^name: alignment-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: alignment-qa"
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

# TASK-24 Requirement: Frontmatter has phase: alignment
if grep -q '^phase: alignment' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: alignment"
else
    echo "FAIL: Frontmatter missing phase: alignment"
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

# TASK-24 Requirement: Reviews traceability validation results
if grep -qi 'traceability' "$SKILL_FILE" && grep -qi 'review' "$SKILL_FILE"; then
    echo "PASS: Documents traceability review"
else
    echo "FAIL: Missing traceability review documentation"
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

# TASK-24 Requirement: Documents escalation reasons (error/gap/conflict)
if grep -q 'error' "$SKILL_FILE" && grep -q 'gap' "$SKILL_FILE" && grep -q 'conflict' "$SKILL_FILE"; then
    echo "PASS: Documents escalation reasons (error/gap/conflict)"
else
    echo "FAIL: Missing escalation reasons (must include error, gap, conflict)"
    exit 1
fi

# TASK-24 Requirement: Documents checklist for validation
if grep -qi 'checklist' "$SKILL_FILE"; then
    echo "PASS: Documents validation checklist"
else
    echo "FAIL: Missing validation checklist documentation"
    exit 1
fi

# TASK-24 Requirement: Outputs yield protocol TOML
if grep -q '\[yield\]' "$SKILL_FILE" || grep -q 'yield.toml' "$SKILL_FILE" || grep -q 'TOML' "$SKILL_FILE"; then
    echo "PASS: Documents yield protocol TOML output"
else
    echo "FAIL: Missing yield protocol TOML documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
