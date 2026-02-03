#!/bin/bash
# context-qa SKILL.md validation tests for TASK-25c
# Run: bash skills/context-qa/SKILL_test.sh

set -e
SKILL_FILE="skills/context-qa/SKILL.md"

echo "=== context-qa SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-25c Requirement: Frontmatter has name field
if grep -q '^name: context-qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: context-qa"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-25c Requirement: Frontmatter has role: qa
if grep -q '^role: qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: qa"
else
    echo "FAIL: Frontmatter missing role: qa"
    exit 1
fi

# TASK-25c Requirement: References QA-TEMPLATE pattern (REVIEW/RETURN)
if grep -qi 'REVIEW' "$SKILL_FILE" && grep -qi 'RETURN' "$SKILL_FILE"; then
    echo "PASS: References QA-TEMPLATE pattern (REVIEW/RETURN)"
else
    echo "FAIL: Missing QA-TEMPLATE pattern (must include REVIEW, RETURN)"
    exit 1
fi

# TASK-25c Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-25c Requirement: Validates all queries were answered
if grep -qi 'queries.*answered\|unanswered.*quer\|missing.*result\|query.*result' "$SKILL_FILE"; then
    echo "PASS: Documents query validation"
else
    echo "FAIL: Missing query validation documentation"
    exit 1
fi

# TASK-25c Requirement: Checks results are relevant to the request
if grep -qi 'relevan' "$SKILL_FILE"; then
    echo "PASS: Documents relevance checking"
else
    echo "FAIL: Missing relevance checking documentation"
    exit 1
fi

# TASK-25c Requirement: Flags contradictions between sources
if grep -qi 'contradict' "$SKILL_FILE"; then
    echo "PASS: Documents contradiction detection"
else
    echo "FAIL: Missing contradiction detection documentation"
    exit 1
fi

# TASK-25c Requirement: Identifies stale or outdated information
if grep -qi 'stale\|outdated\|freshness' "$SKILL_FILE"; then
    echo "PASS: Documents staleness detection"
else
    echo "FAIL: Missing staleness detection documentation"
    exit 1
fi

# TASK-25c Requirement: Documents approved yield
if grep -q 'approved' "$SKILL_FILE"; then
    echo "PASS: Documents approved yield"
else
    echo "FAIL: Missing approved yield documentation"
    exit 1
fi

# TASK-25c Requirement: Documents improvement-request yield
if grep -q 'improvement-request' "$SKILL_FILE"; then
    echo "PASS: Documents improvement-request yield"
else
    echo "FAIL: Missing improvement-request yield documentation"
    exit 1
fi

# TASK-25c Requirement: Outputs yield protocol TOML
if grep -q '\[yield\]' "$SKILL_FILE" && grep -q 'type = ' "$SKILL_FILE"; then
    echo "PASS: Documents yield protocol TOML format"
else
    echo "FAIL: Missing yield protocol TOML documentation"
    exit 1
fi

# TASK-25c Requirement: Documents escalation reasons (error/gap/conflict)
if grep -q 'error' "$SKILL_FILE" && grep -q 'gap' "$SKILL_FILE" && grep -q 'conflict' "$SKILL_FILE"; then
    echo "PASS: Documents escalation reasons (error/gap/conflict)"
else
    echo "FAIL: Missing escalation reasons (must include error, gap, conflict)"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
