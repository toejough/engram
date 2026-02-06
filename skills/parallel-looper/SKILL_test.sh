#!/bin/bash
# parallel-looper SKILL.md validation tests for TASK-25a
# Run: bash skills/parallel-looper/SKILL_test.sh

set -e
SKILL_FILE="skills/parallel-looper/SKILL.md"

echo "=== parallel-looper SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-25a Requirement: Frontmatter has name field
if grep -q '^name: parallel-looper' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: parallel-looper"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-25a Requirement: Frontmatter has role (standalone, not producer/qa since it's an orchestrator)
if grep -q '^role: standalone' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role: standalone"
else
    echo "FAIL: Frontmatter missing role: standalone"
    exit 1
fi

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
fi

# TASK-25a Requirement: Receives list of independent items from LOOPER
if grep -qi 'items' "$SKILL_FILE" && grep -qi 'independent' "$SKILL_FILE"; then
    echo "PASS: Documents receiving independent items from LOOPER"
else
    echo "FAIL: Missing documentation for receiving independent items from LOOPER"
    exit 1
fi

# TASK-25a Requirement: Spawns PAIR LOOP for each item via Task tool (in parallel)
if grep -qi 'PAIR LOOP' "$SKILL_FILE" && grep -qi 'Task tool' "$SKILL_FILE" && grep -qi 'parallel' "$SKILL_FILE"; then
    echo "PASS: Documents spawning PAIR LOOPs via Task tool in parallel"
else
    echo "FAIL: Missing documentation for spawning PAIR LOOPs via Task tool in parallel"
    exit 1
fi

# TASK-25a Requirement: Aggregates results from all parallel PAIR LOOPs
if grep -qi 'aggregate' "$SKILL_FILE" && grep -qi 'results' "$SKILL_FILE"; then
    echo "PASS: Documents aggregating results from parallel PAIR LOOPs"
else
    echo "FAIL: Missing documentation for aggregating results"
    exit 1
fi

# TASK-25a Requirement: Dispatches to consistency-checker for batch QA
if grep -qi 'consistency-checker' "$SKILL_FILE"; then
    echo "PASS: Documents dispatching to consistency-checker"
else
    echo "FAIL: Missing documentation for consistency-checker dispatch"
    exit 1
fi

# TASK-25a Requirement: Handles partial failures (some items fail, others succeed)
if grep -qi 'partial' "$SKILL_FILE" && grep -qi 'fail' "$SKILL_FILE"; then
    echo "PASS: Documents partial failure handling"
else
    echo "FAIL: Missing documentation for partial failure handling"
    exit 1
fi

# TASK-25a Requirement: Outputs yield protocol TOML
if grep -q 'complete' "$SKILL_FILE" && grep -qi 'yield' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield output"
else
    echo "FAIL: Missing complete yield documentation"
    exit 1
fi

# TASK-25a Requirement: Documents TOML format for yield
if grep -q '\[yield\]' "$SKILL_FILE" || grep -q '\[payload\]' "$SKILL_FILE"; then
    echo "PASS: Documents TOML yield format"
else
    echo "FAIL: Missing TOML yield format documentation"
    exit 1
fi

# TASK-25a Requirement: Documents input context format
if grep -qi 'input' "$SKILL_FILE" && grep -qi 'context' "$SKILL_FILE"; then
    echo "PASS: Documents input context format"
else
    echo "FAIL: Missing input context format documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
