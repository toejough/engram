#!/bin/bash
# PRODUCER-TEMPLATE.md validation tests
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

# Frontmatter with role field
if grep -q 'role:' "$TEMPLATE_FILE"; then
    echo "PASS: Frontmatter has role field"
else
    echo "FAIL: Frontmatter missing role field"
    exit 1
fi

# Frontmatter with phase field
if grep -q 'phase:' "$TEMPLATE_FILE"; then
    echo "PASS: Frontmatter has phase field"
else
    echo "FAIL: Frontmatter missing phase field"
    exit 1
fi

# Frontmatter with variant field
if grep -q 'variant:' "$TEMPLATE_FILE"; then
    echo "PASS: Frontmatter has variant field"
else
    echo "FAIL: Frontmatter missing variant field"
    exit 1
fi

# GATHER pattern documented
if grep -qi 'GATHER' "$TEMPLATE_FILE"; then
    echo "PASS: GATHER pattern documented"
else
    echo "FAIL: GATHER pattern NOT documented"
    exit 1
fi

# SYNTHESIZE pattern documented
if grep -qi 'SYNTHESIZE' "$TEMPLATE_FILE"; then
    echo "PASS: SYNTHESIZE pattern documented"
else
    echo "FAIL: SYNTHESIZE pattern NOT documented"
    exit 1
fi

# PRODUCE pattern documented
if grep -qi 'PRODUCE' "$TEMPLATE_FILE"; then
    echo "PASS: PRODUCE pattern documented"
else
    echo "FAIL: PRODUCE pattern NOT documented"
    exit 1
fi

# Team mode: AskUserQuestion documented
if grep -q 'AskUserQuestion' "$TEMPLATE_FILE"; then
    echo "PASS: AskUserQuestion documented"
else
    echo "FAIL: AskUserQuestion not documented"
    exit 1
fi

# Team mode: SendMessage documented
if grep -q 'SendMessage' "$TEMPLATE_FILE"; then
    echo "PASS: SendMessage documented"
else
    echo "FAIL: SendMessage not documented"
    exit 1
fi

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$TEMPLATE_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
fi

# No legacy TOML references
if grep -q 'TOML' "$TEMPLATE_FILE"; then
    echo "FAIL: Legacy TOML reference still present"
    exit 1
else
    echo "PASS: No legacy TOML references"
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
