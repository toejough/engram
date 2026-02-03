#!/bin/bash
# breakdown-producer SKILL.md validation tests for TASK-11
# Run: bash skills/breakdown-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/breakdown-producer/SKILL.md"

echo "=== breakdown-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-11 Requirement: Frontmatter has name field
if grep -q '^name: breakdown-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: breakdown-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-11 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-11 Requirement: Frontmatter has phase: breakdown
if grep -q '^phase: breakdown' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: breakdown"
else
    echo "FAIL: Frontmatter missing phase: breakdown"
    exit 1
fi

# TASK-11 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
if grep -qi 'GATHER' "$SKILL_FILE" && grep -qi 'SYNTHESIZE' "$SKILL_FILE" && grep -qi 'PRODUCE' "$SKILL_FILE"; then
    echo "PASS: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)"
else
    echo "FAIL: Missing PRODUCER-TEMPLATE pattern (must include GATHER, SYNTHESIZE, PRODUCE)"
    exit 1
fi

# TASK-11 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-11 Requirement: Documents complete yield with tasks.md artifact
if grep -q 'complete' "$SKILL_FILE" && grep -q 'tasks.md' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield with tasks.md artifact"
else
    echo "FAIL: Missing complete yield or tasks.md artifact documentation"
    exit 1
fi

# TASK-11 Requirement: Produces TASK-N IDs
if grep -q 'TASK-' "$SKILL_FILE"; then
    echo "PASS: Documents TASK-N ID format"
else
    echo "FAIL: Missing TASK-N ID format documentation"
    exit 1
fi

# TASK-11 Requirement: Includes dependency graph documentation
if grep -qi 'dependency' "$SKILL_FILE" && grep -qi 'graph' "$SKILL_FILE"; then
    echo "PASS: Includes dependency graph documentation"
else
    echo "FAIL: Missing dependency graph documentation"
    exit 1
fi

# TASK-11 Requirement: Documents need-context yield for gathering architecture docs
if grep -q 'need-context' "$SKILL_FILE"; then
    echo "PASS: Documents need-context yield"
else
    echo "FAIL: Missing need-context yield documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
