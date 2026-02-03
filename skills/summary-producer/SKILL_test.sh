#!/bin/bash
# summary-producer SKILL.md validation tests for TASK-24
# Run: bash skills/summary-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/summary-producer/SKILL.md"

echo "=== summary-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-24 Requirement: Frontmatter has name field
if grep -q '^name: summary-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: summary-producer"
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

# TASK-24 Requirement: Frontmatter has phase: summary
if grep -q '^phase: summary' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: summary"
else
    echo "FAIL: Frontmatter missing phase: summary"
    exit 1
fi

# TASK-24 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
if grep -qi 'GATHER' "$SKILL_FILE" && grep -qi 'SYNTHESIZE' "$SKILL_FILE" && grep -qi 'PRODUCE' "$SKILL_FILE"; then
    echo "PASS: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)"
else
    echo "FAIL: Missing PRODUCER-TEMPLATE pattern (must include GATHER, SYNTHESIZE, PRODUCE)"
    exit 1
fi

# TASK-24 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-24 Requirement: Documents complete yield with summary artifacts
if grep -q 'complete' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield"
else
    echo "FAIL: Missing complete yield documentation"
    exit 1
fi

# TASK-24 Requirement: Produces project summary with key decisions
if grep -qi 'decision' "$SKILL_FILE"; then
    echo "PASS: Documents key decisions in summary"
else
    echo "FAIL: Missing key decisions documentation"
    exit 1
fi

# TASK-24 Requirement: Produces project summary with outcomes
if grep -qi 'outcome' "$SKILL_FILE"; then
    echo "PASS: Documents outcomes in summary"
else
    echo "FAIL: Missing outcomes documentation"
    exit 1
fi

# TASK-24 Requirement: Outputs yield protocol TOML
if grep -q '\[yield\]' "$SKILL_FILE" && grep -q 'type = "complete"' "$SKILL_FILE"; then
    echo "PASS: Outputs yield protocol TOML"
else
    echo "FAIL: Missing yield protocol TOML example"
    exit 1
fi

# TASK-24 Requirement: References upstream artifacts
if grep -q 'REQ-' "$SKILL_FILE" || grep -q 'DES-' "$SKILL_FILE" || grep -q 'ARCH-' "$SKILL_FILE" || grep -q 'TASK-' "$SKILL_FILE"; then
    echo "PASS: References upstream artifacts"
else
    echo "FAIL: Missing references to upstream artifacts"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
