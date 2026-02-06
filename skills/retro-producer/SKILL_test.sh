#!/bin/bash
# retro-producer SKILL.md validation tests for TASK-24
# Run: bash skills/retro-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/retro-producer/SKILL.md"

echo "=== retro-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-24 Requirement: Frontmatter has name field
if grep -q '^name: retro-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: retro-producer"
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

# TASK-24 Requirement: Frontmatter has phase: retro
if grep -q '^phase: retro' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: retro"
else
    echo "FAIL: Frontmatter missing phase: retro"
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

# TASK-24 Requirement: Documents complete yield
if grep -q 'complete' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield"
else
    echo "FAIL: Missing complete yield documentation"
    exit 1
fi

# TASK-24 Requirement: Covers "what went well"
if grep -qi 'went well\|successes\|worked well' "$SKILL_FILE"; then
    echo "PASS: Covers what went well / successes"
else
    echo "FAIL: Missing 'what went well' / successes coverage"
    exit 1
fi

# TASK-24 Requirement: Covers "what could improve"
if grep -qi 'could improve\|improvements\|challenges\|pain points' "$SKILL_FILE"; then
    echo "PASS: Covers what could improve / challenges"
else
    echo "FAIL: Missing 'what could improve' / challenges coverage"
    exit 1
fi

# TASK-24 Requirement: Documents process improvement recommendations
if grep -qi 'recommendation\|action item\|improvement' "$SKILL_FILE"; then
    echo "PASS: Documents process improvement recommendations"
else
    echo "FAIL: Missing process improvement recommendations"
    exit 1
fi

# TASK-24 Requirement: Shows TOML yield example
if grep -q '\[yield\]' "$SKILL_FILE" && grep -q 'type = "complete"' "$SKILL_FILE"; then
    echo "PASS: Shows TOML yield example"
else
    echo "FAIL: Missing TOML yield example with [yield] and type"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
