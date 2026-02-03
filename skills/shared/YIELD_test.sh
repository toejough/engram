#!/bin/bash
# YIELD.md validation tests
# Run: bash skills/shared/YIELD_test.sh

set -e
YIELD_FILE="skills/shared/YIELD.md"

echo "=== YIELD.md Validation Tests ==="

# Check file exists
if [[ ! -f "$YIELD_FILE" ]]; then
    echo "FAIL: $YIELD_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# Required yield types
YIELD_TYPES=("complete" "need-user-input" "need-context" "need-decision" "need-agent" "blocked" "error" "approved" "improvement-request" "escalate-phase" "escalate-user")

for type in "${YIELD_TYPES[@]}"; do
    if grep -q "\`$type\`" "$YIELD_FILE"; then
        echo "PASS: Yield type '$type' documented"
    else
        echo "FAIL: Yield type '$type' NOT documented"
        exit 1
    fi
done

# Check TOML examples exist (at least 8)
TOML_COUNT=$(grep -c '```toml' "$YIELD_FILE" || true)
if [[ $TOML_COUNT -ge 8 ]]; then
    echo "PASS: $TOML_COUNT TOML examples found (>= 8 required)"
else
    echo "FAIL: Only $TOML_COUNT TOML examples (>= 8 required)"
    exit 1
fi

# Check required sections
REQUIRED_SECTIONS=("Yield Types" "Yield Format" "Producer Yield" "QA Yield" "Context Serialization" "Query Types")

for section in "${REQUIRED_SECTIONS[@]}"; do
    if grep -qi "$section" "$YIELD_FILE"; then
        echo "PASS: Section '$section' exists"
    else
        echo "FAIL: Section '$section' NOT found"
        exit 1
    fi
done

# Check reference to orchestration-system.md
if grep -q "orchestration-system.md" "$YIELD_FILE"; then
    echo "PASS: References orchestration-system.md"
else
    echo "FAIL: Missing reference to orchestration-system.md"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
