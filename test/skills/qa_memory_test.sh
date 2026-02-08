#!/usr/bin/env bash
# Test: TASK-28 QA memory reads and writes
# Traces to: ARCH-056, REQ-010
#
# These tests verify that the QA skill:
# 1. Queries memory for known failures in LOAD phase
# 2. Persists new failures to memory in RETURN phase
# 3. Only persists error-severity findings (not warnings or approvals)

set -euo pipefail

SKILL_FILE="${HOME}/.claude/skills/qa/SKILL.md"
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

# Test AC-1: LOAD phase includes memory query for known failures
test_load_phase_memory_query() {
    echo "Testing: LOAD phase includes memory query for known failures"

    # Check that LOAD phase section contains projctl memory query
    if grep -A 30 "^### 1\. LOAD Phase" "${SKILL_FILE}" | grep -q "projctl memory query"; then
        pass "LOAD phase contains memory query command"
    else
        fail "LOAD phase does NOT contain memory query command"
    fi

    # Check for pattern matching "known failures in <artifact-type> validation"
    if grep -A 30 "^### 1\. LOAD Phase" "${SKILL_FILE}" | grep -q "known failures"; then
        pass "LOAD phase mentions 'known failures' pattern"
    else
        fail "LOAD phase does NOT mention 'known failures' pattern"
    fi
}

# Test AC-2: RETURN phase includes memory learn on improvement-request
test_return_phase_memory_learn() {
    echo "Testing: RETURN phase includes memory learn on improvement-request"

    # Check that RETURN phase section contains projctl memory learn
    if grep -A 50 "^### 3\. RETURN Phase" "${SKILL_FILE}" | grep -q "projctl memory learn"; then
        pass "RETURN phase contains memory learn command"
    else
        fail "RETURN phase does NOT contain memory learn command"
    fi

    # Check for improvement-request context
    if grep -A 50 "^### 3\. RETURN Phase" "${SKILL_FILE}" | grep -B 5 -A 5 "memory learn" | grep -q "improvement-request"; then
        pass "Memory learn is documented in improvement-request context"
    else
        fail "Memory learn is NOT linked to improvement-request verdict"
    fi
}

# Test AC-3: Only error-severity findings are persisted
test_error_severity_only() {
    echo "Testing: Only error-severity findings are persisted"

    # Check for documentation about severity filtering
    if grep -q "error-severity" "${SKILL_FILE}"; then
        pass "Documentation mentions error-severity filtering"
    else
        fail "Documentation does NOT mention error-severity filtering"
    fi

    # Check that warnings are NOT persisted
    if grep -A 50 "^### 3\. RETURN Phase" "${SKILL_FILE}" | grep -B 3 -A 3 "memory learn" | grep -q -i "error"; then
        pass "Memory learn documentation specifies error-severity condition"
    else
        fail "Memory learn does NOT specify error-severity condition"
    fi
}

# Test AC-4: Message format includes QA failure description
test_message_format() {
    echo "Testing: Memory learn message format includes 'QA failure'"

    # Check for "QA failure" pattern in message format
    if grep -B 5 -A 5 "memory learn" "${SKILL_FILE}" | grep -q "QA failure"; then
        pass "Memory learn message format includes 'QA failure' prefix"
    else
        fail "Memory learn message format does NOT include 'QA failure' prefix"
    fi
}

# Test AC-5: Check ID format in persisted messages
test_message_includes_check_id() {
    echo "Testing: Persisted messages include check-id"

    # Check for check-id or CHECK- pattern in memory learn examples
    if grep -B 5 -A 5 "memory learn" "${SKILL_FILE}" | grep -q -E "(check-id|CHECK-[0-9]+)"; then
        pass "Memory learn includes check-id in message format"
    else
        fail "Memory learn does NOT include check-id in message format"
    fi
}

# Test AC-6: Issue/project ID tagging
test_project_id_tagging() {
    echo "Testing: Memory learn includes project/issue ID tagging"

    # Check for -p flag or project tagging
    if grep -B 5 -A 5 "memory learn" "${SKILL_FILE}" | grep -q "\-p"; then
        pass "Memory learn includes -p flag for project/issue tagging"
    else
        fail "Memory learn does NOT include -p flag for project/issue tagging"
    fi
}

# Test AC-7: grep count for "memory query" returns at least 1
test_grep_count_memory_query() {
    echo "Testing: grep -c 'memory query' returns at least 1"

    count=$(grep -o "memory query" "${SKILL_FILE}" | wc -l | tr -d ' ')
    if [ "$count" -ge 1 ]; then
        pass "Found $count occurrences of 'memory query' (expected >= 1)"
    else
        fail "Found $count occurrences of 'memory query' (expected >= 1)"
    fi
}

# Test AC-8: grep count for "memory learn" returns at least 1
test_grep_count_memory_learn() {
    echo "Testing: grep -c 'memory learn' returns at least 1"

    count=$(grep -o "memory learn" "${SKILL_FILE}" | wc -l | tr -d ' ')
    if [ "$count" -ge 1 ]; then
        pass "Found $count occurrences of 'memory learn' (expected >= 1)"
    else
        fail "Found $count occurrences of 'memory learn' (expected >= 1)"
    fi
}

# Test AC-9: Documentation explains verification backstop role
test_verification_backstop_role() {
    echo "Testing: Documentation explains verification backstop role"

    # QA reads should verify producers addressed known pitfalls
    if grep -B 5 -A 5 "memory query" "${SKILL_FILE}" | grep -q -E "(verify|check|backstop)"; then
        pass "Memory query role explains verification backstop"
    else
        fail "Memory query role does NOT explain verification backstop"
    fi
}

# Test AC-10: Escalate-phase also triggers memory learn
test_escalate_phase_memory_learn() {
    echo "Testing: escalate-phase verdict also triggers memory learn"

    # Check that escalate-phase is mentioned alongside improvement-request for memory learn
    if grep -B 10 -A 10 "memory learn" "${SKILL_FILE}" | grep -q "escalate-phase"; then
        pass "Memory learn applies to escalate-phase verdicts"
    else
        fail "Memory learn does NOT apply to escalate-phase verdicts"
    fi
}

# Run all tests
echo "================================"
echo "TASK-28 QA Memory Test Suite"
echo "Testing: ${SKILL_FILE}"
echo "================================"
echo ""

test_load_phase_memory_query
test_return_phase_memory_learn
test_error_severity_only
test_message_format
test_message_includes_check_id
test_project_id_tagging
test_grep_count_memory_query
test_grep_count_memory_learn
test_verification_backstop_role
test_escalate_phase_memory_learn

echo ""
echo "================================"
if [ $FAILURES -eq 0 ]; then
    echo -e "${GREEN}All tests PASSED${NC}"
    exit 0
else
    echo -e "${RED}$FAILURES test(s) FAILED${NC}"
    exit 1
fi
