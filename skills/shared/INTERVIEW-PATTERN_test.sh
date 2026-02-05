#!/bin/bash
# INTERVIEW-PATTERN.md validation tests
# Tests TASK-1 acceptance criteria
# Run: bash skills/shared/INTERVIEW-PATTERN_test.sh

set -e
PATTERN_FILE="$HOME/.claude/skills/shared/INTERVIEW-PATTERN.md"

echo "=== INTERVIEW-PATTERN.md Validation Tests ==="

# AC-1: File created at ~/.claude/skills/shared/INTERVIEW-PATTERN.md
if [[ -f "$PATTERN_FILE" ]]; then
    echo "PASS: File exists at $PATTERN_FILE"
else
    echo "FAIL: File does not exist at $PATTERN_FILE"
    exit 1
fi

# AC-2: Documents five-phase flow with clear phase boundaries
# Must include all five phases: GATHER, ASSESS, INTERVIEW, SYNTHESIZE, PRODUCE
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

# AC-3: Includes context gathering mechanism descriptions
CONTEXT_MECHANISMS=("territory" "memory" "context-explorer")
for mechanism in "${CONTEXT_MECHANISMS[@]}"; do
    if grep -qi "$mechanism" "$PATTERN_FILE"; then
        echo "PASS: Context gathering mechanism '$mechanism' documented"
    else
        echo "FAIL: Context gathering mechanism '$mechanism' NOT documented"
        exit 1
    fi
done

# Verify territory map command is mentioned
if grep -q "projctl territory map\|territory map" "$PATTERN_FILE"; then
    echo "PASS: Territory map command documented"
else
    echo "FAIL: Territory map command NOT documented"
    exit 1
fi

# Verify memory query is mentioned
if grep -q "projctl memory query\|memory query" "$PATTERN_FILE"; then
    echo "PASS: Memory query command documented"
else
    echo "FAIL: Memory query command NOT documented"
    exit 1
fi

# AC-4: Defines gap assessment approach
# Must include coverage calculation description
if grep -qi "coverage" "$PATTERN_FILE"; then
    echo "PASS: Coverage calculation mentioned"
else
    echo "FAIL: Coverage calculation NOT mentioned"
    exit 1
fi

# Must define depth tiers (small, medium, large)
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

# Must include percentage thresholds
if grep -qE "[0-9]+%" "$PATTERN_FILE"; then
    echo "PASS: Coverage percentage thresholds documented"
else
    echo "FAIL: Coverage percentage thresholds NOT documented"
    exit 1
fi

# AC-5: Specifies error handling patterns for context failures
# Must mention territory map failures
if grep -qi "territory.*fail\|fail.*territory" "$PATTERN_FILE"; then
    echo "PASS: Territory map failure handling documented"
else
    echo "FAIL: Territory map failure handling NOT documented"
    exit 1
fi

# Must mention memory query failures/timeouts
if grep -qi "memory.*\(fail\|timeout\)\|\(fail\|timeout\).*memory" "$PATTERN_FILE"; then
    echo "PASS: Memory query failure handling documented"
else
    echo "FAIL: Memory query failure handling NOT documented"
    exit 1
fi

# Must mention contradictory context
if grep -qi "contradict" "$PATTERN_FILE"; then
    echo "PASS: Contradictory context handling documented"
else
    echo "FAIL: Contradictory context handling NOT documented"
    exit 1
fi

# AC-6: Includes yield context enrichment format with gap analysis metadata
# Must have example or description of gap_analysis section
if grep -q "gap_analysis\|gap-analysis" "$PATTERN_FILE"; then
    echo "PASS: Gap analysis metadata format documented"
else
    echo "FAIL: Gap analysis metadata format NOT documented"
    exit 1
fi

# Must mention yield context enrichment
if grep -qi "yield.*context\|context.*yield" "$PATTERN_FILE"; then
    echo "PASS: Yield context enrichment mentioned"
else
    echo "FAIL: Yield context enrichment NOT mentioned"
    exit 1
fi

# Should include example fields: total_key_questions, questions_answered, coverage_percent
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

# AC-7: Documents when to yield blocked, need-decision, or continue with partial context
YIELD_TYPES=("blocked" "need-decision")
for yield_type in "${YIELD_TYPES[@]}"; do
    if grep -qi "$yield_type" "$PATTERN_FILE"; then
        echo "PASS: Yield type '$yield_type' documented"
    else
        echo "FAIL: Yield type '$yield_type' NOT documented"
        exit 1
    fi
done

# Must mention partial context handling
if grep -qi "partial.*context\|context.*partial" "$PATTERN_FILE"; then
    echo "PASS: Partial context handling documented"
else
    echo "FAIL: Partial context handling NOT documented"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
