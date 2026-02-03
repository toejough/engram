#!/bin/bash
# QA-TEMPLATE.md validation tests for TASK-4
# Run: bash skills/shared/QA-TEMPLATE_test.sh

set -e
TEMPLATE_FILE="skills/shared/QA-TEMPLATE.md"

echo "=== QA-TEMPLATE.md Validation Tests ==="

# Check file exists
if [[ ! -f "$TEMPLATE_FILE" ]]; then
    echo "FAIL: $TEMPLATE_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-4 Requirement: Frontmatter with role field
if grep -q 'role:' "$TEMPLATE_FILE"; then
    echo "PASS: Frontmatter has role field"
else
    echo "FAIL: Frontmatter missing role field"
    exit 1
fi

# TASK-4 Requirement: REVIEW pattern documented
if grep -qi 'REVIEW' "$TEMPLATE_FILE"; then
    echo "PASS: REVIEW pattern documented"
else
    echo "FAIL: REVIEW pattern NOT documented"
    exit 1
fi

# TASK-4 Requirement: RETURN pattern documented
if grep -qi 'RETURN' "$TEMPLATE_FILE"; then
    echo "PASS: RETURN pattern documented"
else
    echo "FAIL: RETURN pattern NOT documented"
    exit 1
fi

# TASK-4 Requirement: Escalation responsibilities
if grep -qi 'escalat' "$TEMPLATE_FILE"; then
    echo "PASS: Escalation documented"
else
    echo "FAIL: Escalation NOT documented"
    exit 1
fi

# TASK-4 Requirement: Error reason type
if grep -q 'error' "$TEMPLATE_FILE"; then
    echo "PASS: Error escalation reason documented"
else
    echo "FAIL: Error escalation reason NOT documented"
    exit 1
fi

# TASK-4 Requirement: Gap reason type
if grep -qi 'gap' "$TEMPLATE_FILE"; then
    echo "PASS: Gap escalation reason documented"
else
    echo "FAIL: Gap escalation reason NOT documented"
    exit 1
fi

# TASK-4 Requirement: Conflict reason type
if grep -qi 'conflict' "$TEMPLATE_FILE"; then
    echo "PASS: Conflict escalation reason documented"
else
    echo "FAIL: Conflict escalation reason NOT documented"
    exit 1
fi

# TASK-4 Requirement: proposed_changes format
if grep -q 'proposed_changes' "$TEMPLATE_FILE"; then
    echo "PASS: proposed_changes format documented"
else
    echo "FAIL: proposed_changes format NOT documented"
    exit 1
fi

# TASK-4 Requirement: Reference to YIELD.md
if grep -q 'YIELD.md' "$TEMPLATE_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# Should specify role = qa
if grep -q 'role:.*qa' "$TEMPLATE_FILE" || grep -q 'role = "qa"' "$TEMPLATE_FILE"; then
    echo "PASS: Specifies qa role"
else
    echo "FAIL: Does not specify qa role"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
