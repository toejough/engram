#!/bin/bash
# intake-evaluator SKILL.md validation tests for TASK-26
# Run: bash skills/intake-evaluator/SKILL_test.sh

set -e
SKILL_FILE="skills/intake-evaluator/SKILL.md"

echo "=== intake-evaluator SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-26 Requirement: Frontmatter has name field
if grep -q '^name: intake-evaluator' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: intake-evaluator"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-26 Requirement: Frontmatter has role: evaluator
if grep -q '^role: evaluator' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: evaluator"
else
    echo "FAIL: Frontmatter missing role: evaluator"
    exit 1
fi

# TASK-26 Requirement: Must be standalone/user-invocable
if grep -q '^user-invocable: true' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has user-invocable: true"
else
    echo "FAIL: Frontmatter missing user-invocable: true"
    exit 1
fi

# TASK-26 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-26 Requirement: Documents classification types (new, adopt, align, single-task)
if grep -q 'new' "$SKILL_FILE" && \
   grep -q 'adopt' "$SKILL_FILE" && \
   grep -q 'align' "$SKILL_FILE" && \
   grep -q 'single-task' "$SKILL_FILE"; then
    echo "PASS: Documents all classification types (new, adopt, align, single-task)"
else
    echo "FAIL: Missing one or more classification types (new, adopt, align, single-task)"
    exit 1
fi

# TASK-26 Requirement: Documents complete yield with classification output
if grep -q 'complete' "$SKILL_FILE" && grep -q 'classification' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield with classification output"
else
    echo "FAIL: Missing complete yield or classification documentation"
    exit 1
fi

# TASK-26 Requirement: Documents need-decision yield for uncertain classification (escalation)
if grep -q 'need-decision' "$SKILL_FILE"; then
    echo "PASS: Documents need-decision yield for escalation"
else
    echo "FAIL: Missing need-decision yield for uncertain classification"
    exit 1
fi

# TASK-26 Requirement: Shows example TOML yield output with classification
if grep -qE '\[yield\]' "$SKILL_FILE" && grep -qE 'type\s*=' "$SKILL_FILE"; then
    echo "PASS: Shows TOML yield output format"
else
    echo "FAIL: Missing TOML yield output examples"
    exit 1
fi

# TASK-26 Requirement: Classification criteria documented
if grep -qi 'criteri' "$SKILL_FILE" || grep -qi 'signal' "$SKILL_FILE" || grep -qi 'indicator' "$SKILL_FILE"; then
    echo "PASS: Documents classification criteria/signals"
else
    echo "FAIL: Missing classification criteria documentation"
    exit 1
fi

# TASK-26 Requirement: Documents confidence threshold for escalation
if grep -qi 'confidence' "$SKILL_FILE" || grep -qi 'uncertain' "$SKILL_FILE" || grep -qi 'ambiguous' "$SKILL_FILE"; then
    echo "PASS: Documents uncertainty/confidence handling"
else
    echo "FAIL: Missing confidence/uncertainty handling documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
