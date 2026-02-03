#!/usr/bin/env bash
# Test for tdd-red-infer-producer SKILL.md
# TDD RED: Run this first to confirm failure before creating the skill

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_FILE="$SCRIPT_DIR/SKILL.md"
ERRORS=()

# Helper to record errors
fail() {
    ERRORS+=("$1")
}

# Test: SKILL.md exists
test_skill_exists() {
    if [[ ! -f "$SKILL_FILE" ]]; then
        fail "SKILL.md does not exist"
        return 1
    fi
    return 0
}

# Test: Has frontmatter with required fields
test_frontmatter() {
    if ! test_skill_exists; then
        return 1
    fi

    # Check frontmatter exists
    if ! head -1 "$SKILL_FILE" | grep -q '^---$'; then
        fail "Missing frontmatter start (---)"
        return 1
    fi

    # Extract frontmatter (between first two ---)
    # Find the line number of the second ---
    end_line=$(awk '/^---$/ {count++; if(count==2) {print NR; exit}}' "$SKILL_FILE")
    frontmatter=$(sed -n "2,$((end_line - 1))p" "$SKILL_FILE")

    # Check name field
    if ! echo "$frontmatter" | grep -q '^name: tdd-red-infer-producer'; then
        fail "Missing or incorrect 'name: tdd-red-infer-producer' in frontmatter"
    fi

    # Check role field
    if ! echo "$frontmatter" | grep -q '^role: producer'; then
        fail "Missing 'role: producer' in frontmatter"
    fi

    # Check phase field
    if ! echo "$frontmatter" | grep -q '^phase: tdd-red'; then
        fail "Missing 'phase: tdd-red' in frontmatter"
    fi

    # Check variant field
    if ! echo "$frontmatter" | grep -q '^variant: infer'; then
        fail "Missing 'variant: infer' in frontmatter"
    fi
}

# Test: References PRODUCER-TEMPLATE pattern (GATHER/SYNTHESIZE/PRODUCE)
test_producer_pattern() {
    if ! test_skill_exists; then
        return 1
    fi

    # Check for GATHER phase reference
    if ! grep -qi 'GATHER' "$SKILL_FILE"; then
        fail "Missing GATHER phase reference"
    fi

    # Check for SYNTHESIZE phase reference
    if ! grep -qi 'SYNTHESIZE' "$SKILL_FILE"; then
        fail "Missing SYNTHESIZE phase reference"
    fi

    # Check for PRODUCE phase reference
    if ! grep -qi 'PRODUCE' "$SKILL_FILE"; then
        fail "Missing PRODUCE phase reference"
    fi

    # Check for PRODUCER-TEMPLATE reference
    if ! grep -q 'PRODUCER-TEMPLATE' "$SKILL_FILE"; then
        fail "Missing reference to PRODUCER-TEMPLATE"
    fi
}

# Test: References YIELD.md
test_yield_reference() {
    if ! test_skill_exists; then
        return 1
    fi

    if ! grep -q 'YIELD.md' "$SKILL_FILE"; then
        fail "Missing reference to YIELD.md"
    fi
}

# Test: Documents analyzing existing implementation to infer tests
test_infer_from_implementation() {
    if ! test_skill_exists; then
        return 1
    fi

    # Should describe analyzing existing code/implementation
    if ! grep -qiE 'existing (code|implementation)|analyze.*implementation|implementation.*analyze' "$SKILL_FILE"; then
        fail "Missing documentation of analyzing existing implementation"
    fi

    # Should mention inferring tests
    if ! grep -qiE 'infer.*test|test.*infer|deduc.*test' "$SKILL_FILE"; then
        fail "Missing documentation of inferring needed tests"
    fi
}

# Test: Documents need-context yield for code exploration
test_need_context_yield() {
    if ! test_skill_exists; then
        return 1
    fi

    if ! grep -qi 'need-context' "$SKILL_FILE"; then
        fail "Missing documentation of need-context yield"
    fi

    # Should mention code exploration context
    if ! grep -qiE 'code|implementation|source|semantic' "$SKILL_FILE"; then
        fail "Missing code exploration context for need-context yield"
    fi
}

# Test: Documents complete yield
test_complete_yield() {
    if ! test_skill_exists; then
        return 1
    fi

    if ! grep -qi 'complete' "$SKILL_FILE"; then
        fail "Missing documentation of complete yield"
    fi

    # Should reference test artifact
    if ! grep -qiE 'test.*file|_test\.go|\.test\.' "$SKILL_FILE"; then
        fail "Missing test file artifact reference"
    fi
}

# Run all tests
main() {
    echo "Testing tdd-red-infer-producer SKILL.md..."
    echo

    test_skill_exists
    test_frontmatter
    test_producer_pattern
    test_yield_reference
    test_infer_from_implementation
    test_need_context_yield
    test_complete_yield

    echo
    if [[ ${#ERRORS[@]} -gt 0 ]]; then
        echo "FAILED: ${#ERRORS[@]} error(s) found:"
        for err in "${ERRORS[@]}"; do
            echo "  - $err"
        done
        exit 1
    else
        echo "PASSED: All tests passed"
        exit 0
    fi
}

main
