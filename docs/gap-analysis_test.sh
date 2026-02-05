#!/bin/bash
# Gap analysis validation tests
# Run: bash docs/gap-analysis_test.sh

set -e
GAP_FILE="docs/gap-analysis.md"

echo "=== Gap Analysis Validation Tests ==="

# Check file exists
if [[ ! -f "$GAP_FILE" ]]; then
    echo "FAIL: $GAP_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# Check all 13 QA skills are analyzed
QA_SKILLS=(
    "pm-qa"
    "design-qa"
    "arch-qa"
    "breakdown-qa"
    "tdd-qa"
    "tdd-red-qa"
    "tdd-green-qa"
    "tdd-refactor-qa"
    "doc-qa"
    "context-qa"
    "alignment-qa"
    "retro-qa"
    "summary-qa"
)

for skill in "${QA_SKILLS[@]}"; do
    if grep -q "$skill" "$GAP_FILE"; then
        echo "PASS: $skill analyzed"
    else
        echo "FAIL: $skill NOT analyzed"
        exit 1
    fi
done

# Check required sections for each analysis
REQUIRED_SECTIONS=("Covered Checks" "Gaps" "Decision Required")
for section in "${REQUIRED_SECTIONS[@]}"; do
    count=$(grep -c "### $section" "$GAP_FILE" || true)
    if [[ $count -ge 13 ]]; then
        echo "PASS: '$section' section appears for all QA skills ($count instances)"
    else
        echo "FAIL: '$section' section missing (only $count instances, need 13+)"
        exit 1
    fi
done

# Check decisions are explicit (ADD or DROP)
add_count=$(grep -c "ADD to" "$GAP_FILE" || true)
drop_count=$(grep -c "DROP" "$GAP_FILE" || true)
decision_count=$((add_count + drop_count))
if [[ $decision_count -ge 10 ]]; then
    echo "PASS: $decision_count explicit decisions (ADD/DROP) documented"
else
    echo "FAIL: Only $decision_count decisions, expected >= 10"
    exit 1
fi

# Check summary tables exist
if grep -q "High Priority" "$GAP_FILE" && grep -q "Medium Priority" "$GAP_FILE"; then
    echo "PASS: Summary priority tables present"
else
    echo "FAIL: Missing priority summary tables"
    exit 1
fi

# Check producer names are mentioned
PRODUCERS=(
    "pm-interview-producer"
    "pm-infer-producer"
    "design-interview-producer"
    "breakdown-producer"
    "tdd-red-producer"
    "tdd-green-producer"
    "doc-producer"
    "context-explorer"
    "alignment-producer"
    "retro-producer"
    "summary-producer"
)

producer_count=0
for producer in "${PRODUCERS[@]}"; do
    if grep -q "$producer" "$GAP_FILE"; then
        producer_count=$((producer_count + 1))
    fi
done

if [[ $producer_count -ge 10 ]]; then
    echo "PASS: $producer_count producers referenced (>= 10 required)"
else
    echo "FAIL: Only $producer_count producers referenced (>= 10 required)"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
