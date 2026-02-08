#!/bin/bash
# context-explorer SKILL.md validation tests for TASK-25
# Run: bash skills/context-explorer/SKILL_test.sh

set -e
SKILL_FILE="skills/context-explorer/SKILL.md"

echo "=== context-explorer SKILL.md Validation Tests ==="

# Check file exists
if [[ ! -f "$SKILL_FILE" ]]; then
    echo "FAIL: $SKILL_FILE does not exist"
    exit 1
fi
echo "PASS: File exists"

# TASK-25 Requirement: Frontmatter has name field
if grep -q '^name: context-explorer' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has name: context-explorer"
else
    echo "FAIL: Frontmatter missing or incorrect name field"
    exit 1
fi

# TASK-25 Requirement: Frontmatter has role (standalone or producer-like)
if grep -q '^role:' "$SKILL_FILE"; then
    echo "PASS: Frontmatter has role field"
else
    echo "FAIL: Frontmatter missing role field"
    exit 1
fi

# No legacy YIELD.md references
if grep -q 'YIELD.md' "$SKILL_FILE"; then
    echo "FAIL: Legacy YIELD.md reference still present"
    exit 1
else
    echo "PASS: No legacy YIELD.md references"
fi

# TASK-25 Requirement: Handles file query type
if grep -q 'file' "$SKILL_FILE" && grep -qi 'Read' "$SKILL_FILE"; then
    echo "PASS: Documents file query type with Read tool"
else
    echo "FAIL: Missing file query type documentation"
    exit 1
fi

# TASK-25 Requirement: Handles memory query type
if grep -q 'memory' "$SKILL_FILE"; then
    echo "PASS: Documents memory query type"
else
    echo "FAIL: Missing memory query type documentation"
    exit 1
fi

# TASK-25 Requirement: Handles territory query type
if grep -q 'territory' "$SKILL_FILE"; then
    echo "PASS: Documents territory query type"
else
    echo "FAIL: Missing territory query type documentation"
    exit 1
fi

# TASK-25 Requirement: Handles web query type
if grep -q 'web' "$SKILL_FILE" && grep -qi 'WebFetch' "$SKILL_FILE"; then
    echo "PASS: Documents web query type with WebFetch tool"
else
    echo "FAIL: Missing web query type documentation"
    exit 1
fi

# TASK-25 Requirement: Handles semantic query type
if grep -q 'semantic' "$SKILL_FILE" && grep -qi 'Task' "$SKILL_FILE"; then
    echo "PASS: Documents semantic query type with Task tool"
else
    echo "FAIL: Missing semantic query type documentation"
    exit 1
fi

# TASK-25 Requirement: Can parallelize queries (Task tool)
if grep -qi 'parallel' "$SKILL_FILE" && grep -qi 'Task' "$SKILL_FILE"; then
    echo "PASS: Documents query parallelization via Task tool"
else
    echo "FAIL: Missing parallelization documentation"
    exit 1
fi

# TASK-25 Requirement: Returns aggregated context
if grep -qi 'aggregat' "$SKILL_FILE"; then
    echo "PASS: Documents aggregated context return"
else
    echo "FAIL: Missing aggregated context documentation"
    exit 1
fi

# TASK-25 Requirement: Documents results delivery via messaging
if grep -qiE 'SendMessage|results.*deliver' "$SKILL_FILE" && grep -qi 'results' "$SKILL_FILE"; then
    echo "PASS: Documents results delivery via messaging"
else
    echo "FAIL: Missing results delivery documentation"
    exit 1
fi

# TASK-25 Requirement: Documents input format (queries)
if grep -q 'queries' "$SKILL_FILE"; then
    echo "PASS: Documents input as queries"
else
    echo "FAIL: Missing input format documentation"
    exit 1
fi

# === TASK-29: Auto-memory enrichment tests ===
# Tests verify documentation changes for auto-memory enrichment feature
# Acceptance Criteria:
#   AC-1: "Auto-memory enrichment" section added after "Execute Queries" section (after line 68)
#   AC-2: Policy: when query list lacks explicit memory query, auto-add one from first semantic/file query's topic
#   AC-3: Skip if: request has memory queries, or topic text < 3 words
#   AC-4: Memory failures are non-blocking
#   AC-5: grep -c "Auto-memory" context-explorer/SKILL.md returns at least 1
#
# Traces to: ARCH-063, REQ-008, DES-026
echo ""
echo "=== TASK-29: Auto-memory enrichment tests ==="

