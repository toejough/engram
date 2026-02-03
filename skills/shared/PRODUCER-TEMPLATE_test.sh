#!/bin/bash
# PRODUCER-TEMPLATE.md validation tests for TASK-3
# Run: bash skills/shared/PRODUCER-TEMPLATE_test.sh

set -e
TEMPLATE_FILE="skills/shared/PRODUCER-TEMPLATE.md"

echo "=== PRODUCER-TEMPLATE.md Validation Tests ==="

# Check file exists
if [[ ! -f "$TEMPLATE_FILE" ]]; then
    echo "FAIL: $TEMPLATE_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-3 Requirement: Frontmatter with role field
if grep -q 'role:' "$TEMPLATE_FILE"; then
    echo "PASS: Frontmatter has role field"
else
    echo "FAIL: Frontmatter missing role field"
    exit 1
fi

# TASK-3 Requirement: Frontmatter with phase field
if grep -q 'phase:' "$TEMPLATE_FILE"; then
    echo "PASS: Frontmatter has phase field"
else
    echo "FAIL: Frontmatter missing phase field"
    exit 1
fi

# TASK-3 Requirement: Frontmatter with variant field
if grep -q 'variant:' "$TEMPLATE_FILE"; then
    echo "PASS: Frontmatter has variant field"
else
    echo "FAIL: Frontmatter missing variant field"
    exit 1
fi

# TASK-3 Requirement: GATHER pattern documented
if grep -qi 'GATHER' "$TEMPLATE_FILE"; then
    echo "PASS: GATHER pattern documented"
else
    echo "FAIL: GATHER pattern NOT documented"
    exit 1
fi

# TASK-3 Requirement: SYNTHESIZE pattern documented
if grep -qi 'SYNTHESIZE' "$TEMPLATE_FILE"; then
    echo "PASS: SYNTHESIZE pattern documented"
else
    echo "FAIL: SYNTHESIZE pattern NOT documented"
    exit 1
fi

# TASK-3 Requirement: PRODUCE pattern documented
if grep -qi 'PRODUCE' "$TEMPLATE_FILE"; then
    echo "PASS: PRODUCE pattern documented"
else
    echo "FAIL: PRODUCE pattern NOT documented"
    exit 1
fi

# TASK-3 Requirement: Yield format section
if grep -qi 'yield' "$TEMPLATE_FILE"; then
    echo "PASS: Yield format section exists"
else
    echo "FAIL: Yield format section missing"
    exit 1
fi

# TASK-3 Requirement: Reference to YIELD.md
if grep -q 'YIELD.md' "$TEMPLATE_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

# Should specify role = producer
if grep -q 'role:.*producer' "$TEMPLATE_FILE" || grep -q 'role = "producer"' "$TEMPLATE_FILE"; then
    echo "PASS: Specifies producer role"
else
    echo "FAIL: Does not specify producer role"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
