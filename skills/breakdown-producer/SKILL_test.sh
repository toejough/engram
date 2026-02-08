#!/bin/bash
# Tests for breakdown-producer SKILL.md

set -e

SKILL_FILE="skills/breakdown-producer/SKILL.md"
REPO_ROOT="$(git rev-parse --show-toplevel)"

cd "$REPO_ROOT"

echo "Running breakdown-producer SKILL.md tests..."

# Test 1: SKILL.md exists
if [[ ! -f "$SKILL_FILE" ]]; then
  echo "FAIL: $SKILL_FILE does not exist"
  exit 1
fi
echo "PASS: SKILL.md exists"

# Test 2: SYNTHESIZE phase mentions simplicity assessment
if ! grep -q "simplicity" "$SKILL_FILE"; then
  echo "FAIL: SYNTHESIZE phase does not mention simplicity assessment"
  exit 1
fi
echo "PASS: SYNTHESIZE phase mentions simplicity"

# Test 3: SYNTHESIZE phase has explicit simplicity check step
if ! grep -A 10 "## SYNTHESIZE Phase" "$SKILL_FILE" | grep -q "simplicity"; then
  echo "FAIL: SYNTHESIZE phase does not include simplicity assessment step"
  exit 1
fi
echo "PASS: SYNTHESIZE phase includes simplicity assessment step"

# Test 4: Simplicity is holistic (SYNTHESIZE), not per-task (no per-task field in template)
if grep -A 50 "## Task Format" "$SKILL_FILE" | grep -q "Simplicity Assessment"; then
  echo "FAIL: Task format should NOT include per-task Simplicity Assessment (moved to SYNTHESIZE)"
  exit 1
fi
echo "PASS: Task format does not include per-task Simplicity Assessment"

# Test 5: No per-task simplicity contract check (CHECK-012 removed)
if grep -q "CHECK-012" "$SKILL_FILE"; then
  echo "FAIL: CHECK-012 (per-task simplicity) should be removed"
  exit 1
fi
echo "PASS: CHECK-012 removed from contract"

# Test 6: Simplicity rationale is in tasks.md header (PRODUCE phase output)
if ! grep -q "Simplicity rationale" "$SKILL_FILE"; then
  echo "FAIL: PRODUCE phase should include simplicity rationale in tasks.md header"
  exit 1
fi
echo "PASS: PRODUCE phase includes simplicity rationale in header"

# Test 7: SYNTHESIZE phase asks the right simplicity question
if ! grep -q "simpler approach" "$SKILL_FILE"; then
  echo "FAIL: SYNTHESIZE phase does not ask about simpler approaches"
  exit 1
fi
echo "PASS: SYNTHESIZE phase asks about simpler approaches"

# Test 8: SYNTHESIZE simplicity step documents alternatives considered
if ! grep -A 10 "Assess simplicity" "$SKILL_FILE" | grep -q "alternatives considered"; then
  echo "FAIL: SYNTHESIZE simplicity step does not mention documenting alternatives"
  exit 1
fi
echo "PASS: SYNTHESIZE simplicity step documents alternatives"

# TASK-32: Memory query integration tests

echo ""
echo "Testing TASK-32: Memory query integration..."

# AC-1: GATHER phase includes memory query for task decomposition patterns
if ! grep -q 'projctl memory query.*task decomposition' "$SKILL_FILE"; then
  echo "FAIL: GATHER phase missing: projctl memory query for task decomposition patterns"
  exit 1
fi
echo "PASS: GATHER phase includes decomposition pattern query"

# AC-2: GATHER phase includes memory query for known failures
if ! grep -q 'projctl memory query.*known failures.*task decomposition' "$SKILL_FILE" && \
   ! grep -q 'projctl memory query.*known failures.*breakdown' "$SKILL_FILE"; then
  echo "FAIL: GATHER phase missing: projctl memory query for known failures in task decomposition"
  exit 1
fi
echo "PASS: GATHER phase includes known failures query"

# AC-3: Graceful degradation documented
if ! grep -iq 'graceful.*degradation' "$SKILL_FILE" && \
   ! grep -iq 'non-blocking' "$SKILL_FILE" && \
   ! grep -iq 'memory unavailable' "$SKILL_FILE" && \
   ! grep -iq 'continue without.*memory' "$SKILL_FILE"; then
  echo "FAIL: Missing graceful degradation documentation for memory queries"
  exit 1
fi
echo "PASS: Graceful degradation documented"

# AC-4: At least 2 memory query occurrences
QUERY_COUNT=$(grep -c 'projctl memory query' "$SKILL_FILE" || true)
if [ "$QUERY_COUNT" -lt 2 ]; then
  echo "FAIL: Found only $QUERY_COUNT memory query occurrences, expected >= 2"
  exit 1
fi
echo "PASS: Found $QUERY_COUNT memory query occurrences (>= 2)"

echo ""
echo "All tests passed!"
