#!/usr/bin/env bash
# Test: TASK-36 - TDD red producers memory reads
# Traces to: TASK-36 AC-1 through AC-5
#
# These tests verify that memory query commands are present in the
# tdd-red-producer and tdd-red-infer-producer SKILL.md files.
# Expected to FAIL in red phase (memory queries not yet added).

set -e

SKILL_DIR="$HOME/.claude/skills"
TDD_RED="$SKILL_DIR/tdd-red-producer/SKILL.md"
TDD_RED_INFER="$SKILL_DIR/tdd-red-infer-producer/SKILL.md"

echo "Running TASK-36 acceptance criteria tests..."
echo

# AC-1: tdd-red-producer GATHER includes memory query for test patterns
echo "Test 1: tdd-red-producer GATHER includes 'projctl memory query' for test patterns"
# Extract GATHER section and check for test patterns query
if sed -n '/^### GATHER$/,/^### SYNTHESIZE$/p' "$TDD_RED" | grep -q 'projctl memory query.*test patterns'; then
    echo "  ✓ PASS: Found test patterns memory query in GATHER"
else
    echo "  ✗ FAIL: Missing test patterns memory query in GATHER section"
    exit 1
fi

# AC-2: tdd-red-producer GATHER includes memory query for known test failures
echo "Test 2: tdd-red-producer GATHER includes 'projctl memory query' for known failures"
# Extract GATHER section and check for known failures query
if sed -n '/^### GATHER$/,/^### SYNTHESIZE$/p' "$TDD_RED" | grep -q 'projctl memory query.*known.*failures'; then
    echo "  ✓ PASS: Found known failures memory query in GATHER"
else
    echo "  ✗ FAIL: Missing known failures memory query in GATHER section"
    exit 1
fi

# AC-3: tdd-red-infer-producer GATHER includes memory query for test inference patterns
echo "Test 3: tdd-red-infer-producer GATHER includes 'projctl memory query' for test inference"
# Extract GATHER section and check for inference patterns query
if sed -n '/^### 1\. GATHER$/,/^### 2\. SYNTHESIZE$/p' "$TDD_RED_INFER" | grep -q 'projctl memory query.*test inference'; then
    echo "  ✓ PASS: Found test inference patterns memory query in GATHER"
else
    echo "  ✗ FAIL: Missing test inference patterns memory query in GATHER section"
    exit 1
fi

# AC-4: tdd-red-producer GATHER has at least 2 memory query commands
echo "Test 4: tdd-red-producer GATHER has at least 2 'projctl memory query' commands"
count=$(sed -n '/^### GATHER$/,/^### SYNTHESIZE$/p' "$TDD_RED" | grep -c "projctl memory query" || true)
if [ "$count" -ge 2 ]; then
    echo "  ✓ PASS: Found $count memory query commands in GATHER (>= 2)"
else
    echo "  ✗ FAIL: Found only $count memory query commands in GATHER (expected >= 2)"
    exit 1
fi

# AC-5: tdd-red-infer-producer GATHER has at least 1 memory query command
echo "Test 5: tdd-red-infer-producer GATHER has at least 1 'projctl memory query' command"
count=$(sed -n '/^### 1\. GATHER$/,/^### 2\. SYNTHESIZE$/p' "$TDD_RED_INFER" | grep -c "projctl memory query" || true)
if [ "$count" -ge 1 ]; then
    echo "  ✓ PASS: Found $count memory query command in GATHER (>= 1)"
else
    echo "  ✗ FAIL: Found only $count memory query command in GATHER (expected >= 1)"
    exit 1
fi

echo
echo "All tests passed!"
