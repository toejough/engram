#!/bin/bash
# consistency-checker SKILL.md validation tests for TASK-25b
# Run: bash skills/consistency-checker/SKILL_test.sh

set -e
SKILL_FILE="skills/consistency-checker/SKILL.md"

echo "=== consistency-checker SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-25b Requirement: Frontmatter has name field
if grep -q '^name: consistency-checker' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: consistency-checker"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-25b Requirement: Frontmatter has role: qa
if grep -q '^role: qa' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: qa"
else
    echo "FAIL: Frontmatter missing role: qa"
    exit 1
fi

# Requirement: Has REVIEW/RETURN workflow pattern
if grep -qi 'REVIEW' "$SKILL_FILE" && grep -qi 'RETURN' "$SKILL_FILE"; then
    echo "PASS: Has REVIEW/RETURN workflow pattern"
else
    echo "FAIL: Missing REVIEW/RETURN workflow pattern"
    exit 1
fi

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
fi

# TASK-25b Requirement: Reviews outputs across all parallel results
if grep -qi 'parallel' "$SKILL_FILE" && grep -qi 'results\|outputs' "$SKILL_FILE"; then
    echo "PASS: Documents reviewing parallel results"
else
    echo "FAIL: Missing documentation for reviewing parallel results"
    exit 1
fi

# TASK-25b Requirement: Applies domain-specific consistency rules
if grep -qi 'consistency' "$SKILL_FILE" && grep -qi 'rules' "$SKILL_FILE"; then
    echo "PASS: Documents domain-specific consistency rules"
else
    echo "FAIL: Missing documentation for domain-specific consistency rules"
    exit 1
fi

# TASK-25b Requirement: Yields approved if consistent
if grep -q 'approved' "$SKILL_FILE"; then
    echo "PASS: Documents approved yield"
else
    echo "FAIL: Missing approved yield documentation"
    exit 1
fi

# TASK-25b Requirement: Yields improvement-request (batch) if inconsistent
if grep -q 'improvement-request' "$SKILL_FILE"; then
    echo "PASS: Documents improvement-request yield"
else
    echo "FAIL: Missing improvement-request yield documentation"
    exit 1
fi

# TASK-25b Requirement: Batch improvement request with multiple issues
if grep -qi 'batch' "$SKILL_FILE" || grep -q 'items\s*=' "$SKILL_FILE"; then
    echo "PASS: Documents batch improvement requests"
else
    echo "FAIL: Missing batch improvement request documentation"
    exit 1
fi

# TASK-25b Requirement: Documents specific inconsistencies
if grep -qi 'inconsistenc' "$SKILL_FILE"; then
    echo "PASS: Documents inconsistencies"
else
    echo "FAIL: Missing inconsistency documentation"
    exit 1
fi

# TASK-25b Requirement: Documents resolutions
if grep -qi 'resolution' "$SKILL_FILE"; then
    echo "PASS: Documents resolutions"
else
    echo "FAIL: Missing resolution documentation"
    exit 1
fi

# TASK-25b Requirement: Outputs yield protocol TOML (has TOML example)
if grep -q '\[yield\]' "$SKILL_FILE" && grep -q 'type =' "$SKILL_FILE"; then
    echo "PASS: Contains yield protocol TOML examples"
else
    echo "FAIL: Missing yield protocol TOML examples"
    exit 1
fi

# TASK-25b Requirement: Documents escalate-phase yield
if grep -q 'escalate-phase' "$SKILL_FILE"; then
    echo "PASS: Documents escalate-phase yield"
else
    echo "FAIL: Missing escalate-phase yield documentation"
    exit 1
fi

# TASK-25b Requirement: Documents escalation reasons (error/gap/conflict)
if grep -q 'error' "$SKILL_FILE" && grep -q 'gap' "$SKILL_FILE" && grep -q 'conflict' "$SKILL_FILE"; then
    echo "PASS: Documents escalation reasons (error/gap/conflict)"
else
    echo "FAIL: Missing escalation reasons (must include error, gap, conflict)"
    exit 1
fi

# TASK-25b Requirement: Documents escalate-user yield
if grep -q 'escalate-user' "$SKILL_FILE"; then
    echo "PASS: Documents escalate-user yield"
else
    echo "FAIL: Missing escalate-user yield documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
