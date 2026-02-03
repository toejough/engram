#!/bin/bash
# arch-interview-producer SKILL.md validation tests for TASK-9
# Run: bash skills/arch-interview-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/arch-interview-producer/SKILL.md"

echo "=== arch-interview-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-9 Requirement: Frontmatter has name field
if grep -q '^name: arch-interview-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: arch-interview-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-9 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-9 Requirement: Frontmatter has phase: arch
if grep -q '^phase: arch' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: arch"
else
    echo "FAIL: Frontmatter missing phase: arch"
    exit 1
fi

# TASK-9 Requirement: Frontmatter has variant: interview
if grep -q '^variant: interview' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has variant: interview"
else
    echo "FAIL: Frontmatter missing variant: interview"
    exit 1
fi

# TASK-9 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
if grep -qi 'GATHER' "$SKILL_FILE" && grep -qi 'SYNTHESIZE' "$SKILL_FILE" && grep -qi 'PRODUCE' "$SKILL_FILE"; then
    echo "PASS: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)"
else
    echo "FAIL: Missing PRODUCER-TEMPLATE pattern (must include GATHER, SYNTHESIZE, PRODUCE)"
    exit 1
fi

# TASK-9 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-9 Requirement: Documents need-user-input yield for architecture questions
if grep -q 'need-user-input' "$SKILL_FILE"; then
    echo "PASS: Documents need-user-input yield"
else
    echo "FAIL: Missing need-user-input yield documentation"
    exit 1
fi

# TASK-9 Requirement: Documents complete yield with architecture.md artifact
if grep -q 'complete' "$SKILL_FILE" && grep -q 'architecture.md' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield with architecture.md artifact"
else
    echo "FAIL: Missing complete yield or architecture.md artifact documentation"
    exit 1
fi

# TASK-9 Requirement: Documents ARCH-N ID format
if grep -q 'ARCH-' "$SKILL_FILE"; then
    echo "PASS: Documents ARCH-N ID format"
else
    echo "FAIL: Missing ARCH-N ID format documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
