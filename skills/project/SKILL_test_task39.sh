#!/usr/bin/env bash
# Test: TASK-39 — Doc and summary producers memory reads
# Traces to: ARCH-055, REQ-008

set -uo pipefail

# Test configuration
readonly DOC_PRODUCER_SKILL="${HOME}/.claude/skills/doc-producer/SKILL.md"
readonly SUMMARY_PRODUCER_SKILL="${HOME}/.claude/skills/summary-producer/SKILL.md"

test_count=0
failed_tests=0

run_test() {
    local test_name="$1"
    local test_command="$2"

    ((test_count++))

    if eval "$test_command" >/dev/null 2>&1; then
        echo "✓ $test_name"
    else
        echo "✗ $test_name"
        ((failed_tests++))
    fi
}

echo "Running TASK-39 acceptance tests..."
echo

# AC1: doc-producer GATHER includes memory query for documentation patterns
run_test "doc-producer GATHER includes 'projctl memory query' for documentation patterns" \
    "grep -q 'projctl memory query.*documentation.*patterns' '$DOC_PRODUCER_SKILL'"

# AC2: summary-producer GATHER includes memory query for summary patterns
run_test "summary-producer GATHER includes 'projctl memory query' for summary patterns" \
    "grep -q 'projctl memory query.*summary.*patterns' '$SUMMARY_PRODUCER_SKILL'"

# AC3: doc-producer SKILL.md contains at least 1 occurrence of "memory query"
run_test "doc-producer SKILL.md contains 'memory query' at least once" \
    "[ \$(grep -c 'memory query' '$DOC_PRODUCER_SKILL') -ge 1 ]"

# AC4: summary-producer SKILL.md contains at least 1 occurrence of "memory query"
run_test "summary-producer SKILL.md contains 'memory query' at least once" \
    "[ \$(grep -c 'memory query' '$SUMMARY_PRODUCER_SKILL') -ge 1 ]"

# Additional verification: Check that memory queries appear in GATHER sections
run_test "doc-producer memory query appears in GATHER phase section" \
    "awk '/^### GATHER/,/^### SYNTHESIZE/ {if (/memory query/) found=1} END {exit !found}' '$DOC_PRODUCER_SKILL'"

run_test "summary-producer memory query appears in GATHER phase section" \
    "awk '/^### GATHER/,/^### SYNTHESIZE/ {if (/memory query/) found=1} END {exit !found}' '$SUMMARY_PRODUCER_SKILL'"

# Semantic check: Ensure graceful degradation is documented
run_test "doc-producer documents non-blocking memory queries (graceful degradation)" \
    "grep -qi 'non-blocking\|graceful\|optional\|if.*unavailable' '$DOC_PRODUCER_SKILL'"

run_test "summary-producer documents non-blocking memory queries (graceful degradation)" \
    "grep -qi 'non-blocking\|graceful\|optional\|if.*unavailable' '$SUMMARY_PRODUCER_SKILL'"

echo
echo "=========================================="
echo "Test Results: $((test_count - failed_tests))/$test_count passed"
echo "=========================================="

if [ $failed_tests -gt 0 ]; then
    echo "FAIL: $failed_tests test(s) failed"
    exit 1
else
    echo "PASS: All tests passed"
    exit 0
fi
