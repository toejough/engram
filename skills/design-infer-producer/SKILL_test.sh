#!/usr/bin/env bash
# Test for design-infer-producer SKILL.md
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
    if ! echo "$frontmatter" | grep -q '^name: design-infer-producer'; then
        fail "Missing or incorrect 'name: design-infer-producer' in frontmatter"
    fi

    # Check role field
    if ! echo "$frontmatter" | grep -q '^role: producer'; then
        fail "Missing 'role: producer' in frontmatter"
    fi

    # Check phase field
    if ! echo "$frontmatter" | grep -q '^phase: design'; then
        fail "Missing 'phase: design' in frontmatter"
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

# Test: No legacy YIELD.md references
test_no_yield_reference() {
    if ! test_skill_exists; then
        return 1
    fi

    if grep -q 'YIELD.md' "$SKILL_FILE"; then
        fail "Legacy YIELD.md reference still present"
    fi
}

# Test: Documents need-context yield for UI/UX analysis
test_need_context_yield() {
    if ! test_skill_exists; then
        return 1
    fi

    if ! grep -qi 'need-context' "$SKILL_FILE"; then
        fail "Missing documentation of need-context yield"
    fi

    # Should mention UI/UX analysis context
    if ! grep -qiE 'UI|UX|interface|visual' "$SKILL_FILE"; then
        fail "Missing UI/UX analysis context for need-context yield"
    fi
}

# Test: Documents complete yield with design.md artifact
test_complete_yield() {
    if ! test_skill_exists; then
        return 1
    fi

    if ! grep -qi 'complete' "$SKILL_FILE"; then
        fail "Missing documentation of complete yield"
    fi

    if ! grep -q 'design.md' "$SKILL_FILE"; then
        fail "Missing design.md artifact reference"
    fi
}

# Test: Describes analyzing existing UI/UX to infer design decisions
test_infer_description() {
    if ! test_skill_exists; then
        return 1
    fi

    # Should describe inference from existing UI/UX
    if ! grep -qiE 'infer|deduce|analyze' "$SKILL_FILE"; then
        fail "Missing description of inference process"
    fi

    # Should mention DES-N IDs
    if ! grep -qE 'DES-[0-9N]' "$SKILL_FILE"; then
        fail "Missing DES-N ID format reference"
    fi
}

# Run all tests
main() {
    echo "Testing design-infer-producer SKILL.md..."
    echo

    test_skill_exists
    test_frontmatter
    test_producer_pattern
    test_no_yield_reference
    test_need_context_yield
    test_complete_yield
    test_infer_description

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
