#!/bin/bash
# pm-infer-producer SKILL.md validation tests for TASK-6
# Run: bash skills/pm-infer-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/pm-infer-producer/SKILL.md"

echo "=== pm-infer-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-6 Requirement: Frontmatter has name field
if grep -q '^name: pm-infer-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: pm-infer-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-6 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-6 Requirement: Frontmatter has phase: pm
if grep -q '^phase: pm' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: pm"
else
    echo "FAIL: Frontmatter missing phase: pm"
    exit 1
fi

# TASK-6 Requirement: Frontmatter has variant: infer
if grep -q '^variant: infer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has variant: infer"
else
    echo "FAIL: Frontmatter missing variant: infer"
    exit 1
fi

# TASK-6 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
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

# TASK-6 Requirement: Documents need-context yield for code exploration
if grep -q 'need-context' "$SKILL_FILE"; then
    echo "PASS: Documents need-context yield"
else
    echo "FAIL: Missing need-context yield documentation"
    exit 1
fi

# TASK-6 Requirement: Documents complete yield with requirements.md artifact
if grep -q 'complete' "$SKILL_FILE" && grep -q 'requirements.md' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield with requirements.md artifact"
else
    echo "FAIL: Missing complete yield or requirements.md artifact documentation"
    exit 1
fi

# TASK-6 Requirement: Documents REQ-N ID format
if grep -q 'REQ-' "$SKILL_FILE"; then
    echo "PASS: Documents REQ-N ID format"
else
    echo "FAIL: Missing REQ-N ID format documentation"
    exit 1
fi

# TASK-6 Requirement: Describes analyzing existing code to infer requirements
if grep -qi 'analy' "$SKILL_FILE" && grep -qi 'code' "$SKILL_FILE" && grep -qi 'infer' "$SKILL_FILE"; then
    echo "PASS: Describes analyzing existing code to infer requirements"
else
    echo "FAIL: Missing description of analyzing existing code to infer requirements"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
