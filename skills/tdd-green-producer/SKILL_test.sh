#!/bin/bash
# tdd-green-producer SKILL.md validation tests for TASK-20
# Run: bash skills/tdd-green-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/tdd-green-producer/SKILL.md"

echo "=== tdd-green-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-20 Requirement: Frontmatter has name field
if grep -q '^name: tdd-green-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: tdd-green-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-20 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-20 Requirement: Frontmatter has phase: tdd-green
if grep -q '^phase: tdd-green' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: tdd-green"
else
    echo "FAIL: Frontmatter missing phase: tdd-green"
    exit 1
fi

# TASK-20 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
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

# TASK-20 Requirement: Documents writing minimal implementation to pass tests
if grep -qi 'minimal' "$SKILL_FILE" && grep -qi 'implementation' "$SKILL_FILE" && grep -qi 'pass' "$SKILL_FILE"; then
    echo "PASS: Documents writing minimal implementation to pass tests"
else
    echo "FAIL: Missing documentation about writing minimal implementation to pass tests"
    exit 1
fi

# TASK-20 Requirement: Documents that all targeted tests must pass
if grep -qi 'all.*tests.*pass\|tests.*must.*pass\|all targeted tests' "$SKILL_FILE"; then
    echo "PASS: Documents that all targeted tests must pass"
else
    echo "FAIL: Missing documentation that all targeted tests must pass"
    exit 1
fi

# TASK-20 Requirement: Documents complete yield
if grep -q 'complete' "$SKILL_FILE" && grep -qi 'yield' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield"
else
    echo "FAIL: Missing complete yield documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
