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

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
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

# TASK-26 Requirement: Documents classification output delivery
if grep -q 'classification' "$SKILL_FILE"; then
    echo "PASS: Documents classification output delivery"
else
    echo "FAIL: Missing classification documentation"
    exit 1
fi

# TASK-26 Requirement: Documents escalation for uncertain classification
if grep -qiE 'escalat|AskUserQuestion|SendMessage' "$SKILL_FILE"; then
    echo "PASS: Documents escalation for uncertain classification"
else
    echo "FAIL: Missing escalation documentation for uncertain classification"
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
