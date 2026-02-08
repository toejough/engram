#!/bin/bash
# next-steps SKILL.md validation tests for TASK-27
# Run: bash skills/next-steps/SKILL_test.sh

set -e
SKILL_FILE="skills/next-steps/SKILL.md"

echo "=== next-steps SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-27 Requirement: Frontmatter has name field
if grep -q '^name: next-steps' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: next-steps"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-27 Requirement: Skill must be user-invocable (standalone)
if grep -q '^user-invocable: true' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has user-invocable: true"
else
    echo "FAIL: Frontmatter missing user-invocable: true (standalone skill)"
    exit 1
fi

# TASK-27 Requirement: Suggests follow-up work based on completed project
if grep -qi 'follow-up\|next.*steps\|suggest' "$SKILL_FILE"; then
    echo "PASS: Documents suggesting follow-up work"
else
    echo "FAIL: Missing documentation about suggesting follow-up work"
    exit 1
fi

# TASK-27 Requirement: References open issues
if grep -qi 'open.*issue\|issues\.md\|ISSUE-' "$SKILL_FILE"; then
    echo "PASS: References open issues"
else
    echo "FAIL: Missing reference to open issues"
    exit 1
fi

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
fi

# Has messaging documentation
if grep -qi 'SendMessage\|message' "$SKILL_FILE"; then
    echo "PASS: Has messaging documentation"
else
    echo "FAIL: Missing messaging documentation"
    exit 1
fi

# TASK-27 Requirement: Documents what inputs it needs (context about completed work)
if grep -qi 'input\|context' "$SKILL_FILE" && grep -qi 'completed\|task\|project' "$SKILL_FILE"; then
    echo "PASS: Documents input context requirements"
else
    echo "FAIL: Missing documentation of input requirements for completed work context"
    exit 1
fi

# TASK-40: Memory query integration tests
echo ""
echo "=== TASK-40: Memory Integration Tests ==="

# AC-1: GATHER phase includes query for past project recommendations
if grep -q 'projctl memory query.*past project recommendations' "$SKILL_FILE"; then
    echo "PASS: GATHER includes query for past project recommendations"
else
    echo "FAIL: Missing 'projctl memory query' for past project recommendations"
    exit 1
fi

# AC-2: GATHER phase includes query for follow-up patterns
if grep -q 'projctl memory query.*follow-up patterns' "$SKILL_FILE"; then
    echo "PASS: GATHER includes query for follow-up patterns"
else
    echo "FAIL: Missing 'projctl memory query' for follow-up patterns"
    exit 1
fi

# AC-3: Graceful degradation documented
if grep -qi 'graceful.*degradation\|memory.*fail\|optional.*memory' "$SKILL_FILE"; then
    echo "PASS: Graceful degradation documented"
else
    echo "FAIL: Missing graceful degradation documentation"
    exit 1
fi

# AC-4: At least 2 memory query commands present
MEMORY_QUERY_COUNT=$(grep -c 'projctl memory query' "$SKILL_FILE" || echo "0")
if [[ "$MEMORY_QUERY_COUNT" -ge 2 ]]; then
    echo "PASS: Contains at least 2 memory query commands (found: $MEMORY_QUERY_COUNT)"
else
    echo "FAIL: Expected at least 2 'projctl memory query' commands, found: $MEMORY_QUERY_COUNT"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
