#!/bin/bash
# design-interview-producer SKILL.md validation tests for TASK-7
# Run: bash skills/design-interview-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/design-interview-producer/SKILL.md"

echo "=== design-interview-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# Check frontmatter exists (starts with ---)
if head -1 "$SKILL_FILE" | grep -q '^---$'; then
    echo "PASS: Frontmatter starts with ---"
else
    echo "FAIL: Frontmatter does not start with ---"
    exit 1
fi

# Check frontmatter has name field
if grep -q '^name:' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name field"
else
    echo "FAIL: Frontmatter missing name field"
    exit 1
fi

# Check frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing 'role: producer'"
    exit 1
fi

# Check frontmatter has phase: design
if grep -q '^phase: design' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: design"
else
    echo "FAIL: Frontmatter missing 'phase: design'"
    exit 1
fi

# Check frontmatter has variant: interview
if grep -q '^variant: interview' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has variant: interview"
else
    echo "FAIL: Frontmatter missing 'variant: interview'"
    exit 1
fi

# Check references PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
if grep -qi 'GATHER' "$SKILL_FILE"; then
    echo "PASS: References GATHER pattern"
else
    echo "FAIL: Missing GATHER pattern"
    exit 1
fi

if grep -qi 'SYNTHESIZE' "$SKILL_FILE"; then
    echo "PASS: References SYNTHESIZE pattern"
else
    echo "FAIL: Missing SYNTHESIZE pattern"
    exit 1
fi

if grep -qi 'PRODUCE' "$SKILL_FILE"; then
    echo "PASS: References PRODUCE pattern"
else
    echo "FAIL: Missing PRODUCE pattern"
    exit 1
fi

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
fi

# Check documents user interaction for design questions
if grep -q 'AskUserQuestion' "$SKILL_FILE"; then
    echo "PASS: Documents AskUserQuestion for design interview"
else
    echo "FAIL: Missing AskUserQuestion documentation"
    exit 1
fi

# Check documents design.md artifact delivery
if grep -q 'design.md' "$SKILL_FILE"; then
    echo "PASS: Documents design.md artifact delivery"
else
    echo "FAIL: Missing design.md artifact documentation"
    exit 1
fi

# Check produces DES-N IDs
if grep -q 'DES-' "$SKILL_FILE"; then
    echo "PASS: Documents DES-N ID format"
else
    echo "FAIL: Missing DES-N ID format documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
