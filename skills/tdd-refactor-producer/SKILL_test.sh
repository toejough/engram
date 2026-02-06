#!/bin/bash
# tdd-refactor-producer SKILL.md validation tests for TASK-21
# Run: bash skills/tdd-refactor-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/tdd-refactor-producer/SKILL.md"

echo "=== tdd-refactor-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-21 Requirement: Frontmatter has name field
if grep -q '^name: tdd-refactor-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: tdd-refactor-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-21 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-21 Requirement: Frontmatter has phase: tdd-refactor
if grep -q '^phase: tdd-refactor' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: tdd-refactor"
else
    echo "FAIL: Frontmatter missing phase: tdd-refactor"
    exit 1
fi

# TASK-21 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
if grep -qi 'GATHER' "$SKILL_FILE" && grep -qi 'SYNTHESIZE' "$SKILL_FILE" && grep -qi 'PRODUCE' "$SKILL_FILE"; then
    echo "PASS: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)"
else
    echo "FAIL: Missing PRODUCER-TEMPLATE pattern (must include GATHER, SYNTHESIZE, PRODUCE)"
    exit 1
fi

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
fi

# TASK-21 Requirement: Documents improving code quality while keeping tests green
if grep -qi 'code quality' "$SKILL_FILE" && grep -qi 'tests green' "$SKILL_FILE"; then
    echo "PASS: Documents improving code quality while keeping tests green"
else
    echo "FAIL: Missing documentation about improving code quality while keeping tests green"
    exit 1
fi

# TASK-21 Requirement: Documents that all tests must still pass after refactor
if grep -qi 'tests must.*pass' "$SKILL_FILE" || grep -qi 'all tests.*pass' "$SKILL_FILE" || grep -qi 'tests.*still pass' "$SKILL_FILE"; then
    echo "PASS: Documents that all tests must still pass after refactor"
else
    echo "FAIL: Missing documentation about tests passing after refactor"
    exit 1
fi

# TASK-21 Requirement: Documents refactored artifact delivery
if grep -qiE 'refactor|improve.*quality' "$SKILL_FILE"; then
    echo "PASS: Documents refactored artifact delivery"
else
    echo "FAIL: Missing refactored artifact delivery documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
