#!/bin/bash
# tdd-producer SKILL.md validation tests for TASK-22b
# Run: bash skills/tdd-producer/SKILL_test.sh

set -e
SKILL_FILE="skills/tdd-producer/SKILL.md"

echo "=== tdd-producer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-22b Requirement: Frontmatter has name field
if grep -q '^name: tdd-producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: tdd-producer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-22b Requirement: Frontmatter has role: producer (composite variant)
if grep -q '^role: producer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: producer"
else
    echo "FAIL: Frontmatter missing role: producer"
    exit 1
fi

# TASK-22b Requirement: Frontmatter has phase: tdd
if grep -q '^phase: tdd' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has phase: tdd"
else
    echo "FAIL: Frontmatter missing phase: tdd"
    exit 1
fi

# TASK-22b Requirement: Follows PRODUCER-TEMPLATE structure (composite variant)
if grep -qi 'GATHER' "$SKILL_FILE" && grep -qi 'SYNTHESIZE' "$SKILL_FILE" && grep -qi 'PRODUCE' "$SKILL_FILE"; then
    echo "PASS: Follows PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)"
else
    echo "FAIL: Missing PRODUCER-TEMPLATE pattern (must include GATHER, SYNTHESIZE, PRODUCE)"
    exit 1
fi

# TASK-22b Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-22b Requirement: Runs RED PAIR LOOP (red-producer + red-qa) internally
if grep -qi 'red-producer' "$SKILL_FILE" && grep -qi 'red-qa' "$SKILL_FILE"; then
    echo "PASS: Documents RED PAIR LOOP (red-producer + red-qa)"
else
    echo "FAIL: Missing RED PAIR LOOP documentation (must reference red-producer and red-qa)"
    exit 1
fi

# TASK-22b Requirement: Runs GREEN PAIR LOOP (green-producer + green-qa) internally
if grep -qi 'green-producer' "$SKILL_FILE" && grep -qi 'green-qa' "$SKILL_FILE"; then
    echo "PASS: Documents GREEN PAIR LOOP (green-producer + green-qa)"
else
    echo "FAIL: Missing GREEN PAIR LOOP documentation (must reference green-producer and green-qa)"
    exit 1
fi

# TASK-22b Requirement: Runs REFACTOR PAIR LOOP (refactor-producer + refactor-qa) internally
if grep -qi 'refactor-producer' "$SKILL_FILE" && grep -qi 'refactor-qa' "$SKILL_FILE"; then
    echo "PASS: Documents REFACTOR PAIR LOOP (refactor-producer + refactor-qa)"
else
    echo "FAIL: Missing REFACTOR PAIR LOOP documentation (must reference refactor-producer and refactor-qa)"
    exit 1
fi

# TASK-22b Requirement: Handles iteration/improvement within each nested pair loop
if grep -qi 'iteration' "$SKILL_FILE" && grep -qi 'improvement' "$SKILL_FILE"; then
    echo "PASS: Documents iteration/improvement handling within nested loops"
else
    echo "FAIL: Missing iteration/improvement handling documentation"
    exit 1
fi

# TASK-22b Requirement: Documents nested pair loop structure
if grep -qi 'pair loop' "$SKILL_FILE" || grep -qi 'nested.*loop' "$SKILL_FILE"; then
    echo "PASS: Documents nested pair loop structure"
else
    echo "FAIL: Missing nested pair loop structure documentation"
    exit 1
fi

# TASK-22b Requirement: Outputs yield protocol TOML after all nested loops complete
if grep -q 'complete' "$SKILL_FILE" && grep -qi 'yield' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield after all nested loops"
else
    echo "FAIL: Missing complete yield documentation"
    exit 1
fi

# TASK-22b Requirement: Is a composite producer (orchestrates other skills)
if grep -qi 'composite' "$SKILL_FILE" || grep -qi 'orchestrat' "$SKILL_FILE"; then
    echo "PASS: Documents composite/orchestration nature"
else
    echo "FAIL: Missing composite/orchestration documentation"
    exit 1
fi

# TASK-22b Requirement: Documents the RED -> GREEN -> REFACTOR sequence
if grep -qi 'RED.*GREEN.*REFACTOR' "$SKILL_FILE" || (grep -qi 'RED' "$SKILL_FILE" && grep -qi 'GREEN' "$SKILL_FILE" && grep -qi 'REFACTOR' "$SKILL_FILE"); then
    echo "PASS: Documents RED -> GREEN -> REFACTOR sequence"
else
    echo "FAIL: Missing RED -> GREEN -> REFACTOR sequence documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
