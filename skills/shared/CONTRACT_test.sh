#!/bin/bash
# CONTRACT.md validation tests
# Run: bash skills/shared/CONTRACT_test.sh

set -e
CONTRACT_FILE="skills/shared/CONTRACT.md"

echo "=== CONTRACT.md Validation Tests ==="

# Check file exists
if [[ ! -f "$CONTRACT_FILE" ]]; then
    echo "FAIL: $CONTRACT_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# Required sections
REQUIRED_SECTIONS=("## Schema" "## Outputs" "## Traces" "## Checks" "## Severity" "## Examples")

for section in "${REQUIRED_SECTIONS[@]}"; do
    if grep -qi "^$section" "$CONTRACT_FILE"; then
        echo "PASS: Section '$section' exists"
    else
        echo "FAIL: Section '$section' NOT found"
        exit 1
    fi
done

# Check schema section has key fields documented
SCHEMA_FIELDS=("outputs" "traces_to" "checks" "id" "description" "severity" "path" "id_format")

for field in "${SCHEMA_FIELDS[@]}"; do
    if grep -q "\`$field\`" "$CONTRACT_FILE"; then
        echo "PASS: Field '$field' documented"
    else
        echo "FAIL: Field '$field' NOT documented"
        exit 1
    fi
done

# Check severity levels documented
if grep -q "error" "$CONTRACT_FILE" && grep -q "warning" "$CONTRACT_FILE"; then
    echo "PASS: Severity levels 'error' and 'warning' documented"
else
    echo "FAIL: Severity levels not properly documented"
    exit 1
fi

# Check at least 2 YAML examples exist (one complete, one minimal)
YAML_COUNT=$(grep -c '```yaml' "$CONTRACT_FILE" || true)
if [[ $YAML_COUNT -ge 2 ]]; then
    echo "PASS: $YAML_COUNT YAML examples found (>= 2 required)"
else
    echo "FAIL: Only $YAML_COUNT YAML examples (>= 2 required)"
    exit 1
fi

# Check complete contract example has all sections (outputs, traces_to, checks)
# Look for a single yaml block that contains all three
if grep -A 30 '```yaml' "$CONTRACT_FILE" | grep -q 'outputs:' && \
   grep -A 30 '```yaml' "$CONTRACT_FILE" | grep -q 'traces_to:' && \
   grep -A 30 '```yaml' "$CONTRACT_FILE" | grep -q 'checks:'; then
    echo "PASS: Complete contract example includes outputs, traces_to, checks"
else
    echo "FAIL: Missing complete contract example with all sections"
    exit 1
fi

# Check reference to DES-001 (contract format design decision)
if grep -q "DES-001" "$CONTRACT_FILE"; then
    echo "PASS: References DES-001 (contract format design)"
else
    echo "FAIL: Missing reference to DES-001"
    exit 1
fi

# Check reference to DES-002 (contract section placement)
if grep -q "DES-002" "$CONTRACT_FILE"; then
    echo "PASS: References DES-002 (contract section placement)"
else
    echo "FAIL: Missing reference to DES-002"
    exit 1
fi

# Check that "## Contract" heading placement is documented
if grep -qi "## Contract" "$CONTRACT_FILE"; then
    echo "PASS: Contract section heading documented"
else
    echo "FAIL: Contract section heading not documented"
    exit 1
fi

# Check producer type examples are present (at least 3 different phases)
PHASE_EXAMPLES=0
for phase in "pm" "design" "arch" "breakdown" "tdd" "doc"; do
    if grep -qi "$phase" "$CONTRACT_FILE"; then
        PHASE_EXAMPLES=$((PHASE_EXAMPLES + 1))
    fi
done
if [[ $PHASE_EXAMPLES -ge 3 ]]; then
    echo "PASS: $PHASE_EXAMPLES producer phases referenced (>= 3 required)"
else
    echo "FAIL: Only $PHASE_EXAMPLES phases referenced (>= 3 required)"
    exit 1
fi

# Check version and evolution policy is mentioned (TASK-1 acceptance criteria)
if grep -qi "version" "$CONTRACT_FILE"; then
    echo "PASS: Version/evolution discussed"
else
    echo "FAIL: Version/evolution policy not mentioned"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
