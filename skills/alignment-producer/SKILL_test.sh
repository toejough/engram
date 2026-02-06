#!/bin/bash
# alignment-producer SKILL.md validation tests for TASK-24
# Run: bash skills/alignment-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/alignment-producer/SKILL.md"

echo "=== alignment-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-24 Requirement: Frontmatter has name field
if grep -q '^name: alignment-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: alignment-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-24 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-24 Requirement: Frontmatter has phase: alignment
if grep -q '^phase: alignment' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: alignment"
else
    echo "FAIL: Frontmatter missing phase: alignment"
    exit 1
fi

# TASK-24 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
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

# TASK-24 Requirement: Documents traceability chain validation (REQ->DES->ARCH->TASK)
if grep -q 'REQ' "$SKILL_FILE" && grep -q 'DES' "$SKILL_FILE" && grep -q 'ARCH' "$SKILL_FILE" && grep -q 'TASK' "$SKILL_FILE"; then
    echo "PASS: Documents traceability chain (REQ, DES, ARCH, TASK)"
else
    echo "FAIL: Missing traceability chain documentation (must mention REQ, DES, ARCH, TASK)"
    exit 1
fi

# TASK-24 Requirement: Identifies broken traces
if grep -qi 'broken' "$SKILL_FILE" || grep -qi 'invalid' "$SKILL_FILE" || grep -qi 'orphan' "$SKILL_FILE"; then
    echo "PASS: Documents broken/invalid trace identification"
else
    echo "FAIL: Missing broken trace identification documentation"
    exit 1
fi

# TASK-24 Requirement: Identifies orphan IDs
if grep -qi 'orphan' "$SKILL_FILE"; then
    echo "PASS: Documents orphan ID identification"
else
    echo "FAIL: Missing orphan ID identification documentation"
    exit 1
fi

# TASK-24 Requirement: Identifies unlinked IDs
if grep -qi 'unlinked' "$SKILL_FILE"; then
    echo "PASS: Documents unlinked ID identification"
else
    echo "FAIL: Missing unlinked ID identification documentation"
    exit 1
fi

# TASK-24 Requirement: Documents alignment results delivery
if grep -qiE 'alignment.*result|validation.*result|SendMessage' "$SKILL_FILE"; then
    echo "PASS: Documents alignment results delivery"
else
    echo "FAIL: Missing alignment results delivery documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
