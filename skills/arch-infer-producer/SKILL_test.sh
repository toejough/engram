#!/bin/bash
# arch-infer-producer SKILL.md validation tests for TASK-10
# Run: bash skills/arch-infer-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/arch-infer-producer/SKILL.md"

echo "=== arch-infer-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-10 Requirement: Frontmatter has name field
if grep -q '^name: arch-infer-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: arch-infer-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-10 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-10 Requirement: Frontmatter has phase: arch
if grep -q '^phase: arch' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: arch"
else
    echo "FAIL: Frontmatter missing phase: arch"
    exit 1
fi

# TASK-10 Requirement: Frontmatter has variant: infer
if grep -q '^variant: infer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has variant: infer"
else
    echo "FAIL: Frontmatter missing variant: infer"
    exit 1
fi

# TASK-10 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
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

# TASK-10 Requirement: Documents architecture.md artifact delivery
if grep -q 'architecture.md' "$SKILL_FILE"; then
    echo "PASS: Documents architecture.md artifact delivery"
else
    echo "FAIL: Missing architecture.md artifact documentation"
    exit 1
fi

# TASK-10 Requirement: Documents ARCH-N ID format
if grep -q 'ARCH-' "$SKILL_FILE"; then
    echo "PASS: Documents ARCH-N ID format"
else
    echo "FAIL: Missing ARCH-N ID format documentation"
    exit 1
fi

# TASK-10 Requirement: Describes analyzing existing code to infer architecture decisions
if grep -qi 'code' "$SKILL_FILE" && grep -qi 'infer' "$SKILL_FILE" && grep -qi 'architecture' "$SKILL_FILE"; then
    echo "PASS: Describes analyzing code to infer architecture"
else
    echo "FAIL: Missing description of analyzing code to infer architecture"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
