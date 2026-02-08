#!/usr/bin/env bash
# Test: TASK-38 Infer producers memory reads
# Traces to: ARCH-055, REQ-008
#
# These tests verify that infer-variant producer skills query memory
# in their GATHER phase to leverage prior project decisions.

set -euo pipefail

PM_SKILL="${HOME}/.claude/skills/pm-infer-producer/SKILL.md"
DESIGN_SKILL="${HOME}/.claude/skills/design-infer-producer/SKILL.md"
ARCH_SKILL="${HOME}/.claude/skills/arch-infer-producer/SKILL.md"
TDD_RED_SKILL="${HOME}/.claude/skills/tdd-red-infer-producer/SKILL.md"
FAILURES=0

# Color output helpers
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    FAILURES=$((FAILURES + 1))
}

# Test AC-1: pm-infer-producer includes memory query
test_pm_infer_memory_query() {
    echo "Testing: pm-infer-producer includes memory query in GATHER phase"

    if [ ! -f "${PM_SKILL}" ]; then
        fail "pm-infer-producer SKILL.md not found at ${PM_SKILL}"
        return
    fi

    if grep -q "memory query" "${PM_SKILL}"; then
        count=$(grep -c "memory query" "${PM_SKILL}")
        pass "Found $count occurrences of 'memory query' in pm-infer-producer (expected >= 1)"
    else
        fail "Found 0 occurrences of 'memory query' in pm-infer-producer (expected >= 1)"
    fi
}

# Test AC-2: design-infer-producer includes memory query
test_design_infer_memory_query() {
    echo "Testing: design-infer-producer includes memory query in GATHER phase"

    if [ ! -f "${DESIGN_SKILL}" ]; then
        fail "design-infer-producer SKILL.md not found at ${DESIGN_SKILL}"
        return
    fi

    if grep -q "memory query" "${DESIGN_SKILL}"; then
        count=$(grep -c "memory query" "${DESIGN_SKILL}")
        pass "Found $count occurrences of 'memory query' in design-infer-producer (expected >= 1)"
    else
        fail "Found 0 occurrences of 'memory query' in design-infer-producer (expected >= 1)"
    fi
}

# Test AC-3: arch-infer-producer includes memory query
test_arch_infer_memory_query() {
    echo "Testing: arch-infer-producer includes memory query in GATHER phase"

    if [ ! -f "${ARCH_SKILL}" ]; then
        fail "arch-infer-producer SKILL.md not found at ${ARCH_SKILL}"
        return
    fi

    if grep -q "memory query" "${ARCH_SKILL}"; then
        count=$(grep -c "memory query" "${ARCH_SKILL}")
        pass "Found $count occurrences of 'memory query' in arch-infer-producer (expected >= 1)"
    else
        fail "Found 0 occurrences of 'memory query' in arch-infer-producer (expected >= 1)"
    fi
}

# Test AC-4: memory query appears in GATHER phase sections
test_gather_phase_placement() {
    echo "Testing: memory query appears in GATHER phase sections"

    local all_valid=true

    # Check pm-infer-producer
    if grep -A 50 "^### 1. GATHER" "${PM_SKILL}" | grep -q "memory query"; then
        pass "pm-infer-producer: memory query in GATHER phase"
    else
        fail "pm-infer-producer: memory query NOT in GATHER phase"
        all_valid=false
    fi

    # Check design-infer-producer
    if grep -A 50 "^### 1. GATHER" "${DESIGN_SKILL}" | grep -q "memory query"; then
        pass "design-infer-producer: memory query in GATHER phase"
    else
        fail "design-infer-producer: memory query NOT in GATHER phase"
        all_valid=false
    fi

    # Check arch-infer-producer
    if grep -A 50 "^### 1. GATHER" "${ARCH_SKILL}" | grep -q "memory query"; then
        pass "arch-infer-producer: memory query in GATHER phase"
    else
        fail "arch-infer-producer: memory query NOT in GATHER phase"
        all_valid=false
    fi
}

# Test AC-5: projctl command format is correct
test_projctl_command_format() {
    echo "Testing: projctl memory query command format is correct"

    local all_valid=true

    # Check for "projctl memory query" format in pm-infer-producer
    if grep "projctl memory query" "${PM_SKILL}" &>/dev/null; then
        pass "pm-infer-producer: uses 'projctl memory query' format"
    else
        fail "pm-infer-producer: does NOT use 'projctl memory query' format"
        all_valid=false
    fi

    # Check for "projctl memory query" format in design-infer-producer
    if grep "projctl memory query" "${DESIGN_SKILL}" &>/dev/null; then
        pass "design-infer-producer: uses 'projctl memory query' format"
    else
        fail "design-infer-producer: does NOT use 'projctl memory query' format"
        all_valid=false
    fi

    # Check for "projctl memory query" format in arch-infer-producer
    if grep "projctl memory query" "${ARCH_SKILL}" &>/dev/null; then
        pass "arch-infer-producer: uses 'projctl memory query' format"
    else
        fail "arch-infer-producer: does NOT use 'projctl memory query' format"
        all_valid=false
    fi
}

# Test AC-6: tdd-red-infer-producer also includes memory query (bonus coverage)
test_tdd_red_infer_memory_query() {
    echo "Testing: tdd-red-infer-producer includes memory query in GATHER phase"

    if [ ! -f "${TDD_RED_SKILL}" ]; then
        fail "tdd-red-infer-producer SKILL.md not found at ${TDD_RED_SKILL}"
        return
    fi

    if grep -q "memory query" "${TDD_RED_SKILL}"; then
        count=$(grep -c "memory query" "${TDD_RED_SKILL}")
        pass "Found $count occurrences of 'memory query' in tdd-red-infer-producer (expected >= 1)"
    else
        fail "Found 0 occurrences of 'memory query' in tdd-red-infer-producer (expected >= 1)"
    fi
}

# Run all tests
echo "========================================"
echo "TASK-38 Infer Producers Memory Test Suite"
echo "========================================"
echo ""

test_pm_infer_memory_query
test_design_infer_memory_query
test_arch_infer_memory_query
test_gather_phase_placement
test_projctl_command_format
test_tdd_red_infer_memory_query

echo ""
echo "========================================"
if [ $FAILURES -eq 0 ]; then
    echo -e "${GREEN}All tests PASSED${NC}"
    exit 0
else
    echo -e "${RED}$FAILURES test(s) FAILED${NC}"
    exit 1
fi
