#!/bin/bash
# CONTEXT.md validation tests for TASK-2 updates
# Run: bash skills/shared/CONTEXT_test.sh

set -e
CONTEXT_FILE="skills/shared/CONTEXT.md"

echo "=== CONTEXT.md Validation Tests ==="

# Check file exists
if [[ ! -f "$CONTEXT_FILE" ]]; then
    echo "FAIL: $CONTEXT_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-2 Requirement: [output] section with yield_path field
if grep -q '\[output\]' "$CONTEXT_FILE"; then
    echo "PASS: [output] section exists"
else
    echo "FAIL: [output] section NOT found"
    exit 1
fi

if grep -q 'yield_path' "$CONTEXT_FILE"; then
    echo "PASS: yield_path field documented"
else
    echo "FAIL: yield_path field NOT documented"
    exit 1
fi

# TASK-2 Requirement: Query result injection documentation
if grep -qi 'query.*result' "$CONTEXT_FILE" || grep -qi 'result.*injection' "$CONTEXT_FILE"; then
    echo "PASS: Query result injection documented"
else
    echo "FAIL: Query result injection NOT documented"
    exit 1
fi

# TASK-2 Requirement: Reference to need-context resumption
if grep -q 'need-context' "$CONTEXT_FILE"; then
    echo "PASS: References need-context"
else
    echo "FAIL: Missing reference to need-context"
    exit 1
fi

# Check TOML example contains [output] section
TOML_OUTPUT_COUNT=$(grep -A 50 '```toml' "$CONTEXT_FILE" | grep -c '\[output\]' || true)
if [[ $TOML_OUTPUT_COUNT -ge 1 ]]; then
    echo "PASS: TOML example includes [output] section"
else
    echo "FAIL: No TOML example with [output] section"
    exit 1
fi

# Check reference to YIELD.md
if grep -q 'YIELD.md' "$CONTEXT_FILE"; then
    echo "PASS: References YIELD.md"
else
    echo "FAIL: Missing reference to YIELD.md"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
