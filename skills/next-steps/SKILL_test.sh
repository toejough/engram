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

# Has completion message example
if grep -qi 'complete' "$SKILL_FILE" && grep -qi 'SendMessage\|completion message' "$SKILL_FILE"; then
    echo "PASS: Has completion message documentation"
else
    echo "FAIL: Missing completion message documentation"
    exit 1
fi

# TASK-27 Requirement: Documents what inputs it needs (context about completed work)
if grep -qi 'input\|context' "$SKILL_FILE" && grep -qi 'completed\|task\|project' "$SKILL_FILE"; then
    echo "PASS: Documents input context requirements"
else
    echo "FAIL: Missing documentation of input requirements for completed work context"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
