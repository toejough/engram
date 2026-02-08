#!/bin/bash
# pm-interview-producer memory query tests for TASK-24
# Run: bash .claude/skills/pm-interview-producer/SKILL_test_TASK-24.sh

set -e
SKILL_FILE="$HOME/.claude/skills/pm-interview-producer/SKILL.md"

echo "=== TASK-24: pm-interview-producer Memory Query Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# AC-1: GATHER phase includes memory query for prior requirements
if grep -q 'projctl memory query.*prior requirements' "$SKILL_FILE"; then
    echo "PASS: GATHER includes memory query for prior requirements"
else
    echo "FAIL: Missing memory query for prior requirements"
    exit 1
fi

# AC-2: GATHER phase includes memory query for decisions
if grep -q 'projctl memory query.*decisions' "$SKILL_FILE"; then
    echo "PASS: GATHER includes memory query for decisions"
else
    echo "FAIL: Missing memory query for decisions"
    exit 1
fi

# AC-3: GATHER phase includes memory query for known validation failures
if grep -q 'projctl memory query.*known failures.*validation' "$SKILL_FILE" || \
   grep -q 'projctl memory query.*validation failures' "$SKILL_FILE" || \
   grep -q 'projctl memory query.*known.*failures.*requirements' "$SKILL_FILE"; then
    echo "PASS: GATHER includes memory query for known validation failures"
else
    echo "FAIL: Missing memory query for known validation failures"
    exit 1
fi

# AC-4: Memory queries appear in GATHER phase section
# Extract GATHER phase section and verify memory queries are there
if awk '/^### 1\. GATHER Phase/,/^### 2\. SYNTHESIZE Phase/' "$SKILL_FILE" | grep -q 'projctl memory query'; then
    echo "PASS: Memory queries are in GATHER phase section"
else
    echo "FAIL: Memory queries not found in GATHER phase section"
    exit 1
fi

# AC-4: Memory queries run BEFORE interview questions (AskUserQuestion)
# Extract GATHER phase and verify memory query appears before AskUserQuestion mention
GATHER_SECTION=$(awk '/^### 1\. GATHER Phase/,/^### 2\. SYNTHESIZE Phase/' "$SKILL_FILE")
MEMORY_LINE=$(echo "$GATHER_SECTION" | grep -n 'projctl memory query' | head -1 | cut -d: -f1)
INTERVIEW_LINE=$(echo "$GATHER_SECTION" | grep -n 'AskUserQuestion' | head -1 | cut -d: -f1)

if [[ -n "$MEMORY_LINE" && -n "$INTERVIEW_LINE" && $MEMORY_LINE -lt $INTERVIEW_LINE ]]; then
    echo "PASS: Memory queries run BEFORE interview questions"
else
    echo "FAIL: Memory queries must appear before AskUserQuestion in GATHER phase"
    exit 1
fi

# AC-5: At least 3 occurrences of "memory query"
MEMORY_QUERY_COUNT=$(grep -c 'memory query' "$SKILL_FILE" || true)
if [[ $MEMORY_QUERY_COUNT -ge 3 ]]; then
    echo "PASS: At least 3 occurrences of 'memory query' found ($MEMORY_QUERY_COUNT)"
else
    echo "FAIL: Expected at least 3 occurrences of 'memory query', found $MEMORY_QUERY_COUNT"
    exit 1
fi

# Verify traceability comments
if grep -q 'ARCH-055' "$SKILL_FILE" || grep -q 'REQ-008' "$SKILL_FILE"; then
    echo "PASS: Traceability to ARCH-055 or REQ-008 documented"
else
    echo "WARN: Consider adding traceability comments for ARCH-055, REQ-008"
fi

echo ""
echo "=== All TASK-24 tests passed ==="
