#!/bin/bash
# context-explorer SKILL.md validation tests for TASK-25
# Run: bash skills/context-explorer/SKILL_test.sh

set -e
SKILL_FILE="skills/context-explorer/SKILL.md"

echo "=== context-explorer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-25 Requirement: Frontmatter has name field
if grep -q '^name: context-explorer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: context-explorer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-25 Requirement: Frontmatter has role (standalone or producer-like)
if grep -q '^role:' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role field"
else
    echo "FAIL: Frontmatter missing role field"
    exit 1
fi

# TASK-25 Requirement: References YIELD.md
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# TASK-25 Requirement: Handles file query type
if grep -q 'file' "$SKILL_FILE" && grep -qi 'Read' "$SKILL_FILE"; then
    echo "PASS: Documents file query type with Read tool"
else
    echo "FAIL: Missing file query type documentation"
    exit 1
fi

# TASK-25 Requirement: Handles memory query type
if grep -q 'memory' "$SKILL_FILE"; then
    echo "PASS: Documents memory query type"
else
    echo "FAIL: Missing memory query type documentation"
    exit 1
fi

# TASK-25 Requirement: Handles territory query type
if grep -q 'territory' "$SKILL_FILE"; then
    echo "PASS: Documents territory query type"
else
    echo "FAIL: Missing territory query type documentation"
    exit 1
fi

# TASK-25 Requirement: Handles web query type
if grep -q 'web' "$SKILL_FILE" && grep -qi 'WebFetch' "$SKILL_FILE"; then
    echo "PASS: Documents web query type with WebFetch tool"
else
    echo "FAIL: Missing web query type documentation"
    exit 1
fi

# TASK-25 Requirement: Handles semantic query type
if grep -q 'semantic' "$SKILL_FILE" && grep -qi 'Task' "$SKILL_FILE"; then
    echo "PASS: Documents semantic query type with Task tool"
else
    echo "FAIL: Missing semantic query type documentation"
    exit 1
fi

# TASK-25 Requirement: Can parallelize queries (Task tool)
if grep -qi 'parallel' "$SKILL_FILE" && grep -qi 'Task' "$SKILL_FILE"; then
    echo "PASS: Documents query parallelization via Task tool"
else
    echo "FAIL: Missing parallelization documentation"
    exit 1
fi

# TASK-25 Requirement: Returns aggregated context
if grep -qi 'aggregat' "$SKILL_FILE"; then
    echo "PASS: Documents aggregated context return"
else
    echo "FAIL: Missing aggregated context documentation"
    exit 1
fi

# TASK-25 Requirement: Outputs yield protocol TOML (complete with results)
if grep -q 'complete' "$SKILL_FILE" && grep -q '\[yield\]' "$SKILL_FILE"; then
    echo "PASS: Documents complete yield with TOML format"
else
    echo "FAIL: Missing complete yield documentation with TOML"
    exit 1
fi

# TASK-25 Requirement: Outputs results in payload
if grep -q '\[payload\]' "$SKILL_FILE" && grep -qi 'results' "$SKILL_FILE"; then
    echo "PASS: Documents results in yield payload"
else
    echo "FAIL: Missing results in yield payload documentation"
    exit 1
fi

# TASK-25 Requirement: Documents input format (queries from need-context)
if grep -q 'queries' "$SKILL_FILE" && grep -qi 'need-context' "$SKILL_FILE"; then
    echo "PASS: Documents input as queries from need-context"
else
    echo "FAIL: Missing input format documentation"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
