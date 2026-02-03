#!/bin/bash
# pm-interview-producer SKILL.md validation tests for TASK-5
# Run: bash skills/pm-interview-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/pm-interview-producer/SKILL.md"

echo "=== pm-interview-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-5 Requirement: Frontmatter has name field
if grep -q '^name: pm-interview-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: pm-interview-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-5 Requirement: Frontmatter has role: producer
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-5 Requirement: Frontmatter has phase: pm
if grep -q '^phase: pm' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: pm"
else
    echo "FAIL: Frontmatter missing phase: pm"
    exit 1
fi

# TASK-5 Requirement: Frontmatter has variant: interview
if grep -q '^variant: interview' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has variant: interview"
else
    echo "FAIL: Frontmatter missing variant: interview"
    exit 1
fi

# TASK-5 Requirement: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
if grep -qi 'GATHER' "$SKILL_FILE" && grep -qi 'SYNTHESIZE' "$SKILL_FILE" && grep -qi 'PRODUCE' "$SKILL_FILE"; then
    echo "PASS: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)"
else
    echo "FAIL: Missing PRODUCER-TEMPLATE pattern (must include GATHER, SYNTHESIZE, PRODUCE)"
    exit 1
fi

# TASK-5 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-5 Requirement: Documents need-user-input yield for interview questions
if grep -q 'need-user-input' "$SKILL_FILE"; then
    echo "PASS: Documents need-user-input yield"
else
    echo "FAIL: Missing need-user-input yield documentation"
    exit 1
fi

# TASK-5 Requirement: Documents need-context yield for gathering existing docs
if grep -q 'need-context' "$SKILL_FILE"; then
    echo "PASS: Documents need-context yield"
else
    echo "FAIL: Missing need-context yield documentation"
    exit 1
fi

# TASK-5 Requirement: Documents complete yield with requirements.md artifact
if grep -q 'complete' "$SKILL_FILE" && grep -q 'requirements.md' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield with requirements.md artifact"
else
    echo "FAIL: Missing complete yield or requirements.md artifact documentation"
    exit 1
fi

# TASK-5 Requirement: Documents REQ-N ID format
if grep -q 'REQ-' "$SKILL_FILE"; then
    echo "PASS: Documents REQ-N ID format"
else
    echo "FAIL: Missing REQ-N ID format documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
