#!/bin/bash
# INTERVIEW-PATTERN.md validation tests
# Run: bash skills/shared/INTERVIEW-PATTERN_test.sh

set -e
PATTERN_FILE="skills/shared/INTERVIEW-PATTERN.md"

echo "=== INTERVIEW-PATTERN.md Validation Tests ==="

# File exists
if [[ -f "$PATTERN_FILE" ]]; then
    echo "PASS: File exists at $PATTERN_FILE"
else
    echo "FAIL: File does not exist at $PATTERN_FILE"
    exit 1
fi

# Documents five-phase flow with clear phase boundaries
REQUIRED_PHASES=("GATHER" "ASSESS" "INTERVIEW" "SYNTHESIZE" "PRODUCE")
for phase in "${REQUIRED_PHASES[@]}"; do
    if grep -q "$phase" "$PATTERN_FILE"; then
        echo "PASS: Phase '$phase' documented"
    else
        echo "FAIL: Phase '$phase' NOT documented"
        exit 1
    fi
done

# Verify phase boundaries are clear (each phase should have a heading)
PHASE_HEADING_COUNT=$(grep -c "^##.*\(GATHER\|ASSESS\|INTERVIEW\|SYNTHESIZE\|PRODUCE\)" "$PATTERN_FILE" || true)
if [[ $PHASE_HEADING_COUNT -ge 5 ]]; then
    echo "PASS: Five-phase flow has clear phase boundaries ($PHASE_HEADING_COUNT headings found)"
else
    echo "FAIL: Phase boundaries unclear (found $PHASE_HEADING_COUNT headings, need >= 5)"
    exit 1
fi

# Includes context gathering mechanism descriptions
CONTEXT_MECHANISMS=("territory" "memory")
for mechanism in "${CONTEXT_MECHANISMS[@]}"; do
    if grep -qi "$mechanism" "$PATTERN_FILE"; then
        echo "PASS: Context gathering mechanism '$mechanism' documented"
    else
        echo "FAIL: Context gathering mechanism '$mechanism' NOT documented"
        exit 1
    fi
done

# Territory map command documented
if grep -q "projctl territory map\|territory map" "$PATTERN_FILE"; then
    echo "PASS: Territory map command documented"
else
    echo "FAIL: Territory map command NOT documented"
    exit 1
fi

# Memory query documented
if grep -q "projctl memory query\|memory query" "$PATTERN_FILE"; then
    echo "PASS: Memory query command documented"
else
    echo "FAIL: Memory query command NOT documented"
    exit 1
fi

# Defines gap assessment approach with coverage calculation
if grep -qi "coverage" "$PATTERN_FILE"; then
    echo "PASS: Coverage calculation mentioned"
else
    echo "FAIL: Coverage calculation NOT mentioned"
    exit 1
fi

# Depth tiers (small, medium, large)
DEPTH_TIERS=("small" "medium" "large")
TIER_COUNT=0
for tier in "${DEPTH_TIERS[@]}"; do
    if grep -qi "$tier.*gap\|gap.*$tier" "$PATTERN_FILE"; then
        TIER_COUNT=$((TIER_COUNT + 1))
    fi
done
if [[ $TIER_COUNT -ge 3 ]]; then
    echo "PASS: Depth tiers documented (found $TIER_COUNT/3 tiers)"
else
    echo "FAIL: Depth tiers incomplete (found $TIER_COUNT/3 tiers)"
    exit 1
fi

# Percentage thresholds
if grep -qE "[0-9]+%" "$PATTERN_FILE"; then
    echo "PASS: Coverage percentage thresholds documented"
else
    echo "FAIL: Coverage percentage thresholds NOT documented"
    exit 1
fi

# Error handling: territory map failures
if grep -qi "territory.*fail\|fail.*territory" "$PATTERN_FILE"; then
    echo "PASS: Territory map failure handling documented"
else
    echo "FAIL: Territory map failure handling NOT documented"
    exit 1
fi

# Error handling: memory query failures/timeouts
if grep -qi "memory.*\(fail\|timeout\)\|\(fail\|timeout\).*memory" "$PATTERN_FILE"; then
    echo "PASS: Memory query failure handling documented"
else
    echo "FAIL: Memory query failure handling NOT documented"
    exit 1
fi

# Contradictory context handling
if grep -qi "contradict" "$PATTERN_FILE"; then
    echo "PASS: Contradictory context handling documented"
else
    echo "FAIL: Contradictory context handling NOT documented"
    exit 1
fi

# Gap analysis metadata format
if grep -q "gap_analysis\|gap analysis\|Gap Analysis" "$PATTERN_FILE"; then
    echo "PASS: Gap analysis metadata format documented"
else
    echo "FAIL: Gap analysis metadata format NOT documented"
    exit 1
fi

# Gap analysis fields documented
GAP_FIELDS=("total_key_questions\|total.*questions" "questions_answered\|answered" "coverage_percent\|coverage.*percent")
FIELD_COUNT=0
for field_pattern in "${GAP_FIELDS[@]}"; do
    if grep -qi "$field_pattern" "$PATTERN_FILE"; then
        FIELD_COUNT=$((FIELD_COUNT + 1))
    fi
done
if [[ $FIELD_COUNT -ge 2 ]]; then
    echo "PASS: Gap analysis fields documented ($FIELD_COUNT/3 found)"
else
    echo "FAIL: Gap analysis fields incomplete ($FIELD_COUNT/3 found)"
    exit 1
fi

# Blocker handling documented
if grep -qi "blocker" "$PATTERN_FILE"; then
    echo "PASS: Blocker handling documented"
else
    echo "FAIL: Blocker handling NOT documented"
    exit 1
fi

# Partial context handling
if grep -qi "partial.*context\|context.*partial" "$PATTERN_FILE"; then
    echo "PASS: Partial context handling documented"
else
    echo "FAIL: Partial context handling NOT documented"
    exit 1
fi

# Team mode: AskUserQuestion documented
if grep -q 'AskUserQuestion' "$PATTERN_FILE"; then
    echo "PASS: AskUserQuestion documented"
else
    echo "FAIL: AskUserQuestion not documented"
    exit 1
fi

# Team mode: SendMessage documented
if grep -q 'SendMessage' "$PATTERN_FILE"; then
    echo "PASS: SendMessage documented"
else
    echo "FAIL: SendMessage not documented"
    exit 1
fi

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$PATTERN_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
fi

echo ""
echo "=== All tests passed ==="
