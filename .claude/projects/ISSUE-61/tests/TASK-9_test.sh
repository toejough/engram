#!/usr/bin/env bash
# TASK-9 Tests: Validation log structure and content

set -euo pipefail

VALIDATION_LOG=".claude/projects/ISSUE-61/validation-log.md"

# Test 1: Validation log file exists
test_validation_log_exists() {
    if [[ ! -f "$VALIDATION_LOG" ]]; then
        echo "FAIL: Validation log file does not exist at $VALIDATION_LOG"
        return 1
    fi
    echo "PASS: Validation log file exists"
}

# Test 2: At least 2 validation sessions documented (AC-1)
test_minimum_validation_sessions() {
    local count
    count=$(grep -c "^## Validation Session" "$VALIDATION_LOG" || true)
    if [[ $count -lt 2 ]]; then
        echo "FAIL: Expected at least 2 validation sessions, found $count"
        return 1
    fi
    echo "PASS: Found $count validation sessions (minimum 2 required)"
}

# Test 3: Different issue types validated (AC-1)
test_different_issue_types() {
    local content
    content=$(cat "$VALIDATION_LOG")

    # Check for at least two different issue type keywords
    local has_feature=false
    local has_refactoring=false

    if echo "$content" | grep -q -i "new feature\|feature"; then
        has_feature=true
    fi
    if echo "$content" | grep -q -i "refactor"; then
        has_refactoring=true
    fi

    if [[ "$has_feature" == false ]] || [[ "$has_refactoring" == false ]]; then
        echo "FAIL: Expected different issue types (feature and refactoring), found feature=$has_feature, refactoring=$has_refactoring"
        return 1
    fi
    echo "PASS: Found different issue types documented"
}

# Test 4: Required metadata fields present (AC-2)
test_required_metadata_fields() {
    local content
    content=$(cat "$VALIDATION_LOG")

    local missing_fields=()

    if ! echo "$content" | grep -q "Issue ID:"; then
        missing_fields+=("Issue ID")
    fi
    if ! echo "$content" | grep -q "Gap Size:"; then
        missing_fields+=("Gap Size")
    fi
    if ! echo "$content" | grep -q "Question Count:"; then
        missing_fields+=("Question Count")
    fi
    if ! echo "$content" | grep -q "User Feedback:"; then
        missing_fields+=("User Feedback")
    fi

    if [[ ${#missing_fields[@]} -gt 0 ]]; then
        echo "FAIL: Missing required metadata fields: ${missing_fields[*]}"
        return 1
    fi
    echo "PASS: All required metadata fields present"
}

# Test 5: Context gathering verification (AC-3)
test_context_gathering_verification() {
    local content
    content=$(cat "$VALIDATION_LOG")

    if ! echo "$content" | grep -q -i "context gathering"; then
        echo "FAIL: No context gathering verification documented"
        return 1
    fi

    if ! echo "$content" | grep -q -i "error\|success\|complet"; then
        echo "FAIL: Context gathering status not documented"
        return 1
    fi

    echo "PASS: Context gathering verification documented"
}

# Test 6: Depth tier matching (AC-4)
test_depth_tier_matching() {
    local content
    content=$(cat "$VALIDATION_LOG")

    if ! echo "$content" | grep -q -i "depth\|tier\|expected"; then
        echo "FAIL: No depth tier matching verification documented"
        return 1
    fi

    echo "PASS: Depth tier matching verification documented"
}

# Test 7: Question appropriateness assessment (AC-5)
test_question_appropriateness() {
    local content
    content=$(cat "$VALIDATION_LOG")

    if ! echo "$content" | grep -q -i "question.*appropriate\|redundant\|sparse"; then
        echo "FAIL: No question appropriateness assessment documented"
        return 1
    fi

    echo "PASS: Question appropriateness assessment documented"
}

# Test 8: Yield metadata debugging (AC-6)
test_yield_metadata_debugging() {
    local content
    content=$(cat "$VALIDATION_LOG")

    if ! echo "$content" | grep -q -i "yield.*metadata\|debug"; then
        echo "FAIL: No yield metadata debugging verification documented"
        return 1
    fi

    echo "PASS: Yield metadata debugging verification documented"
}

# Test 9: Adjustments documented (AC-7)
test_adjustments_documented() {
    local content
    content=$(cat "$VALIDATION_LOG")

    if ! echo "$content" | grep -q -i "adjustment\|weight\|question"; then
        echo "FAIL: No adjustments or recommendations documented"
        return 1
    fi

    echo "PASS: Adjustments or recommendations documented"
}

# Test 10: Validation summary section exists (AC-8)
test_validation_summary_section() {
    if ! grep -q "^## Summary" "$VALIDATION_LOG" && ! grep -q "^## Validation Summary" "$VALIDATION_LOG"; then
        echo "FAIL: No summary section found in validation log"
        return 1
    fi
    echo "PASS: Validation summary section exists"
}

# Run all tests
main() {
    local failed=0
    local passed=0

    echo "Running TASK-9 validation log tests..."
    echo

    for test_func in \
        test_validation_log_exists \
        test_minimum_validation_sessions \
        test_different_issue_types \
        test_required_metadata_fields \
        test_context_gathering_verification \
        test_depth_tier_matching \
        test_question_appropriateness \
        test_yield_metadata_debugging \
        test_adjustments_documented \
        test_validation_summary_section
    do
        echo "Running: $test_func" >&2
        if $test_func; then
            ((passed++)) || true
        else
            ((failed++)) || true
        fi
        echo
    done

    echo "=========================================="
    echo "TASK-9 Tests: $passed passed, $failed failed"
    echo "=========================================="

    if [[ $failed -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
