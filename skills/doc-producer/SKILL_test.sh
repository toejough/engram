#!/bin/bash
# doc-producer SKILL.md validation tests for TASK-12
# Run: bash skills/doc-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/doc-producer/SKILL.md"

echo "=== doc-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-12 Requirement: Frontmatter has name field
if grep -q '^name: doc-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: doc-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-12 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-12 Requirement: Frontmatter has phase: doc
if grep -q '^phase: doc' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: doc"
else
    echo "FAIL: Frontmatter missing phase: doc"
    exit 1
fi

# TASK-12 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
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

# TASK-12 Requirement: Documents documentation artifact delivery
if grep -qi 'README\|API docs\|user guide\|documentation' "$SKILL_FILE"; then
    echo "PASS: Documents documentation artifact delivery"
else
    echo "FAIL: Missing documentation artifact delivery documentation"
    exit 1
fi

# TASK-12 Requirement: Traces to REQ-N, DES-N, ARCH-N
if grep -q 'REQ-' "$SKILL_FILE" && grep -q 'DES-' "$SKILL_FILE" && grep -q 'ARCH-' "$SKILL_FILE"; then
    echo "PASS: Documents tracing to REQ-N, DES-N, ARCH-N"
else
    echo "FAIL: Missing traceability to REQ-N, DES-N, ARCH-N"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
