#!/bin/bash
# tdd-red-producer SKILL.md validation tests for TASK-18
# Run: bash skills/tdd-red-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/tdd-red-producer/SKILL.md"

echo "=== tdd-red-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-18 Requirement: Frontmatter has name field
if grep -q '^name: tdd-red-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: tdd-red-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-18 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-18 Requirement: Frontmatter has phase: tdd-red
if grep -q '^phase: tdd-red' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: tdd-red"
else
    echo "FAIL: Frontmatter missing phase: tdd-red"
    exit 1
fi

# TASK-18 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
if grep -qi 'GATHER' "$SKILL_FILE" && grep -qi 'SYNTHESIZE' "$SKILL_FILE" && grep -qi 'PRODUCE' "$SKILL_FILE"; then
    echo "PASS: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)"
else
    echo "FAIL: Missing PRODUCER-TEMPLATE pattern (must include GATHER, SYNTHESIZE, PRODUCE)"
    exit 1
fi

# TASK-18 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-18 Requirement: Documents writing failing tests
if grep -qi 'failing test' "$SKILL_FILE" || grep -qi 'write.*test' "$SKILL_FILE"; then
    echo "PASS: Documents writing failing tests"
else
    echo "FAIL: Missing documentation about writing failing tests"
    exit 1
fi

# TASK-18 Requirement: Documents that tests must fail (verifies correct red state)
if grep -qi 'must fail' "$SKILL_FILE" || grep -qi 'tests.*fail' "$SKILL_FILE"; then
    echo "PASS: Documents that tests must fail (verifies correct red state)"
else
    echo "FAIL: Missing documentation that tests must fail"
    exit 1
fi

# TASK-18 Requirement: Documents complete yield
if grep -q 'complete' "$SKILL_FILE" && grep -qi 'yield' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield"
else
    echo "FAIL: Missing complete yield documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