# AC-5: grep -c "Auto-memory" returns at least 1
if grep -q "Auto-memory" "$SKILL_FILE" 2>/dev/null; then
    AUTO_MEMORY_COUNT=$(grep -c "Auto-memory" "$SKILL_FILE" 2>/dev/null)
else
    AUTO_MEMORY_COUNT=0
fi
if [[ $AUTO_MEMORY_COUNT -ge 1 ]]; then
    echo "PASS: Auto-memory section exists (count: $AUTO_MEMORY_COUNT)"
else
    echo "FAIL: Auto-memory section not found (expected count >= 1, got: $AUTO_MEMORY_COUNT)"
    exit 1
fi

# AC-1: Auto-memory section appears after line 68 (after Execute Queries section)
if grep -q "Auto-memory" "$SKILL_FILE" 2>/dev/null; then
    AUTO_MEMORY_LINE=$(grep -n "Auto-memory" "$SKILL_FILE" 2>/dev/null | head -1 | cut -d: -f1)
else
    AUTO_MEMORY_LINE=0
fi
if [[ $AUTO_MEMORY_LINE -gt 68 ]]; then
    echo "PASS: Auto-memory section after line 68 (at line $AUTO_MEMORY_LINE)"
else
    echo "FAIL: Auto-memory section not positioned correctly (expected >68, got: $AUTO_MEMORY_LINE)"
    exit 1
fi

# AC-2: Policy documented - auto-add memory query when missing
if grep -A 10 "Auto-memory" "$SKILL_FILE" | grep -qi "auto.*memory query\|memory query.*auto"; then
    echo "PASS: Policy for auto-adding memory query documented"
else
    echo "FAIL: Auto-add memory query policy not documented"
    exit 1
fi

# AC-2: Derives query from first semantic/file query topic
if grep -A 15 "Auto-memory" "$SKILL_FILE" | grep -qi "semantic\|file"; then
    echo "PASS: Derives memory query from semantic/file query"
else
    echo "FAIL: Derivation from semantic/file query not documented"
    exit 1
fi

# AC-3: Skip condition - already has memory queries
if grep -A 20 "Auto-memory" "$SKILL_FILE" | grep -Eqi "skip.*memory|already.*memory|has.*memory.*query"; then
    echo "PASS: Skip condition when memory queries exist documented"
else
    echo "FAIL: Skip condition for existing memory queries not documented"
    exit 1
fi

# AC-3: Skip condition - topic text < 3 words
if grep -A 20 "Auto-memory" "$SKILL_FILE" | grep -Eq "3 word|<3|< 3|\<\s*3"; then
    echo "PASS: Skip condition for short topics (< 3 words) documented"
else
    echo "FAIL: Skip condition for topic length not documented"
    exit 1
fi

# AC-4: Memory failures are non-blocking
if grep -A 20 "Auto-memory" "$SKILL_FILE" | grep -Eqi "non-blocking|non blocking|failure.*not block|not block.*failure"; then
    echo "PASS: Memory failures documented as non-blocking"
else
    echo "FAIL: Non-blocking failure handling not documented"
    exit 1
fi

# Section structure check - should be a proper markdown section
if grep -E "^###? .*Auto-memory" "$SKILL_FILE" > /dev/null; then
    echo "PASS: Auto-memory has proper section header"
else
    echo "FAIL: Auto-memory section header not properly formatted"
    exit 1
fi

# Workflow integration check - should be between Execute Queries and Aggregate Results
EXECUTE_LINE=$(grep -n "### 2. Execute Queries" "$SKILL_FILE" 2>/dev/null | cut -d: -f1)
AGGREGATE_LINE=$(grep -n "### 3. Aggregate Results" "$SKILL_FILE" 2>/dev/null | cut -d: -f1)
if [[ $AUTO_MEMORY_LINE -gt $EXECUTE_LINE ]] && [[ $AUTO_MEMORY_LINE -lt $AGGREGATE_LINE ]]; then
    echo "PASS: Auto-memory section positioned between Execute Queries and Aggregate Results"
else
    echo "FAIL: Auto-memory section not in correct workflow position (should be between lines $EXECUTE_LINE and $AGGREGATE_LINE)"
    exit 1
fi

echo ""
echo "=== All tests passed ==="
